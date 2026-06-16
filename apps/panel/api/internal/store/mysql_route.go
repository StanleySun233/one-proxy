package store

import (
	"context"
	"database/sql"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/panel/api/internal/features/proxy/domain"
	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/store/deleteplan"
)

func (s *MySQLStore) ListRouteRules() []proxy.RouteRule {
	items, err := s.proxyRepository().listRouteRules(context.Background())
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) ListRouteRuleGroups() []proxy.RouteRuleGroup {
	items, err := s.proxyRepository().listRouteRuleGroups(context.Background())
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) ListRouteRuleGroupsForTenant(tenantCtx domain.TenantAuthContext) []proxy.RouteRuleGroup {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return s.ListRouteRuleGroups()
	}
	items, err := s.proxyRepository().listRouteRuleGroupsForTenant(context.Background(), tenantCtx)
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) ListRouteRulesForTenant(tenantCtx domain.TenantAuthContext) []proxy.RouteRule {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return s.ListRouteRules()
	}
	items, err := s.proxyRepository().listRouteRulesForTenant(context.Background(), tenantCtx, false)
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) ListPolicyRouteRulesForTenant(tenantCtx domain.TenantAuthContext) []proxy.RouteRule {
	items, err := s.proxyRepository().listRouteRulesForTenant(context.Background(), tenantCtx, true)
	if err != nil {
		return nil
	}
	return items
}

func (s *MySQLStore) CreateRouteRuleGroupForTenant(tenantCtx domain.TenantAuthContext, input proxy.CreateRouteRuleGroupInput) (proxy.RouteRuleGroup, error) {
	groupID, err := s.nextID("route_rule_group")
	if err != nil {
		return proxy.RouteRuleGroup{}, err
	}
	item := proxy.RouteRuleGroup{
		ID:          groupID,
		Name:        input.Name,
		Description: input.Description,
		Enabled:     true,
		CreateID:    tenantCtx.Account.ID,
		OwnerID:     tenantCtx.Account.ID,
	}
	return item, s.proxyRepository().createRouteRuleGroup(context.Background(), item, tenantCtx.ActiveTenant.TenantID)
}

func (s *MySQLStore) UpdateRouteRuleGroup(groupID string, input proxy.UpdateRouteRuleGroupInput) (proxy.RouteRuleGroup, error) {
	return s.proxyRepository().updateRouteRuleGroup(context.Background(), groupID, input)
}

func (s *MySQLStore) GetRouteRuleGroupDeleteImpact(groupID string) (proxy.RouteRuleGroupDeleteImpact, error) {
	plan, err := s.proxyRepository().buildRouteRuleGroupDeletePlan(context.Background(), groupID, true)
	if err != nil {
		return proxy.RouteRuleGroupDeleteImpact{GroupID: groupID}, err
	}
	impact := routeRuleGroupDeleteImpactFromPlan(plan)
	if len(impact.Delete.Group) == 0 {
		return impact, sql.ErrNoRows
	}
	return impact, nil
}

func (s *MySQLStore) DeleteRouteRuleGroup(groupID string) error {
	plan, err := s.proxyRepository().buildRouteRuleGroupDeletePlan(context.Background(), groupID, false)
	if err != nil {
		return err
	}
	_, err = deleteplan.NewMySQLExecutor(s.db).Execute(context.Background(), plan)
	return err
}

func (s *MySQLStore) RouteRuleGroupBindingPermission(tenantCtx domain.TenantAuthContext, groupID string) (domain.BindingPermission, bool) {
	return s.tenantResourcePermission(tenantCtx, "tenant_route_rule_groups", "route_rule_group_id", groupID)
}

func (s *MySQLStore) CountRouteRuleGroupBindings(groupID string) int {
	return s.countTenantResourceBindings("tenant_route_rule_groups", "route_rule_group_id", groupID)
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
		GroupID:          input.GroupID,
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
	return item, s.proxyRepository().createRouteRule(context.Background(), item)
}

func (s *MySQLStore) CreateRouteRuleForTenant(tenantCtx domain.TenantAuthContext, input proxy.CreateRouteRuleInput) (proxy.RouteRule, error) {
	ruleID, err := s.nextID("route_rule")
	if err != nil {
		return proxy.RouteRule{}, err
	}
	item := proxy.RouteRule{
		ID:               ruleID,
		GroupID:          input.GroupID,
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
	return item, s.proxyRepository().createRouteRule(context.Background(), item)
}

func (s *MySQLStore) UpdateRouteRule(ruleID string, input proxy.UpdateRouteRuleInput) (proxy.RouteRule, error) {
	return s.proxyRepository().updateRouteRule(context.Background(), ruleID, input)
}

func (s *MySQLStore) DeleteRouteRule(ruleID string) error {
	return s.proxyRepository().deleteRouteRule(context.Background(), ruleID)
}

func (s *MySQLStore) RouteRuleBindingPermission(tenantCtx domain.TenantAuthContext, ruleID string) (domain.BindingPermission, bool) {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return domain.BindingPermissionManage, true
	}
	var permission domain.BindingPermission
	err := s.db.QueryRow(
		`SELECT trg.permission
		 FROM route_rules rr
		 JOIN tenant_route_rule_groups trg ON trg.route_rule_group_id = rr.group_id
		 WHERE rr.id = ? AND trg.tenant_id = ?`,
		ruleID, tenantCtx.ActiveTenant.TenantID,
	).Scan(&permission)
	if err != nil || (permission != domain.BindingPermissionUse && permission != domain.BindingPermissionManage) {
		return "", false
	}
	return permission, true
}
