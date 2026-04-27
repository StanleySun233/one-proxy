package domain

type Overview struct {
	Nodes        OverviewNodes        `json:"nodes"`
	Policies     OverviewPolicies     `json:"policies"`
	Certificates OverviewCertificates `json:"certificates"`
}

type OverviewNodes struct {
	Healthy  int `json:"healthy"`
	Degraded int `json:"degraded"`
}

type OverviewPolicies struct {
	ActiveRevision string `json:"activeRevision"`
	PublishedAt    string `json:"publishedAt"`
}

type OverviewCertificates struct {
	RenewSoon int `json:"renewSoon"`
}

type Certificate struct {
	ID        string `json:"id"`
	OwnerType string `json:"ownerType"`
	OwnerID   string `json:"ownerId"`
	CertType  string `json:"certType"`
	Provider  string `json:"provider"`
	Status    string `json:"status"`
	NotBefore string `json:"notBefore"`
	NotAfter  string `json:"notAfter"`
}
