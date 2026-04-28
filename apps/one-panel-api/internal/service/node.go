package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/controlrelay"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/network"
)

type remoteNodeAttachInput struct {
	Password        string   `json:"password"`
	NewPassword     string   `json:"newPassword"`
	ControlPlaneURL string   `json:"controlPlaneUrl"`
	NodeID          string   `json:"nodeId"`
	NodeAccessToken string   `json:"nodeAccessToken"`
	NodeName        string   `json:"nodeName"`
	NodeMode        string   `json:"nodeMode"`
	NodeScopeKey    string   `json:"nodeScopeKey"`
	NodeParentID    string   `json:"nodeParentId"`
	NodePublicHost  string   `json:"nodePublicHost"`
	NodePublicPort  int      `json:"nodePublicPort"`
	LocalIPs        []string `json:"localIps"`
}

type remoteNodeAttachResult struct {
	ConnectionStatus    string   `json:"connectionStatus"`
	LocalIPs            []string `json:"localIps"`
	NodeListenAddr      string   `json:"nodeListenAddr"`
	NodeHTTPSListenAddr string   `json:"nodeHttpsListenAddr"`
	ControlPlaneBound   bool     `json:"controlPlaneBound"`
	MustRotatePassword  bool     `json:"mustRotatePassword"`
}

type remoteEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func (c *ControlPlane) Nodes() []domain.Node {
	return c.store.ListNodes()
}

func (c *ControlPlane) CreateNode(input domain.CreateNodeInput) (domain.Node, error) {
	if err := validateNodeInput(input.Name, input.Mode, input.ScopeKey); err != nil {
		return domain.Node{}, err
	}
	return c.store.CreateNode(input)
}

func (c *ControlPlane) ConnectNode(input domain.ConnectNodeInput) (domain.ConnectedNodeResult, error) {
	if input.Address == "" || input.Password == "" || input.ControlPlaneURL == "" {
		return domain.ConnectedNodeResult{}, invalidInput("invalid_connect_node_payload")
	}
	if err := validateNodeInput(input.Name, input.Mode, input.ScopeKey); err != nil {
		return domain.ConnectedNodeResult{}, err
	}
	if parsedURL, err := url.Parse(input.ControlPlaneURL); err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return domain.ConnectedNodeResult{}, invalidInput("invalid_control_plane_url")
	}
	publicHost := input.PublicHost
	publicPort := input.PublicPort
	if publicHost == "" || publicPort <= 0 {
		host, port, err := splitAddress(input.Address)
		if err != nil {
			return domain.ConnectedNodeResult{}, invalidInput("invalid_node_address")
		}
		if publicHost == "" {
			publicHost = host
		}
		if publicPort <= 0 {
			publicPort = port
		}
	}
	node, err := c.store.CreateNode(domain.CreateNodeInput{
		Name:         input.Name,
		Mode:         input.Mode,
		ScopeKey:     input.ScopeKey,
		ParentNodeID: input.ParentNodeID,
		PublicHost:   publicHost,
		PublicPort:   publicPort,
	})
	if err != nil {
		return domain.ConnectedNodeResult{}, err
	}
	if input.ParentNodeID != "" {
		if _, err := c.store.CreateNodeLink(domain.CreateNodeLinkInput{
			SourceNodeID: input.ParentNodeID,
			TargetNodeID: node.ID,
			LinkType:     domain.LinkTypeManaged,
			TrustState:   domain.TrustStateActive,
		}); err != nil {
			_ = c.store.DeleteNode(node.ID)
			return domain.ConnectedNodeResult{}, err
		}
	}
	issued, err := c.store.ProvisionNodeAccess(node.ID)
	if err != nil {
		_ = c.store.DeleteNode(node.ID)
		return domain.ConnectedNodeResult{}, err
	}
	result, err := connectRemoteNode(input.Address, remoteNodeAttachInput{
		Password:        input.Password,
		NewPassword:     input.NewPassword,
		ControlPlaneURL: input.ControlPlaneURL,
		NodeID:          issued.Node.ID,
		NodeAccessToken: issued.AccessToken,
		NodeName:        issued.Node.Name,
		NodeMode:        issued.Node.Mode,
		NodeScopeKey:    issued.Node.ScopeKey,
		NodeParentID:    issued.Node.ParentNodeID,
		NodePublicHost:  issued.Node.PublicHost,
		NodePublicPort:  issued.Node.PublicPort,
		LocalIPs:        network.LocalIPs(),
	})
	if err != nil {
		_ = c.store.DeleteNode(node.ID)
		return domain.ConnectedNodeResult{}, invalidInput(err.Error())
	}
	return domain.ConnectedNodeResult{
		Node:                issued.Node,
		ConnectionStatus:    result.ConnectionStatus,
		LocalIPs:            result.LocalIPs,
		NodeListenAddr:      result.NodeListenAddr,
		NodeHTTPSListenAddr: result.NodeHTTPSListenAddr,
		ControlPlaneBound:   result.ControlPlaneBound,
		MustRotatePassword:  result.MustRotatePassword,
	}, nil
}

