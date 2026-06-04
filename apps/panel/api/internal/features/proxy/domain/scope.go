package proxy

type Scope struct {
	ID          string `json:"id"`
	CreateID    string `json:"createId"`
	OwnerID     string `json:"ownerId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Permission  string `json:"permission,omitempty"`
}

type CreateScopeInput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateScopeInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
