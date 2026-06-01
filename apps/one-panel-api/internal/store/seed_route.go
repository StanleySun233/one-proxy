package store

import "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"

func (s *SeedStore) ListRouteRules() []domain.RouteRule {
	return []domain.RouteRule{}
}

func (s *SeedStore) CreateRouteRule(input domain.CreateRouteRuleInput) (domain.RouteRule, error) {
	return domain.RouteRule{
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

func (s *SeedStore) UpdateRouteRule(ruleID string, input domain.UpdateRouteRuleInput) (domain.RouteRule, error) {
	return domain.RouteRule{
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
