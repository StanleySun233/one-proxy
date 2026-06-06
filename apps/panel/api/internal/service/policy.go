package service

import (
	"net/http"
	"strings"
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
	rules := c.store.ListRouteRulesForTenant(scopedTenantCtx)
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
					group.DirectCIDRs = append(group.DirectCIDRs, value)
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
		Account:             account,
		PolicyRevision:      overview.Policies.ActiveRevision,
		FetchedAt:           fetchedAt,
		ProxyToken:          proxyToken,
		ProxyTokenExpiresAt: proxyTokenExpiresAt,
		Groups:              groups,
	}, true
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
