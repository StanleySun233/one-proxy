package httpapi

import (
	"net/http"

	proxyhttpapi "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/httpapi"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/service"
)

type Router struct {
	mux     *http.ServeMux
	service *service.ControlPlane
}

func NewRouter(cfg HTTPConfig, service *service.ControlPlane) http.Handler {
	router := &Router{
		mux:     http.NewServeMux(),
		service: service,
	}
	router.routes(cfg)
	return withObservability(router.mux)
}

type HTTPConfig struct {
	HTTPAddr    string
	DBBackend   string
	EnvFilePath string
}

func (r *Router) routes(cfg HTTPConfig) {
	r.mux.HandleFunc("/api/setup/status", r.handleSetupStatus)
	r.mux.HandleFunc("/healthz", r.handleHealthz(cfg))

	r.mux.HandleFunc("/api/enums", r.handleEnums)
	r.mux.HandleFunc("/api/auth/login", r.handleLogin)
	r.mux.HandleFunc("/api/auth/refresh", r.handleRefresh)
	r.mux.HandleFunc("/api/auth/logout", r.requireAccount(r.handleLogout))
	r.mux.HandleFunc("/api/audit/business/events", r.requireAccount(r.handleAuditBusinessEvents))
	r.mux.HandleFunc("/api/audit/network/sessions", r.requireAccount(r.handleAuditNetworkSessions))
	r.mux.HandleFunc("/api/audit/dashboard", r.requireAccount(r.handleAuditDashboard))
	r.mux.HandleFunc("/api/audit/proxy/sessions", r.requireAccount(r.handleAuditProxySessions))
	r.mux.HandleFunc("/api/audit/proxy/events", r.requireAccount(r.handleAuditProxyEvents))
	r.mux.HandleFunc("/api/proxy/extension/bootstrap", r.requireAccount(r.handleExtensionBootstrap))
	r.mux.HandleFunc("/api/proxy/extension/direct/session", r.requireAccount(r.handleClientDirectSession))
	r.mux.HandleFunc("/api/overview", r.requireAccount(r.handleOverview))
	r.mux.HandleFunc("/api/tenants", r.requireAccount(r.handleTenants))
	r.mux.HandleFunc("/api/tenants/", r.requireAccount(r.handleTenantByID))
	r.mux.HandleFunc("/api/grants/tenants", r.requireAccount(r.handleGrantTenants))
	r.mux.HandleFunc("/api/grants", r.requireAccount(r.handleGrants))
	r.mux.HandleFunc("/api/grants/", r.requireAccount(r.handleGrantByID))
	r.mux.HandleFunc("/api/accounts", r.requireAccount(r.handleAccounts))
	r.mux.HandleFunc("/api/accounts/", r.requireAccount(r.handleAccountByID))
	r.mux.HandleFunc("/api/groups", r.requireAccount(r.handleGroups))
	r.mux.HandleFunc("/api/groups/", r.requireAccount(r.handleGroupByID))
	r.mux.HandleFunc("/api/nodes", r.requireAccount(r.handleNodes))
	r.mux.HandleFunc("/api/nodes/", r.requireAccount(r.handleNodeByID))
	r.mux.HandleFunc("/api/nodes/transports", r.requireAccount(r.handleNodeTransports))
	r.mux.HandleFunc("/api/nodes/bootstrap/token", r.requireAccount(r.handleNodeBootstrapToken))
	r.mux.HandleFunc("/api/nodes/bootstrap/parent-url/probe", r.requireAccount(r.handleNodeParentURLProbe))
	r.mux.HandleFunc("/api/nodes/bootstrap/tokens/unconsumed", r.requireAccount(r.handleUnconsumedBootstrapTokens))
	r.mux.HandleFunc("/api/nodes/bootstrap/tokens/", r.requireAccount(r.handleBootstrapTokenByID))
	r.mux.HandleFunc("/api/nodes/enroll", r.handleNodeEnroll)
	r.mux.HandleFunc("/api/nodes/exchange", r.handleNodeExchange)
	r.mux.HandleFunc("/api/nodes/pending", r.requireAccount(r.handlePendingNodes))
	r.mux.HandleFunc("/api/policies/revisions", r.requireAccount(r.handlePolicyRevisions))
	r.mux.HandleFunc("/api/policies/publish", r.requireAccount(r.handlePolicyPublish))
	r.mux.HandleFunc("/api/nodes/health", r.requireAccount(r.handleNodeHealth))
	r.mux.HandleFunc("/api/nodes/health/history", r.requireAccount(r.handleNodeHealthHistory))
	r.mux.HandleFunc("/api/nodes/sla", r.requireAccount(r.handleNodeSLA))
	r.mux.HandleFunc("/api/node/agent/policy", r.requireNode(r.handleNodeAgentPolicy))
	r.mux.HandleFunc("/api/node/agent/auth/validate", r.requireNode(r.handleNodeAgentAuthValidate))
	r.mux.HandleFunc("/api/node/agent/heartbeat", r.requireNode(r.handleNodeAgentHeartbeat))
	r.mux.HandleFunc("/api/node/agent/cert/renew", r.requireNode(r.handleNodeAgentCertRenew))
	r.mux.HandleFunc("/api/node/agent/transports", r.requireNode(r.handleNodeAgentTransport))
	r.mux.HandleFunc("/api/node/agent/direct/candidates", r.requireNode(r.handleDirectCandidates))
	r.mux.HandleFunc("/api/node/agent/direct/link/plan", r.requireNode(r.handleDirectLinkPlan))
	r.mux.HandleFunc("/api/node/agent/direct/status", r.requireNode(r.handleDirectStatus))
	r.mux.HandleFunc("/api/node/agent/direct/client/session/validate", r.requireNode(r.handleClientDirectSessionValidate))
	r.mux.HandleFunc("/api/node/agent/proxy/token/authenticate", r.requireNode(r.handleProxyTokenAuthenticate))
	r.mux.HandleFunc("/api/node/agent/proxy/token/validate", r.requireNode(r.handleProxyTokenValidate))
	r.mux.HandleFunc("/api/node/agent/proxy/sessions", r.requireNode(r.handleNodeAgentProxySessions))
	proxyhttpapi.Register(r.mux, r.requireAccount, r.service.Proxy(), r.service, r.service)
}
