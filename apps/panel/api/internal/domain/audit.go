package domain

import "time"

const (
	AuditOutcomeSuccess = "success"
	AuditOutcomeFailure = "failure"
	AuditOutcomeDenied  = "denied"

	NetworkDecisionAllow = "allow"
	NetworkDecisionDeny  = "deny"
)

type BusinessAuditEvent struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenantId"`
	OccurredAt   time.Time `json:"occurredAt"`
	ActorType    string    `json:"actorType"`
	ActorID      string    `json:"actorId"`
	ActorName    string    `json:"actorName"`
	ActorIP      string    `json:"actorIp"`
	ActorAgent   string    `json:"actorAgent"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resourceType"`
	ResourceID   string    `json:"resourceId"`
	ResourceName string    `json:"resourceName"`
	Outcome      string    `json:"outcome"`
	Reason       string    `json:"reason"`
	RequestID    string    `json:"requestId"`
	BeforeJSON   string    `json:"beforeJson"`
	AfterJSON    string    `json:"afterJson"`
	MetadataJSON string    `json:"metadataJson"`
}

type CreateBusinessAuditEventInput struct {
	TenantID     string
	OccurredAt   time.Time
	ActorType    string
	ActorID      string
	ActorName    string
	ActorIP      string
	ActorAgent   string
	Action       string
	ResourceType string
	ResourceID   string
	ResourceName string
	Outcome      string
	Reason       string
	RequestID    string
	BeforeJSON   string
	AfterJSON    string
	MetadataJSON string
}

type BusinessAuditQuery struct {
	TenantID     string
	ActorID      string
	ActorType    string
	ResourceType string
	ResourceID   string
	Action       string
	Outcome      string
	From         time.Time
	To           time.Time
	Limit        int
}

type BusinessAuditSummary struct {
	Total        int64             `json:"total"`
	OutcomeCount map[string]int64  `json:"outcomeCount"`
	ActionCount  map[string]int64  `json:"actionCount"`
	ResourceType map[string]int64  `json:"resourceType"`
	ActorCount   []AuditActorCount `json:"actorCount"`
}

type AuditActorCount struct {
	ActorType string `json:"actorType"`
	ActorID   string `json:"actorId"`
	ActorName string `json:"actorName"`
	Count     int64  `json:"count"`
}

type BusinessAuditEventsResult struct {
	Items   []BusinessAuditEvent `json:"items"`
	Summary BusinessAuditSummary `json:"summary"`
}

type NetworkAuditSession struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenantId"`
	StartedAt          time.Time `json:"startedAt"`
	EndedAt            time.Time `json:"endedAt"`
	ActorType          string    `json:"actorType"`
	ActorID            string    `json:"actorId"`
	TokenID            string    `json:"tokenId"`
	SourceIP           string    `json:"sourceIp"`
	EntryNodeID        string    `json:"entryNodeId"`
	ExitNodeID         string    `json:"exitNodeId"`
	TargetHost         string    `json:"targetHost"`
	TargetPort         int       `json:"targetPort"`
	Scheme             string    `json:"scheme"`
	Method             string    `json:"method"`
	RouteID            string    `json:"routeId"`
	ScopeID            string    `json:"scopeId"`
	ChainID            string    `json:"chainId"`
	GovernanceMode     string    `json:"governanceMode"`
	PolicyRevision     string    `json:"policyRevision"`
	MatchedRuleID      string    `json:"matchedRuleId"`
	MatchedRuleType    string    `json:"matchedRuleType"`
	MatchedRulePattern string    `json:"matchedRulePattern"`
	MatchedAction      string    `json:"matchedAction"`
	DecisionSource     string    `json:"decisionSource"`
	Decision           string    `json:"decision"`
	DenyReason         string    `json:"denyReason"`
	BytesIn            int64     `json:"bytesIn"`
	BytesOut           int64     `json:"bytesOut"`
	DurationMs         int64     `json:"durationMs"`
	StatusCode         int       `json:"statusCode"`
	ErrorCode          string    `json:"errorCode"`
	ReceivedAt         time.Time `json:"receivedAt"`
	MetadataJSON       string    `json:"metadataJson"`
}

type CreateNetworkAuditSessionInput struct {
	ID                 string
	TenantID           string
	StartedAt          time.Time
	EndedAt            time.Time
	ActorType          string
	ActorID            string
	TokenID            string
	SourceIP           string
	EntryNodeID        string
	ExitNodeID         string
	TargetHost         string
	TargetPort         int
	Scheme             string
	Method             string
	RouteID            string
	ScopeID            string
	ChainID            string
	GovernanceMode     string
	PolicyRevision     string
	MatchedRuleID      string
	MatchedRuleType    string
	MatchedRulePattern string
	MatchedAction      string
	DecisionSource     string
	Decision           string
	DenyReason         string
	BytesIn            int64
	BytesOut           int64
	DurationMs         int64
	StatusCode         int
	ErrorCode          string
	ReceivedAt         time.Time
	MetadataJSON       string
}

type NetworkAuditQuery struct {
	TenantID       string
	ActorID        string
	TokenID        string
	NodeID         string
	TargetHost     string
	RouteID        string
	ScopeID        string
	ChainID        string
	DenyReason     string
	PolicyRevision string
	MatchedRuleID  string
	DecisionSource string
	Decision       string
	From           time.Time
	To             time.Time
	Limit          int
}

type NetworkAuditSummary struct {
	Total           int64                `json:"total"`
	BytesIn         int64                `json:"bytesIn"`
	BytesOut        int64                `json:"bytesOut"`
	DurationAvgMs   int64                `json:"durationAvgMs"`
	DecisionCount   map[string]int64     `json:"decisionCount"`
	DenyReasonCount map[string]int64     `json:"denyReasonCount"`
	TopTargets      []AuditTargetTraffic `json:"topTargets"`
	UserTraffic     []AuditActorTraffic  `json:"userTraffic"`
	NodeTraffic     []AuditNodeTraffic   `json:"nodeTraffic"`
	TenantTraffic   []AuditTenantTraffic `json:"tenantTraffic"`
	RecentBusiness  []BusinessAuditEvent `json:"recentBusinessEvents"`
}

type NetworkAuditSessionsResult struct {
	Items   []NetworkAuditSession `json:"items"`
	Summary NetworkAuditSummary   `json:"summary"`
}

type AuditDashboardQuery struct {
	TenantID string
	From     time.Time
	To       time.Time
}

type AuditDashboard struct {
	NetworkAuditSummary
}

type AuditTargetTraffic struct {
	TargetHost string `json:"targetHost"`
	BytesIn    int64  `json:"bytesIn"`
	BytesOut   int64  `json:"bytesOut"`
	Count      int64  `json:"count"`
}

type AuditActorTraffic struct {
	ActorID  string `json:"actorId"`
	BytesIn  int64  `json:"bytesIn"`
	BytesOut int64  `json:"bytesOut"`
	Count    int64  `json:"count"`
}

type AuditNodeTraffic struct {
	NodeID   string `json:"nodeId"`
	BytesIn  int64  `json:"bytesIn"`
	BytesOut int64  `json:"bytesOut"`
	Count    int64  `json:"count"`
}

type AuditTenantTraffic struct {
	TenantID string `json:"tenantId"`
	BytesIn  int64  `json:"bytesIn"`
	BytesOut int64  `json:"bytesOut"`
	Count    int64  `json:"count"`
}
