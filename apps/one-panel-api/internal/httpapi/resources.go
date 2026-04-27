package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
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

func (r *Router) handleExtensionBootstrap(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.ExtensionBootstrap(account))
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

func (r *Router) handleNodeTransports(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.NodeTransports())
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
	account, ok := accountFromContext(req.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "invalid_access_token")
		return
	}
	if account.MustRotatePassword && account.ID != accountID {
		writeError(w, http.StatusForbidden, "password_rotation_required")
		return
	}
	switch req.Method {
	case http.MethodPatch:
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
	case http.MethodDelete:
		if err := r.service.DeleteAccount(accountID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "PATCH, DELETE")
	}
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

type rejectNodePayload struct {
	Reason string `json:"reason"`
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

func (r *Router) handleChains(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		includeDetails := req.URL.Query().Get("details") == "true"
		if includeDetails {
			writeSuccess(w, http.StatusOK, r.service.ChainsWithDetails())
		} else {
			writeSuccess(w, http.StatusOK, r.service.Chains())
		}
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
	if strings.HasSuffix(req.URL.Path, "/probe") {
		r.handleChainProbe(w, req)
		return
	}
	chainID := resourceID(req.URL.Path, "/api/v1/chains/")
	if chainID == "" {
		writeError(w, http.StatusBadRequest, "missing_chain_id")
		return
	}
	switch req.Method {
	case http.MethodGet:
		item, err := r.service.GetChain(chainID)
		if err != nil {
			writeServiceError(w, req, err, "get_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
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
		writeMethodNotAllowed(w, "GET, PATCH, DELETE")
	}
}

func (r *Router) handleChainProbe(w http.ResponseWriter, req *http.Request) {
	chainID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/v1/chains/"), "/probe")
	chainID = strings.TrimSuffix(chainID, "/")
	if chainID == "" {
		writeError(w, http.StatusBadRequest, "missing_chain_id")
		return
	}
	switch req.Method {
	case http.MethodGet:
		item, ok := r.service.LatestChainProbe(chainID)
		if !ok {
			writeError(w, http.StatusNotFound, "chain_probe_not_found")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodPost:
		item, err := r.service.ProbeChain(chainID)
		if err != nil {
			writeServiceError(w, req, err, "probe_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleChainValidate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.ValidateChainInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	result, err := r.service.ValidateChain(payload)
	if err != nil {
		writeServiceError(w, req, err, "validation_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleChainPreview(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.PreviewChainInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	result, err := r.service.PreviewChain(payload)
	if err != nil {
		writeServiceError(w, req, err, "preview_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleNodeScopes(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.NodeScopes())
}

func (r *Router) handleRouteRules(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		includeDetails := req.URL.Query().Get("details") == "true"
		if includeDetails {
			writeSuccess(w, http.StatusOK, r.service.RouteRulesWithDetails())
		} else {
			writeSuccess(w, http.StatusOK, r.service.RouteRules())
		}
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
	case http.MethodGet:
		item, err := r.service.GetRouteRule(ruleID)
		if err != nil {
			writeServiceError(w, req, err, "get_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
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
		writeMethodNotAllowed(w, "GET, PATCH, DELETE")
	}
}

func (r *Router) handleMatchTypes(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	writeSuccess(w, http.StatusOK, r.service.MatchTypes())
}

func (r *Router) handleRouteRuleValidate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		writeMethodNotAllowed(w, "POST")
		return
	}
	var payload domain.ValidateRouteRuleInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	result, err := r.service.ValidateRouteRule(payload)
	if err != nil {
		writeServiceError(w, req, err, "validation_failed")
		return
	}
	writeSuccess(w, http.StatusOK, result)
}

func (r *Router) handleRouteRuleSuggestions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		writeMethodNotAllowed(w, "GET")
		return
	}
	matchType := req.URL.Query().Get("match_type")
	if matchType == "" {
		writeError(w, http.StatusBadRequest, "missing_match_type")
		return
	}
	query := req.URL.Query().Get("query")
	result := r.service.RouteRuleSuggestions(matchType, query)
	writeSuccess(w, http.StatusOK, result)
}

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

func (r *Router) handleGroups(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		groups, err := r.service.ListGroups()
		if err != nil {
			writeServiceError(w, req, err, "list_failed")
			return
		}
		writeSuccess(w, http.StatusOK, groups)
	case http.MethodPost:
		var payload domain.CreateGroupInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.CreateGroup(payload)
		if err != nil {
			writeServiceError(w, req, err, "create_failed")
			return
		}
		writeSuccess(w, http.StatusCreated, item)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (r *Router) handleGroupByID(w http.ResponseWriter, req *http.Request) {
	if strings.HasSuffix(req.URL.Path, "/accounts") {
		r.handleGroupAccounts(w, req)
		return
	}
	if strings.HasSuffix(req.URL.Path, "/scopes") {
		r.handleGroupScopes(w, req)
		return
	}
	groupID := resourceID(req.URL.Path, "/api/v1/groups/")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "missing_group_id")
		return
	}
	switch req.Method {
	case http.MethodGet:
		item, err := r.service.GetGroup(groupID)
		if err != nil {
			writeServiceError(w, req, err, "get_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodPut:
		var payload domain.UpdateGroupInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		item, err := r.service.UpdateGroup(groupID, payload)
		if err != nil {
			writeServiceError(w, req, err, "update_failed")
			return
		}
		writeSuccess(w, http.StatusOK, item)
	case http.MethodDelete:
		if err := r.service.DeleteGroup(groupID); err != nil {
			writeServiceError(w, req, err, "delete_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "deleted"})
	default:
		writeMethodNotAllowed(w, "GET, PUT, DELETE")
	}
}

func (r *Router) handleGroupAccounts(w http.ResponseWriter, req *http.Request) {
	groupID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/v1/groups/"), "/accounts")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "missing_group_id")
		return
	}
	switch req.Method {
	case http.MethodGet:
		items, err := r.service.ListGroupAccounts(groupID)
		if err != nil {
			writeServiceError(w, req, err, "list_failed")
			return
		}
		writeSuccess(w, http.StatusOK, items)
	case http.MethodPut:
		var payload domain.SetGroupAccountsInput
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if err := r.service.SetGroupAccounts(groupID, payload); err != nil {
			writeServiceError(w, req, err, "set_failed")
			return
		}
		writeSuccess(w, http.StatusOK, map[string]any{"status": "updated"})
	default:
		writeMethodNotAllowed(w, "GET, PUT")
	}
}

func (r *Router) handleGroupScopes(w http.ResponseWriter, req *http.Request) {
	groupID := strings.TrimSuffix(resourceID(req.URL.Path, "/api/v1/groups/"), "/scopes")
	if groupID == "" {
		writeError(w, http.StatusBadRequest, "missing_group_id")
		return
	}
	if req.Method != http.MethodPut {
		writeMethodNotAllowed(w, "PUT")
		return
	}
	var payload domain.SetGroupScopesInput
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	if err := r.service.SetGroupScopes(groupID, payload); err != nil {
		writeServiceError(w, req, err, "set_failed")
		return
	}
	writeSuccess(w, http.StatusOK, map[string]any{"status": "updated"})
}
