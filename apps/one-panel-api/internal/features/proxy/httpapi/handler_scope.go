package proxyhttpapi

import (
	"encoding/json"
	proxy "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/proxy/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/httpctx"
	"net/http"
	"strings"
)

func (r *Router) handleScopes(w http.ResponseWriter, req *http.Request) {
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.Scopes(tenantCtx))
	case http.MethodPost:
		var payload proxy.CreateScopeInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request_body")
			return
		}
		item, err := r.service.CreateScope(tenantCtx, payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}

func (r *Router) handleScopeByID(w http.ResponseWriter, req *http.Request) {
	scopeID := strings.TrimPrefix(req.URL.Path, "/api/proxy/scopes/")
	if scopeID == "" {
		writeError(w, http.StatusNotFound, "scope_not_found")
		return
	}
	tenantCtx, ok := httpctx.TenantAuth(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload proxy.UpdateScopeInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request_body")
			return
		}
		item, err := r.service.UpdateScope(tenantCtx, scopeID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteScope(tenantCtx, scopeID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		w.Header().Set("Allow", "PATCH, DELETE")
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
	}
}
