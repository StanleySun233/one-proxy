package domain

type Group struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type CreateGroupInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
}

type UpdateGroupInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Enabled     *bool   `json:"enabled"`
}

type GroupDetail struct {
	Group
	Accounts []Account `json:"accounts"`
	Scopes   []string  `json:"scopes"`
}

type SetGroupAccountsInput struct {
	AccountIDs []string `json:"accountIds"`
}

type SetGroupScopesInput struct {
	ScopeKeys []string `json:"scopeKeys"`
}
