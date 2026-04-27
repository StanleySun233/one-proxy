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

type ConnectNodeInput struct {
	Address         string `json:"address"`
	Password        string `json:"password"`
	NewPassword     string `json:"newPassword"`
	Name            string `json:"name"`
	Mode            string `json:"mode"`
	ScopeKey        string `json:"scopeKey"`
	ParentNodeID    string `json:"parentNodeId"`
	PublicHost      string `json:"publicHost"`
	PublicPort      int    `json:"publicPort"`
	ControlPlaneURL string `json:"controlPlaneUrl"`
}

type ConnectedNodeResult struct {
	Node                Node     `json:"node"`
	ConnectionStatus    string   `json:"connectionStatus"`
	LocalIPs            []string `json:"localIps"`
	NodeListenAddr      string   `json:"nodeListenAddr"`
	NodeHTTPSListenAddr string   `json:"nodeHttpsListenAddr"`
	ControlPlaneBound   bool     `json:"controlPlaneBound"`
	MustRotatePassword  bool     `json:"mustRotatePassword"`
}

type NodeLink struct {
	ID           string `json:"id"`
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

type NodeAccessPath struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Mode         string   `json:"mode"`
	TargetNodeID string   `json:"targetNodeId"`
	EntryNodeID  string   `json:"entryNodeId"`
	RelayNodeIDs []string `json:"relayNodeIds"`
	TargetHost   string   `json:"targetHost"`
	TargetPort   int      `json:"targetPort"`
	Enabled      bool     `json:"enabled"`
}

type CreateNodeAccessPathInput struct {
	Name         string   `json:"name"`
	Mode         string   `json:"mode"`
	TargetNodeID string   `json:"targetNodeId"`
	EntryNodeID  string   `json:"entryNodeId"`
	RelayNodeIDs []string `json:"relayNodeIds"`
	TargetHost   string   `json:"targetHost"`
	TargetPort   int      `json:"targetPort"`
}

type UpdateNodeAccessPathInput struct {
	Name         string   `json:"name"`
	Mode         string   `json:"mode"`
	TargetNodeID string   `json:"targetNodeId"`
	EntryNodeID  string   `json:"entryNodeId"`
	RelayNodeIDs []string `json:"relayNodeIds"`
	TargetHost   string   `json:"targetHost"`
	TargetPort   int      `json:"targetPort"`
	Enabled      bool     `json:"enabled"`
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
	ID         string `json:"id"`
	Token      string `json:"token"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	NodeName   string `json:"nodeName"`
	ExpiresAt  string `json:"expiresAt"`
	CreatedAt  string `json:"createdAt"`
}

type CreateBootstrapTokenInput struct {
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	NodeName   string `json:"nodeName"`
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

type NodeScope struct {
	ScopeKey      string `json:"scopeKey"`
	OwnerNodeID   string `json:"ownerNodeId"`
	OwnerNodeName string `json:"ownerNodeName"`
	Description   string `json:"description"`
}
