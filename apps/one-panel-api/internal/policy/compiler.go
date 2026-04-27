package policy

import (
	"encoding/json"
	"fmt"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

type GroupScopeEntry struct {
	GroupName string   `json:"groupName"`
	ScopeKeys []string `json:"scopeKeys"`
	AccountIDs []string `json:"accountIds"`
}

type Snapshot struct {
	Nodes      []domain.Node      `json:"nodes"`
	Links      []domain.NodeLink  `json:"links"`
	Chains     []domain.Chain     `json:"chains"`
	RouteRules []domain.RouteRule `json:"routeRules"`
	Groups     []GroupScopeEntry  `json:"groups"`
}

func Compile(nodes []domain.Node, links []domain.NodeLink, chains []domain.Chain, rules []domain.RouteRule, groups []GroupScopeEntry) (string, error) {
	activeNodes := make([]domain.Node, 0, len(nodes))
	nodeSet := make(map[string]domain.Node, len(nodes))
	for _, node := range nodes {
		if !node.Enabled || node.Status == "pending" {
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
	compiledRules := make([]domain.RouteRule, 0, len(rules))
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if rule.ActionType != "chain" && rule.ActionType != "direct" {
			return "", fmt.Errorf("rule %s has invalid_action_type", rule.ID)
		}
		switch rule.ActionType {
		case "chain":
			if _, ok := chainSet[rule.ChainID]; !ok {
				return "", fmt.Errorf("rule %s references unknown_chain %s", rule.ID, rule.ChainID)
			}
		case "direct":
			if rule.DestinationScope == "" {
				return "", fmt.Errorf("rule %s missing_destination_scope", rule.ID)
			}
		}
		compiledRules = append(compiledRules, rule)
	}
	compiledLinks := make([]domain.NodeLink, 0, len(links))
	for _, link := range links {
		if _, ok := nodeSet[link.SourceNodeID]; !ok {
			continue
		}
		if _, ok := nodeSet[link.TargetNodeID]; !ok {
			continue
		}
		compiledLinks = append(compiledLinks, link)
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

func CompileForNode(nodeID string, nodes []domain.Node, links []domain.NodeLink, chains []domain.Chain, rules []domain.RouteRule, groups []GroupScopeEntry) (string, error) {
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
	filteredChains := make([]domain.Chain, 0)
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
	filteredRules := make([]domain.RouteRule, 0)
	for _, rule := range snapshot.RouteRules {
		if rule.ActionType == "chain" {
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
	for _, link := range snapshot.Links {
		if link.SourceNodeID == nodeID || link.TargetNodeID == nodeID {
			filteredLinks = append(filteredLinks, link)
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
