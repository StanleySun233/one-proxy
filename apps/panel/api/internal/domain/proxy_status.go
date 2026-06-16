package domain

import "time"

const (
	ProxySessionStatusOK      = "ok"
	ProxySessionStatusSlow    = "slow"
	ProxySessionStatusError   = "error"
	ProxySessionStatusUnknown = "unknown"
)

type ProxySessionMetric struct {
	ID                  string            `json:"id"`
	TenantID            string            `json:"tenantId"`
	NodeID              string            `json:"nodeId"`
	ChainID             string            `json:"chainId"`
	ScopeID             string            `json:"scopeId"`
	RouteID             string            `json:"routeId"`
	GovernanceMode      string            `json:"governanceMode"`
	PolicyRevision      string            `json:"policyRevision"`
	MatchedRuleID       string            `json:"matchedRuleId"`
	MatchedRuleType     string            `json:"matchedRuleType"`
	MatchedRulePattern  string            `json:"matchedRulePattern"`
	MatchedAction       string            `json:"matchedAction"`
	DecisionSource      string            `json:"decisionSource"`
	TargetHost          string            `json:"targetHost"`
	TargetPort          int               `json:"targetPort"`
	Protocol            string            `json:"protocol"`
	StartedAt           time.Time         `json:"startedAt"`
	EndedAt             *time.Time        `json:"endedAt,omitempty"`
	UploadBytes         int64             `json:"uploadBytes"`
	DownloadBytes       int64             `json:"downloadBytes"`
	LatencyMs           int64             `json:"latencyMs"`
	StatusCode          int               `json:"statusCode,omitempty"`
	ReceiveTSMs         int64             `json:"receiveTsMs,omitempty"`
	ForwardTSMs         int64             `json:"forwardTsMs,omitempty"`
	ResponseReceiveTSMs int64             `json:"responseReceiveTsMs,omitempty"`
	ResponseForwardTSMs int64             `json:"responseForwardTsMs,omitempty"`
	NodeProcessMs       int64             `json:"nodeProcessMs,omitempty"`
	ResponseProcessMs   int64             `json:"responseProcessMs,omitempty"`
	NodeTimings         []ProxyNodeTiming `json:"nodeTimings,omitempty"`
	LinkTimings         []ProxyLinkTiming `json:"linkTimings,omitempty"`
	Status              string            `json:"status"`
	ErrorCode           string            `json:"errorCode"`
	ErrorMessage        string            `json:"errorMessage"`
	CacheStatus         string            `json:"cacheStatus,omitempty"`
	CacheStoredAt       *time.Time        `json:"cacheStoredAt,omitempty"`
	ReceivedAt          time.Time         `json:"receivedAt"`
}

type ProxyNodeTiming struct {
	NodeID               string `json:"nodeId"`
	ProcessAvgMs         int64  `json:"processAvgMs"`
	ResponseProcessAvgMs int64  `json:"responseProcessAvgMs"`
	SampleTSMs           int64  `json:"sampleTsMs"`
	Count                int    `json:"count"`
}

type ProxyLinkTiming struct {
	FromNodeID string `json:"fromNodeId"`
	ToNodeID   string `json:"toNodeId"`
	RTTMs      int64  `json:"rttMs"`
	SampleTSMs int64  `json:"sampleTsMs"`
	Count      int    `json:"count"`
}

type ProxySessionIngestInput struct {
	Sessions []ProxySessionMetric `json:"sessions"`
}

type ProxySessionIngestResult struct {
	Status string `json:"status"`
}

type ProxyPageStatusQuery struct {
	Host    string
	RouteID string
	ChainID string
}

type ProxyPageStatus struct {
	Status           string            `json:"status"`
	LatencyMs        int64             `json:"latencyMs"`
	UploadBytes      int64             `json:"uploadBytes"`
	DownloadBytes    int64             `json:"downloadBytes"`
	RequestCount     int               `json:"requestCount"`
	FailureCount     int               `json:"failureCount"`
	LastErrorCode    string            `json:"lastErrorCode"`
	LastErrorMessage string            `json:"lastErrorMessage"`
	CacheStatus      string            `json:"cacheStatus,omitempty"`
	CacheStoredAt    *time.Time        `json:"cacheStoredAt,omitempty"`
	LastSeenAt       time.Time         `json:"lastSeenAt"`
	NodeTimings      []ProxyNodeTiming `json:"nodeTimings"`
	LinkTimings      []ProxyLinkTiming `json:"linkTimings"`
	PolicyRevision   string            `json:"policyRevision"`
	Correlated       bool              `json:"correlated"`
}

type ProxyAuditQuery struct {
	Host    string
	ChainID string
	RouteID string
	NodeID  string
	Status  string
	Level   string
	From    time.Time
	To      time.Time
	Limit   int
}

type ProxyAuditSessionsResult struct {
	Sessions []ProxySessionMetric `json:"sessions"`
}

type ProxyAuditEvent struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenantId"`
	NodeID     string    `json:"nodeId"`
	ChainID    string    `json:"chainId"`
	RouteID    string    `json:"routeId"`
	TargetHost string    `json:"targetHost"`
	Level      string    `json:"level"`
	Code       string    `json:"code"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurredAt"`
	SessionID  string    `json:"sessionId"`
}

type ProxyAuditEventsResult struct {
	Events []ProxyAuditEvent `json:"events"`
}
