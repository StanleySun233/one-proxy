package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (r *Router) handleRouteRules(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		includeDetails := req.URL.Query().Get("details") == "true"
		if includeDetails {
			writeSuccess(w, http.StatusOK, r.service.RouteRulesWithDetails())
		} else {
			writeSuccess(w, http.StatusOK, r.service.RouteRules())
		}
	case http.MethodPost:
		var payload domain.CreateRouteRuleInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateRouteRule(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleRouteRuleByID(w http.ResponseWriter, req *http.Request) {
	ruleID := resourceID(req.URL.Path, "/api/v1/route-rules/")
	if ruleID == "" {
		writeError(w, http.StatusBadRequest, "missing_rule_id")
		return
	}
	switch req.Method {
	case http.MethodGet:
		item, err := r.service.GetRouteRule(ruleID)
		if err != nil {
			writeServiceError(w, req, err, "get_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodPatch:
		var payload domain.UpdateRouteRuleInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateRouteRule(ruleID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteRouteRule(ruleID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
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
	var payload domain.ValidateRouteRuleInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	result, err := r.service.ValidateRouteRule(payload)
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
	matchType := req.URL.Query().Get("match_type")
	if matchType == "" {
		writeError(w, http.StatusBadRequest, "missing_match_type")
		return
	}
	query := req.URL.Query().Get("query")
	result := r.service.RouteRuleSuggestions(matchType, query)
	writeSuccess(w, http.StatusOK, result)
}
