package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (r *Router) handleNodeAccessPaths(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.NodeAccessPaths())
	case http.MethodPost:
		var payload domain.CreateNodeAccessPathInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateNodeAccessPath(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleNodeAccessPathByID(w http.ResponseWriter, req *http.Request) {
	pathID := resourceID(req.URL.Path, "/api/v1/node-access-paths/")
	if pathID == "" {
		writeError(w, http.StatusBadRequest, "missing_path_id")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload domain.UpdateNodeAccessPathInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateNodeAccessPath(pathID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteNodeAccessPath(pathID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "PATCH, DELETE")
	}
}
