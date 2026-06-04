package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (r *Router) handleAuditProxySessions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	query, ok := auditQueryFromRequest(w, req)
	if !ok {
		return
	}
	writeSuccess(w, http.StatusOK, r.service.AuditProxySessions(tenantCtx, query))
}

func (r *Router) handleAuditProxyEvents(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	query, ok := auditQueryFromRequest(w, req)
	if !ok {
		return
	}
	writeSuccess(w, http.StatusOK, r.service.AuditProxyEvents(tenantCtx, query))
}

func auditQueryFromRequest(w http.ResponseWriter, req *http.Request) (domain.ProxyAuditQuery, bool) {
	values := req.URL.Query()
	query := domain.ProxyAuditQuery{
		Host:    values.Get("host"),
		ChainID: values.Get("chainId"),
		RouteID: values.Get("routeId"),
		NodeID:  values.Get("nodeId"),
		Status:  values.Get("status"),
		Level:   values.Get("level"),
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 0 {
			writeError(w, http.StatusBadRequest, "invalid_limit")
			return domain.ProxyAuditQuery{}, false
		}
		query.Limit = limit
	}
	from, ok := parseAuditTime(w, values.Get("from"), "invalid_from")
	if !ok {
		return domain.ProxyAuditQuery{}, false
	}
	to, ok := parseAuditTime(w, values.Get("to"), "invalid_to")
	if !ok {
		return domain.ProxyAuditQuery{}, false
	}
	query.From = from
	query.To = to
	return query, true
}

func parseAuditTime(w http.ResponseWriter, raw string, code string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, true
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, code)
		return time.Time{}, false
	}
	return value, true
}
