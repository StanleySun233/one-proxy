package service

import (
	"strings"
	"time"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (c *ControlPlane) Overview() domain.Overview {
	return c.store.GetOverview()
}

func (c *ControlPlane) ExtensionBootstrap(account domain.Account) domain.ExtensionBootstrap {
	nodes := c.store.ListNodes()
	rules := c.store.ListRouteRules()
	chains := c.store.ListChains()
	overview := c.store.GetOverview()
	fetchedAt := time.Now().UTC().Format(time.RFC3339)
	chainsByID := make(map[string]domain.Chain, len(chains))
	for _, chain := range chains {
		chainsByID[chain.ID] = chain
	}
	nodesByID := make(map[string]domain.Node, len(nodes))
	for _, node := range nodes {
		nodesByID[node.ID] = node
	}

	filteredNodes := nodes
	filteredRules := rules
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
			filteredNodes = make([]domain.Node, 0)
			for _, node := range nodes {
				if allowedScopes[node.ScopeKey] {
					filteredNodes = append(filteredNodes, node)
				}
			}
			filteredRules = make([]domain.RouteRule, 0)
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
		}
	}

	groups := make([]domain.ExtensionGroup, 0)
	for _, node := range filteredNodes {
		if !node.Enabled || node.PublicHost == "" || node.PublicPort <= 0 {
			continue
		}
		if node.Mode != domain.NodeModeEdge && node.ParentNodeID != "" {
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
			Topology:      extensionTopology(nodesByID, []string{node.ID}),
		}
		for _, rule := range filteredRules {
			if !rule.Enabled {
				continue
			}
			topology := group.Topology
			if rule.ActionType == domain.ActionTypeChain {
				chain, ok := chainsByID[rule.ChainID]
				if !ok || !chain.Enabled || len(chain.Hops) == 0 || chain.Hops[0] != node.ID {
					continue
				}
				topology = extensionTopology(nodesByID, chain.Hops)
			}
			if rule.ActionType == domain.ActionTypeDirect && rule.MatchType != domain.MatchTypeDefault && rule.DestinationScope != node.ScopeKey {
				continue
			}
			group.Routes = append(group.Routes, domain.ExtensionRoute{
				ID:               rule.ID,
				Priority:         rule.Priority,
				MatchType:        rule.MatchType,
				MatchValue:       rule.MatchValue,
				ActionType:       rule.ActionType,
				ChainID:          rule.ChainID,
				DestinationScope: rule.DestinationScope,
				Topology:         topology,
			})
			value := strings.TrimSpace(rule.MatchValue)
			switch rule.MatchType {
			case domain.MatchTypeDefault:
				if rule.ActionType == domain.ActionTypeChain {
					group.ProxyDefault = true
				}
			case domain.MatchTypeDomain:
				if value == "" {
					continue
				}
				if rule.ActionType == domain.ActionTypeDirect {
					group.DirectHosts = append(group.DirectHosts, value)
				} else if rule.ActionType == domain.ActionTypeChain {
					group.ProxyHosts = append(group.ProxyHosts, value)
				}
			case domain.MatchTypeDomainSuffix:
				if value == "" {
					continue
				}
				pattern := value
				if strings.HasPrefix(pattern, ".") {
					pattern = "*" + pattern
				}
				if rule.ActionType == domain.ActionTypeDirect {
					group.DirectHosts = append(group.DirectHosts, pattern)
				} else if rule.ActionType == domain.ActionTypeChain {
					group.ProxyHosts = append(group.ProxyHosts, pattern)
				}
			case domain.MatchTypeIPCIDR:
				if value == "" {
					continue
				}
				if rule.ActionType == domain.ActionTypeDirect {
					group.ProxyCIDRs = append(group.ProxyCIDRs, value)
				} else if rule.ActionType == domain.ActionTypeChain {
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

func extensionTopology(nodesByID map[string]domain.Node, nodeIDs []string) []domain.ExtensionTopologyNode {
	result := make([]domain.ExtensionTopologyNode, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		node, ok := nodesByID[nodeID]
		if !ok {
			continue
		}
		result = append(result, domain.ExtensionTopologyNode{
			ID:         node.ID,
			Name:       node.Name,
			Mode:       node.Mode,
			ScopeKey:   node.ScopeKey,
			PublicHost: node.PublicHost,
			PublicPort: node.PublicPort,
		})
	}
	return result
}
