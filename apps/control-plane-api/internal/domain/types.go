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

type Account struct {
	ID                 string `json:"id"`
	Account            string `json:"account"`
	Role               string `json:"role"`
	Status             string `json:"status"`
	MustRotatePassword bool   `json:"mustRotatePassword"`
}

type NodeLink struct {
	ID           string `json:"id"`
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

type Certificate struct {
	ID        string `json:"id"`
	OwnerType string `json:"ownerType"`
	OwnerID   string `json:"ownerId"`
	CertType  string `json:"certType"`
	Provider  string `json:"provider"`
	Status    string `json:"status"`
	NotBefore string `json:"notBefore"`
	NotAfter  string `json:"notAfter"`
}

type CreateNodeLinkInput struct {
	SourceNodeID string `json:"sourceNodeId"`
	TargetNodeID string `json:"targetNodeId"`
	LinkType     string `json:"linkType"`
	TrustState   string `json:"trustState"`
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

type CreateAccountInput struct {
	Account  string `json:"account"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UpdateAccountInput struct {
	Password string `json:"password"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

type CreateNodeInput struct {
	Name         string `json:"name"`
	Mode         string `json:"mode"`
	ScopeKey     string `json:"scopeKey"`
	ParentNodeID string `json:"parentNodeId"`
	PublicHost   string `json:"publicHost"`
	PublicPort   int    `json:"publicPort"`
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

type CreateChainInput struct {
	Name             string   `json:"name"`
	DestinationScope string   `json:"destinationScope"`
	Hops             []string `json:"hops"`
}

type UpdateChainInput struct {
	Name             string   `json:"name"`
	DestinationScope string   `json:"destinationScope"`
	Hops             []string `json:"hops"`
	Enabled          bool     `json:"enabled"`
}

type CreateRouteRuleInput struct {
	Priority         int    `json:"priority"`
	MatchType        string `json:"matchType"`
	MatchValue       string `json:"matchValue"`
	ActionType       string `json:"actionType"`
	ChainID          string `json:"chainId"`
	DestinationScope string `json:"destinationScope"`
}

type UpdateRouteRuleInput struct {
	Priority         int    `json:"priority"`
	MatchType        string `json:"matchType"`
	MatchValue       string `json:"matchValue"`
	ActionType       string `json:"actionType"`
	ChainID          string `json:"chainId"`
	DestinationScope string `json:"destinationScope"`
	Enabled          bool   `json:"enabled"`
}

type CreateBootstrapTokenInput struct {
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
}

type BootstrapToken struct {
	ID         string `json:"id"`
	Token      string `json:"token"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	ExpiresAt  string `json:"expiresAt"`
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

type RefreshSessionInput struct {
	RefreshToken string `json:"refreshToken"`
}

type LogoutInput struct {
	RefreshToken string `json:"refreshToken"`
}

type LoginResult struct {
	Account            Account `json:"account"`
	AccessToken        string  `json:"accessToken"`
	RefreshToken       string  `json:"refreshToken"`
	ExpiresAt          string  `json:"expiresAt"`
	MustRotatePassword bool    `json:"mustRotatePassword"`
}

type PolicyRevision struct {
	ID            string `json:"id"`
	Version       string `json:"version"`
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
	AssignedNodes int    `json:"assignedNodes"`
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

type NodeHealth struct {
	NodeID           string            `json:"nodeId"`
	HeartbeatAt      string            `json:"heartbeatAt"`
	PolicyRevisionID string            `json:"policyRevisionId"`
	ListenerStatus   map[string]string `json:"listenerStatus"`
	CertStatus       map[string]string `json:"certStatus"`
}

type Overview struct {
	Nodes        OverviewNodes        `json:"nodes"`
	Policies     OverviewPolicies     `json:"policies"`
	Certificates OverviewCertificates `json:"certificates"`
}

type OverviewNodes struct {
	Healthy  int `json:"healthy"`
	Degraded int `json:"degraded"`
}

type OverviewPolicies struct {
	ActiveRevision string `json:"activeRevision"`
	PublishedAt    string `json:"publishedAt"`
}

type OverviewCertificates struct {
	RenewSoon int `json:"renewSoon"`
}
