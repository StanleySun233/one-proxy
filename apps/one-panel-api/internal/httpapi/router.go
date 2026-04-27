package httpapi

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/network"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/service"
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
	HTTPAddr  string
	DBBackend string
}

func (r *Router) routes(cfg HTTPConfig) {
	r.mux.HandleFunc("/api/v1/setup/status", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]any{"configured": true})
	})

	r.mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]any{
			"status":    "ok",
			"httpAddr":  cfg.HTTPAddr,
			"dbBackend": cfg.DBBackend,
			"localIPs":  network.LocalIPs(),
		})
	})

	r.mux.HandleFunc("/api/v1/auth/login", r.handleLogin)
	r.mux.HandleFunc("/api/v1/auth/refresh", r.handleRefresh)
	r.mux.HandleFunc("/api/v1/auth/logout", r.requireAccount(r.handleLogout))
	r.mux.HandleFunc("/api/v1/extension/bootstrap", r.requireAccount(r.handleExtensionBootstrap))
	r.mux.HandleFunc("/api/v1/overview", r.requireAccount(r.handleOverview))
	r.mux.HandleFunc("/api/v1/accounts", r.requireAccount(r.handleAccounts))
	r.mux.HandleFunc("/api/v1/accounts/", r.requireAccount(r.handleAccountByID))
	r.mux.HandleFunc("/api/v1/groups", r.requireAccount(r.handleGroups))
	r.mux.HandleFunc("/api/v1/groups/", r.requireAccount(r.handleGroupByID))
	r.mux.HandleFunc("/api/v1/node-links", r.requireAccount(r.handleNodeLinks))
	r.mux.HandleFunc("/api/v1/node-access-paths", r.requireAccount(r.handleNodeAccessPaths))
	r.mux.HandleFunc("/api/v1/node-access-paths/", r.requireAccount(r.handleNodeAccessPathByID))
	r.mux.HandleFunc("/api/v1/node-onboarding-tasks", r.requireAccount(r.handleNodeOnboardingTasks))
	r.mux.HandleFunc("/api/v1/node-onboarding-tasks/", r.requireAccount(r.handleNodeOnboardingTaskByID))
	r.mux.HandleFunc("/api/v1/nodes", r.requireAccount(r.handleNodes))
	r.mux.HandleFunc("/api/v1/nodes/", r.requireAccount(r.handleNodeByID))
	r.mux.HandleFunc("/api/v1/node-transports", r.requireAccount(r.handleNodeTransports))
	r.mux.HandleFunc("/api/v1/nodes/connect", r.requireAccount(r.handleNodeConnect))
	r.mux.HandleFunc("/api/v1/nodes/approve/", r.requireAccount(r.handleNodeApprove))
	r.mux.HandleFunc("/api/v1/nodes/bootstrap-token", r.requireAccount(r.handleNodeBootstrapToken))
	r.mux.HandleFunc("/api/v1/nodes/enroll", r.handleNodeEnroll)
	r.mux.HandleFunc("/api/v1/nodes/exchange", r.handleNodeExchange)
	r.mux.HandleFunc("/api/v1/nodes/approvals", r.requireAccount(r.handleNodeEnrollmentApprovals))
	r.mux.HandleFunc("/api/v1/nodes/approvals/", r.requireAccount(r.handleNodeEnrollmentApprovalByID))
	r.mux.HandleFunc("/api/v1/nodes/scopes", r.requireAccount(r.handleNodeScopes))
	r.mux.HandleFunc("/api/v1/chains", r.requireAccount(r.handleChains))
	r.mux.HandleFunc("/api/v1/chains/validate", r.requireAccount(r.handleChainValidate))
	r.mux.HandleFunc("/api/v1/chains/preview", r.requireAccount(r.handleChainPreview))
	r.mux.HandleFunc("/api/v1/chains/", r.requireAccount(r.handleChainByID))
	r.mux.HandleFunc("/api/v1/route-rules", r.requireAccount(r.handleRouteRules))
	r.mux.HandleFunc("/api/v1/route-rules/match-types", r.requireAccount(r.handleMatchTypes))
	r.mux.HandleFunc("/api/v1/route-rules/validate", r.requireAccount(r.handleRouteRuleValidate))
	r.mux.HandleFunc("/api/v1/route-rules/suggestions", r.requireAccount(r.handleRouteRuleSuggestions))
	r.mux.HandleFunc("/api/v1/route-rules/", r.requireAccount(r.handleRouteRuleByID))
	r.mux.HandleFunc("/api/v1/policies/revisions", r.requireAccount(r.handlePolicyRevisions))
	r.mux.HandleFunc("/api/v1/policies/publish", r.requireAccount(r.handlePolicyPublish))
	r.mux.HandleFunc("/api/v1/certificates", r.requireAccount(r.handleCertificates))
	r.mux.HandleFunc("/api/v1/nodes/health", r.requireAccount(r.handleNodeHealth))
	r.mux.HandleFunc("/api/v1/nodes/health/history", r.requireAccount(r.handleNodeHealthHistory))
	r.mux.HandleFunc("/api/v1/node-agent/policy", r.requireNode(r.handleNodeAgentPolicy))
	r.mux.HandleFunc("/api/v1/node-agent/heartbeat", r.requireNode(r.handleNodeAgentHeartbeat))
	r.mux.HandleFunc("/api/v1/node-agent/cert/renew", r.requireNode(r.handleNodeAgentCertRenew))
	r.mux.HandleFunc("/api/v1/node-agent/transports", r.requireNode(r.handleNodeAgentTransport))
}
