package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/config"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/controlrelay"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/network"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/policy"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/store"
)

type ControlPlane struct {
	store             store.Store
	sessionTTL        time.Duration
	bootstrapTokenTTL time.Duration
	nodeHeartbeatTTL  time.Duration
	publicRenewWindow time.Duration
}

func NewControlPlane(store store.Store, cfg config.Config) *ControlPlane {
	return &ControlPlane{
		store:             store,
		sessionTTL:        parseDuration(cfg.SessionTTL, 12*time.Hour),
		bootstrapTokenTTL: parseDuration(cfg.BootstrapTokenTTL, 15*time.Minute),
		nodeHeartbeatTTL:  parseDuration(cfg.NodeHeartbeatTTL, 2*time.Minute),
		publicRenewWindow: parseDuration(cfg.PublicCertRenewWindow, 7*24*time.Hour),
	}
}

func (c *ControlPlane) IsInitialized() bool {
	return c.store.IsInitialized()
}

func (c *ControlPlane) ReinitializeStore(adminPassword string) error {
	return c.store.ReinitializeStore(adminPassword)
}

func (c *ControlPlane) Overview() domain.Overview {
	return c.store.GetOverview()
}

func (c *ControlPlane) Accounts() []domain.Account {
	return c.store.ListAccounts()
}

func (c *ControlPlane) ExtensionBootstrap(account domain.Account) domain.ExtensionBootstrap {
	nodes := c.store.ListNodes()
	rules := c.store.ListRouteRules()
	overview := c.store.GetOverview()
	fetchedAt := time.Now().UTC().Format(time.RFC3339)

	filteredNodes := nodes
	filteredRules := rules
	if account.Role != "super_admin" {
		accountGroups, err := c.store.ListAccountGroups(account.ID)
		if err == nil && len(accountGroups) > 0 {
			allowedScopes := make(map[string]bool)
			for _, g := range accountGroups {
				scopes, _ := c.store.GetGroupScopes(g.ID)
				for _, scope := range scopes {
					allowedScopes[scope] = true
				}
			}
			filteredNodes = make([]domain.Node, 0)
			for _, node := range nodes {
				if allowedScopes[node.ScopeKey] {
					filteredNodes = append(filteredNodes, node)
				}
			}
			filteredRules = make([]domain.RouteRule, 0)
			for _, rule := range rules {
				if rule.DestinationScope == "" || allowedScopes[rule.DestinationScope] {
					filteredRules = append(filteredRules, rule)
				}
			}
		}
	}

	groups := make([]domain.ExtensionGroup, 0)
	for _, node := range filteredNodes {
		if !node.Enabled || node.PublicHost == "" || node.PublicPort <= 0 {
			continue
		}
		if node.Mode != "edge" && node.ParentNodeID != "" {
			continue
		}
		group := domain.ExtensionGroup{
			ID:            node.ID,
			Name:          node.Name,
			EntryNodeID:   node.ID,
			EntryNodeName: node.Name,
			ProxyScheme:   "PROXY",
			ProxyHost:     node.PublicHost,
			ProxyPort:     node.PublicPort,
		}
		for _, rule := range filteredRules {
			if !rule.Enabled {
				continue
			}
			value := strings.TrimSpace(rule.MatchValue)
			if value == "" {
				continue
			}
			switch rule.MatchType {
			case "domain":
				if rule.ActionType == "direct" {
					group.DirectHosts = append(group.DirectHosts, value)
				} else if rule.ActionType == "chain" {
					group.ProxyHosts = append(group.ProxyHosts, value)
				}
			case "domain_suffix":
				pattern := "*" + value
				if rule.ActionType == "direct" {
					group.DirectHosts = append(group.DirectHosts, pattern)
				} else if rule.ActionType == "chain" {
					group.ProxyHosts = append(group.ProxyHosts, pattern)
				}
			case "cidr":
				if rule.ActionType == "direct" {
					group.DirectCIDRs = append(group.DirectCIDRs, value)
				} else if rule.ActionType == "chain" {
					group.ProxyCIDRs = append(group.ProxyCIDRs, value)
				}
			}
		}
		group.ProxyHosts = uniqueStrings(group.ProxyHosts)
		group.ProxyCIDRs = uniqueStrings(group.ProxyCIDRs)
		group.DirectHosts = uniqueStrings(group.DirectHosts)
		group.DirectCIDRs = uniqueStrings(group.DirectCIDRs)
		groups = append(groups, group)
	}
	return domain.ExtensionBootstrap{
		Account:        account,
		PolicyRevision: overview.Policies.ActiveRevision,
		FetchedAt:      fetchedAt,
		Groups:         groups,
	}
}

func (c *ControlPlane) CreateAccount(input domain.CreateAccountInput) (domain.Account, error) {
	if input.Account == "" || input.Password == "" || input.Role == "" {
		return domain.Account{}, invalidInput("invalid_account_payload")
	}
	return c.store.CreateAccount(input)
}

