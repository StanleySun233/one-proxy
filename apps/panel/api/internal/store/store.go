package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
)

type AccountStore interface {
	ListAccounts() []domain.Account
	CreateAccount(input domain.CreateAccountInput) (domain.Account, error)
	UpdateAccount(accountID string, input domain.UpdateAccountInput) (domain.Account, error)
	DeleteAccount(accountID string) error
}

type SessionStore interface {
	Authenticate(account string, password string) (domain.LoginResult, bool)
	AuthenticateAccessToken(accessToken string) (domain.Account, bool)
	RefreshSession(refreshToken string) (domain.LoginResult, bool)
	Logout(accessToken string) bool
}

type TenantStore interface {
	ListTenants(account domain.Account) []domain.Tenant
	ListAllTenants() []domain.Tenant
	GetTenant(tenantID string) (domain.Tenant, bool)
	CreateTenant(name string, initialAdminAccountID string, createID string) (domain.Tenant, error)
	UpdateTenant(tenantID string, name string) (domain.Tenant, error)
	DeleteTenant(tenantID string) error
	ListTenantMemberships(accountID string) []domain.TenantMembership
	ListTenantMembers(tenantID string) []domain.TenantMembership
	GetTenantMembership(accountID string, tenantID string) (domain.TenantMembership, bool)
	UpsertTenantMembership(tenantID string, accountID string, role domain.TenantRole, createID string) (domain.TenantMembership, error)
	DeleteTenantMembership(tenantID string, accountID string) error
	ListTenantResourceBindings(resourceType domain.ResourceType, resourceID string) ([]domain.TenantResourceBinding, error)
	UpsertTenantResourceBinding(resourceType domain.ResourceType, resourceID string, tenantID string, permission domain.BindingPermission, createID string) (domain.TenantResourceBinding, error)
	DeleteTenantResourceBinding(resourceType domain.ResourceType, resourceID string, tenantID string) error
	TenantResourceBindingPermission(tenantCtx domain.TenantAuthContext, resourceType domain.ResourceType, resourceID string) (domain.BindingPermission, bool)
	CountTenantResourceManageBindings(resourceType domain.ResourceType, resourceID string) int
}

type NodeStore interface {
	ListNodes() []domain.Node
	ListNodesForTenant(tenantCtx domain.TenantAuthContext) []domain.Node
	CreateNode(input domain.CreateNodeInput) (domain.Node, error)
	CreateNodeForTenant(tenantCtx domain.TenantAuthContext, input domain.CreateNodeInput) (domain.Node, error)
	UpdateNode(nodeID string, input domain.UpdateNodeInput) (domain.Node, error)
	DeleteNode(nodeID string) error
	NodeBindingPermission(tenantCtx domain.TenantAuthContext, nodeID string) (domain.BindingPermission, bool)
	CountNodeBindings(nodeID string) int
	ProvisionNodeAccess(nodeID string) (domain.ApproveNodeEnrollmentResult, error)
	ListNodeTransports() []domain.NodeTransport
	UpsertNodeTransport(input domain.UpsertNodeTransportInput) (domain.NodeTransport, error)
	ListNodeLinks() []domain.NodeLink
	ListNodeLinksForTenant(tenantCtx domain.TenantAuthContext) []domain.NodeLink
	CreateNodeLink(input domain.CreateNodeLinkInput) (domain.NodeLink, error)
	CreateNodeLinkForTenant(tenantCtx domain.TenantAuthContext, input domain.CreateNodeLinkInput) (domain.NodeLink, error)
	UpdateNodeLink(linkID string, input domain.UpdateNodeLinkInput) (domain.NodeLink, error)
	DeleteNodeLink(linkID string) error
	NodeLinkBindingPermission(tenantCtx domain.TenantAuthContext, linkID string) (domain.BindingPermission, bool)
	CountNodeLinkBindings(linkID string) int
	ListNodeAccessPaths() []domain.NodeAccessPath
	ListNodeAccessPathsForTenant(tenantCtx domain.TenantAuthContext) []domain.NodeAccessPath
	CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error)
	CreateNodeAccessPathForTenant(tenantCtx domain.TenantAuthContext, input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error)
	UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error)
	DeleteNodeAccessPath(pathID string) error
	NodeAccessPathBindingPermission(tenantCtx domain.TenantAuthContext, pathID string) (domain.BindingPermission, bool)
	CountNodeAccessPathBindings(pathID string) int
	ListNodeOnboardingTasks() []domain.NodeOnboardingTask
	CreateNodeOnboardingTask(accountID string, input domain.CreateNodeOnboardingTaskInput) (domain.NodeOnboardingTask, error)
	UpdateNodeOnboardingTaskStatus(taskID string, status string, statusMessage string) (domain.NodeOnboardingTask, error)
	CreateBootstrapToken(input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error)
	CreateBootstrapTokenForTenant(tenantCtx domain.TenantAuthContext, input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error)
	ListUnconsumedBootstrapTokens() []domain.BootstrapToken
	DeleteBootstrapToken(tokenID string) error
	EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error)
	ApproveNodeEnrollment(nodeID string, reviewedBy string) (domain.ApproveNodeEnrollmentResult, error)
	ExchangeNodeEnrollment(input domain.ExchangeNodeEnrollmentInput) (domain.ApproveNodeEnrollmentResult, error)
	ListPendingNodes() []domain.Node
	RejectNodeEnrollment(nodeID string, reviewedBy string, reason string) error
	AuthenticateNodeToken(accessToken string) (string, bool)
}

