package httpapi

import (
	"net/http"
	"time"
)

func (r *Router) handleNodeHealth(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.NodeHealth(tenantCtx))
}

func (r *Router) handleNodeHealthHistory(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	nodeID := req.URL.Query().Get("nodeId")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "missing_node_id")
		return
	}
	windowStr := req.URL.Query().Get("window")
	window := 24 * time.Hour
	if windowStr != "" {
		if parsed, err := time.ParseDuration(windowStr); err == nil {
			window = parsed
		}
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	items, err := r.service.NodeHealthHistory(tenantCtx, nodeID, window)
	if err != nil {
		writeServiceError(w, req, err, "history_fetch_failed")
		return
	}
	writeSuccess(w, http.StatusOK, items)
}