func (c *ControlPlane) ListGroups() ([]domain.Group, error) {
	return c.store.ListGroups()
}

func (c *ControlPlane) CreateGroup(input domain.CreateGroupInput) (domain.Group, error) {
	if input.Name == "" {
		return domain.Group{}, invalidInput("invalid_group_payload")
	}
	return c.store.CreateGroup(input)
}

func (c *ControlPlane) GetGroup(id string) (domain.GroupDetail, error) {
	if id == "" {
		return domain.GroupDetail{}, invalidInput("missing_group_id")
	}
	group, err := c.store.GetGroup(id)
	if err != nil {
		return domain.GroupDetail{}, err
	}
	scopes, _ := c.store.GetGroupScopes(id)
	detail := domain.GroupDetail{
		Group:   group,
		Scopes:  scopes,
	}
	return detail, nil
}

func (c *ControlPlane) UpdateGroup(id string, input domain.UpdateGroupInput) (domain.Group, error) {
	if id == "" {
		return domain.Group{}, invalidInput("missing_group_id")
	}
	return c.store.UpdateGroup(id, input)
}

func (c *ControlPlane) DeleteGroup(id string) error {
	if id == "" {
		return invalidInput("missing_group_id")
	}
	return c.store.DeleteGroup(id)
}

func (c *ControlPlane) ListGroupAccounts(groupID string) ([]domain.Account, error) {
	if groupID == "" {
		return nil, invalidInput("missing_group_id")
	}
	return c.store.ListGroupAccounts(groupID)
}

func (c *ControlPlane) SetGroupAccounts(groupID string, input domain.SetGroupAccountsInput) error {
	if groupID == "" {
		return invalidInput("missing_group_id")
	}
	return c.store.SetGroupAccounts(groupID, input.AccountIDs)
}

