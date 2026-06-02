package store

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/link/domain"

func (s *SeedStore) ListRouteRules() []link.RouteRule {
	return []link.RouteRule{}
}

func (s *SeedStore) CreateRouteRule(input link.CreateRouteRuleInput) (link.RouteRule, error) {
	return link.RouteRule{
		ID:               s.nextID("route_rule"),
		Priority:         input.Priority,
		MatchType:        input.MatchType,
		MatchValue:       input.MatchValue,
		ActionType:       input.ActionType,
		ChainID:          input.ChainID,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
	}, nil
}

func (s *SeedStore) UpdateRouteRule(ruleID string, input link.UpdateRouteRuleInput) (link.RouteRule, error) {
	return link.RouteRule{
		ID:               ruleID,
		Priority:         input.Priority,
		MatchType:        input.MatchType,
		MatchValue:       input.MatchValue,
		ActionType:       input.ActionType,
		ChainID:          input.ChainID,
		DestinationScope: input.DestinationScope,
		Enabled:          input.Enabled,
	}, nil
}

func (s *SeedStore) DeleteRouteRule(ruleID string) error {
	_ = ruleID
	return nil
}
