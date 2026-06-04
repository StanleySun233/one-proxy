package domain

type PolicyRevision struct {
	ID                string   `json:"id"`
	TenantID          string   `json:"tenantId"`
	Version           string   `json:"version"`
	Status            string   `json:"status"`
	CreatedAt         string   `json:"createdAt"`
	AssignedNodes     int      `json:"assignedNodes"`
	AffectedTenantIDs []string `json:"affectedTenantIds"`
}

type NodeAgentPolicy struct {
	NodeID           string `json:"nodeId"`
	PolicyRevisionID string `json:"policyRevisionId"`
	PayloadJSON      string `json:"payloadJson"`
}
