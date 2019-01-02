package BlockchainC2

// AgentInfoMsg is used to transfer agent information to the server
type AgentInfoMsg struct {
	Username string `json:"username"`
	Hostname string `json:"hostname"`
}
