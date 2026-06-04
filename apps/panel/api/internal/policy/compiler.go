package policy

import (
	"encoding/json"
	"fmt"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
)

type GroupScopeEntry struct {
	GroupName  string   `json:"groupName"`
	ScopeKeys  []string `json:"scopeKeys"`
	AccountIDs []string `json:"accountIds"`
}

type Snapshot struct {
	TenantID   string            `json:"tenantId,omitempty"`
	Nodes      []domain.Node     `json:"nodes"`
	Links      []domain.NodeLink `json:"links"`
	Chains     []proxy.Chain     `json:"chains"`
	RouteRules []proxy.RouteRule `json:"routeRules"`
	Groups     []GroupScopeEntry `json:"groups"`
}

func CompileForTenant(tenantID string, nodes []domain.Node, links []domain.NodeLink, chains []proxy.Chain, rules []proxy.RouteRule, groups []GroupScopeEntry) (string, error) {
	raw, err := Compile(nodes, links, chains, rules, groups)
	if err != nil {
		return "", err
	}
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return "", err
	}
	snapshot.TenantID = tenantID
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func Compile(nodes []domain.Node, links []domain.NodeLink, chains []proxy.Chain, rules []proxy.RouteRule, groups []GroupScopeEntry) (string, error) {
	activeNodes := make([]domain.Node, 0, len(nodes))
	nodeSet := make(map[string]domain.Node, len(nodes))
	for _, node := range nodes {
		if !node.Enabled || node.Status == domain.NodeStatusPending {
			continue
		}
		activeNodes = append(activeNodes, node)
		nodeSet[node.ID] = node
	}
	for _, chain := range chains {
		if !chain.Enabled {
			continue
		}
		if len(chain.Hops) == 0 {
			return "", fmt.Errorf("chain %s has no hops", chain.ID)
		}
		seen := map[string]struct{}{}
		for _, hop := range chain.Hops {
			if _, ok := nodeSet[hop]; !ok {
				return "", fmt.Errorf("chain %s references unknown_or_disabled_node %s", chain.ID, hop)
			}
			if _, ok := seen[hop]; ok {
				return "", fmt.Errorf("chain %s contains loop at %s", chain.ID, hop)
			}
			seen[hop] = struct{}{}
		}
		lastHop := chain.Hops[len(chain.Hops)-1]
		if nodeSet[lastHop].ScopeKey != chain.DestinationScope {
			return "", fmt.Errorf("chain %s destination_scope_mismatch", chain.ID)
		}
	}
	chainSet := make(map[string]struct{}, len(chains))
	for _, chain := range chains {
		if chain.Enabled {
			chainSet[chain.ID] = struct{}{}
		}
	}
	compiledRules := make([]proxy.RouteRule, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.ActionType != domain.ActionTypeChain && rule.ActionType != domain.ActionTypeDirect {
			return "", fmt.Errorf("rule %s has invalid_action_type", rule.ID)
		}
		switch rule.ActionType {
		case domain.ActionTypeChain:
			if _, ok := chainSet[rule.ChainID]; !ok {
				return "", fmt.Errorf("rule %s references unknown_chain %s", rule.ID, rule.ChainID)
			}
		case domain.ActionTypeDirect:
			if rule.DestinationScope == "" {
				return "", fmt.Errorf("rule %s missing_destination_scope", rule.ID)
			}
		}
		compiledRules = append(compiledRules, rule)
	}
	compiledLinks := make([]domain.NodeLink, 0, len(links))
	for _, nodeLink := range links {
		if _, ok := nodeSet[nodeLink.SourceNodeID]; !ok {
			continue
		}
		if _, ok := nodeSet[nodeLink.TargetNodeID]; !ok {
			continue
		}
		compiledLinks = append(compiledLinks, nodeLink)
	}
	payload, err := json.Marshal(Snapshot{
		Nodes:      activeNodes,
		Links:      compiledLinks,
		Chains:     chains,
		RouteRules: compiledRules,
		Groups:     groups,
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func CompileForNode(nodeID string, nodes []domain.Node, links []domain.NodeLink, chains []proxy.Chain, rules []proxy.RouteRule, groups []GroupScopeEntry) (string, error) {
	raw, err := Compile(nodes, links, chains, rules, groups)
	if err != nil {
		return "", err
	}
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		return "", err
	}
	currentScope := ""
	for _, node := range snapshot.Nodes {
		if node.ID == nodeID {
			currentScope = node.ScopeKey
			break
		}
	}
	filteredChains := make([]proxy.Chain, 0)
	visibleChainIDs := make(map[string]struct{})
	for _, chain := range snapshot.Chains {
		include := chain.DestinationScope == currentScope
		if !include {
			for _, hop := range chain.Hops {
				if hop == nodeID {
					include = true
					break
				}
			}
		}
		if include {
			filteredChains = append(filteredChains, chain)
			visibleChainIDs[chain.ID] = struct{}{}
		}
	}
	filteredRules := make([]proxy.RouteRule, 0)
	for _, rule := range snapshot.RouteRules {
		if rule.ActionType == domain.ActionTypeChain {
			if _, ok := visibleChainIDs[rule.ChainID]; ok {
				filteredRules = append(filteredRules, rule)
			}
			continue
		}
		if rule.DestinationScope == currentScope {
			filteredRules = append(filteredRules, rule)
		}
	}
	filteredLinks := make([]domain.NodeLink, 0)
	for _, nodeLink := range snapshot.Links {
		if nodeLink.SourceNodeID == nodeID || nodeLink.TargetNodeID == nodeID {
			filteredLinks = append(filteredLinks, nodeLink)
		}
	}
	payload, err := json.Marshal(Snapshot{
		Nodes:      snapshot.Nodes,
		Links:      filteredLinks,
		Chains:     filteredChains,
		RouteRules: filteredRules,
		Groups:     snapshot.Groups,
	})
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
