package httpapi

import (
	"encoding/json"
	"net/http"
)

type proxyTokenValidateRequest struct {
	TokenHash string `json:"tokenHash"`
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
	result := r.service.ValidateProxyTokenHash(payload.TokenHash)
	if !result.Valid {
		writeError(w, http.StatusUnauthorized, "invalid_proxy_token")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}
