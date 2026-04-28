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
)

// Account status constants
const (
	AccountStatusActive   = "active"
	AccountStatusDisabled = "disabled"
)

// Path mode constants
const (
	PathModeDirect       = "direct"
	PathModeRelayChain   = "relay_chain"
	PathModeUpstreamPull = "upstream_pull"
)

// Task status constants
const (
	TaskStatusPlanned    = "planned"
	TaskStatusPending    = "pending"
	TaskStatusConnected  = "connected"
	TaskStatusFailed     = "failed"
	TaskStatusCancelled  = "cancelled"
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
	TransportTypeChildWS         = "child_ws"
	TransportTypeReverseWS       = "reverse_ws"
)

// Transport status constants
const (
	TransportStatusConnected  = "connected"
	TransportStatusAvailable  = "available"
	TransportStatusDegraded   = "degraded"
	TransportStatusFailed     = "failed"
	TransportStatusPending    = "pending"
)

// Cert status constants
const (
	CertStatusHealthy  = "healthy"
	CertStatusRenewSoon = "renew-soon"
	CertStatusExpired  = "expired"
	CertStatusRenewed  = "renewed"
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
	MatchTypeIPCIDR       = "ip_cidr"
	MatchTypeIPRange      = "ip_range"
	MatchTypePort         = "port"
	MatchTypeURLRegex     = "url_regex"
	MatchTypeDefault      = "default"
)