func (c *ControlPlane) UpdateNode(nodeID string, input domain.UpdateNodeInput) (domain.Node, error) {
	if nodeID == "" {
		return domain.Node{}, invalidInput("missing_node_id")
	}
	if err := validateNodeInput(input.Name, input.Mode, input.ScopeKey); err != nil {
		return domain.Node{}, err
	}
	return c.store.UpdateNode(nodeID, input)
}

func (c *ControlPlane) DeleteNode(nodeID string) error {
	return c.store.DeleteNode(nodeID)
}

func (c *ControlPlane) NodeLinks() []domain.NodeLink {
	return c.store.ListNodeLinks()
}

func (c *ControlPlane) CreateNodeLink(input domain.CreateNodeLinkInput) (domain.NodeLink, error) {
	if input.SourceNodeID == "" || input.TargetNodeID == "" || input.LinkType == "" || input.TrustState == "" {
		return domain.NodeLink{}, invalidInput("invalid_node_link_payload")
	}
	return c.store.CreateNodeLink(input)
}

func (c *ControlPlane) NodeTransports() []domain.NodeTransport {
	return c.store.ListNodeTransports()
}

func (c *ControlPlane) UpsertNodeTransport(input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	if input.NodeID == "" || input.TransportType == "" || input.Direction == "" || input.Address == "" || input.Status == "" {
		return domain.NodeTransport{}, invalidInput("invalid_node_transport_payload")
	}
	return c.store.UpsertNodeTransport(input)
}

func (c *ControlPlane) UpsertNodeAgentTransport(nodeID string, input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	input.NodeID = nodeID
	return c.UpsertNodeTransport(input)
}

func (c *ControlPlane) NodeAccessPaths() []domain.NodeAccessPath {
	return c.store.ListNodeAccessPaths()
}

func (c *ControlPlane) CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if err := c.validateNodeAccessPath(input.Name, input.Mode, input.TargetHost, input.TargetPort); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return c.store.CreateNodeAccessPath(input)
}

func (c *ControlPlane) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if pathID == "" {
		return domain.NodeAccessPath{}, invalidInput("missing_path_id")
	}
	if err := c.validateNodeAccessPath(input.Name, input.Mode, input.TargetHost, input.TargetPort); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return c.store.UpdateNodeAccessPath(pathID, input)
}

func (c *ControlPlane) DeleteNodeAccessPath(pathID string) error {
	if pathID == "" {
		return invalidInput("missing_path_id")
	}
	return c.store.DeleteNodeAccessPath(pathID)
}

func (c *ControlPlane) NodeOnboardingTasks() []domain.NodeOnboardingTask {
	return c.store.ListNodeOnboardingTasks()
}

