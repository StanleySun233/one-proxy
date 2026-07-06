package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (r *Router) handleRemoteCredentials(w http.ResponseWriter, req *http.Request) {
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		protocol := strings.TrimSpace(req.URL.Query().Get("protocol"))
		writeSuccess(w, http.StatusOK, r.service.RemoteCredentials(account, tenantCtx, protocol))
	case http.MethodPost:
		var payload domain.CreateRemoteCredentialInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateRemoteCredential(account, tenantCtx, payload)
		if err != nil {
			writeServiceError(w, req, err, "remote_credential_create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleRemoteCredentialByID(w http.ResponseWriter, req *http.Request) {
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	credentialID := strings.Trim(strings.TrimPrefix(req.URL.Path, "/api/remote/credentials/"), "/")
	if credentialID == "" {
		writeError(w, http.StatusNotFound, "remote_credential_not_found")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload domain.UpdateRemoteCredentialInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateRemoteCredential(account, tenantCtx, credentialID, payload)
		if err != nil {
			writeServiceError(w, req, err, "remote_credential_update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteRemoteCredential(account, tenantCtx, credentialID); err != nil {
			writeServiceError(w, req, err, "remote_credential_delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]string{"id": credentialID})
	default:
		writeMethodNotAllowed(w, "PATCH, DELETE")
	}
}

func (r *Router) handleRemoteSessions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	var payload domain.RemoteSessionInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	session, err := r.service.CreateRemoteSession(account, tenantCtx, payload)
	if err != nil {
		writeServiceError(w, req, err, "remote_session_create_failed")
		return
	}
	writeSuccess(w, http.StatusCreated, session)
}

func (r *Router) handleRemoteSessionByID(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	rest := strings.Trim(strings.TrimPrefix(req.URL.Path, "/api/remote/sessions/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "tunnel" {
		writeError(w, http.StatusNotFound, "remote_session_not_found")
		return
	}
	r.service.ServeRemoteTunnel(w, req, parts[0], strings.TrimSpace(req.URL.Query().Get("token")))
}
