package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/domain"
)

func (r *Router) handleNodeAgentPolicy(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	item, ok := r.service.NodeAgentPolicy(nodeID)
	if !ok {
		writeError(w, http.StatusNotFound, "policy_not_found")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}

func (r *Router) handleNodeAgentHeartbeat(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.NodeHeartbeatInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	payload.NodeID = nodeID
	item, err := r.service.UpsertNodeHeartbeat(payload)
	if err != nil {
		writeServiceError(w, req, err, "heartbeat_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}

func (r *Router) handleNodeAgentCertRenew(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.NodeCertRenewInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	nodeID, ok := nodeIDFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_node_token")
		return
	}
	payload.NodeID = nodeID
	item, err := r.service.RenewNodeCertificate(payload)
	if err != nil {
		writeServiceError(w, req, err, "renew_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}