func (c *ControlPlane) CreateNodeOnboardingTask(accountID string, input domain.CreateNodeOnboardingTaskInput) (domain.NodeOnboardingTask, error) {
	if accountID == "" {
		return domain.NodeOnboardingTask{}, unauthorized("invalid_access_token")
	}
	if err := c.validateNodeOnboardingTask(input.Mode, input.PathID, input.TargetHost, input.TargetPort); err != nil {
		return domain.NodeOnboardingTask{}, err
	}
	if input.Mode != domain.PathModeDirect && !hasNodeAccessPath(c.store.ListNodeAccessPaths(), input.PathID) {
		return domain.NodeOnboardingTask{}, invalidInput("invalid_node_access_path")
	}
	item, err := c.store.CreateNodeOnboardingTask(accountID, input)
	if err != nil {
		return domain.NodeOnboardingTask{}, err
	}
	switch input.Mode {
	case domain.PathModeDirect:
		status, message := probeDirectNodeTarget(input.TargetHost, input.TargetPort)
		updated, updateErr := c.store.UpdateNodeOnboardingTaskStatus(item.ID, status, message)
		if updateErr != nil {
			return item, nil
		}
		return updated, nil
	case domain.PathModeRelayChain:
		status, message := c.probeRelayPath(input.PathID)
		updated, updateErr := c.store.UpdateNodeOnboardingTaskStatus(item.ID, status, message)
		if updateErr != nil {
			return item, nil
		}
		return updated, nil
	case domain.PathModeUpstreamPull:
		updated, updateErr := c.store.UpdateNodeOnboardingTaskStatus(item.ID, domain.TaskStatusPending, "waiting_for_target_node_pull")
		if updateErr != nil {
			return item, nil
		}
		return updated, nil
	default:
		return item, nil
	}
}

func (c *ControlPlane) UpdateNodeOnboardingTaskStatus(taskID string, input domain.UpdateNodeOnboardingTaskStatusInput) (domain.NodeOnboardingTask, error) {
	if taskID == "" {
		return domain.NodeOnboardingTask{}, invalidInput("missing_task_id")
	}
	if input.Status == "" {
		return domain.NodeOnboardingTask{}, invalidInput("invalid_task_status")
	}
	if !c.isValidEnum("task_status", input.Status) {
		return domain.NodeOnboardingTask{}, invalidInput("invalid_task_status")
	}
	return c.store.UpdateNodeOnboardingTaskStatus(taskID, input.Status, input.StatusMessage)
}

func (c *ControlPlane) NodeScopes() []domain.NodeScope {
	nodes := c.store.ListNodes()
	scopeMap := make(map[string]domain.NodeScope)

	for _, node := range nodes {
		if node.ScopeKey == "" {
			continue
		}
		if _, exists := scopeMap[node.ScopeKey]; !exists {
			scopeMap[node.ScopeKey] = domain.NodeScope{
				ScopeKey:      node.ScopeKey,
				OwnerNodeID:   node.ID,
				OwnerNodeName: node.Name,
				Description:   fmt.Sprintf("Scope managed by %s", node.Name),
			}
		}
	}

	scopes := make([]domain.NodeScope, 0, len(scopeMap))
	for _, scope := range scopeMap {
		scopes = append(scopes, scope)
	}

	return scopes
}

func (c *ControlPlane) PendingNodeEnrollments() []domain.Node {
	return c.store.ListPendingNodes()
}

func (c *ControlPlane) RejectNodeEnrollment(nodeID string, accountID string, reason string) error {
	if nodeID == "" {
		return invalidInput("missing_node_id")
	}
	if accountID == "" {
		return unauthorized("invalid_access_token")
	}
	return c.store.RejectNodeEnrollment(nodeID, accountID, reason)
}

func (c *ControlPlane) CreateBootstrapToken(input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	if input.TargetType == "" {
		return domain.BootstrapToken{}, invalidInput("invalid_bootstrap_payload")
	}
	return c.store.CreateBootstrapToken(input)
}

func (c *ControlPlane) UnconsumedBootstrapTokens() []domain.BootstrapToken {
	return c.store.ListUnconsumedBootstrapTokens()
}

