package domain

type Node struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Mode         string `json:"mode"`
	ScopeKey     string `json:"scopeKey"`
	ParentNodeID string `json:"parentNodeId"`
	Enabled      bool   `json:"enabled"`
	Status       string `json:"status"`
	PublicHost   string `json:"publicHost,omitempty"`
	PublicPort   int    `json:"publicPort,omitempty"`
}

type NodeLink struct {
	ID           string `json:"id"`
	SourceNodeID string `json:"sourceNodeId"`
	TargetNodeID string `json:"targetNodeId"`
	LinkType     string `json:"linkType"`
	TrustState   string `json:"trustState"`
}

type Chain struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	DestinationScope string   `json:"destinationScope"`
	Enabled          bool     `json:"enabled"`
	Hops             []string `json:"hops"`
}

type RouteRule struct {
	ID               string `json:"id"`
	Priority         int    `json:"priority"`
	MatchType        string `json:"matchType"`
	MatchValue       string `json:"matchValue"`
	ActionType       string `json:"actionType"`
	ChainID          string `json:"chainId,omitempty"`
	DestinationScope string `json:"destinationScope,omitempty"`
	Enabled          bool   `json:"enabled"`
}

type NodeAgentPolicy struct {
	NodeID           string `json:"nodeId"`
	PolicyRevisionID string `json:"policyRevisionId"`
	PayloadJSON      string `json:"payloadJson"`
}

type NodeHeartbeatInput struct {
	NodeID           string            `json:"nodeId"`
	PolicyRevisionID string            `json:"policyRevisionId"`
	ListenerStatus   map[string]string `json:"listenerStatus"`
	CertStatus       map[string]string `json:"certStatus"`
}

type NodeHealth struct {
	NodeID           string            `json:"nodeId"`
	HeartbeatAt      string            `json:"heartbeatAt"`
	PolicyRevisionID string            `json:"policyRevisionId"`
	ListenerStatus   map[string]string `json:"listenerStatus"`
	CertStatus       map[string]string `json:"certStatus"`
}

type NodeTransport struct {
	ID              string            `json:"id"`
	NodeID          string            `json:"nodeId"`
	TransportType   string            `json:"transportType"`
	Direction       string            `json:"direction"`
	Address         string            `json:"address"`
	Status          string            `json:"status"`
	ParentNodeID    string            `json:"parentNodeId"`
	ConnectedAt     string            `json:"connectedAt"`
	LastHeartbeatAt string            `json:"lastHeartbeatAt"`
	LatencyMs       int               `json:"latencyMs"`
	Details         map[string]string `json:"details"`
}

type UpsertNodeTransportInput struct {
	NodeID          string            `json:"nodeId"`
	TransportType   string            `json:"transportType"`
	Direction       string            `json:"direction"`
	Address         string            `json:"address"`
	Status          string            `json:"status"`
	ParentNodeID    string            `json:"parentNodeId"`
	ConnectedAt     string            `json:"connectedAt"`
	LastHeartbeatAt string            `json:"lastHeartbeatAt"`
	LatencyMs       int               `json:"latencyMs"`
	Details         map[string]string `json:"details"`
}

type NodeCertRenewInput struct {
	NodeID   string `json:"nodeId"`
	CertType string `json:"certType"`
}

type NodeCertRenewResult struct {
	NodeID   string `json:"nodeId"`
	CertType string `json:"certType"`
	Status   string `json:"status"`
	NotAfter string `json:"notAfter"`
}

type ExchangeNodeEnrollmentInput struct {
	NodeID           string `json:"nodeId"`
	EnrollmentSecret string `json:"enrollmentSecret"`
}

type EnrollNodeInput struct {
	Token        string `json:"token"`
	Name         string `json:"name"`
	Mode         string `json:"mode"`
	ScopeKey     string `json:"scopeKey"`
	ParentNodeID string `json:"parentNodeId"`
	PublicHost   string `json:"publicHost"`
	PublicPort   int    `json:"publicPort"`
}

type EnrollNodeResult struct {
	Node             Node   `json:"node"`
	EnrollmentSecret string `json:"enrollmentSecret"`
	ApprovalState    string `json:"approvalState"`
}

type ApproveNodeEnrollmentResult struct {
	Node          Node   `json:"node"`
	AccessToken   string `json:"accessToken"`
	TrustMaterial string `json:"trustMaterial"`
	ExpiresAt     string `json:"expiresAt"`
}

type NodeBootstrapAttachInput struct {
	Password        string   `json:"password"`
	NewPassword     string   `json:"newPassword"`
	ControlPlaneURL string   `json:"controlPlaneUrl"`
	NodeID          string   `json:"nodeId"`
	NodeAccessToken string   `json:"nodeAccessToken"`
	NodeName        string   `json:"nodeName"`
	NodeMode        string   `json:"nodeMode"`
	NodeScopeKey    string   `json:"nodeScopeKey"`
	NodeParentID    string   `json:"nodeParentId"`
	NodePublicHost  string   `json:"nodePublicHost"`
	NodePublicPort  int      `json:"nodePublicPort"`
	LocalIPs        []string `json:"localIps"`
}

type NodeBootstrapAttachResult struct {
	ConnectionStatus    string   `json:"connectionStatus"`
	LocalIPs            []string `json:"localIps"`
	NodeListenAddr      string   `json:"nodeListenAddr"`
	NodeHTTPSListenAddr string   `json:"nodeHttpsListenAddr"`
	ControlPlaneBound   bool     `json:"controlPlaneBound"`
	MustRotatePassword  bool     `json:"mustRotatePassword"`
}

type NodeBootstrapPasswordRotateInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
