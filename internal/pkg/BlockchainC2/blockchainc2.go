package BlockchainC2

import (
	"fmt"
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Maximum message length to send to a contract
const MAX_MSG_LEN = 16000

// BlockchainC2 arguments passed from contract event handler
type BlockchainC2 struct {
	AgentID string `json:"AgentID"`
	MsgID   int    `json:"MsgID"`
	Data    string `json:"Data"`
}

// DeployContract deploys a new smart contract to the blockchain to support BlockchainC2
// key - keychain in JSON format of account to deploy contact with
// password - password used to decrypt the keychain
// endpoint - JSON-RPC endpoint
func DeployContract(key string, password string, endpoint string) string {

	fmt.Println("[*] Connecting to WS JSON-RPC endpoint")
	conn, err := ethclient.Dial(endpoint)
	if err != nil {
		log.Fatal("[!] Whoops something went wrong!", err)
		return ""
	}

	auth, err := bind.NewTransactor(strings.NewReader(key), password)
	if err != nil {
		log.Fatalf("[!] Failed to create authorized transactor: %v", err)
		return ""
	}

	contractAddr, _, _, err := DeployEventC2(auth, conn)
	if err != nil {
		log.Fatalf("[!] Failed to deploy contract to blockchain: ", err)
		return ""
	}
	return contractAddr.Hex()
}
