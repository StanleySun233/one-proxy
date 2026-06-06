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

func (r *Router) handleAuditBusinessEvents(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	query, ok := businessAuditQueryFromRequest(w, req)
	if !ok {
		return
	}
	result, err := r.service.BusinessAuditEvents(tenantCtx, query)
	if err != nil {
		writeServiceError(w, req, err, "audit_business_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleAuditNetworkSessions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	query, ok := networkAuditQueryFromRequest(w, req)
	if !ok {
		return
	}
	result, err := r.service.NetworkAuditSessions(tenantCtx, query)
	if err != nil {
		writeServiceError(w, req, err, "audit_network_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleAuditDashboard(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	values := req.URL.Query()
	from, ok := parseAuditTime(w, values.Get("from"), "invalid_from")
	if !ok {
		return
	}
	to, ok := parseAuditTime(w, values.Get("to"), "invalid_to")
	if !ok {
		return
	}
	result, err := r.service.AuditDashboard(tenantCtx, domain.AuditDashboardQuery{
		TenantID: values.Get("tenantId"),
		From:     from,
		To:       to,
	})
	if err != nil {
		writeServiceError(w, req, err, "audit_dashboard_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
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

func businessAuditQueryFromRequest(w http.ResponseWriter, req *http.Request) (domain.BusinessAuditQuery, bool) {
	values := req.URL.Query()
	query := domain.BusinessAuditQuery{
		TenantID:     values.Get("tenantId"),
		ActorID:      values.Get("actorId"),
		ActorType:    values.Get("actorType"),
		ResourceType: values.Get("resourceType"),
		ResourceID:   values.Get("resourceId"),
		Action:       values.Get("action"),
		Outcome:      values.Get("outcome"),
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 0 {
			writeError(w, http.StatusBadRequest, "invalid_limit")
			return domain.BusinessAuditQuery{}, false
		}
		query.Limit = limit
	}
	from, ok := parseAuditTime(w, values.Get("from"), "invalid_from")
	if !ok {
		return domain.BusinessAuditQuery{}, false
	}
	to, ok := parseAuditTime(w, values.Get("to"), "invalid_to")
	if !ok {
		return domain.BusinessAuditQuery{}, false
	}
	query.From = from
	query.To = to
	return query, true
}

func networkAuditQueryFromRequest(w http.ResponseWriter, req *http.Request) (domain.NetworkAuditQuery, bool) {
	values := req.URL.Query()
	query := domain.NetworkAuditQuery{
		TenantID:       values.Get("tenantId"),
		ActorID:        values.Get("actorId"),
		TokenID:        values.Get("tokenId"),
		NodeID:         values.Get("nodeId"),
		TargetHost:     values.Get("targetHost"),
		RouteID:        values.Get("routeId"),
		ScopeID:        values.Get("scopeId"),
		ChainID:        values.Get("chainId"),
		DenyReason:     values.Get("denyReason"),
		PolicyRevision: values.Get("policyRevision"),
		MatchedRuleID:  values.Get("matchedRuleId"),
		DecisionSource: values.Get("decisionSource"),
		Decision:       values.Get("decision"),
	}
	if raw := values.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 0 {
			writeError(w, http.StatusBadRequest, "invalid_limit")
			return domain.NetworkAuditQuery{}, false
		}
		query.Limit = limit
	}
	from, ok := parseAuditTime(w, values.Get("from"), "invalid_from")
	if !ok {
		return domain.NetworkAuditQuery{}, false
	}
	to, ok := parseAuditTime(w, values.Get("to"), "invalid_to")
	if !ok {
		return domain.NetworkAuditQuery{}, false
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
