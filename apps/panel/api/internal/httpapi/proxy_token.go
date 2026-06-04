package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/auth"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

type proxyTokenValidateRequest struct {
	TokenHash    string `json:"tokenHash"`
	Token        string `json:"token"`
	AccessPathID string `json:"accessPathId"`
}

func (r *Router) handleProxyTokenValidate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload proxyTokenValidateRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	tokenHash := payload.TokenHash
	if tokenHash == "" && payload.Token != "" {
		tokenHash = auth.TokenHash(payload.Token)
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	result := r.service.ValidateProxyTokenHash(tokenHash, nodeID)
	if !result.Valid {
		writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
		return
	}
	if result.ActiveTenantID == nil && len(result.TenantMemberships) == 1 {
		result.ActiveTenantID = &result.TenantMemberships[0].TenantID
	}
	if result.ActiveTenantID == nil && !proxyTokenAllowsTenantlessProxy(result) {
		writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func proxyTokenAllowsTenantlessProxy(result domain.ProxyTokenValidation) bool {
	if result.Account.Role == domain.AccountRoleSuperAdmin {
		return true
	}
	for _, membership := range result.TenantMemberships {
		if membership.Role == domain.TenantRoleAdmin {
			return true
		}
	}
	return false
}
