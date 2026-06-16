package store

import (
	"context"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
)

func (s *MySQLStore) ListRouteRules() []proxy.RouteRule {
	items, err := s.proxyRepository().listRouteRules(context.Background())
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) ListRouteRulesForTenant(tenantCtx domain.TenantAuthContext) []proxy.RouteRule {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return s.ListRouteRules()
	}
	items, err := s.proxyRepository().listRouteRulesForTenant(context.Background(), tenantCtx)
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) CreateRouteRule(input proxy.CreateRouteRuleInput) (proxy.RouteRule, error) {
	ownerID, err := s.defaultOwnerAccountID()
	if err != nil {
		return proxy.RouteRule{}, err
	}
	ruleID, err := s.nextID("route_rule")
	if err != nil {
		return proxy.RouteRule{}, err
	}
	item := proxy.RouteRule{
		ID:               ruleID,
		CreateID:         ownerID,
		OwnerID:          ownerID,
		Priority:         input.Priority,
		MatchType:        input.MatchType,
		MatchValue:       input.MatchValue,
		ActionType:       input.ActionType,
		ChainID:          input.ChainID,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
	}
	return item, s.proxyRepository().createRouteRule(context.Background(), item, "")
}

func (s *MySQLStore) CreateRouteRuleForTenant(tenantCtx domain.TenantAuthContext, input proxy.CreateRouteRuleInput) (proxy.RouteRule, error) {
	ruleID, err := s.nextID("route_rule")
	if err != nil {
		return proxy.RouteRule{}, err
	}
	item := proxy.RouteRule{
		ID:               ruleID,
		CreateID:         tenantCtx.Account.ID,
		OwnerID:          tenantCtx.Account.ID,
		Priority:         input.Priority,
		MatchType:        input.MatchType,
		MatchValue:       input.MatchValue,
		ActionType:       input.ActionType,
		ChainID:          input.ChainID,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
	}
	return item, s.proxyRepository().createRouteRule(context.Background(), item, tenantCtx.ActiveTenant.TenantID)
}

func (s *MySQLStore) UpdateRouteRule(ruleID string, input proxy.UpdateRouteRuleInput) (proxy.RouteRule, error) {
	return s.proxyRepository().updateRouteRule(context.Background(), ruleID, input)
}

func (s *MySQLStore) DeleteRouteRule(ruleID string) error {
	return s.proxyRepository().deleteRouteRule(context.Background(), ruleID)
}

func (s *MySQLStore) RouteRuleBindingPermission(tenantCtx domain.TenantAuthContext, ruleID string) (domain.BindingPermission, bool) {
	return s.tenantResourcePermission(tenantCtx, "tenant_route_rules", "route_rule_id", ruleID)
}

func (s *MySQLStore) CountRouteRuleBindings(ruleID string) int {
	return s.countTenantResourceBindings("tenant_route_rules", "route_rule_id", ruleID)
}