func (c *ControlPlane) SetGroupScopes(groupID string, input domain.SetGroupScopesInput) error {
	if groupID == "" {
		return invalidInput("missing_group_id")
	}
	return c.store.SetGroupScopes(groupID, input.ScopeKeys)
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

func (c *ControlPlane) NodeAccessPaths() []domain.NodeAccessPath {
	return c.store.ListNodeAccessPaths()
}

func (c *ControlPlane) CreateNodeAccessPath(input domain.CreateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if err := validateNodeAccessPath(input.Name, input.Mode, input.TargetHost, input.TargetPort); err != nil {
		return domain.NodeAccessPath{}, err
	}
	return c.store.CreateNodeAccessPath(input)
}

func (c *ControlPlane) UpdateNodeAccessPath(pathID string, input domain.UpdateNodeAccessPathInput) (domain.NodeAccessPath, error) {
	if pathID == "" {
		return domain.NodeAccessPath{}, invalidInput("missing_path_id")
	}
	if err := validateNodeAccessPath(input.Name, input.Mode, input.TargetHost, input.TargetPort); err != nil {
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
	if err := validateNodeOnboardingTask(input.Mode, input.PathID, input.TargetHost, input.TargetPort); err != nil {
		return domain.NodeOnboardingTask{}, err
	}
	if input.Mode != "direct" && !hasNodeAccessPath(c.store.ListNodeAccessPaths(), input.PathID) {
		return domain.NodeOnboardingTask{}, invalidInput("invalid_node_access_path")
	}
	item, err := c.store.CreateNodeOnboardingTask(accountID, input)
	if err != nil {
		return domain.NodeOnboardingTask{}, err
	}
	switch input.Mode {
	case "direct":
		status, message := probeDirectNodeTarget(input.TargetHost, input.TargetPort)
		updated, updateErr := c.store.UpdateNodeOnboardingTaskStatus(item.ID, status, message)
		if updateErr != nil {
			return item, nil
		}
		return updated, nil
	case "relay_chain":
		status, message := c.probeRelayPath(input.PathID)
		updated, updateErr := c.store.UpdateNodeOnboardingTaskStatus(item.ID, status, message)
		if updateErr != nil {
			return item, nil
		}
		return updated, nil
	case "upstream_pull":
		updated, updateErr := c.store.UpdateNodeOnboardingTaskStatus(item.ID, "pending", "waiting_for_target_node_pull")
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
	switch input.Status {
	case "planned", "pending", "connected", "failed", "cancelled":
	default:
		return domain.NodeOnboardingTask{}, invalidInput("invalid_task_status")
	}
	return c.store.UpdateNodeOnboardingTaskStatus(taskID, input.Status, input.StatusMessage)
}

func (c *ControlPlane) Certificates() []domain.Certificate {
	return c.store.ListCertificates()
}

func (c *ControlPlane) UpdateAccount(accountID string, input domain.UpdateAccountInput) (domain.Account, error) {
	if accountID == "" {
		return domain.Account{}, invalidInput("missing_account_id")
	}
	return c.store.UpdateAccount(accountID, input)
}

func (c *ControlPlane) DeleteAccount(accountID string) error {
	if accountID == "" {
		return invalidInput("missing_account_id")
	}
	return c.store.DeleteAccount(accountID)
}

func (c *ControlPlane) Login(account string, password string) (domain.LoginResult, bool) {
	return c.store.Authenticate(account, password)
}

func (c *ControlPlane) AuthenticateAccessToken(accessToken string) (domain.Account, bool) {
	return c.store.AuthenticateAccessToken(accessToken)
}

func (c *ControlPlane) RefreshSession(refreshToken string) (domain.LoginResult, bool) {
	return c.store.RefreshSession(refreshToken)
}

func (c *ControlPlane) Logout(accessToken string) bool {
	return c.store.Logout(accessToken)
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
			LinkType:     "managed",
			TrustState:   "active",
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

func (c *ControlPlane) Chains() []domain.Chain {
	return c.store.ListChains()
}

func (c *ControlPlane) ChainsWithDetails() []domain.ChainWithDetails {
	chains := c.store.ListChains()
	nodes := c.store.ListNodes()
	result := make([]domain.ChainWithDetails, 0, len(chains))

	for _, chain := range chains {
		hopDetails := make([]domain.ChainHopDetail, 0, len(chain.Hops))
		for _, hopID := range chain.Hops {
			node, ok := nodeByID(nodes, hopID)
			if ok {
				hopDetails = append(hopDetails, domain.ChainHopDetail{
					NodeID:   node.ID,
					NodeName: node.Name,
					Mode:     node.Mode,
				})
			}
		}

		result = append(result, domain.ChainWithDetails{
			ID:               chain.ID,
			Name:             chain.Name,
			DestinationScope: chain.DestinationScope,
			Enabled:          chain.Enabled,
			Hops:             chain.Hops,
			HopDetails:       hopDetails,
		})
	}

	return result
}

func (c *ControlPlane) GetChain(chainID string) (domain.ChainWithDetails, error) {
	if chainID == "" {
		return domain.ChainWithDetails{}, invalidInput("missing_chain_id")
	}

	chains := c.store.ListChains()
	chain, ok := chainByID(chains, chainID)
	if !ok {
		return domain.ChainWithDetails{}, invalidInput("chain_not_found")
	}

	nodes := c.store.ListNodes()
	hopDetails := make([]domain.ChainHopDetail, 0, len(chain.Hops))
	for _, hopID := range chain.Hops {
		node, ok := nodeByID(nodes, hopID)
		if ok {
			hopDetails = append(hopDetails, domain.ChainHopDetail{
				NodeID:   node.ID,
				NodeName: node.Name,
				Mode:     node.Mode,
			})
		}
	}

	return domain.ChainWithDetails{
		ID:               chain.ID,
		Name:             chain.Name,
		DestinationScope: chain.DestinationScope,
		Enabled:          chain.Enabled,
		Hops:             chain.Hops,
		HopDetails:       hopDetails,
	}, nil
}

func (c *ControlPlane) LatestChainProbe(chainID string) (domain.ChainProbeResult, bool) {
	if chainID == "" {
		return domain.ChainProbeResult{}, false
	}
	return c.store.GetChainProbeResult(chainID)
}

func (c *ControlPlane) ProbeChain(chainID string) (domain.ChainProbeResult, error) {
	if chainID == "" {
		return domain.ChainProbeResult{}, invalidInput("missing_chain_id")
	}
	chain, ok := chainByID(c.store.ListChains(), chainID)
	if !ok {
		return domain.ChainProbeResult{}, invalidInput("invalid_chain_id")
	}
	nodes := c.store.ListNodes()
	transports := c.store.ListNodeTransports()
	result := domain.ChainProbeResult{
		ChainID:      chainID,
		Status:       "connected",
		Message:      "chain_transport_ready",
		ResolvedHops: make([]domain.ChainProbeHop, 0, len(chain.Hops)),
		ProbedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	prevHopID := ""
	for _, hopID := range chain.Hops {
		node, ok := nodeByID(nodes, hopID)
		if !ok || !node.Enabled {
			result.Status = "failed"
			result.Message = "chain_blocked"
			result.BlockingNodeID = hopID
			result.BlockingReason = "unknown_or_disabled_node"
			return c.store.SaveChainProbeResult(toChainProbeInput(result))
		}
		transport, ok := resolveProbeTransport(node, prevHopID, transports)
		if !ok {
			result.Status = "failed"
			result.Message = "chain_blocked"
			result.BlockingNodeID = node.ID
			if prevHopID == "" {
				result.BlockingReason = "missing_entry_transport"
			} else {
				result.BlockingReason = "missing_parent_transport"
			}
			return c.store.SaveChainProbeResult(toChainProbeInput(result))
		}
		result.ResolvedHops = append(result.ResolvedHops, domain.ChainProbeHop{
			NodeID:        node.ID,
			NodeName:      node.Name,
			TransportType: transport.TransportType,
			Address:       transport.Address,
			Status:        transport.Status,
		})
		prevHopID = node.ID
	}
	if len(result.ResolvedHops) > 0 && (result.ResolvedHops[0].TransportType == "public_http" || result.ResolvedHops[0].TransportType == "public_https") {
		probeResult, err := controlrelay.Execute(result.ResolvedHops[0].Address, controlrelay.ProbeRequest{
			RemainingHopNodeIDs: chain.Hops[1:],
		})
		if err != nil {
			result.Status = "failed"
			result.Message = "chain_probe_failed"
			result.BlockingNodeID = chain.Hops[0]
			result.BlockingReason = "probe_dispatch_failed"
			return c.store.SaveChainProbeResult(toChainProbeInput(result))
		}
		result.Status = probeResult.Status
		result.Message = probeResult.Message
		if probeResult.Status != "connected" && result.BlockingReason == "" && len(chain.Hops) > 0 {
			result.BlockingNodeID = chain.Hops[len(chain.Hops)-1]
			result.BlockingReason = probeResult.Message
		}
	}
	return c.store.SaveChainProbeResult(toChainProbeInput(result))
}

func (c *ControlPlane) CreateChain(input domain.CreateChainInput) (domain.Chain, error) {
	if input.Name == "" || input.DestinationScope == "" || len(input.Hops) == 0 {
		return domain.Chain{}, invalidInput("invalid_chain_payload")
	}
	return c.store.CreateChain(input)
}

func (c *ControlPlane) UpdateChain(chainID string, input domain.UpdateChainInput) (domain.Chain, error) {
	if chainID == "" || input.Name == "" || input.DestinationScope == "" || len(input.Hops) == 0 {
		return domain.Chain{}, invalidInput("invalid_chain_payload")
	}
	return c.store.UpdateChain(chainID, input)
}

func (c *ControlPlane) DeleteChain(chainID string) error {
	return c.store.DeleteChain(chainID)
}

func (c *ControlPlane) ValidateChain(input domain.ValidateChainInput) (domain.ChainValidationResult, error) {
	result := domain.ChainValidationResult{
		Valid:           true,
		Errors:          []string{},
		Warnings:        []string{},
		HopConnectivity: []domain.HopConnectivity{},
	}

	if len(input.Hops) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "Chain must have at least one hop")
		return result, nil
	}

	nodes := c.store.ListNodes()
	links := c.store.ListNodeLinks()

	firstHopNode, ok := nodeByID(nodes, input.Hops[0])
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("First hop node %s not found", input.Hops[0]))
		return result, nil
	}

	if firstHopNode.Mode != "edge" {
		result.Valid = false
		result.Errors = append(result.Errors, "First hop must be an edge node")
	}

	for i := 0; i < len(input.Hops)-1; i++ {
		fromNodeID := input.Hops[i]
		toNodeID := input.Hops[i+1]
		reachable := false

		for _, link := range links {
			if link.SourceNodeID == fromNodeID && link.TargetNodeID == toNodeID {
				reachable = true
				break
			}
		}

		result.HopConnectivity = append(result.HopConnectivity, domain.HopConnectivity{
			From:      fromNodeID,
			To:        toNodeID,
			Reachable: reachable,
		})

		if !reachable {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Node %s cannot reach node %s", fromNodeID, toNodeID))
		}
	}

	if len(input.Hops) > 0 {
		finalHopNodeID := input.Hops[len(input.Hops)-1]
		finalHopNode, ok := nodeByID(nodes, finalHopNodeID)
		if !ok {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Final hop node %s not found", finalHopNodeID))
		} else {
			scopeValid := finalHopNode.ScopeKey == input.DestinationScope
			result.ScopeOwnership = domain.ScopeOwnership{
				Scope:       input.DestinationScope,
				OwnerNodeID: finalHopNodeID,
				Valid:       scopeValid,
			}

			if !scopeValid {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Scope %s is not owned by final hop node %s", input.DestinationScope, finalHopNodeID))
			}
		}
	}

	return result, nil
}

func (c *ControlPlane) PreviewChain(input domain.PreviewChainInput) (domain.ChainPreviewResult, error) {
	nodes := c.store.ListNodes()
	hopDetails := make([]domain.ChainHopDetail, 0, len(input.Hops))
	routingPath := "user"

	for _, hopID := range input.Hops {
		node, ok := nodeByID(nodes, hopID)
		if !ok {
			return domain.ChainPreviewResult{}, invalidInput(fmt.Sprintf("node %s not found", hopID))
		}

		hopDetails = append(hopDetails, domain.ChainHopDetail{
			NodeID:   node.ID,
			NodeName: node.Name,
			Mode:     node.Mode,
		})

		routingPath += " → " + node.Name
	}

	routingPath += fmt.Sprintf(" → target(%s)", input.DestinationScope)

	return domain.ChainPreviewResult{
		CompiledConfig: domain.CompiledChainConfig{
			ChainID:          "preview",
			Name:             input.Name,
			Hops:             hopDetails,
			DestinationScope: input.DestinationScope,
			RoutingPath:      routingPath,
		},
	}, nil
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

func (c *ControlPlane) RouteRules() []domain.RouteRule {
	return c.store.ListRouteRules()
}

func (c *ControlPlane) RouteRulesWithDetails() []domain.RouteRuleWithDetails {
	rules := c.store.ListRouteRules()
	chains := c.ChainsWithDetails()
	chainMap := make(map[string]domain.ChainWithDetails)
	for _, chain := range chains {
		chainMap[chain.ID] = chain
	}

	result := make([]domain.RouteRuleWithDetails, 0, len(rules))
	for _, rule := range rules {
		item := domain.RouteRuleWithDetails{
			ID:               rule.ID,
			Priority:         rule.Priority,
			MatchType:        rule.MatchType,
			MatchValue:       rule.MatchValue,
			ActionType:       rule.ActionType,
			ChainID:          rule.ChainID,
			DestinationScope: rule.DestinationScope,
			Enabled:          rule.Enabled,
		}
		if rule.ChainID != "" {
			if chain, ok := chainMap[rule.ChainID]; ok {
				item.Chain = &chain
			}
		}
		result = append(result, item)
	}
	return result
}

func (c *ControlPlane) GetRouteRule(ruleID string) (domain.RouteRuleWithDetails, error) {
	if ruleID == "" {
		return domain.RouteRuleWithDetails{}, invalidInput("missing_rule_id")
	}

	rules := c.RouteRulesWithDetails()
	for _, rule := range rules {
		if rule.ID == ruleID {
			return rule, nil
		}
	}
	return domain.RouteRuleWithDetails{}, invalidInput("route_rule_not_found")
}

func (c *ControlPlane) MatchTypes() []domain.MatchType {
	domainRegex := "^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$"
	domainSuffixRegex := "^\\*\\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$"
	ipCIDRRegex := "^([0-9]{1,3}\\.){3}[0-9]{1,3}/[0-9]{1,2}$"
	ipRangeRegex := "^([0-9]{1,3}\\.){3}[0-9]{1,3}-([0-9]{1,3}\\.){3}[0-9]{1,3}$"
	portRegex := "^[0-9]{1,5}$"

	return []domain.MatchType{
		{
			Type:            "domain",
			Label:           "Domain",
			Description:     "Match exact domain name",
			Placeholder:     "example.com",
			ValidationRegex: &domainRegex,
		},
		{
			Type:            "domain_suffix",
			Label:           "Domain Suffix",
			Description:     "Match domain suffix (e.g., *.example.com)",
			Placeholder:     "*.example.com",
			ValidationRegex: &domainSuffixRegex,
		},
		{
			Type:            "ip_cidr",
			Label:           "IP CIDR",
			Description:     "Match IP address range in CIDR notation",
			Placeholder:     "10.0.0.0/24",
			ValidationRegex: &ipCIDRRegex,
		},
		{
			Type:            "ip_range",
			Label:           "IP Range",
			Description:     "Match IP address range",
			Placeholder:     "10.0.0.1-10.0.0.255",
			ValidationRegex: &ipRangeRegex,
		},
		{
			Type:            "port",
			Label:           "Port",
			Description:     "Match port number",
			Placeholder:     "8080",
			ValidationRegex: &portRegex,
		},
		{
			Type:            "url_regex",
			Label:           "URL Regex",
			Description:     "Match URL using regular expression",
			Placeholder:     "^https://api\\..*",
			ValidationRegex: nil,
		},
		{
			Type:            "default",
			Label:           "Default",
			Description:     "Catch-all rule (lowest priority)",
			Placeholder:     "*",
			ValidationRegex: nil,
		},
	}
}

func (c *ControlPlane) CreateRouteRule(input domain.CreateRouteRuleInput) (domain.RouteRule, error) {
	if err := validateRouteRule(input.ActionType, input.ChainID, input.DestinationScope, input.MatchType, input.MatchValue); err != nil {
		return domain.RouteRule{}, err
	}
	return c.store.CreateRouteRule(input)
}

func (c *ControlPlane) UpdateRouteRule(ruleID string, input domain.UpdateRouteRuleInput) (domain.RouteRule, error) {
	if ruleID == "" {
		return domain.RouteRule{}, invalidInput("missing_rule_id")
	}
	if err := validateRouteRule(input.ActionType, input.ChainID, input.DestinationScope, input.MatchType, input.MatchValue); err != nil {
		return domain.RouteRule{}, err
	}
	return c.store.UpdateRouteRule(ruleID, input)
}

func (c *ControlPlane) DeleteRouteRule(ruleID string) error {
	return c.store.DeleteRouteRule(ruleID)
}

func (c *ControlPlane) ValidateRouteRule(input domain.ValidateRouteRuleInput) (domain.RouteRuleValidationResult, error) {
	result := domain.RouteRuleValidationResult{
		Valid:   true,
		Errors:  []string{},
		Warnings: []string{},
	}

	result.MatchValueValidation = validateMatchValue(input.MatchType, input.MatchValue)
	if !result.MatchValueValidation.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, result.MatchValueValidation.Message)
	}

	chains := c.store.ListChains()
	var matchedChain *domain.Chain
	for _, chain := range chains {
		if chain.ID == input.ChainID {
			c := chain
			matchedChain = &c
			break
		}
	}
	if matchedChain == nil {
		result.ChainValidation = domain.ChainValidation{
			Valid:        false,
			ChainEnabled: false,
		}
		result.Errors = append(result.Errors, "chain_not_found")
	} else {
		result.ChainValidation = domain.ChainValidation{
			Valid:        true,
			ChainEnabled: matchedChain.Enabled,
			ChainHops:    matchedChain.Hops,
		}
		if !matchedChain.Enabled {
			result.Warnings = append(result.Warnings, "Selected chain is disabled")
		}
	}

	scopes := c.NodeScopes()
	var matchedScope *domain.NodeScope
	for _, scope := range scopes {
		if scope.ScopeKey == input.DestinationScope {
			s := scope
			matchedScope = &s
			break
		}
	}
	if matchedScope == nil {
		result.ScopeValidation = domain.ScopeValidation{
			Valid:       false,
			ScopeExists: false,
		}
		result.Errors = append(result.Errors, "scope_not_found")
	} else {
		matchesFinalHop := false
		if matchedChain != nil && len(matchedChain.Hops) > 0 {
			matchesFinalHop = matchedChain.Hops[len(matchedChain.Hops)-1] == matchedScope.OwnerNodeID
		}
		result.ScopeValidation = domain.ScopeValidation{
			Valid:                true,
			ScopeExists:          true,
			ScopeOwnerNodeID:     matchedScope.OwnerNodeID,
			MatchesChainFinalHop: matchesFinalHop,
		}
		if !matchesFinalHop && matchedChain != nil && len(matchedChain.Hops) > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Scope %s is not owned by chain's final hop node", input.DestinationScope))
		}
	}

	rules := c.store.ListRouteRules()
	for _, rule := range rules {
		if rule.Priority == input.Priority {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Priority %d conflicts with existing rule", input.Priority))
			break
		}
	}

	return result, nil
}

