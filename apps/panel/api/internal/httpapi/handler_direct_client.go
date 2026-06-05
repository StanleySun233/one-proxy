package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

func (r *Router) handleClientDirectSession(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	var payload domain.ClientDirectSessionInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	result, err := r.service.ClientDirectSession(tenantCtx, payload)
	if err != nil {
		writeServiceError(w, req, err, "direct_session_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}
