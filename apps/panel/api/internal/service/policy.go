package service

import (
	"net/http"
	"time"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
)

func (c *ControlPlane) PolicyRevisions(tenantCtx domain.TenantAuthContext) []domain.PolicyRevision {
	if tenantCtx.ActiveTenant.TenantID == "" && tenantCtx.SuperAdmin {
		return c.store.ListPolicyRevisions()
	}
	return c.store.ListPolicyRevisionsForTenant(tenantCtx)
}

func (c *ControlPlane) PublishPolicy(tenantCtx domain.TenantAuthContext, accountID string) (domain.PolicyRevision, error) {
	if accountID == "" {
		return domain.PolicyRevision{}, unauthorized("invalid_access_token")
	}
	if tenantCtx.ActiveTenant.TenantID == "" {
		return domain.PolicyRevision{}, invalidInput("tenant_required")
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return domain.PolicyRevision{}, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	item, err := c.store.PublishPolicy(tenantCtx, accountID)
	if err != nil {
		return domain.PolicyRevision{}, invalidInput("invalid_policy_graph")
	}
	if len(item.AffectedTenantIDs) == 0 {
		item.AffectedTenantIDs = []string{tenantCtx.ActiveTenant.TenantID}
	}
	return item, nil
}

func (c *ControlPlane) NodeAgentPolicy(nodeID string) (domain.NodeAgentPolicy, bool) {
	return c.store.GetNodeAgentPolicy(nodeID)
}

func (c *ControlPlane) ExtensionBootstrapForTenant(account domain.Account, tenantCtx domain.TenantAuthContext) (domain.ExtensionBootstrap, bool) {
	proxyToken, proxyTokenExpiresAt, ok := c.IssueProxyToken(account, tenantCtx)
	if !ok {
		return domain.ExtensionBootstrap{}, false
	}
	scopedTenantCtx := tenantCtx
	scopedTenantCtx.SuperAdmin = false
	nodes := c.store.ListNodesForTenant(scopedTenantCtx)
	chains := c.store.ListChainsForTenant(scopedTenantCtx)
	rules := c.store.ListPolicyRouteRulesForTenant(scopedTenantCtx)
	accessPaths := c.store.ListNodeAccessPathsForTenant(scopedTenantCtx)
	if bootstrapStore, ok := c.store.(interface {
		ExtensionBootstrapResourcesForTenant(domain.TenantAuthContext) ([]domain.Node, []proxy.Chain, []proxy.RouteRule)
	}); ok {
		nodes, chains, rules = bootstrapStore.ExtensionBootstrapResourcesForTenant(tenantCtx)
	}
	nodes = extensionBootstrapNodes(nodes, c.store.ListNodes(), chains, rules)
	overview := c.store.GetOverview()
	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	chainsByID := make(map[string]proxy.Chain, len(chains))
	for _, chain := range chains {
		chainsByID[chain.ID] = chain
	}
	nodesByID := make(map[string]domain.Node, len(nodes))
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}

	filteredRules := rules
	filteredAccessPaths := accessPaths
	if account.Role != domain.AccountRoleSuperAdmin {
		accountGroups, err := c.store.ListAccountGroups(account.ID)
		if err == nil && len(accountGroups) > 0 {
			allowedScopes := make(map[string]bool)
			for _, g := range accountGroups {
				scopes, _ := c.store.GetGroupScopes(g.ID)
				for _, scope := range scopes {
					allowedScopes[scope] = true
				}
			}
			filteredRules = make([]proxy.RouteRule, 0)
			for _, rule := range rules {
				if rule.ActionType == domain.ActionTypeDirect && allowedScopes[rule.DestinationScope] {
					filteredRules = append(filteredRules, rule)
					continue
				}
				chain, ok := chainsByID[rule.ChainID]
				if rule.ActionType == domain.ActionTypeChain && ok && allowedScopes[chain.DestinationScope] {
					filteredRules = append(filteredRules, rule)
				}
			}
			filteredAccessPaths = make([]domain.NodeAccessPath, 0)
			for _, path := range accessPaths {
				chain, ok := chainsByID[path.ChainID]
				if ok && allowedScopes[chain.DestinationScope] {
					filteredAccessPaths = append(filteredAccessPaths, path)
				}
			}
		}
	}

	return domain.ExtensionBootstrap{
		SchemaVersion:       "v2.1.0",
		Account:             account,
		Tenant:              domain.ExtensionTenant{ID: tenantCtx.ActiveTenant.TenantID, Name: tenantCtx.ActiveTenant.TenantName},
		PolicyRevision:      overview.Policies.ActiveRevision,
		FetchedAt:           fetchedAt,
		ProxyToken:          proxyToken,
		ProxyTokenExpiresAt: proxyTokenExpiresAt,
		Nodes:               nodes,
		AccessPaths:         extensionAccessPaths(filteredAccessPaths, chainsByID, nodesByID, fetchedAt),
		Routes:              extensionRoutes(filteredRules, filteredAccessPaths, chainsByID, nodesByID),
		RouteEvaluation: domain.ExtensionRouteEvaluation{
			DefaultClientMode:     "direct",
			DefaultNodeMode:       "deny",
			RuleOrder:             "priority_asc_then_id_asc",
			NoMatchNodeDenyReason: "route_not_found",
			SupportedMatchTypes:   []string{"domain", "domain_suffix", "ip", "ip_cidr", "protocol", "default"},
			SupportedActions:      []string{"chain", "direct", "deny"},
		},
	}, true
}