func validateMatchValue(matchType, matchValue string) domain.MatchValueValidation {
	switch matchType {
	case "domain":
		pattern := `^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`
		matched, _ := regexp.MatchString(pattern, matchValue)
		if matched {
			return domain.MatchValueValidation{Valid: true, Format: "domain", Message: "Valid domain format"}
		}
		return domain.MatchValueValidation{Valid: false, Format: "domain", Message: "Invalid domain format"}
	case "domain_suffix":
		if !strings.HasPrefix(matchValue, ".") && !strings.HasPrefix(matchValue, "*.") {
			return domain.MatchValueValidation{Valid: false, Format: "domain_suffix", Message: "Domain suffix must start with . or *."}
		}
		return domain.MatchValueValidation{Valid: true, Format: "domain_suffix", Message: "Valid domain suffix"}
	case "ip_cidr":
		_, _, err := net.ParseCIDR(matchValue)
		if err != nil {
			return domain.MatchValueValidation{Valid: false, Format: "ip_cidr", Message: "Invalid CIDR notation"}
		}
		return domain.MatchValueValidation{Valid: true, Format: "ip_cidr", Message: "Valid CIDR notation"}
	case "ip_range":
		parts := strings.SplitN(matchValue, "-", 2)
		if len(parts) != 2 || net.ParseIP(strings.TrimSpace(parts[0])) == nil || net.ParseIP(strings.TrimSpace(parts[1])) == nil {
			return domain.MatchValueValidation{Valid: false, Format: "ip_range", Message: "Invalid IP range format"}
		}
		return domain.MatchValueValidation{Valid: true, Format: "ip_range", Message: "Valid IP range"}
	case "port":
		p, err := strconv.Atoi(matchValue)
		if err != nil || p < 1 || p > 65535 {
			return domain.MatchValueValidation{Valid: false, Format: "port", Message: "Port must be between 1 and 65535"}
		}
		return domain.MatchValueValidation{Valid: true, Format: "port", Message: "Valid port"}
	case "url_regex":
		_, err := regexp.Compile(matchValue)
		if err != nil {
			return domain.MatchValueValidation{Valid: false, Format: "url_regex", Message: fmt.Sprintf("Invalid regex: %s", err.Error())}
		}
		return domain.MatchValueValidation{Valid: true, Format: "url_regex", Message: "Valid regex pattern"}
	case "default":
		return domain.MatchValueValidation{Valid: true, Format: "default", Message: "Default match type"}
	default:
		return domain.MatchValueValidation{Valid: false, Format: matchType, Message: "Unknown match type"}
	}
}

