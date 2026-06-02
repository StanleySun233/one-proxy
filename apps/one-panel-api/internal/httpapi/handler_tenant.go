package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/service"
)

func (r *Router) handleTenants(w http.ResponseWriter, req *http.Request) {
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, map[string]any{"tenants": r.service.Tenants(account)})
	case http.MethodPost:
		var payload service.CreateTenantInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateTenant(account, payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleTenantByID(w http.ResponseWriter, req *http.Request) {
	tenantID, suffix := tenantPath(req.URL.Path)
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "missing_tenant_id")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok && account.Role != domain.AccountRoleSuperAdmin {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	if suffix == "" {
		r.handleTenantDetail(w, req, account, tenantCtx, tenantID)
		return
	}
	if suffix == "memberships" || strings.HasPrefix(suffix, "memberships/") {
		r.handleTenantMemberships(w, req, account, tenantCtx, tenantID, strings.TrimPrefix(suffix, "memberships"))
		return
	}
	writeError(w, http.StatusNotFound, "resource_not_found")
}

func (r *Router) handleTenantDetail(w http.ResponseWriter, req *http.Request, account domain.Account, tenantCtx domain.TenantAuthContext, tenantID string) {
	switch req.Method {
	case http.MethodGet:
		item, err := r.service.Tenant(account, tenantCtx, tenantID)
		if err != nil {
			writeServiceError(w, req, err, "tenant_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"tenant": item})
	case http.MethodPatch:
		var payload service.UpdateTenantInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateTenant(account, tenantCtx, tenantID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"tenant": item})
	case http.MethodDelete:
		if err := r.service.DeleteTenant(account, tenantCtx, tenantID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess[any](w, http.StatusOK, nil)
	default:
		writeMethodNotAllowed(w, "GET, PATCH, DELETE")
	}
}

func (r *Router) handleTenantMemberships(w http.ResponseWriter, req *http.Request, account domain.Account, tenantCtx domain.TenantAuthContext, tenantID string, suffix string) {
	accountID := strings.TrimPrefix(suffix, "/")
	if accountID == "" {
		if req.Method != http.MethodGet {
			writeMethodNotAllowed(w, "GET")
			return
		}
		items, err := r.service.TenantMembers(account, tenantCtx, tenantID)
		if err != nil {
			writeServiceError(w, req, err, "memberships_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"memberships": items})
		return
	}
	switch req.Method {
	case http.MethodPut:
		var payload service.UpsertTenantMembershipInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpsertTenantMembership(account, tenantCtx, tenantID, accountID, payload)
		if err != nil {
			writeServiceError(w, req, err, "membership_upsert_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"membership": item})
	case http.MethodDelete:
		if err := r.service.DeleteTenantMembership(account, tenantCtx, tenantID, accountID); err != nil {
			writeServiceError(w, req, err, "membership_delete_failed")
			return
		}
		writeSuccess[any](w, http.StatusOK, nil)
	default:
		writeMethodNotAllowed(w, "PUT, DELETE")
	}
}

func tenantPath(path string) (string, string) {
	raw := strings.Trim(resourceID(path, "/api/v1/tenants/"), "/")
	if raw == "" {
		return "", ""
	}
	parts := strings.SplitN(raw, "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
