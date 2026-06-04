package domain

type TenantRole string

const (
	TenantRoleAdmin TenantRole = "tenant_admin"
	TenantRoleUser  TenantRole = "user"
)

type BindingPermission string

const (
	BindingPermissionManage BindingPermission = "manage"
	BindingPermissionUse    BindingPermission = "use"
	BindingPermissionView   BindingPermission = "view"
)

type ResourceType string

const (
	ResourceTypeNode       ResourceType = "node"
	ResourceTypeNodeLink   ResourceType = "node_link"
	ResourceTypeScope      ResourceType = "scope"
	ResourceTypeChain      ResourceType = "chain"
	ResourceTypeRouteRule  ResourceType = "route_rule"
	ResourceTypeAccessPath ResourceType = "access_path"
)

type Tenant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type TenantMembership struct {
	TenantID   string     `json:"tenantId"`
	TenantName string     `json:"tenantName"`
	Role       TenantRole `json:"role"`
	JoinedAt   string     `json:"joinedAt"`
}

type TenantAuthContext struct {
	Account      Account          `json:"account"`
	ActiveTenant TenantMembership `json:"activeTenant"`
	SuperAdmin   bool             `json:"superAdmin"`
}

type TenantResourceBinding struct {
	TenantID     string            `json:"tenantId"`
	TenantName   string            `json:"tenantName"`
	ResourceType string            `json:"resourceType"`
	ResourceID   string            `json:"resourceId"`
	Permission   BindingPermission `json:"permission"`
	CreateID     string            `json:"createId"`
	CreatedAt    string            `json:"createdAt"`
}

type UpsertTenantResourceBindingInput struct {
	Permission BindingPermission `json:"permission"`
}
