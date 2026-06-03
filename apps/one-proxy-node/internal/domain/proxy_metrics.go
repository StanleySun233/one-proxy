package domain

const (
	ProxySessionProtocolHTTP    = "http"
	ProxySessionProtocolConnect = "connect"
	ProxySessionStatusOK        = "ok"
	ProxySessionStatusError     = "error"
)

type ProxySessionMetric struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenantId"`
	NodeID        string `json:"nodeId"`
	ChainID       string `json:"chainId"`
	RouteID       string `json:"routeId"`
	TargetHost    string `json:"targetHost"`
	TargetPort    int    `json:"targetPort"`
	Protocol      string `json:"protocol"`
	StartedAt     string `json:"startedAt"`
	EndedAt       string `json:"endedAt"`
	UploadBytes   int64  `json:"uploadBytes"`
	DownloadBytes int64  `json:"downloadBytes"`
	LatencyMs     int64  `json:"latencyMs"`
	Status        string `json:"status"`
	ErrorCode     string `json:"errorCode"`
	ErrorMessage  string `json:"errorMessage"`
}

type ProxySessionMetricsInput struct {
	Sessions []ProxySessionMetric `json:"sessions"`
}

type ProxySessionMetricsResult struct {
	Status string `json:"status"`
}
