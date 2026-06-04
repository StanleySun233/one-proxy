package httpapi

import (
	"encoding/json"
	"net/http"
)

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	var payload loginRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	result, ok := r.service.Login(payload.Account, payload.Password)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_credentials")
		return
	}

	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleRefresh(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	refreshToken := bearerToken(req)
	if refreshToken == "" {
		var payload struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		refreshToken = payload.RefreshToken
	}

	result, ok := r.service.RefreshSession(refreshToken)
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_refresh_token")
		return
	}

	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleLogout(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}

	accessToken := bearerToken(req)
	if !r.service.Logout(accessToken) {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}

	writeSuccess(w, http.StatusOK, map[string]any{"status": "logged_out"})
}
