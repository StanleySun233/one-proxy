package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/domain"
)

func resourceID(path string, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}

func (r *Router) handleOverview(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.Overview())
}

func (r *Router) handleAccounts(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.Accounts())
	case http.MethodPost:
		var payload domain.CreateAccountInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateAccount(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleNodeLinks(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.NodeLinks())
	case http.MethodPost:
		var payload domain.CreateNodeLinkInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateNodeLink(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

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

func (r *Router) handleNodeOnboardingTasks(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.NodeOnboardingTasks())
	case http.MethodPost:
		account, ok := accountFromContext(req.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid_access_token")
			return
		}
		var payload domain.CreateNodeOnboardingTaskInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateNodeOnboardingTask(account.ID, payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleNodeOnboardingTaskByID(w http.ResponseWriter, req *http.Request) {
	taskID := resourceID(req.URL.Path, "/api/v1/node-onboarding-tasks/")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "missing_task_id")
		return
	}
	if req.Method != http.MethodPatch {
		writeMethodNotAllowed(w, "PATCH")
		return
	}
	var payload domain.UpdateNodeOnboardingTaskStatusInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	item, err := r.service.UpdateNodeOnboardingTaskStatus(taskID, payload)
	if err != nil {
		writeServiceError(w, req, err, "update_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}

func (r *Router) handleCertificates(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.Certificates())
}

func (r *Router) handleAccountByID(w http.ResponseWriter, req *http.Request) {
	accountID := resourceID(req.URL.Path, "/api/v1/accounts/")
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "missing_account_id")
		return
	}
	if req.Method != http.MethodPatch {
		writeMethodNotAllowed(w, "PATCH")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	if account.MustRotatePassword && account.ID != accountID {
		writeError(w, http.StatusForbidden, "password_rotation_required")
		return
	}
	var payload domain.UpdateAccountInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	item, err := r.service.UpdateAccount(accountID, payload)
	if err != nil {
		writeServiceError(w, req, err, "update_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
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
	item, err := r.service.ApproveNodeEnrollment(nodeID)
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

func (r *Router) handleChains(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.Chains())
	case http.MethodPost:
		var payload domain.CreateChainInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateChain(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleChainByID(w http.ResponseWriter, req *http.Request) {
	chainID := resourceID(req.URL.Path, "/api/v1/chains/")
	if chainID == "" {
		writeError(w, http.StatusBadRequest, "missing_chain_id")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload domain.UpdateChainInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateChain(chainID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteChain(chainID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "PATCH, DELETE")
	}
}

func (r *Router) handleRouteRules(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		writeSuccess(w, http.StatusOK, r.service.RouteRules())
	case http.MethodPost:
		var payload domain.CreateRouteRuleInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateRouteRule(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleRouteRuleByID(w http.ResponseWriter, req *http.Request) {
	ruleID := resourceID(req.URL.Path, "/api/v1/route-rules/")
	if ruleID == "" {
		writeError(w, http.StatusBadRequest, "missing_rule_id")
		return
	}
	switch req.Method {
	case http.MethodPatch:
		var payload domain.UpdateRouteRuleInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateRouteRule(ruleID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteRouteRule(ruleID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "PATCH, DELETE")
	}
}

func (r *Router) handleNodeHealth(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.NodeHealth())
}

func (r *Router) handlePolicyRevisions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.PolicyRevisions())
}

func (r *Router) handlePolicyPublish(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	item, err := r.service.PublishPolicy(account.ID)
	if err != nil {
		writeServiceError(w, req, err, "publish_failed")
		return
	}
	writeSuccess(w, http.StatusOK, item)
}
