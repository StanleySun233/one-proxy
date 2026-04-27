package store

import (
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
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

type NodeStore interface {
	ListNodes() []domain.Node
	CreateNode(input domain.CreateNodeInput) (domain.Node, error)
	UpdateNode(nodeID string, input domain.UpdateNodeInput) (domain.Node, error)
	DeleteNode(nodeID string) error
	ProvisionNodeAccess(nodeID string) (domain.ApproveNodeEnrollmentResult, error)
	ListNodeTransports() []domain.NodeTransport
	UpsertNodeTransport(input domain.UpsertNodeTransportInput) (domain.NodeTransport, error)
	ListNodeLinks() []domain.NodeLink
	CreateNodeLink(input domain.CreateNodeLinkInput) (domain.NodeLink, error)
	ListNodeAccessPaths() []domain.NodeAccessPath
	CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error)
	UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error)
	DeleteNodeAccessPath(pathID string) error
	ListNodeOnboardingTasks() []domain.NodeOnboardingTask
	CreateNodeOnboardingTask(accountID string, input domain.CreateNodeOnboardingTaskInput) (domain.NodeOnboardingTask, error)
	UpdateNodeOnboardingTaskStatus(taskID string, status string, statusMessage string) (domain.NodeOnboardingTask, error)
	CreateBootstrapToken(input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error)
	ListUnconsumedBootstrapTokens() []domain.BootstrapToken
	EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error)
	ApproveNodeEnrollment(nodeID string, reviewedBy string) (domain.ApproveNodeEnrollmentResult, error)
	ExchangeNodeEnrollment(input domain.ExchangeNodeEnrollmentInput) (domain.ApproveNodeEnrollmentResult, error)
	ListPendingNodes() []domain.Node
	RejectNodeEnrollment(nodeID string, reviewedBy string, reason string) error
	AuthenticateNodeToken(accessToken string) (string, bool)
}

type ChainStore interface {
	ListChains() []domain.Chain
	CreateChain(input domain.CreateChainInput) (domain.Chain, error)
	UpdateChain(chainID string, input domain.UpdateChainInput) (domain.Chain, error)
	DeleteChain(chainID string) error
	GetChainProbeResult(chainID string) (domain.ChainProbeResult, bool)
	SaveChainProbeResult(input domain.SaveChainProbeResultInput) (domain.ChainProbeResult, error)
}

type RouteStore interface {
	ListRouteRules() []domain.RouteRule
	CreateRouteRule(input domain.CreateRouteRuleInput) (domain.RouteRule, error)
	UpdateRouteRule(ruleID string, input domain.UpdateRouteRuleInput) (domain.RouteRule, error)
	DeleteRouteRule(ruleID string) error
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

type PolicyStore interface {
	ListPolicyRevisions() []domain.PolicyRevision
	PublishPolicy(accountID string) (domain.PolicyRevision, error)
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

type Store interface {
	IsInitialized() bool
	ReinitializeStore(adminPassword string) error
	GetOverview() domain.Overview
	ListCertificates() []domain.Certificate
	ListFieldEnums() ([]domain.FieldEnum, error)
	ListFieldEnumsByField(field string) ([]domain.FieldEnum, error)

	AccountStore
	SessionStore
	NodeStore
	ChainStore
	RouteStore
	HealthStore
	GroupStore
	PolicyStore
	MaintenanceStore
}
