package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

type matchTypeMeta struct {
	Placeholder     string `json:"placeholder"`
	ValidationRegex string `json:"validationRegex"`
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
	items, _ := c.store.ListFieldEnumsByField("match_type")
	result := make([]domain.MatchType, 0, len(items))
	for _, item := range items {
		mt := domain.MatchType{
			Type:        item.Value,
			Label:       item.Name,
			Description: item.Name,
		}
		if item.Meta != nil && *item.Meta != "" {
			var meta matchTypeMeta
			if json.Unmarshal([]byte(*item.Meta), &meta) == nil {
				mt.Placeholder = meta.Placeholder
				if meta.ValidationRegex != "" {
					re := meta.ValidationRegex
					mt.ValidationRegex = &re
				}
			}
		}
		result = append(result, mt)
	}
	return result
}

func (c *ControlPlane) CreateRouteRule(input domain.CreateRouteRuleInput) (domain.RouteRule, error) {
	if err := c.validateRouteRule(input.ActionType, input.ChainID, input.DestinationScope, input.MatchType, input.MatchValue); err != nil {
		return domain.RouteRule{}, err
	}
	return c.store.CreateRouteRule(input)
}

func (c *ControlPlane) UpdateRouteRule(ruleID string, input domain.UpdateRouteRuleInput) (domain.RouteRule, error) {
	if ruleID == "" {
		return domain.RouteRule{}, invalidInput("missing_rule_id")
	}
	if err := c.validateRouteRule(input.ActionType, input.ChainID, input.DestinationScope, input.MatchType, input.MatchValue); err != nil {
		return domain.RouteRule{}, err
	}
	return c.store.UpdateRouteRule(ruleID, input)
}

func (c *ControlPlane) DeleteRouteRule(ruleID string) error {
	return c.store.DeleteRouteRule(ruleID)
}

func (c *ControlPlane) ValidateRouteRule(input domain.ValidateRouteRuleInput) (domain.RouteRuleValidationResult, error) {
	result := domain.RouteRuleValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	result.MatchValueValidation = c.validateMatchValue(input.MatchType, input.MatchValue)
	if !result.MatchValueValidation.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, result.MatchValueValidation.Message)
	}
	if !c.isValidEnum("action_type", input.ActionType) {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid_action_type")
	}

	var matchedChain *domain.Chain
	if input.ActionType == domain.ActionTypeChain {
		chains := c.store.ListChains()
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
			result.Valid = false
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
	}

	if input.ActionType == domain.ActionTypeChain && matchedChain != nil {
		input.DestinationScope = matchedChain.DestinationScope
	}
	if input.ActionType == domain.ActionTypeDirect && input.DestinationScope == "" {
		result.ScopeValidation = domain.ScopeValidation{Valid: false}
		result.Valid = false
		result.Errors = append(result.Errors, "scope_not_found")
	} else if input.DestinationScope == "" {
		result.ScopeValidation = domain.ScopeValidation{Valid: true}
	} else if !c.scopeExists(input.DestinationScope) {
		result.ScopeValidation = domain.ScopeValidation{
			Valid:       false,
			ScopeExists: false,
		}
		result.Valid = false
		result.Errors = append(result.Errors, "scope_not_found")
	} else {
		matchesFinalHop := false
		ownerNodeID := ""
		if matchedChain != nil && len(matchedChain.Hops) > 0 {
			finalHopID := matchedChain.Hops[len(matchedChain.Hops)-1]
			ownerNodeID = finalHopID
			for _, node := range c.store.ListNodes() {
				if node.ID == finalHopID {
					matchesFinalHop = node.ScopeKey == input.DestinationScope
					break
				}
			}
		}
		result.ScopeValidation = domain.ScopeValidation{
			Valid:                true,
			ScopeExists:          true,
			ScopeOwnerNodeID:     ownerNodeID,
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
