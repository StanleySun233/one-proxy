package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (r *Router) handleGroups(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		groups, err := r.service.ListGroups()
		if err != nil {
			writeServiceError(w, req, err, "list_failed")
			return
		}
		writeSuccess(w, http.StatusOK, groups)
	case http.MethodPost:
		var payload domain.CreateGroupInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateGroup(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleGroupByID(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "/accounts") {
		r.handleGroupAccounts(w, req)
		return
	}
	if strings.HasSuffix(req.URL.Path, "/scopes") {
		r.handleGroupScopes(w, req)
		return
	}
	groupID := resourceID(req.URL.Path, "/api/v1/groups/")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "missing_group_id")
		return
	}
	switch req.Method {
	case http.MethodGet:
		item, err := r.service.GetGroup(groupID)
		if err != nil {
			writeServiceError(w, req, err, "get_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodPut:
		var payload domain.UpdateGroupInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateGroup(groupID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteGroup(groupID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "GET, PUT, DELETE")
	}
}

func (r *Router) handleGroupAccounts(w http.ResponseWriter, req *http.Request) {
	groupID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/v1/groups/"), "/accounts")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "missing_group_id")
		return
	}
	switch req.Method {
	case http.MethodGet:
		items, err := r.service.ListGroupAccounts(groupID)
		if err != nil {
			writeServiceError(w, req, err, "list_failed")
			return
		}
		writeSuccess(w, http.StatusOK, items)
	case http.MethodPut:
		var payload domain.SetGroupAccountsInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if err := r.service.SetGroupAccounts(groupID, payload); err != nil {
			writeServiceError(w, req, err, "set_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "updated"})
	default:
		writeMethodNotAllowed(w, "GET, PUT")
	}
}

func (r *Router) handleGroupScopes(w http.ResponseWriter, req *http.Request) {
	groupID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/v1/groups/"), "/scopes")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "missing_group_id")
		return
	}
	if req.Method != http.MethodPut {
		writeMethodNotAllowed(w, "PUT")
		return
	}
	var payload domain.SetGroupScopesInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if err := r.service.SetGroupScopes(groupID, payload); err != nil {
		writeServiceError(w, req, err, "set_failed")
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"status": "updated"})
}
