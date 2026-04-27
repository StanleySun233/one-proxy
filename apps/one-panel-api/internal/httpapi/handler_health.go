package httpapi

import (
	"net/http"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (r *Router) handleNodeHealth(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.NodeHealth())
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
	items, err := r.service.NodeHealthHistory(nodeID, window)
	if err != nil {
		writeServiceError(w, req, err, "history_fetch_failed")
		return
	}
	writeSuccess(w, http.StatusOK, items)
}