func (c *ControlPlane) RouteRuleSuggestions(matchType string, query string) domain.RouteRuleSuggestionResult {
	rules := c.store.ListRouteRules()
	seen := make(map[string]struct{})
	var suggestions []string

	for _, rule := range rules {
		if rule.MatchType != matchType {
			continue
		}
		if rule.MatchValue == "" {
			continue
		}
		if query != "" && !strings.HasPrefix(strings.ToLower(rule.MatchValue), strings.ToLower(query)) {
			continue
		}
		if _, ok := seen[rule.MatchValue]; ok {
			continue
		}
		seen[rule.MatchValue] = struct{}{}
		suggestions = append(suggestions, rule.MatchValue)
	}

	if suggestions == nil {
		suggestions = []string{}
	}

	return domain.RouteRuleSuggestionResult{
		MatchType:   matchType,
		Suggestions: suggestions,
	}
}

func (c *ControlPlane) NodeHealth() []domain.NodeHealth {
	return c.store.ListNodeHealth()
}

func (c *ControlPlane) NodeHealthHistory(nodeID string, window time.Duration) ([]domain.NodeHealth, error) {
	if nodeID == "" {
		return nil, invalidInput("missing_node_id")
	}
	if window <= 0 || window > 7*24*time.Hour {
		window = 24 * time.Hour
	}
	return c.store.ListNodeHealthHistory(nodeID, window)
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

func toChainProbeInput(result domain.ChainProbeResult) domain.SaveChainProbeResultInput {
	return domain.SaveChainProbeResultInput{
		ChainID:        result.ChainID,
		Status:         result.Status,
		Message:        result.Message,
		ResolvedHops:   result.ResolvedHops,
		BlockingNodeID: result.BlockingNodeID,
		BlockingReason: result.BlockingReason,
		TargetHost:     result.TargetHost,
		TargetPort:     result.TargetPort,
		ProbedAt:       result.ProbedAt,
	}
}

func resolveProbeTransport(node domain.Node, prevHopID string, transports []domain.NodeTransport) (domain.NodeTransport, bool) {
	if prevHopID != "" {
		for _, transport := range transports {
			if transport.NodeID != node.ID || transport.ParentNodeID != prevHopID {
				continue
			}
			if transport.Status != "connected" {
				continue
			}
			if strings.HasPrefix(transport.TransportType, "reverse_ws") || strings.HasPrefix(transport.TransportType, "child_ws") {
				return transport, true
			}
		}
	}
	for _, transport := range transports {
		if transport.NodeID != node.ID {
			continue
		}
		if transport.TransportType == "public_https" || transport.TransportType == "public_http" {
			return transport, true
		}
	}
	return domain.NodeTransport{}, false
}

func (c *ControlPlane) CreateBootstrapToken(input domain.CreateBootstrapTokenInput) (domain.BootstrapToken, error) {
	if input.TargetType == "" {
		return domain.BootstrapToken{}, invalidInput("invalid_bootstrap_payload")
	}
	return c.store.CreateBootstrapToken(input)
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

func (c *ControlPlane) PolicyRevisions() []domain.PolicyRevision {
	return c.store.ListPolicyRevisions()
}

func (c *ControlPlane) PublishPolicy(accountID string) (domain.PolicyRevision, error) {
	if accountID == "" {
		return domain.PolicyRevision{}, unauthorized("invalid_access_token")
	}
	if _, err := policy.Compile(c.store.ListNodes(), c.store.ListNodeLinks(), c.store.ListChains(), c.store.ListRouteRules()); err != nil {
		return domain.PolicyRevision{}, invalidInput("invalid_policy_graph")
	}
	return c.store.PublishPolicy(accountID)
}

func (c *ControlPlane) AuthenticateNodeToken(accessToken string) (string, bool) {
	return c.store.AuthenticateNodeToken(accessToken)
}

func (c *ControlPlane) NodeAgentPolicy(nodeID string) (domain.NodeAgentPolicy, bool) {
	return c.store.GetNodeAgentPolicy(nodeID)
}

func (c *ControlPlane) UpsertNodeHeartbeat(input domain.NodeHeartbeatInput) (domain.NodeHealth, error) {
	if input.NodeID == "" {
		return domain.NodeHealth{}, invalidInput("missing_node_id")
	}
	return c.store.UpsertNodeHeartbeat(input)
}

func (c *ControlPlane) UpsertNodeAgentTransport(nodeID string, input domain.UpsertNodeTransportInput) (domain.NodeTransport, error) {
	input.NodeID = nodeID
	return c.UpsertNodeTransport(input)
}

func (c *ControlPlane) RenewNodeCertificate(input domain.NodeCertRenewInput) (domain.NodeCertRenewResult, error) {
	if input.NodeID == "" || input.CertType == "" {
		return domain.NodeCertRenewResult{}, invalidInput("invalid_cert_renew_payload")
	}
	return c.store.RenewNodeCertificate(input)
}

func (c *ControlPlane) RunMaintenance() error {
	if _, err := c.store.CleanupExpiredSessions(); err != nil {
		return err
	}
	if _, err := c.store.CleanupExpiredBootstrapTokens(); err != nil {
		return err
	}
	if _, err := c.store.CleanupExpiredNodeTokens(); err != nil {
		return err
	}
	if err := c.store.RefreshCertificateStatus(c.publicRenewWindow); err != nil {
		return err
	}
	for _, cert := range c.store.ListCertificates() {
		if cert.OwnerType != "node" || cert.CertType != "public" {
			continue
		}
		if cert.Status != "renew-soon" && cert.Status != "expired" {
			continue
		}
		if _, err := c.store.RenewNodeCertificate(domain.NodeCertRenewInput{
			NodeID:   cert.OwnerID,
			CertType: cert.CertType,
		}); err != nil {
			return err
		}
	}
	if err := c.store.RefreshNodeStatus(c.nodeHeartbeatTTL); err != nil {
		return err
	}
	if removed, err := c.store.CleanupNodeHealthHistory(7 * 24 * time.Hour); err != nil {
		log.Printf("maintenance: failed to cleanup node health history: %v", err)
	} else if removed > 0 {
		log.Printf("maintenance: cleaned up %d stale health history rows", removed)
	}
	return nil
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

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

func validateNodeInput(name string, mode string, scopeKey string) error {
	if name == "" || mode == "" || scopeKey == "" {
		return invalidInput("invalid_node_payload")
	}
	return nil
}

func validateRouteRule(actionType string, chainID string, destinationScope string, matchType string, matchValue string) error {
	if matchType == "" || matchValue == "" || actionType == "" {
		return invalidInput("invalid_route_rule_payload")
	}
	switch actionType {
	case "chain":
		if chainID == "" {
			return invalidInput("invalid_route_rule_payload")
		}
	case "direct":
		if destinationScope == "" {
			return invalidInput("invalid_route_rule_payload")
		}
	default:
		return invalidInput("invalid_route_rule_payload")
	}
	return nil
}

func validateNodeAccessPath(name string, mode string, targetHost string, targetPort int) error {
	if name == "" || mode == "" {
		return invalidInput("invalid_node_access_path_payload")
	}
	switch mode {
	case "direct", "relay_chain":
		if targetHost == "" || targetPort <= 0 {
			return invalidInput("invalid_node_access_path_payload")
		}
	case "upstream_pull":
	default:
		return invalidInput("invalid_node_access_path_payload")
	}
	return nil
}

func validateNodeOnboardingTask(mode string, pathID string, targetHost string, targetPort int) error {
	switch mode {
	case "direct":
		if targetHost == "" || targetPort <= 0 {
			return invalidInput("invalid_node_onboarding_task_payload")
		}
	case "relay_chain", "upstream_pull":
		if pathID == "" {
			return invalidInput("invalid_node_onboarding_task_payload")
		}
	default:
		return invalidInput("invalid_node_onboarding_task_payload")
	}
	return nil
}

func probeDirectNodeTarget(targetHost string, targetPort int) (string, string) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/healthz", targetHost, targetPort))
	if err != nil {
		return "failed", "target_unreachable"
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return "failed", "target_unhealthy"
	}
	return "connected", "target_reachable"
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
		return "failed", "invalid_node_access_path"
	}
	relayURLs, ok := relayURLsForPath(c.store.ListNodes(), path)
	if !ok || len(relayURLs) == 0 {
		return "failed", "invalid_relay_chain"
	}
	result, err := controlrelay.Execute(relayURLs[0], controlrelay.ProbeRequest{
		RemainingRelayURLs: relayURLs[1:],
		TargetHost:         path.TargetHost,
		TargetPort:         path.TargetPort,
	})
	if err != nil {
		return "failed", "relay_probe_failed"
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

func chainByID(items []domain.Chain, chainID string) (domain.Chain, bool) {
	for _, item := range items {
		if item.ID == chainID {
			return item, true
		}
	}
	return domain.Chain{}, false
}