type ChainStore interface {
	ListChains() []proxy.Chain
	ListChainsForTenant(tenantCtx domain.TenantAuthContext) []proxy.Chain
	CreateChain(input proxy.CreateChainInput) (proxy.Chain, error)
	CreateChainForTenant(tenantCtx domain.TenantAuthContext, input proxy.CreateChainInput) (proxy.Chain, error)
	UpdateChain(chainID string, input proxy.UpdateChainInput) (proxy.Chain, error)
	DeleteChain(chainID string) error
	ChainBindingPermission(tenantCtx domain.TenantAuthContext, chainID string) (domain.BindingPermission, bool)
	CountChainBindings(chainID string) int
	GetChainProbeResult(chainID string) (proxy.ChainProbeResult, bool)
	SaveChainProbeResult(input proxy.SaveChainProbeResultInput) (proxy.ChainProbeResult, error)
}

type RouteStore interface {
	ListRouteRules() []proxy.RouteRule
	ListRouteRulesForTenant(tenantCtx domain.TenantAuthContext) []proxy.RouteRule
	CreateRouteRule(input proxy.CreateRouteRuleInput) (proxy.RouteRule, error)
	CreateRouteRuleForTenant(tenantCtx domain.TenantAuthContext, input proxy.CreateRouteRuleInput) (proxy.RouteRule, error)
	UpdateRouteRule(ruleID string, input proxy.UpdateRouteRuleInput) (proxy.RouteRule, error)
	DeleteRouteRule(ruleID string) error
	RouteRuleBindingPermission(tenantCtx domain.TenantAuthContext, ruleID string) (domain.BindingPermission, bool)
	CountRouteRuleBindings(ruleID string) int
}

type HealthStore interface {
	ListNodeHealth() []domain.NodeHealth
	ListNodeHealthHistory(nodeID string, window time.Duration) ([]domain.NodeHealth, error)
	UpsertNodeHeartbeat(input domain.NodeHeartbeatInput) (domain.NodeHealth, error)
	RenewNodeCertificate(input domain.NodeCertRenewInput) (domain.NodeCertRenewResult, error)
}

type GroupStore interface {
	CreateGroup(input domain.CreateGroupInput) (domain.Group, error)
	UpdateGroup(id string, input domain.UpdateGroupInput) (domain.Group, error)
	DeleteGroup(id string) error
	GetGroup(id string) (domain.Group, error)
	ListGroups() ([]domain.Group, error)
	ListAccountGroups(accountID string) ([]domain.Group, error)
	AddAccountToGroup(accountID, groupID string) error
	RemoveAccountFromGroup(accountID, groupID string) error
	ListGroupAccounts(groupID string) ([]domain.Account, error)
	SetGroupAccounts(groupID string, accountIDs []string) error
	GetGroupScopes(groupID string) ([]string, error)
	SetGroupScopes(groupID string, scopeKeys []string) error
}

type ScopeStore interface {
	ListScopes() []proxy.Scope
	ListScopesForTenant(tenantCtx domain.TenantAuthContext) []proxy.Scope
	CreateScope(input proxy.CreateScopeInput) (proxy.Scope, error)
	CreateScopeForTenant(tenantCtx domain.TenantAuthContext, input proxy.CreateScopeInput) (proxy.Scope, error)
	UpdateScope(scopeID string, input proxy.UpdateScopeInput) (proxy.Scope, error)
	DeleteScope(scopeID string) error
	ScopeBindingPermission(tenantCtx domain.TenantAuthContext, scopeID string) (domain.BindingPermission, bool)
	CountScopeBindings(scopeID string) int
}

type PolicyStore interface {
	ListPolicyRevisions() []domain.PolicyRevision
	ListPolicyRevisionsForTenant(tenantCtx domain.TenantAuthContext) []domain.PolicyRevision
	PublishPolicy(tenantCtx domain.TenantAuthContext, accountID string) (domain.PolicyRevision, error)
	GetNodeAgentPolicy(nodeID string) (domain.NodeAgentPolicy, bool)
}

type MaintenanceStore interface {
	CleanupExpiredSessions() (int64, error)
	CleanupExpiredBootstrapTokens() (int64, error)
	CleanupExpiredNodeTokens() (int64, error)
	RefreshCertificateStatus(window time.Duration) error
	RefreshNodeStatus(staleAfter time.Duration) error
	CleanupNodeHealthHistory(retention time.Duration) (int64, error)
}

type AuditStore interface {
	CreateBusinessAuditEvent(input domain.CreateBusinessAuditEventInput) (domain.BusinessAuditEvent, error)
	ListBusinessAuditEvents(query domain.BusinessAuditQuery) (domain.BusinessAuditEventsResult, error)
	CreateNetworkAuditSession(input domain.CreateNetworkAuditSessionInput) (domain.NetworkAuditSession, error)
	ListNetworkAuditSessions(query domain.NetworkAuditQuery) (domain.NetworkAuditSessionsResult, error)
	GetAuditDashboard(query domain.AuditDashboardQuery) (domain.AuditDashboard, error)
}

type Store interface {
	IsInitialized() bool
	ReinitializeStore(adminPassword string) error
	GetOverview() domain.Overview
	ListCertificates() []domain.Certificate
	ListFieldEnums() ([]domain.FieldEnum, error)
	ListFieldEnumsByField(field string) ([]domain.FieldEnum, error)

	AccountStore
	SessionStore
	TenantStore
	NodeStore
	ChainStore
	RouteStore
	HealthStore
	GroupStore
	ScopeStore
	PolicyStore
	MaintenanceStore
	AuditStore
}
