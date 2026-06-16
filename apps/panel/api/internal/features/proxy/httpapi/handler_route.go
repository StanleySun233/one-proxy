package proxyhttpapi

import (
	"encoding/json"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/httpctx"
	"net/http"
	"strings"
)

func (r *Router) handleRouteRuleGroups(w http.ResponseWriter, req *http.Request) {
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.RouteRuleGroups(tenantCtx))
	case http.MethodPost:
		var payload proxy.CreateRouteRuleGroupInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateRouteRuleGroup(tenantCtx, payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
			Action:       "proxy.route_group.create",
			ResourceType: "route_rule_group",
			ResourceID:   item.ID,
			ResourceName: item.Name,
		})
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleRouteRuleGroupByID(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "/delete-impact") {
		r.handleRouteRuleGroupDeleteImpact(w, req)
		return
	}
	groupID := resourceID(req.URL.Path, "/api/proxy/route-groups/")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "missing_route_rule_group_id")
		return
	}
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		item, err := r.service.GetRouteRuleGroup(tenantCtx, groupID)
		if err != nil {
			writeServiceError(w, req, err, "get_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodPatch:
		var payload proxy.UpdateRouteRuleGroupInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateRouteRuleGroup(tenantCtx, groupID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
			Action:       "proxy.route_group.update",
			ResourceType: "route_rule_group",
			ResourceID:   item.ID,
			ResourceName: item.Name,
		})
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteRouteRuleGroup(tenantCtx, groupID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
			Action:       "proxy.route_group.delete",
			ResourceType: "route_rule_group",
			ResourceID:   groupID,
		})
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "GET, PATCH, DELETE")
	}
}

func (r *Router) handleRouteRuleGroupDeleteImpact(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	groupID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/proxy/route-groups/"), "/delete-impact")
	groupID = strings.TrimSuffix(groupID, "/")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "missing_route_rule_group_id")
		return
	}
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	result, err := r.service.RouteRuleGroupDeleteImpact(tenantCtx, groupID)
	if err != nil {
		writeServiceError(w, req, err, "delete_impact_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleRouteRules(w http.ResponseWriter, req *http.Request) {
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		includeDetails := req.URL.Query().Get("details") == "true"
		if includeDetails {
			writeSuccess(w, http.StatusOK, r.service.RouteRulesWithDetails(tenantCtx))
		} else {
			writeSuccess(w, http.StatusOK, r.service.RouteRules(tenantCtx))
		}
	case http.MethodPost:
		var payload proxy.CreateRouteRuleInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateRouteRule(tenantCtx, payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
			Action:       "proxy.route.create",
			ResourceType: "route",
			ResourceID:   item.ID,
			ResourceName: item.MatchValue,
		})
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleRouteRuleByID(w http.ResponseWriter, req *http.Request) {
	ruleID := resourceID(req.URL.Path, "/api/proxy/routes/")
	if ruleID == "" {
		writeError(w, http.StatusBadRequest, "missing_rule_id")
		return
	}
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		item, err := r.service.GetRouteRule(tenantCtx, ruleID)
		if err != nil {
			writeServiceError(w, req, err, "get_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodPatch:
		var payload proxy.UpdateRouteRuleInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateRouteRule(tenantCtx, ruleID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
			Action:       "proxy.route.update",
			ResourceType: "route",
			ResourceID:   item.ID,
			ResourceName: item.MatchValue,
		})
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteRouteRule(tenantCtx, ruleID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
			Action:       "proxy.route.delete",
			ResourceType: "route",
			ResourceID:   ruleID,
		})
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "GET, PATCH, DELETE")
	}
}

func (r *Router) handleRouteRuleValidate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	var payload proxy.ValidateRouteRuleInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	result, err := r.service.ValidateRouteRule(tenantCtx, payload)
	if err != nil {
		writeServiceError(w, req, err, "validation_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleRouteRuleSuggestions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	matchType := req.URL.Query().Get("match_type")
	if matchType == "" {
		writeError(w, http.StatusBadRequest, "missing_match_type")
		return
	}
	query := req.URL.Query().Get("query")
	result := r.service.RouteRuleSuggestions(tenantCtx, matchType, query)
	writeSuccess(w, http.StatusOK, result)
}
