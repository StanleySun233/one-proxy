package linkservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/link/domain"
)

type matchTypeMeta struct {
	Placeholder     string `json:"placeholder"`
	ValidationRegex string `json:"validationRegex"`
}

func (s *Service) RouteRules(tenantCtx domain.TenantAuthContext) []link.RouteRule {
	return s.store.ListRouteRulesForTenant(tenantCtx)
}

func (s *Service) RouteRulesWithDetails(tenantCtx domain.TenantAuthContext) []link.RouteRuleWithDetails {
	rules := s.store.ListRouteRulesForTenant(tenantCtx)
	chains := s.ChainsWithDetails(tenantCtx)
	chainMap := make(map[string]link.ChainWithDetails)
	for _, chain := range chains {
		chainMap[chain.ID] = chain
	}

	result := make([]link.RouteRuleWithDetails, 0, len(rules))
	for _, rule := range rules {
		item := link.RouteRuleWithDetails{
			ID:               rule.ID,
			CreateID:         rule.CreateID,
			OwnerID:          rule.OwnerID,
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

func (s *Service) GetRouteRule(tenantCtx domain.TenantAuthContext, ruleID string) (link.RouteRuleWithDetails, error) {
	if ruleID == "" {
		return link.RouteRuleWithDetails{}, invalidInput("missing_rule_id")
	}

	rules := s.RouteRulesWithDetails(tenantCtx)
	for _, rule := range rules {
		if rule.ID == ruleID {
			return rule, nil
		}
	}
	return link.RouteRuleWithDetails{}, invalidInput("route_rule_not_found")
}

func (s *Service) MatchTypes() []link.MatchType {
	items, _ := s.store.ListFieldEnumsByField("match_type")
	result := make([]link.MatchType, 0, len(items))
	for _, item := range items {
		mt := link.MatchType{
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

func (s *Service) CreateRouteRule(tenantCtx domain.TenantAuthContext, input link.CreateRouteRuleInput) (link.RouteRule, error) {
	if err := requireActiveTenant(tenantCtx); err != nil {
		return link.RouteRule{}, err
	}
	if !tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.Role != domain.TenantRoleAdmin {
		return link.RouteRule{}, newError(http.StatusForbidden, "tenant_role_forbidden")
	}
	if err := s.validateRouteRule(tenantCtx, input.ActionType, input.ChainID, input.DestinationScope, input.MatchType, input.MatchValue); err != nil {
		return link.RouteRule{}, err
	}
	return s.store.CreateRouteRuleForTenant(tenantCtx, input)
}

func (s *Service) UpdateRouteRule(tenantCtx domain.TenantAuthContext, ruleID string, input link.UpdateRouteRuleInput) (link.RouteRule, error) {
	if ruleID == "" {
		return link.RouteRule{}, invalidInput("missing_rule_id")
	}
	if err := s.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return s.store.RouteRuleBindingPermission(tenantCtx, ruleID)
	}); err != nil {
		return link.RouteRule{}, err
	}
	if err := s.validateRouteRule(tenantCtx, input.ActionType, input.ChainID, input.DestinationScope, input.MatchType, input.MatchValue); err != nil {
		return link.RouteRule{}, err
	}
	return s.store.UpdateRouteRule(ruleID, input)
}

func (s *Service) DeleteRouteRule(tenantCtx domain.TenantAuthContext, ruleID string) error {
	if err := s.requireTenantResourceManage(tenantCtx, func() (domain.BindingPermission, bool) {
		return s.store.RouteRuleBindingPermission(tenantCtx, ruleID)
	}); err != nil {
		return err
	}
	if !(tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "") && s.store.CountRouteRuleBindings(ruleID) > 1 {
		return newError(http.StatusConflict, "shared_resource_delete_forbidden")
	}
	return s.store.DeleteRouteRule(ruleID)
}

func (s *Service) ValidateRouteRule(tenantCtx domain.TenantAuthContext, input link.ValidateRouteRuleInput) (link.RouteRuleValidationResult, error) {
	result := link.RouteRuleValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	result.MatchValueValidation = s.validateMatchValue(input.MatchType, input.MatchValue)
	if !result.MatchValueValidation.Valid {
		result.Valid = false
		result.Errors = append(result.Errors, result.MatchValueValidation.Message)
	}
	if !s.isValidEnum("action_type", input.ActionType) {
		result.Valid = false
		result.Errors = append(result.Errors, "invalid_action_type")
	}

	var matchedChain *link.Chain
	if input.ActionType == domain.ActionTypeChain {
		chains := s.store.ListChainsForTenant(tenantCtx)
		for _, chain := range chains {
			if chain.ID == input.ChainID {
				c := chain
				matchedChain = &c
				break
			}
		}
		if matchedChain == nil {
			result.ChainValidation = link.ChainValidation{
				Valid:        false,
				ChainEnabled: false,
			}
			result.Valid = false
			result.Errors = append(result.Errors, "chain_not_found")
		} else {
			result.ChainValidation = link.ChainValidation{
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
		result.ScopeValidation = link.ScopeValidation{Valid: false}
		result.Valid = false
		result.Errors = append(result.Errors, "scope_not_found")
	} else if input.DestinationScope == "" {
		result.ScopeValidation = link.ScopeValidation{Valid: true}
	} else if !s.tenantScopeExists(tenantCtx, input.DestinationScope) {
		result.ScopeValidation = link.ScopeValidation{
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
			for _, node := range s.store.ListNodesForTenant(tenantCtx) {
				if node.ID == finalHopID {
					matchesFinalHop = node.ScopeKey == input.DestinationScope
					break
				}
			}
		}
		result.ScopeValidation = link.ScopeValidation{
			Valid:                true,
			ScopeExists:          true,
			ScopeOwnerNodeID:     ownerNodeID,
			MatchesChainFinalHop: matchesFinalHop,
		}
		if !matchesFinalHop && matchedChain != nil && len(matchedChain.Hops) > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Scope %s is not owned by chain's final hop node", input.DestinationScope))
		}
	}

	rules := s.store.ListRouteRulesForTenant(tenantCtx)
	for _, rule := range rules {
		if rule.Priority == input.Priority {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Priority %d conflicts with existing rule", input.Priority))
			break
		}
	}

	return result, nil
}

func (s *Service) RouteRuleSuggestions(tenantCtx domain.TenantAuthContext, matchType string, query string) link.RouteRuleSuggestionResult {
	rules := s.store.ListRouteRulesForTenant(tenantCtx)
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

	return link.RouteRuleSuggestionResult{
		MatchType:   matchType,
		Suggestions: suggestions,
	}
}
