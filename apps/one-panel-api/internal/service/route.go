package service

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
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

func (c *ControlPlane) validateMatchValue(matchType, matchValue string) domain.MatchValueValidation {
	if !c.isValidEnum("match_type", matchType) {
		return domain.MatchValueValidation{Valid: false, Format: matchType, Message: "Unknown match type"}
	}
	switch matchType {
	case domain.MatchTypeDomain:
		pattern := `^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`
		matched, _ := regexp.MatchString(pattern, matchValue)
		if matched {
			return domain.MatchValueValidation{Valid: true, Format: "domain", Message: "Valid domain format"}
		}
		return domain.MatchValueValidation{Valid: false, Format: "domain", Message: "Invalid domain format"}
	case domain.MatchTypeDomainSuffix:
		if !strings.HasPrefix(matchValue, ".") && !strings.HasPrefix(matchValue, "*.") {
			return domain.MatchValueValidation{Valid: false, Format: "domain_suffix", Message: "Domain suffix must start with . or *."}
		}
		return domain.MatchValueValidation{Valid: true, Format: "domain_suffix", Message: "Valid domain suffix"}
	case domain.MatchTypeIPCIDR:
		_, _, err := net.ParseCIDR(matchValue)
		if err != nil {
			return domain.MatchValueValidation{Valid: false, Format: "ip_cidr", Message: "Invalid CIDR notation"}
		}
		return domain.MatchValueValidation{Valid: true, Format: "ip_cidr", Message: "Valid CIDR notation"}
	case domain.MatchTypeIPRange:
		parts := strings.SplitN(matchValue, "-", 2)
		if len(parts) != 2 || net.ParseIP(strings.TrimSpace(parts[0])) == nil || net.ParseIP(strings.TrimSpace(parts[1])) == nil {
			return domain.MatchValueValidation{Valid: false, Format: "ip_range", Message: "Invalid IP range format"}
		}
		return domain.MatchValueValidation{Valid: true, Format: "ip_range", Message: "Valid IP range"}
	case domain.MatchTypePort:
		p, err := strconv.Atoi(matchValue)
		if err != nil || p < 1 || p > 65535 {
			return domain.MatchValueValidation{Valid: false, Format: "port", Message: "Port must be between 1 and 65535"}
		}
		return domain.MatchValueValidation{Valid: true, Format: "port", Message: "Valid port"}
	case domain.MatchTypeURLRegex:
		_, err := regexp.Compile(matchValue)
		if err != nil {
			return domain.MatchValueValidation{Valid: false, Format: "url_regex", Message: fmt.Sprintf("Invalid regex: %s", err.Error())}
		}
		return domain.MatchValueValidation{Valid: true, Format: "url_regex", Message: "Valid regex pattern"}
	case domain.MatchTypeDefault:
		return domain.MatchValueValidation{Valid: true, Format: "default", Message: "Default match type"}
	}
	return domain.MatchValueValidation{Valid: false, Format: matchType, Message: "Unknown match type"}
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

func (c *ControlPlane) validateRouteRule(actionType string, chainID string, destinationScope string, matchType string, matchValue string) error {
	if matchType == "" || matchValue == "" || actionType == "" {
		return invalidInput("invalid_route_rule_payload")
	}
	if !c.isValidEnum("action_type", actionType) {
		return invalidInput("invalid_route_rule_payload")
	}
	switch actionType {
	case domain.ActionTypeChain:
		if chainID == "" {
			return invalidInput("invalid_route_rule_payload")
		}
	case domain.ActionTypeDirect:
		if destinationScope == "" {
			return invalidInput("invalid_route_rule_payload")
		}
	}
	return nil
}
