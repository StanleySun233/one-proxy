package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (r *Router) handleAccounts(w http.ResponseWriter, req *http.Request) {
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	switch req.Method {
	case http.MethodGet:
		tenantCtx, tenantOK := tenantAuthContextFromContext(req.Context())
		if account.Role != domain.AccountRoleSuperAdmin && (!tenantOK || tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin) {
			writeError(w, http.StatusForbidden, "account_role_forbidden")
			return
		}
		writeSuccess(w, http.StatusOK, r.service.Accounts())
	case http.MethodPost:
		if account.Role != domain.AccountRoleSuperAdmin {
			writeError(w, http.StatusForbidden, "account_role_forbidden")
			return
		}
		var payload domain.CreateAccountInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateAccount(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleAccountByID(w http.ResponseWriter, req *http.Request) {
	accountID := resourceID(req.URL.Path, "/api/accounts/")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "missing_account_id")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	if account.MustRotatePassword && account.ID != accountID {
		writeError(w, http.StatusForbidden, "password_rotation_required")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload domain.UpdateAccountInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if account.Role != domain.AccountRoleSuperAdmin {
			if account.ID != accountID || payload.Password == "" || payload.Role != "" || payload.Status != "" {
				writeError(w, http.StatusForbidden, "account_role_forbidden")
				return
			}
		}
		item, err := r.service.UpdateAccount(accountID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if account.Role != domain.AccountRoleSuperAdmin {
			writeError(w, http.StatusForbidden, "account_role_forbidden")
			return
		}
		if err := r.service.DeleteAccount(accountID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "PATCH, DELETE")
	}
}
