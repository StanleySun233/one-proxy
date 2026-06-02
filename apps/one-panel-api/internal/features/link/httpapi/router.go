package linkhttpapi

import (
	"net/http"
	"strings"

	linkservice "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/link/service"
)

type AccountGuard func(http.HandlerFunc) http.HandlerFunc

type Router struct {
	mux     *http.ServeMux
	service *linkservice.Service
	guard   AccountGuard
}

func Register(mux *http.ServeMux, guard AccountGuard, service *linkservice.Service) {
	router := &Router{
		mux:     mux,
		service: service,
		guard:   guard,
	}
	router.routes()
}

func (r *Router) routes() {
	r.mux.HandleFunc("/api/v1/chains/scopes", r.guard(r.handleScopes))
	r.mux.HandleFunc("/api/v1/chains/scopes/", r.guard(r.handleScopeByID))
	r.mux.HandleFunc("/api/v1/chains/node-links", r.guard(r.handleNodeLinks))
	r.mux.HandleFunc("/api/v1/chains/node-links/", r.guard(r.handleNodeLinkByID))
	r.mux.HandleFunc("/api/v1/chains", r.guard(r.handleChains))
	r.mux.HandleFunc("/api/v1/chains/validate", r.guard(r.handleChainValidate))
	r.mux.HandleFunc("/api/v1/chains/preview", r.guard(r.handleChainPreview))
	r.mux.HandleFunc("/api/v1/chains/", r.guard(r.handleChainByID))
	r.mux.HandleFunc("/api/v1/chains/routes", r.guard(r.handleRouteRules))
	r.mux.HandleFunc("/api/v1/chains/routes/validate", r.guard(r.handleRouteRuleValidate))
	r.mux.HandleFunc("/api/v1/chains/routes/suggestions", r.guard(r.handleRouteRuleSuggestions))
	r.mux.HandleFunc("/api/v1/chains/routes/", r.guard(r.handleRouteRuleByID))
}

func resourceID(path string, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}
