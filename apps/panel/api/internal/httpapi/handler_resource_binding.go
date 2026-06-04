package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (r *Router) handleGrants(w http.ResponseWriter, req *http.Request) {
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	tenantCtx, _ := tenantAuthContextFromContext(req.Context())
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	resourceType := domain.ResourceType(req.URL.Query().Get("resourceType"))
	resourceID := strings.TrimSpace(req.URL.Query().Get("resourceId"))
	items, err := r.service.ResourceBindings(account, tenantCtx, resourceType, resourceID)
	if err != nil {
		writeServiceError(w, req, err, "resource_bindings_failed")
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"bindings": items})
}

func (r *Router) handleGrantTenants(w http.ResponseWriter, req *http.Request) {
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	tenantCtx, _ := tenantAuthContextFromContext(req.Context())
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	items, err := r.service.GrantTenants(account, tenantCtx)
	if err != nil {
		writeServiceError(w, req, err, "grant_tenants_failed")
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"tenants": items})
}

func (r *Router) handleGrantByID(w http.ResponseWriter, req *http.Request) {
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	tenantCtx, _ := tenantAuthContextFromContext(req.Context())
	resourceType, resourceID, tenantID := resourceBindingPath(req.URL.Path)
	if resourceType == "" || resourceID == "" || tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_resource_binding_path")
		return
	}
	switch req.Method {
	case http.MethodPut:
		var payload domain.UpsertTenantResourceBindingInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpsertResourceBinding(account, tenantCtx, domain.ResourceType(resourceType), resourceID, tenantID, payload)
		if err != nil {
			writeServiceError(w, req, err, "resource_binding_upsert_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"binding": item})
	case http.MethodDelete:
		if err := r.service.DeleteResourceBinding(account, tenantCtx, domain.ResourceType(resourceType), resourceID, tenantID); err != nil {
			writeServiceError(w, req, err, "resource_binding_delete_failed")
			return
		}
		writeSuccess[any](w, http.StatusOK, nil)
	default:
		writeMethodNotAllowed(w, "PUT, DELETE")
	}
}

func resourceBindingPath(path string) (string, string, string) {
	raw := strings.Trim(resourceID(path, "/api/grants/"), "/")
	if raw == "" {
		return "", "", ""
	}
	parts := strings.Split(raw, "/")
	if len(parts) != 3 {
		return "", "", ""
	}
	return parts[0], parts[1], parts[2]
}
