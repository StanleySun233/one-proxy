package service

import (
	"fmt"
	"net/http"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/controlrelay"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

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
