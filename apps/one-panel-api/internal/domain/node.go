package domain

type Node struct {
	ID           string `json:"id"`
	CreateID     string `json:"createId"`
	OwnerID      string `json:"ownerId"`
	Name         string `json:"name"`
	Mode         string `json:"mode"`
	ScopeKey     string `json:"scopeKey"`
	ParentNodeID string `json:"parentNodeId"`
	Enabled      bool   `json:"enabled"`
	Status       string `json:"status"`
	PublicHost   string `json:"publicHost,omitempty"`
	PublicPort   int    `json:"publicPort,omitempty"`
	ReviewedBy   string `json:"reviewedBy,omitempty"`
	ReviewedAt   string `json:"reviewedAt,omitempty"`
	RejectReason string `json:"rejectReason,omitempty"`
}

type CreateNodeInput struct {
	Name         string `json:"name"`
	Mode         string `json:"mode"`
	ScopeKey     string `json:"scopeKey"`
	ParentNodeID string `json:"parentNodeId"`
	PublicHost   string `json:"publicHost"`
	PublicPort   int    `json:"publicPort"`
}

type UpdateNodeInput struct {
	Name         string `json:"name"`
	Mode         string `json:"mode"`
	ScopeKey     string `json:"scopeKey"`
	ParentNodeID string `json:"parentNodeId"`
	PublicHost   string `json:"publicHost"`
	PublicPort   int    `json:"publicPort"`
	Enabled      bool   `json:"enabled"`
	Status       string `json:"status"`
}

type NodeLink struct {
	ID           string `json:"id"`
	CreateID     string `json:"createId"`
	OwnerID      string `json:"ownerId"`
	SourceNodeID string `json:"sourceNodeId"`
	TargetNodeID string `json:"targetNodeId"`
	LinkType     string `json:"linkType"`
	TrustState   string `json:"trustState"`
}

type CreateNodeLinkInput struct {
	SourceNodeID string `json:"sourceNodeId"`
	TargetNodeID string `json:"targetNodeId"`
	LinkType     string `json:"linkType"`
	TrustState   string `json:"trustState"`
}

type UpdateNodeLinkInput struct {
	SourceNodeID string `json:"sourceNodeId"`
	TargetNodeID string `json:"targetNodeId"`
	LinkType     string `json:"linkType"`
	TrustState   string `json:"trustState"`
}

type NodeAccessPath struct {
	ID             string            `json:"id"`
	CreateID       string            `json:"createId"`
	OwnerID        string            `json:"ownerId"`
	ChainID        string            `json:"chainId"`
	Name           string            `json:"name"`
	Mode           string            `json:"mode"`
	Protocol       string            `json:"protocol"`
	ServiceType    string            `json:"serviceType"`
	TargetNodeID   string            `json:"targetNodeId"`
	EntryNodeID    string            `json:"entryNodeId"`
	RelayNodeIDs   []string          `json:"relayNodeIds"`
	ListenHost     string            `json:"listenHost"`
	ListenPort     int               `json:"listenPort"`
	TargetProtocol string            `json:"targetProtocol"`
	TargetHost     string            `json:"targetHost"`
	TargetPort     int               `json:"targetPort"`
	TargetSNI      string            `json:"targetSni"`
	TLSMode        string            `json:"tlsMode"`
	AuthMode       string            `json:"authMode"`
	Options        map[string]string `json:"options"`
	Enabled        bool              `json:"enabled"`
}

type CreateNodeAccessPathInput struct {
	ChainID        string            `json:"chainId"`
	Name           string            `json:"name"`
	Mode           string            `json:"mode"`
	Protocol       string            `json:"protocol"`
	ServiceType    string            `json:"serviceType"`
	TargetNodeID   string            `json:"targetNodeId"`
	EntryNodeID    string            `json:"entryNodeId"`
	RelayNodeIDs   []string          `json:"relayNodeIds"`
	ListenHost     string            `json:"listenHost"`
	ListenPort     int               `json:"listenPort"`
	TargetProtocol string            `json:"targetProtocol"`
	TargetHost     string            `json:"targetHost"`
	TargetPort     int               `json:"targetPort"`
	TargetSNI      string            `json:"targetSni"`
	TLSMode        string            `json:"tlsMode"`
	AuthMode       string            `json:"authMode"`
	Options        map[string]string `json:"options"`
}

type UpdateNodeAccessPathInput struct {
	ChainID        string            `json:"chainId"`
	Name           string            `json:"name"`
	Mode           string            `json:"mode"`
	Protocol       string            `json:"protocol"`
	ServiceType    string            `json:"serviceType"`
	TargetNodeID   string            `json:"targetNodeId"`
	EntryNodeID    string            `json:"entryNodeId"`
	RelayNodeIDs   []string          `json:"relayNodeIds"`
	ListenHost     string            `json:"listenHost"`
	ListenPort     int               `json:"listenPort"`
	TargetProtocol string            `json:"targetProtocol"`
	TargetHost     string            `json:"targetHost"`
	TargetPort     int               `json:"targetPort"`
	TargetSNI      string            `json:"targetSni"`
	TLSMode        string            `json:"tlsMode"`
	AuthMode       string            `json:"authMode"`
	Options        map[string]string `json:"options"`
	Enabled        bool              `json:"enabled"`
}

type NodeOnboardingTask struct {
	ID                   string `json:"id"`
	Mode                 string `json:"mode"`
	PathID               string `json:"pathId"`
	TargetNodeID         string `json:"targetNodeId"`
	TargetHost           string `json:"targetHost"`
	TargetPort           int    `json:"targetPort"`
	Status               string `json:"status"`
	StatusMessage        string `json:"statusMessage"`
	RequestedByAccountID string `json:"requestedByAccountId,omitempty"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
}

type CreateNodeOnboardingTaskInput struct {
	Mode         string `json:"mode"`
	PathID       string `json:"pathId"`
	TargetNodeID string `json:"targetNodeId"`
	TargetHost   string `json:"targetHost"`
	TargetPort   int    `json:"targetPort"`
}

type UpdateNodeOnboardingTaskStatusInput struct {
	Status        string `json:"status"`
	StatusMessage string `json:"statusMessage"`
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

type BootstrapToken struct {
	ID           string `json:"id"`
	Token        string `json:"token"`
	TargetType   string `json:"targetType"`
	TargetID     string `json:"targetId"`
	NodeName     string `json:"nodeName"`
	NodeMode     string `json:"nodeMode"`
	ScopeKey     string `json:"scopeKey"`
	ParentNodeID string `json:"parentNodeId"`
	PublicHost   string `json:"publicHost"`
	PublicPort   int    `json:"publicPort"`
	ExpiresAt    string `json:"expiresAt"`
	CreatedAt    string `json:"createdAt"`
}

type CreateBootstrapTokenInput struct {
	TargetType   string `json:"targetType"`
	TargetID     string `json:"targetId"`
	NodeName     string `json:"nodeName"`
	NodeMode     string `json:"nodeMode"`
	ScopeKey     string `json:"scopeKey"`
	ParentNodeID string `json:"parentNodeId"`
	PublicHost   string `json:"publicHost"`
	PublicPort   int    `json:"publicPort"`
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

type ExchangeNodeEnrollmentInput struct {
	NodeID           string `json:"nodeId"`
	EnrollmentSecret string `json:"enrollmentSecret"`
}
