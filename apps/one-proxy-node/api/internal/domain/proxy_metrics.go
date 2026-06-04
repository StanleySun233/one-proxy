package domain

const (
	ProxySessionProtocolHTTP    = "http"
	ProxySessionProtocolConnect = "connect"
	ProxySessionStatusOK        = "ok"
	ProxySessionStatusError     = "error"
)

type ProxySessionMetric struct {
	ID                  string            `json:"id"`
	TenantID            string            `json:"tenantId"`
	NodeID              string            `json:"nodeId"`
	ChainID             string            `json:"chainId"`
	RouteID             string            `json:"routeId"`
	TargetHost          string            `json:"targetHost"`
	TargetPort          int               `json:"targetPort"`
	Protocol            string            `json:"protocol"`
	StartedAt           string            `json:"startedAt"`
	EndedAt             string            `json:"endedAt"`
	UploadBytes         int64             `json:"uploadBytes"`
	DownloadBytes       int64             `json:"downloadBytes"`
	LatencyMs           int64             `json:"latencyMs"`
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

type ProxySessionMetricsInput struct {
	Sessions []ProxySessionMetric `json:"sessions"`
}

type ProxySessionMetricsResult struct {
	Status string `json:"status"`
}