func (c *ControlPlane) EnrollNode(input domain.EnrollNodeInput) (domain.EnrollNodeResult, error) {
	if input.Token == "" {
		return domain.EnrollNodeResult{}, invalidInput("missing_bootstrap_token")
	}
	if err := validateNodeInput(input.Name, input.Mode, input.ScopeKey); err != nil {
		return domain.EnrollNodeResult{}, err
	}
	return c.store.EnrollNode(input)
}

func (c *ControlPlane) ApproveNodeEnrollment(nodeID string, reviewedBy string) (domain.ApproveNodeEnrollmentResult, error) {
	if nodeID == "" {
		return domain.ApproveNodeEnrollmentResult{}, invalidInput("missing_node_id")
	}
	item, err := c.store.ApproveNodeEnrollment(nodeID, reviewedBy)
	if err != nil {
		if strings.Contains(err.Error(), "node_not_pending") {
			return domain.ApproveNodeEnrollmentResult{}, invalidInput("node_not_pending")
		}
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	return item, nil
}

func (c *ControlPlane) ExchangeNodeEnrollment(input domain.ExchangeNodeEnrollmentInput) (domain.ApproveNodeEnrollmentResult, error) {
	if input.NodeID == "" || input.EnrollmentSecret == "" {
		return domain.ApproveNodeEnrollmentResult{}, invalidInput("invalid_enrollment_exchange_payload")
	}
	item, err := c.store.ExchangeNodeEnrollment(input)
	if err != nil {
		if strings.Contains(err.Error(), "node_enrollment_pending") {
			return domain.ApproveNodeEnrollmentResult{}, invalidInput("node_enrollment_pending")
		}
		if strings.Contains(err.Error(), "invalid_enrollment_secret") {
			return domain.ApproveNodeEnrollmentResult{}, invalidInput("invalid_enrollment_secret")
		}
		return domain.ApproveNodeEnrollmentResult{}, err
	}
	return item, nil
}

func (c *ControlPlane) AuthenticateNodeToken(accessToken string) (string, bool) {
	return c.store.AuthenticateNodeToken(accessToken)
}

func validateNodeInput(name string, mode string, scopeKey string) error {
	if name == "" || mode == "" || scopeKey == "" {
		return invalidInput("invalid_node_payload")
	}
	return nil
}

func (c *ControlPlane) validateNodeAccessPath(name string, mode string, targetHost string, targetPort int) error {
	if name == "" || mode == "" {
		return invalidInput("invalid_node_access_path_payload")
	}
	if !c.isValidEnum("path_mode", mode) {
		return invalidInput("invalid_node_access_path_payload")
	}
	switch mode {
	case domain.PathModeDirect, domain.PathModeRelayChain:
		if targetHost == "" || targetPort <= 0 {
			return invalidInput("invalid_node_access_path_payload")
		}
	}
	return nil
}

func (c *ControlPlane) validateNodeOnboardingTask(mode string, pathID string, targetHost string, targetPort int) error {
	if !c.isValidEnum("path_mode", mode) {
		return invalidInput("invalid_node_onboarding_task_payload")
	}
	switch mode {
	case domain.PathModeDirect:
		if targetHost == "" || targetPort <= 0 {
			return invalidInput("invalid_node_onboarding_task_payload")
		}
	case domain.PathModeRelayChain, domain.PathModeUpstreamPull:
		if pathID == "" {
			return invalidInput("invalid_node_onboarding_task_payload")
		}
	}
	return nil
}

func probeDirectNodeTarget(targetHost string, targetPort int) (string, string) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/healthz", targetHost, targetPort))
	if err != nil {
		return domain.ProbeResultStatusFailed, "target_unreachable"
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return domain.ProbeResultStatusFailed, "target_unhealthy"
	}
	return domain.ProbeResultStatusConnected, "target_reachable"
}

func hasNodeAccessPath(items []domain.NodeAccessPath, pathID string) bool {
	for _, item := range items {
		if item.ID == pathID {
			return true
		}
	}
	return false
}

