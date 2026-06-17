package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

type rejectNodePayload struct {
	Reason string `json:"reason"`
}

func (r *Router) handleNodes(w http.ResponseWriter, req *http.Request) {
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.Nodes(tenantCtx))
	default:
		writeMethodNotAllowed(w, "GET")
	}
}

func (r *Router) handleNodeByID(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "/reject") {
		r.handleNodeReject(w, req)
		return
	}
	if strings.HasSuffix(req.URL.Path, "/access/manage") {
		r.handleNodeManageAccess(w, req)
		return
	}
	if strings.HasSuffix(req.URL.Path, "/delete-impact") {
		r.handleNodeDeleteImpact(w, req)
		return
	}
	if strings.HasSuffix(req.URL.Path, "/approve") {
		r.handleNodeApprove(w, req)
		return
	}
	nodeID := resourceID(req.URL.Path, "/api/nodes/")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "missing_node_id")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload domain.UpdateNodeInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateNode(tenantCtx, nodeID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
			Action:       "node.update",
			ResourceType: "node",
			ResourceID:   item.ID,
			ResourceName: item.Name,
		})
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteNode(tenantCtx, nodeID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
			Action:       "node.delete",
			ResourceType: "node",
			ResourceID:   nodeID,
		})
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "PATCH, DELETE")
	}
}

func (r *Router) handleNodeManageAccess(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	nodeID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/nodes/"), "/access/manage")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "missing_node_id")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	result := r.service.NodeManageAccess(tenantCtx, nodeID)
	if !result.Allowed {
		writeError(w, http.StatusForbidden, result.Reason)
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleNodeDeleteImpact(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	nodeID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/nodes/"), "/delete-impact")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "missing_node_id")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	result, err := r.service.NodeDeleteImpact(tenantCtx, nodeID)
	if err != nil {
		writeServiceError(w, req, err, "delete_impact_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleNodeApprove(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	nodeID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/nodes/"), "/approve")
	if nodeID == "" {
		writeError(w, http.StatusBadRequest, "missing_node_id")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	item, err := r.service.ApproveNodeEnrollment(tenantCtx, nodeID, account.ID)
	if err != nil {
		writeServiceError(w, req, err, "approve_failed")
		return
	}
	r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
		Action:       "node.approve",
		ResourceType: "node",
		ResourceID:   nodeID,
		ResourceName: item.Node.Name,
	})
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
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	item, err := r.service.CreateBootstrapToken(tenantCtx, payload)
	if err != nil {
		writeServiceError(w, req, err, "create_failed")
		return
	}
	r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
		Action:       "node.bootstrap.create",
		ResourceType: "bootstrap_token",
		ResourceID:   item.ID,
		ResourceName: item.TargetID,
	})
	writeSuccess(w, http.StatusCreated, item)
}

func (r *Router) handleNodeParentURLProbe(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.ProbeNodeParentURLInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	item, err := r.service.ProbeNodeParentURL(req.Context(), tenantCtx, payload)
	if err != nil {
		writeServiceError(w, req, err, "parent_url_probe_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}

func (r *Router) handleUnconsumedBootstrapTokens(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	items, err := r.service.UnconsumedBootstrapTokens(tenantCtx)
	if err != nil {
		writeServiceError(w, req, err, "list_failed")
		return
	}
	writeSuccess(w, http.StatusOK, items)
}

func (r *Router) handleBootstrapTokenByID(w http.ResponseWriter, req *http.Request) {
	tokenID := resourceID(req.URL.Path, "/api/nodes/bootstrap/tokens/")
	if tokenID == "" {
		writeError(w, http.StatusBadRequest, "missing_bootstrap_token_id")
		return
	}
	switch req.Method {
	case http.MethodDelete:
		tenantCtx, ok := tenantAuthContextFromContext(req.Context())
		if !ok {
			writeError(w, http.StatusBadRequest, "tenant_required")
			return
		}
		if err := r.service.DeleteBootstrapToken(tenantCtx, tokenID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
			Action:       "node.bootstrap.delete",
			ResourceType: "bootstrap_token",
			ResourceID:   tokenID,
		})
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "DELETE")
	}
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
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	items, err := r.service.PendingNodeEnrollments(tenantCtx)
	if err != nil {
		writeServiceError(w, req, err, "list_failed")
		return
	}
	writeSuccess(w, http.StatusOK, items)
}

func (r *Router) handleNodeReject(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	nodeID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/nodes/"), "/reject")
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
	tenantCtx, ok := tenantAuthContextFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "tenant_required")
		return
	}
	if err := r.service.RejectNodeEnrollment(tenantCtx, nodeID, account.ID, payload.Reason); err != nil {
		writeServiceError(w, req, err, "reject_failed")
		return
	}
	r.recordBusinessAudit(req, domain.CreateBusinessAuditEventInput{
		Action:       "node.reject",
		ResourceType: "node",
		ResourceID:   nodeID,
		Reason:       payload.Reason,
	})
	writeSuccess(w, http.StatusOK, map[string]any{"status": "rejected"})
}
