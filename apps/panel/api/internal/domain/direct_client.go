package domain

type ClientDirectSessionInput struct {
	AccessPathID string `json:"accessPathId"`
	ClientID     string `json:"clientId"`
	TargetHost   string `json:"targetHost"`
	TargetPort   int    `json:"targetPort"`
}

type ClientDirectSession struct {
	SessionID      string            `json:"sessionId"`
	AccessPathID   string            `json:"accessPathId"`
	TargetNodeID   string            `json:"targetNodeId"`
	TargetHost     string            `json:"targetHost"`
	TargetPort     int               `json:"targetPort"`
	RelayEntryHost string            `json:"relayEntryHost"`
	RelayEntryPort int               `json:"relayEntryPort"`
	PunchToken     string            `json:"punchToken"`
	ExpiresAt      string            `json:"expiresAt"`
	NodeCandidates []DirectCandidate `json:"nodeCandidates"`
}

type ClientDirectSessionValidateInput struct {
	SessionID  string `json:"sessionId"`
	PunchToken string `json:"punchToken"`
	TargetHost string `json:"targetHost"`
	TargetPort int    `json:"targetPort"`
}

type ClientDirectSessionValidateResult struct {
	Valid      bool   `json:"valid"`
	TargetHost string `json:"targetHost"`
	TargetPort int    `json:"targetPort"`
}
