package domain

// FieldEnum represents a configurable field enumeration value stored in the database.
type FieldEnum struct {
	ID    string  `json:"id"`
	Field string  `json:"field"`
	Value string  `json:"value"`
	Name  string  `json:"name"`
	Meta  *string `json:"meta,omitempty"`
}

// Node mode constants
const (
	NodeModeEdge  = "edge"
	NodeModeRelay = "relay"
)

// Node status constants
const (
	NodeStatusHealthy  = "healthy"
	NodeStatusDegraded = "degraded"
	NodeStatusPending  = "pending"
	NodeStatusInactive = "inactive"
)

// Account role constants
const (
	AccountRoleSuperAdmin = "super_admin"
	AccountRoleUser       = "user"
)

// Account status constants
const (
	AccountStatusActive   = "active"
	AccountStatusDisabled = "disabled"
)

// Path mode constants
const (
	PathModeForward = "forward"
	PathModeReverse = "reverse"
	PathModeDirect  = "direct"
	PathModeTCP     = "tcp"
	PathModeUDP     = "udp"
)

// Access protocol constants
const (
	AccessProtocolHTTP    = "http"
	AccessProtocolHTTPS   = "https"
	AccessProtocolConnect = "connect"
	AccessProtocolTCP     = "tcp"
	AccessProtocolUDP     = "udp"
	AccessProtocolQUIC    = "quic"
)

// Access service type constants
const (
	AccessServiceHTTPForwardProxy = "http_forward_proxy"
	AccessServiceReverseProxy     = "reverse_proxy"
	AccessServiceTCPAccess        = "tcp_access"
	AccessServiceUDPAccess        = "udp_access"
	AccessServiceDirectQUIC       = "direct_quic"
)

// TLS mode constants
const (
	TLSModePassthrough  = "passthrough"
	TLSModeTerminate    = "terminate"
	TLSModeDirectVerify = "direct_verify"
)

// Access auth mode constants
const (
	AccessAuthProxyToken = "proxy_token"
)

// Task status constants
const (
	TaskStatusPlanned   = "planned"
	TaskStatusPending   = "pending"
	TaskStatusConnected = "connected"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

// Action type constants
const (
	ActionTypeChain  = "chain"
	ActionTypeDirect = "direct"
)

// Link type constants
const (
	LinkTypeParentChild = "parent_child"
	LinkTypeRelay       = "relay"
	LinkTypeManaged     = "managed"
)

// Trust state constants
const (
	TrustStateTrusted = "trusted"
	TrustStateActive  = "active"
)

// Transport type constants
const (
	TransportTypePublicHTTP      = "public_http"
	TransportTypePublicHTTPS     = "public_https"
	TransportTypeReverseWSParent = "reverse_ws_parent"
	TransportTypeDirectRelay     = "direct_relay"
	TransportTypeChildWS         = "child_ws"
	TransportTypeReverseWS       = "reverse_ws"
)

// Transport status constants
const (
	TransportStatusConnected = "connected"
	TransportStatusAvailable = "available"
	TransportStatusDegraded  = "degraded"
	TransportStatusFailed    = "failed"
	TransportStatusPending   = "pending"
)

// Cert status constants
const (
	CertStatusHealthy   = "healthy"
	CertStatusDegraded  = "degraded"
	CertStatusRenewSoon = "renew-soon"
	CertStatusExpired   = "expired"
	CertStatusRenewed   = "renewed"
)

// Cert type constants
const (
	CertTypePublic   = "public"
	CertTypeInternal = "internal"
)

// Bootstrap target type constants
const (
	BootstrapTargetTypeNode = "node"
)

// Trust material status constants
const (
	TrustMaterialStatusActive   = "active"
	TrustMaterialStatusRotated  = "rotated"
	TrustMaterialStatusPending  = "pending"
	TrustMaterialStatusConsumed = "consumed"
)

// Probe result status constants
const (
	ProbeResultStatusConnected = "connected"
	ProbeResultStatusFailed    = "failed"
)

// Policy status constants
const (
	PolicyStatusPublished = "published"
)

// Listener status constants
const (
	ListenerStatusUp       = "up"
	ListenerStatusDegraded = "degraded"
	ListenerStatusDown     = "0"
)

// Approval state constants
const (
	ApprovalStatePending  = "pending"
	ApprovalStateApproved = "approved"
	ApprovalStateRejected = "rejected"
)

// Match type constants
const (
	MatchTypeDomain       = "domain"
	MatchTypeDomainSuffix = "domain_suffix"
	MatchTypeIP           = "ip"
	MatchTypeIPCIDR       = "ip_cidr"
	MatchTypeProtocol     = "protocol"
	MatchTypeDefault      = "default"
)