func extensionAccessPaths(paths []domain.NodeAccessPath, chainsByID map[string]proxy.Chain, nodesByID map[string]domain.Node, fetchedAt string) []domain.ExtensionAccessPath {
	result := make([]domain.ExtensionAccessPath, 0, len(paths))
	for _, path := range paths {
		chain, ok := chainsByID[path.ChainID]
		if !ok || !chain.Enabled {
			continue
		}
		result = append(result, domain.ExtensionAccessPath{
			ID:             path.ID,
			Name:           path.Name,
			ChainID:        path.ChainID,
			Mode:           path.Mode,
			Protocol:       path.Protocol,
			ServiceType:    path.ServiceType,
			TargetNodeID:   path.TargetNodeID,
			EntryNodeID:    path.EntryNodeID,
			RelayNodeIDs:   append([]string(nil), path.RelayNodeIDs...),
			ListenHost:     path.ListenHost,
			ListenPort:     path.ListenPort,
			TargetProtocol: path.TargetProtocol,
			TargetHost:     path.TargetHost,
			TargetPort:     path.TargetPort,
			TargetSNI:      path.TargetSNI,
			TLSMode:        path.TLSMode,
			AuthMode:       path.AuthMode,
			Enabled:        path.Enabled,
			Options:        cloneStringMap(path.Options),
			Topology:       extensionTopology(nodesByID, chain.Hops, extensionTransport(path)),
			Health: domain.ExtensionPathHealth{
				Status:    extensionPathStatus(path, chain, nodesByID),
				Reason:    extensionPathReason(path, chain, nodesByID),
				CheckedAt: fetchedAt,
			},
		})
	}
	return result
}

func extensionRoutes(rules []proxy.RouteRule, paths []domain.NodeAccessPath, chainsByID map[string]proxy.Chain, nodesByID map[string]domain.Node) []domain.ExtensionRoute {
	pathsByChainID := make(map[string]domain.NodeAccessPath, len(paths))
	for _, path := range paths {
		if path.Enabled {
			if _, ok := pathsByChainID[path.ChainID]; !ok {
				pathsByChainID[path.ChainID] = path
			}
		}
	}
	result := make([]domain.ExtensionRoute, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		path := pathsByChainID[rule.ChainID]
		if rule.ActionType == domain.ActionTypeChain && path.ID == "" {
			continue
		}
		chain := chainsByID[rule.ChainID]
		topology := extensionTopology(nodesByID, chain.Hops, extensionTransport(path))
		result = append(result, domain.ExtensionRoute{
			ID:               rule.ID,
			Priority:         rule.Priority,
			MatchType:        rule.MatchType,
			MatchValue:       rule.MatchValue,
			ActionType:       rule.ActionType,
			ChainID:          rule.ChainID,
			AccessPathID:     path.ID,
			DestinationScope: rule.DestinationScope,
			Enabled:          rule.Enabled,
			Topology:         topology,
		})
	}
	return result
}

func extensionTopology(nodesByID map[string]domain.Node, nodeIDs []string, transport string) []domain.ExtensionTopologyHop {
	result := make([]domain.ExtensionTopologyHop, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		node, ok := nodesByID[nodeID]
		if !ok {
			continue
		}
		result = append(result, domain.ExtensionTopologyHop{
			NodeID:     node.ID,
			NodeName:   node.Name,
			Mode:       node.Mode,
			ScopeKey:   node.ScopeKey,
			PublicHost: node.PublicHost,
			PublicPort: node.PublicPort,
			Transport:  transport,
		})
	}
	return result
}

func extensionTransport(path domain.NodeAccessPath) string {
	switch path.Mode {
	case "direct":
		return "direct_quic"
	case "tcp":
		return "tcp_access"
	case "udp":
		return "udp_access"
	case "reverse":
		return "reverse_ws"
	default:
		if path.Protocol == "https" {
			return "public_https"
		}
		return "public_http"
	}
}

func extensionPathStatus(path domain.NodeAccessPath, chain proxy.Chain, nodesByID map[string]domain.Node) string {
	if !path.Enabled || !chain.Enabled {
		return "disabled"
	}
	for _, nodeID := range chain.Hops {
		node, ok := nodesByID[nodeID]
		if !ok || !node.Enabled {
			return "blocked"
		}
	}
	return "ready"
}

func extensionPathReason(path domain.NodeAccessPath, chain proxy.Chain, nodesByID map[string]domain.Node) string {
	if !path.Enabled {
		return "access_path_disabled"
	}
	if !chain.Enabled {
		return "chain_disabled"
	}
	for _, nodeID := range chain.Hops {
		node, ok := nodesByID[nodeID]
		if !ok || !node.Enabled {
			return "node_unavailable"
		}
	}
	return "ok"
}

func cloneStringMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func extensionBootstrapNodes(scoped []domain.Node, all []domain.Node, chains []proxy.Chain, rules []proxy.RouteRule) []domain.Node {
	byID := make(map[string]domain.Node, len(all))
	byScope := make(map[string][]domain.Node)
	for _, node := range all {
		byID[node.ID] = node
		byScope[node.ScopeKey] = append(byScope[node.ScopeKey], node)
	}
	included := make(map[string]bool, len(scoped))
	result := make([]domain.Node, 0, len(scoped))
	add := func(node domain.Node) {
		if node.ID == "" || included[node.ID] {
			return
		}
		included[node.ID] = true
		result = append(result, node)
	}
	for _, node := range scoped {
		add(node)
	}
	chainsByID := make(map[string]proxy.Chain, len(chains))
	for _, chain := range chains {
		chainsByID[chain.ID] = chain
	}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.ActionType == domain.ActionTypeChain {
			chain, ok := chainsByID[rule.ChainID]
			if !ok || !chain.Enabled {
				continue
			}
			for _, nodeID := range chain.Hops {
				add(byID[nodeID])
			}
			continue
		}
		if rule.ActionType == domain.ActionTypeDirect {
			for _, node := range byScope[rule.DestinationScope] {
				add(node)
			}
		}
	}
	return result
}
