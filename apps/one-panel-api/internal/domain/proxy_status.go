package domain

import "time"

const (
	ProxySessionStatusOK      = "ok"
	ProxySessionStatusSlow    = "slow"
	ProxySessionStatusError   = "error"
	ProxySessionStatusUnknown = "unknown"
)

type ProxySessionMetric struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenantId"`
	NodeID        string     `json:"nodeId"`
	ChainID       string     `json:"chainId"`
	RouteID       string     `json:"routeId"`
	TargetHost    string     `json:"targetHost"`
	TargetPort    int        `json:"targetPort"`
	Protocol      string     `json:"protocol"`
	StartedAt     time.Time  `json:"startedAt"`
	EndedAt       *time.Time `json:"endedAt,omitempty"`
	UploadBytes   int64      `json:"uploadBytes"`
	DownloadBytes int64      `json:"downloadBytes"`
	LatencyMs     int64      `json:"latencyMs"`
	Status        string     `json:"status"`
	ErrorCode     string     `json:"errorCode"`
	ErrorMessage  string     `json:"errorMessage"`
	ReceivedAt    time.Time  `json:"receivedAt"`
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
	Status           string    `json:"status"`
	LatencyMs        int64     `json:"latencyMs"`
	UploadBytes      int64     `json:"uploadBytes"`
	DownloadBytes    int64     `json:"downloadBytes"`
	RequestCount     int       `json:"requestCount"`
	FailureCount     int       `json:"failureCount"`
	LastErrorCode    string    `json:"lastErrorCode"`
	LastErrorMessage string    `json:"lastErrorMessage"`
	LastSeenAt       time.Time `json:"lastSeenAt"`
	PolicyRevision   string    `json:"policyRevision"`
	Correlated       bool      `json:"correlated"`
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
