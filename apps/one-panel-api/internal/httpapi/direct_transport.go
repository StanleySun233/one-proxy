package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (r *Router) handleDirectCandidates(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.DirectCandidatesInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	result, err := r.service.UpsertDirectCandidates(nodeID, payload)
	if err != nil {
		writeServiceError(w, req, err, "direct_candidates_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleDirectLinkPlan(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	result, err := r.service.DirectLinkPlan(nodeID)
	if err != nil {
		writeServiceError(w, req, err, "direct_link_plan_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleDirectStatus(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.DirectStatusInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	result, err := r.service.UpsertDirectStatus(nodeID, payload)
	if err != nil {
		writeServiceError(w, req, err, "direct_status_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}
