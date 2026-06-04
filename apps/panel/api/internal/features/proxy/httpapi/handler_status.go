package proxyhttpapi

import (
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/httpctx"
)

func (r *Router) handleExtensionPageStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	host := req.URL.Query().Get("host")
	if host == "" {
		writeError(w, http.StatusBadRequest, "missing_host")
		return
	}
	writeSuccess(w, http.StatusOK, r.status.ExtensionPageStatus(tenantCtx, domain.ProxyPageStatusQuery{
		Host:    host,
		RouteID: req.URL.Query().Get("routeId"),
		ChainID: req.URL.Query().Get("chainId"),
	}))
}
