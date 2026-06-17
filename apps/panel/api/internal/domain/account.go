package domain

type Account struct {
	ID                 string `json:"id"`
	Account            string `json:"account"`
	Role               string `json:"role"`
	Status             string `json:"status"`
	MustRotatePassword bool   `json:"mustRotatePassword"`
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

type LoginResult struct {
	Account             Account            `json:"account"`
	AccessToken         string             `json:"accessToken"`
	RefreshToken        string             `json:"refreshToken"`
	ExpiresAt           string             `json:"expiresAt"`
	ProxyToken          string             `json:"proxyToken"`
	ProxyTokenExpiresAt string             `json:"proxyTokenExpiresAt"`
	MustRotatePassword  bool               `json:"mustRotatePassword"`
	TenantMemberships   []TenantMembership `json:"tenantMemberships"`
	ActiveTenantID      *string            `json:"activeTenantId"`
}

type ProxyTokenRecord struct {
	Account           Account            `json:"account"`
	ExpiresAt         string             `json:"expiresAt"`
	TenantMemberships []TenantMembership `json:"tenantMemberships"`
	ActiveTenantID    *string            `json:"activeTenantId"`
}

type ProxyTokenValidation struct {
	Valid             bool               `json:"valid"`
	Account           Account            `json:"account"`
	ExpiresAt         string             `json:"expiresAt"`
	CacheTTLSeconds   int                `json:"cacheTtlSeconds"`
	TenantMemberships []TenantMembership `json:"tenantMemberships"`
	ActiveTenantID    *string            `json:"activeTenantId"`
	AllowLocalProxy   bool               `json:"allowLocalProxy"`
}

type RefreshSessionInput struct {
	RefreshToken string `json:"refreshToken"`
}

type LogoutInput struct {
	RefreshToken string `json:"refreshToken"`
}

type ExtensionBootstrap struct {
	SchemaVersion       string                   `json:"schemaVersion"`
	Account             Account                  `json:"account"`
	Tenant              ExtensionTenant          `json:"tenant"`
	PolicyRevision      string                   `json:"policyRevision"`
	FetchedAt           string                   `json:"fetchedAt"`
	ProxyToken          string                   `json:"proxyToken"`
	ProxyTokenExpiresAt string                   `json:"proxyTokenExpiresAt"`
	Nodes               []Node                   `json:"nodes"`
	AccessPaths         []ExtensionAccessPath    `json:"accessPaths"`
	Routes              []ExtensionRoute         `json:"routes"`
	RouteEvaluation     ExtensionRouteEvaluation `json:"routeEvaluation"`
}

type ExtensionTenant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ExtensionTopologyHop struct {
	NodeID     string `json:"nodeId"`
	NodeName   string `json:"nodeName"`
	Mode       string `json:"mode"`
	ScopeKey   string `json:"scopeKey"`
	PublicHost string `json:"publicHost,omitempty"`
	PublicPort int    `json:"publicPort,omitempty"`
	Transport  string `json:"transport"`
}

type ExtensionAccessPath struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	ChainID        string                 `json:"chainId"`
	Mode           string                 `json:"mode"`
	Protocol       string                 `json:"protocol"`
	ServiceType    string                 `json:"serviceType"`
	TargetNodeID   string                 `json:"targetNodeId"`
	EntryNodeID    string                 `json:"entryNodeId"`
	RelayNodeIDs   []string               `json:"relayNodeIds"`
	ListenHost     string                 `json:"listenHost"`
	ListenPort     int                    `json:"listenPort"`
	TargetProtocol string                 `json:"targetProtocol"`
	TargetHost     string                 `json:"targetHost"`
	TargetPort     int                    `json:"targetPort"`
	TargetSNI      string                 `json:"targetSni"`
	TLSMode        string                 `json:"tlsMode"`
	AuthMode       string                 `json:"authMode"`
	Enabled        bool                   `json:"enabled"`
	Options        map[string]string      `json:"options"`
	Topology       []ExtensionTopologyHop `json:"topology"`
	Health         ExtensionPathHealth    `json:"health"`
}

type ExtensionPathHealth struct {
	Status    string `json:"status"`
	Reason    string `json:"reason"`
	CheckedAt string `json:"checkedAt"`
}

type ExtensionRoute struct {
	ID               string                 `json:"id"`
	Priority         int                    `json:"priority"`
	MatchType        string                 `json:"matchType"`
	MatchValue       string                 `json:"matchValue"`
	ActionType       string                 `json:"actionType"`
	ChainID          string                 `json:"chainId"`
	AccessPathID     string                 `json:"accessPathId"`
	DestinationScope string                 `json:"destinationScope"`
	Enabled          bool                   `json:"enabled"`
	Topology         []ExtensionTopologyHop `json:"topology"`
}

type ExtensionRouteEvaluation struct {
	DefaultClientMode     string   `json:"defaultClientMode"`
	DefaultNodeMode       string   `json:"defaultNodeMode"`
	RuleOrder             string   `json:"ruleOrder"`
	NoMatchNodeDenyReason string   `json:"noMatchNodeDenyReason"`
	SupportedMatchTypes   []string `json:"supportedMatchTypes"`
	SupportedActions      []string `json:"supportedActions"`
}
