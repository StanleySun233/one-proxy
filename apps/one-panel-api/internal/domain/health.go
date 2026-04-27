package domain

type NodeHealth struct {
	NodeID           string            `json:"nodeId"`
	HeartbeatAt      string            `json:"heartbeatAt"`
	PolicyRevisionID string            `json:"policyRevisionId"`
	ListenerStatus   map[string]string `json:"listenerStatus"`
	CertStatus       map[string]string `json:"certStatus"`
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
