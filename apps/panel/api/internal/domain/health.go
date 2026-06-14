package domain

import "time"

type NodeHealth struct {
	NodeID                    string            `json:"nodeId"`
	HeartbeatAt               string            `json:"heartbeatAt"`
	PolicyRevisionID          string            `json:"policyRevisionId"`
	ListenerStatus            map[string]string `json:"listenerStatus"`
	CertStatus                map[string]string `json:"certStatus"`
	ProxyTokenCacheTTLSeconds int               `json:"proxyTokenCacheTtlSeconds"`
}

type NodeHeartbeatInput struct {
	NodeID           string            `json:"nodeId"`
	HeartbeatTs      int64             `json:"heartbeatTs"`
	PolicyRevisionID string            `json:"policyRevisionId"`
	ListenerStatus   map[string]string `json:"listenerStatus"`
	CertStatus       map[string]string `json:"certStatus"`
}

func (input NodeHeartbeatInput) HeartbeatTime(fallback time.Time) time.Time {
	if input.HeartbeatTs <= 0 {
		return fallback.UTC()
	}
	return time.UnixMilli(input.HeartbeatTs).UTC()
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
