package domain

type DirectCandidate struct {
	Type       string `json:"type"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
	Protocol   string `json:"protocol"`
	StunServer string `json:"stunServer,omitempty"`
	Priority   int    `json:"priority,omitempty"`
}

type DirectCandidatesInput struct {
	UDPListenPort int               `json:"udpListenPort"`
	NATType       string            `json:"natType"`
	Candidates    []DirectCandidate `json:"candidates"`
	ObservedAt    string            `json:"observedAt"`
}

type DirectCandidatesResult struct {
	NodeID         string `json:"nodeId"`
	CandidateCount int    `json:"candidateCount"`
	UpdatedAt      string `json:"updatedAt"`
}

type DirectLinkPlan struct {
	LinkID             string            `json:"linkId"`
	PeerNodeID         string            `json:"peerNodeId"`
	Role               string            `json:"role"`
	PreferredTransport string            `json:"preferredTransport"`
	FallbackTransport  string            `json:"fallbackTransport"`
	PunchToken         string            `json:"punchToken"`
	ExpiresAt          string            `json:"expiresAt"`
	PeerCandidates     []DirectCandidate `json:"peerCandidates"`
}

type DirectLinkPlanResult struct {
	NodeID string           `json:"nodeId"`
	Links  []DirectLinkPlan `json:"links"`
}

type DirectStatusInput struct {
	LinkID            string          `json:"linkId"`
	PeerNodeID        string          `json:"peerNodeId"`
	TransportType     string          `json:"transportType"`
	Status            string          `json:"status"`
	SelectedCandidate DirectCandidate `json:"selectedCandidate"`
	RTTMs             int             `json:"rttMs"`
	LastProbeAt       string          `json:"lastProbeAt"`
	FallbackReason    string          `json:"fallbackReason"`
}

type DirectStatusResult struct {
	LinkID     string `json:"linkId"`
	PeerNodeID string `json:"peerNodeId"`
	Status     string `json:"status"`
	UpdatedAt  string `json:"updatedAt"`
}