func (c *ControlPlane) probeRelayPath(pathID string) (string, string) {
	path, ok := nodeAccessPathByID(c.store.ListNodeAccessPaths(), pathID)
	if !ok || !path.Enabled {
		return domain.ProbeResultStatusFailed, "invalid_node_access_path"
	}
	relayURLs, ok := relayURLsForPath(c.store.ListNodes(), path)
	if !ok || len(relayURLs) == 0 {
		return domain.ProbeResultStatusFailed, "invalid_relay_chain"
	}
	result, err := controlrelay.Execute(relayURLs[0], controlrelay.ProbeRequest{
		RemainingRelayURLs: relayURLs[1:],
		TargetHost:         path.TargetHost,
		TargetPort:         path.TargetPort,
	})
	if err != nil {
		return domain.ProbeResultStatusFailed, "relay_probe_failed"
	}
	return result.Status, result.Message
}

func nodeAccessPathByID(items []domain.NodeAccessPath, pathID string) (domain.NodeAccessPath, bool) {
	for _, item := range items {
		if item.ID == pathID {
			return item, true
		}
	}
	return domain.NodeAccessPath{}, false
}

func relayURLsForPath(nodes []domain.Node, path domain.NodeAccessPath) ([]string, bool) {
	relayIDs := normalizeNodeRelayIDs(path)
	if len(relayIDs) == 0 {
		return nil, false
	}
	urls := make([]string, 0, len(relayIDs))
	for _, relayID := range relayIDs {
		node, ok := nodeByID(nodes, relayID)
		if !ok || node.PublicHost == "" || node.PublicPort <= 0 {
			return nil, false
		}
		urls = append(urls, fmt.Sprintf("http://%s:%d", node.PublicHost, node.PublicPort))
	}
	return urls, true
}

func normalizeNodeRelayIDs(path domain.NodeAccessPath) []string {
	if len(path.RelayNodeIDs) > 0 {
		return path.RelayNodeIDs
	}
	if path.EntryNodeID != "" {
		return []string{path.EntryNodeID}
	}
	return nil
}

func nodeByID(items []domain.Node, nodeID string) (domain.Node, bool) {
	for _, item := range items {
		if item.ID == nodeID {
			return item, true
		}
	}
	return domain.Node{}, false
}

func connectRemoteNode(address string, input remoteNodeAttachInput) (remoteNodeAttachResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return remoteNodeAttachResult{}, err
	}
	targetURL := normalizeNodeAddress(address) + "/api/v1/node/bootstrap/attach"
	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return remoteNodeAttachResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return remoteNodeAttachResult{}, fmt.Errorf("node_connect_failed")
	}
	defer resp.Body.Close()
	var envelope remoteEnvelope[remoteNodeAttachResult]
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return remoteNodeAttachResult{}, fmt.Errorf("node_connect_failed")
	}
	if resp.StatusCode >= http.StatusBadRequest || envelope.Code != 0 {
		if envelope.Message != "" {
			return remoteNodeAttachResult{}, fmt.Errorf(envelope.Message)
		}
		return remoteNodeAttachResult{}, fmt.Errorf("node_connect_failed")
	}
	return envelope.Data, nil
}

func normalizeNodeAddress(address string) string {
	trimmed := strings.TrimSpace(address)
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return strings.TrimRight(trimmed, "/")
	}
	return "http://" + strings.TrimRight(trimmed, "/")
}

func splitAddress(address string) (string, int, error) {
	normalized := strings.TrimPrefix(strings.TrimPrefix(normalizeNodeAddress(address), "http://"), "https://")
	hostPort := strings.SplitN(normalized, "/", 2)[0]
	parts := strings.Split(hostPort, ":")
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("invalid_address")
	}
	port, err := parsePort(parts[len(parts)-1])
	if err != nil {
		return "", 0, err
	}
	host := strings.Join(parts[:len(parts)-1], ":")
	return host, port, nil
}

func parsePort(raw string) (int, error) {
	var port int
	if _, err := fmt.Sscanf(raw, "%d", &port); err != nil || port <= 0 {
		return 0, fmt.Errorf("invalid_port")
	}
	return port, nil
}
