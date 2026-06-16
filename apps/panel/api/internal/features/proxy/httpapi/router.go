package proxyhttpapi

import (
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxyservice "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/service"
)

type AccountGuard func(http.HandlerFunc) http.HandlerFunc

type StatusService interface {
	ExtensionPageStatus(domain.TenantAuthContext, domain.ProxyPageStatusQuery) domain.ProxyPageStatus
}

type AuditService interface {
	RecordBusinessAuditEvent(domain.CreateBusinessAuditEventInput) (domain.BusinessAuditEvent, error)
}

type Router struct {
	mux     *http.ServeMux
	service *proxyservice.Service
	status  StatusService
	audit   AuditService
	guard   AccountGuard
}

func Register(mux *http.ServeMux, guard AccountGuard, service *proxyservice.Service, status StatusService, audit AuditService) {
	router := &Router{
		mux:     mux,
		service: service,
		status:  status,
		audit:   audit,
		guard:   guard,
	}
	router.routes()
}

func (r *Router) routes() {
	r.mux.HandleFunc("/api/proxy/paths", r.guard(r.handleAccessPaths))
	r.mux.HandleFunc("/api/proxy/paths/", r.guard(r.handleAccessPathByID))
	r.mux.HandleFunc("/api/proxy/scopes", r.guard(r.handleScopes))
	r.mux.HandleFunc("/api/proxy/scopes/", r.guard(r.handleScopeByID))
	r.mux.HandleFunc("/api/proxy/links", r.guard(r.handleNodeLinks))
	r.mux.HandleFunc("/api/proxy/links/", r.guard(r.handleNodeLinkByID))
	r.mux.HandleFunc("/api/proxy/extension/page/status", r.guard(r.handleExtensionPageStatus))
	r.mux.HandleFunc("/api/proxy/route-groups", r.guard(r.handleRouteRuleGroups))
	r.mux.HandleFunc("/api/proxy/route-groups/", r.guard(r.handleRouteRuleGroupByID))
	r.mux.HandleFunc("/api/proxy", r.guard(r.handleChains))
	r.mux.HandleFunc("/api/proxy/validate", r.guard(r.handleChainValidate))
	r.mux.HandleFunc("/api/proxy/preview", r.guard(r.handleChainPreview))
	r.mux.HandleFunc("/api/proxy/", r.guard(r.handleChainByID))
	r.mux.HandleFunc("/api/proxy/routes", r.guard(r.handleRouteRules))
	r.mux.HandleFunc("/api/proxy/routes/validate", r.guard(r.handleRouteRuleValidate))
	r.mux.HandleFunc("/api/proxy/routes/suggestions", r.guard(r.handleRouteRuleSuggestions))
	r.mux.HandleFunc("/api/proxy/routes/", r.guard(r.handleRouteRuleByID))
}

func resourceID(path string, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}
