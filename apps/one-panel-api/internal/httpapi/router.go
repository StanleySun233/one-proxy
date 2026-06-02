package httpapi

import (
	"net/http"

	linkhttpapi "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/link/httpapi"
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
	HTTPAddr    string
	DBBackend   string
	EnvFilePath string
}

func (r *Router) routes(cfg HTTPConfig) {
	r.mux.HandleFunc("/api/v1/setup/status", r.handleSetupStatus)
	r.mux.HandleFunc("/healthz", r.handleHealthz(cfg))

	r.mux.HandleFunc("/api/v1/enums", r.handleEnums)
	r.mux.HandleFunc("/api/v1/auth/login", r.handleLogin)
	r.mux.HandleFunc("/api/v1/auth/refresh", r.handleRefresh)
	r.mux.HandleFunc("/api/v1/auth/logout", r.requireAccount(r.handleLogout))
	r.mux.HandleFunc("/api/v1/extension/bootstrap", r.requireAccount(r.handleExtensionBootstrap))
	r.mux.HandleFunc("/api/v1/overview", r.requireAccount(r.handleOverview))
	r.mux.HandleFunc("/api/v1/accounts", r.requireAccount(r.handleAccounts))
	r.mux.HandleFunc("/api/v1/accounts/", r.requireAccount(r.handleAccountByID))
	r.mux.HandleFunc("/api/v1/groups", r.requireAccount(r.handleGroups))
	r.mux.HandleFunc("/api/v1/groups/", r.requireAccount(r.handleGroupByID))
	r.mux.HandleFunc("/api/v1/nodes", r.requireAccount(r.handleNodes))
	r.mux.HandleFunc("/api/v1/nodes/", r.requireAccount(r.handleNodeByID))
	r.mux.HandleFunc("/api/v1/node-transports", r.requireAccount(r.handleNodeTransports))
	r.mux.HandleFunc("/api/v1/nodes/approve/", r.requireAccount(r.handleNodeApprove))
	r.mux.HandleFunc("/api/v1/nodes/bootstrap-token", r.requireAccount(r.handleNodeBootstrapToken))
	r.mux.HandleFunc("/api/v1/nodes/bootstrap-tokens/unconsumed", r.requireAccount(r.handleUnconsumedBootstrapTokens))
	r.mux.HandleFunc("/api/v1/nodes/bootstrap-tokens/", r.requireAccount(r.handleBootstrapTokenByID))
	r.mux.HandleFunc("/api/v1/nodes/enroll", r.handleNodeEnroll)
	r.mux.HandleFunc("/api/v1/nodes/exchange", r.handleNodeExchange)
	r.mux.HandleFunc("/api/v1/nodes/pending", r.requireAccount(r.handlePendingNodes))
	r.mux.HandleFunc("/api/v1/policies/revisions", r.requireAccount(r.handlePolicyRevisions))
	r.mux.HandleFunc("/api/v1/policies/publish", r.requireAccount(r.handlePolicyPublish))
	r.mux.HandleFunc("/api/v1/nodes/health", r.requireAccount(r.handleNodeHealth))
	r.mux.HandleFunc("/api/v1/nodes/health/history", r.requireAccount(r.handleNodeHealthHistory))
	r.mux.HandleFunc("/api/v1/node-agent/policy", r.requireNode(r.handleNodeAgentPolicy))
	r.mux.HandleFunc("/api/v1/node-agent/heartbeat", r.requireNode(r.handleNodeAgentHeartbeat))
	r.mux.HandleFunc("/api/v1/node-agent/cert/renew", r.requireNode(r.handleNodeAgentCertRenew))
	r.mux.HandleFunc("/api/v1/node-agent/transports", r.requireNode(r.handleNodeAgentTransport))
	r.mux.HandleFunc("/api/v1/node-agent/proxy-token/validate", r.requireNode(r.handleProxyTokenValidate))
	linkhttpapi.Register(r.mux, r.requireAccount, r.service.Link())
}
