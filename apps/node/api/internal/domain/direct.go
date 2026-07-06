package domain

const (
	TransportTypeRelayWSParent      = "relay_ws_parent"
	TransportTypeDirectUDPCandidate = "direct_udp_candidate"
	TransportTypeDirectQUIC         = "direct_quic"
	TransportTypeDirectRelay        = "direct_relay"
	TransportDirectionPeer          = "peer"
	CandidateTypeHost               = "host"
	CandidateTypeServerReflexive    = "srflx"
	CandidateProtocolUDP            = "udp"
	NATTypeUnknown                  = "unknown"
	NATTypeEndpointIndependent      = "endpoint_independent_mapping"
	NATTypeAddressDependent         = "address_dependent_mapping"
	NATTypeBlocked                  = "blocked"
	DirectStatusProbing             = "probing"
	DirectStatusConnected           = "connected"
	DirectStatusFailed              = "failed"
	DirectStatusFallback            = "fallback"
)

type DirectCandidate struct {
	Type       string `json:"type"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
	Protocol   string `json:"protocol"`
	STUNServer string `json:"stunServer,omitempty"`
	Priority   int    `json:"priority,omitempty"`
}

type DirectNodeIdentity struct {
	NodeID                       string `json:"nodeId"`
	ServerName                   string `json:"serverName"`
	CertificateFingerprintSHA256 string `json:"certificateFingerprintSha256"`
	TrustMaterial                string `json:"trustMaterial"`
}

type ReportDirectCandidatesInput struct {
	UDPListenPort  int                `json:"udpListenPort"`
	NATType        string             `json:"natType"`
	Candidates     []DirectCandidate  `json:"candidates"`
	ObservedAt     string             `json:"observedAt"`
	DirectIdentity DirectNodeIdentity `json:"directIdentity"`
}

type ReportDirectCandidatesResult struct {
	NodeID         string `json:"nodeId"`
	CandidateCount int    `json:"candidateCount"`
	UpdatedAt      string `json:"updatedAt"`
}

type DirectLinkPlan struct {
	NodeID string           `json:"nodeId"`
	Links  []DirectLinkItem `json:"links"`
}

type DirectLinkItem struct {
	LinkID             string             `json:"linkId"`
	PeerNodeID         string             `json:"peerNodeId"`
	Role               string             `json:"role"`
	PreferredTransport string             `json:"preferredTransport"`
	FallbackTransport  string             `json:"fallbackTransport"`
	PunchToken         string             `json:"punchToken"`
	ExpiresAt          string             `json:"expiresAt"`
	PeerCandidates     []DirectCandidate  `json:"peerCandidates"`
	PeerIdentity       DirectNodeIdentity `json:"peerIdentity"`
}

type ReportDirectStatusInput struct {
	LinkID            string          `json:"linkId"`
	PeerNodeID        string          `json:"peerNodeId"`
	TransportType     string          `json:"transportType"`
	Status            string          `json:"status"`
	SelectedCandidate DirectCandidate `json:"selectedCandidate,omitempty"`
	RTTMs             int             `json:"rttMs,omitempty"`
	LastProbeAt       string          `json:"lastProbeAt,omitempty"`
	FallbackReason    string          `json:"fallbackReason,omitempty"`
}

type ReportDirectStatusResult struct {
	LinkID     string `json:"linkId"`
	PeerNodeID string `json:"peerNodeId"`
	Status     string `json:"status"`
	UpdatedAt  string `json:"updatedAt"`
}
