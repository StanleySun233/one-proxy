package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

type rejectNodePayload struct {
	Reason string `json:"reason"`
}

func (r *Router) handleNodes(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.Nodes())
	case http.MethodPost:
		var payload domain.CreateNodeInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateNode(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleNodeByID(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "/reject") {
		r.handleNodeReject(w, req)
		return
	}
	nodeID := resourceID(req.URL.Path, "/api/v1/nodes/")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "missing_node_id")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload domain.UpdateNodeInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateNode(nodeID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteNode(nodeID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "PATCH, DELETE")
	}
}

func (r *Router) handleNodeConnect(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.ConnectNodeInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	item, err := r.service.ConnectNode(payload)
	if err != nil {
		writeServiceError(w, req, err, "connect_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}

func (r *Router) handleNodeApprove(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	nodeID := resourceID(req.URL.Path, "/api/v1/nodes/approve/")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "missing_node_id")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	item, err := r.service.ApproveNodeEnrollment(nodeID, account.ID)
	if err != nil {
		writeServiceError(w, req, err, "approve_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}

func (r *Router) handleNodeBootstrapToken(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.CreateBootstrapTokenInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	item, err := r.service.CreateBootstrapToken(payload)
	if err != nil {
		writeServiceError(w, req, err, "create_failed")
		return
	}
	writeSuccess(w, http.StatusCreated, item)
}

func (r *Router) handleUnconsumedBootstrapTokens(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.UnconsumedBootstrapTokens())
}

func (r *Router) handleNodeEnroll(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.EnrollNodeInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	item, err := r.service.EnrollNode(payload)
	if err != nil {
		writeServiceError(w, req, err, "enroll_failed")
		return
	}
	writeSuccess(w, http.StatusCreated, item)
}

func (r *Router) handleNodeExchange(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.ExchangeNodeEnrollmentInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	item, err := r.service.ExchangeNodeEnrollment(payload)
	if err != nil {
		writeServiceError(w, req, err, "exchange_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}

func (r *Router) handlePendingNodes(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.PendingNodeEnrollments())
}

func (r *Router) handleNodeReject(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	nodeID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/v1/nodes/"), "/reject")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "missing_node_id")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var payload rejectNodePayload
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if err := r.service.RejectNodeEnrollment(nodeID, account.ID, payload.Reason); err != nil {
		writeServiceError(w, req, err, "reject_failed")
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"status": "rejected"})
}
