package proxyhttpapi

import (
	"net/http"
	"strings"

	proxyservice "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/proxy/service"
)

type AccountGuard func(http.HandlerFunc) http.HandlerFunc

type Router struct {
	mux     *http.ServeMux
	service *proxyservice.Service
	guard   AccountGuard
}

func Register(mux *http.ServeMux, guard AccountGuard, service *proxyservice.Service) {
	router := &Router{
		mux:     mux,
		service: service,
		guard:   guard,
	}
	router.routes()
}

func (r *Router) routes() {
	r.mux.HandleFunc("/api/v1/proxy/access-paths", r.guard(r.handleAccessPaths))
	r.mux.HandleFunc("/api/v1/proxy/access-paths/", r.guard(r.handleAccessPathByID))
	r.mux.HandleFunc("/api/v1/proxy/scopes", r.guard(r.handleScopes))
	r.mux.HandleFunc("/api/v1/proxy/scopes/", r.guard(r.handleScopeByID))
	r.mux.HandleFunc("/api/v1/proxy/node-links", r.guard(r.handleNodeLinks))
	r.mux.HandleFunc("/api/v1/proxy/node-links/", r.guard(r.handleNodeLinkByID))
	r.mux.HandleFunc("/api/v1/proxy", r.guard(r.handleChains))
	r.mux.HandleFunc("/api/v1/proxy/validate", r.guard(r.handleChainValidate))
	r.mux.HandleFunc("/api/v1/proxy/preview", r.guard(r.handleChainPreview))
	r.mux.HandleFunc("/api/v1/proxy/", r.guard(r.handleChainByID))
	r.mux.HandleFunc("/api/v1/proxy/routes", r.guard(r.handleRouteRules))
	r.mux.HandleFunc("/api/v1/proxy/routes/validate", r.guard(r.handleRouteRuleValidate))
	r.mux.HandleFunc("/api/v1/proxy/routes/suggestions", r.guard(r.handleRouteRuleSuggestions))
	r.mux.HandleFunc("/api/v1/proxy/routes/", r.guard(r.handleRouteRuleByID))
}

func resourceID(path string, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}
