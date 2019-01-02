package BlockchainC2

// Constants for Agent -> Server commands
const (
	AgentToServerPing            = 0
	AgentToServerJoin            = 1
	AgentToServerData            = 2
	AgentToServerInfo            = 3
	AgentToServerFileDownload    = 4
	AgentToServerCryptoHandshake = 9
)

// Constants for Server -> Agent commands
const (
	ServerToAgentCryptoHandshake = 9900
	ServerToAgentExecuteCommand  = 9901
	ServerToAgentSleep           = 9902
	ServerToAgentExit            = 9903
	ServerToAgentInfo            = 9904
	ServerToAgentFileDownload    = 9905
)

// Agent status
const (
	Handshake = 0
	Running   = 1
	Exited    = 2
)
