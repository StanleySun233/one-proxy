package linkhttpapi

import (
	"encoding/json"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/link/domain"
	"net/http"
	"strings"
)

func (r *Router) handleScopes(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.Scopes())
	case http.MethodPost:
		var payload link.CreateScopeInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request_body")
			return
		}
		item, err := r.service.CreateScope(payload)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

func (r *Router) handleScopeByID(w http.ResponseWriter, req *http.Request) {
	scopeID := strings.TrimPrefix(req.URL.Path, "/api/v1/chains/scopes/")
	if scopeID == "" {
		writeError(w, http.StatusNotFound, "scope_not_found")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload link.UpdateScopeInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request_body")
			return
		}
		item, err := r.service.UpdateScope(scopeID, payload)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteScope(scopeID); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeSuccess(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		w.Header().Set("Allow", "PATCH, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}
