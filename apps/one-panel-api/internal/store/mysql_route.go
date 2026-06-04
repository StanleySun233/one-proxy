package store

import (
	"database/sql"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
	proxy "github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/features/proxy/domain"
)

func (s *MySQLStore) ListRouteRules() []proxy.RouteRule {
	rows, err := s.db.Query(
		`SELECT id, create_id, owner_id, priority, match_type, match_value, action_type, COALESCE(chain_id, ''), COALESCE(destination_scope, ''), enabled
		 FROM route_rules ORDER BY priority ASC`,
	)
	return s.scanRouteRules(rows, err)
}

func (s *MySQLStore) ListRouteRulesForTenant(tenantCtx domain.TenantAuthContext) []proxy.RouteRule {
	if tenantCtx.SuperAdmin && tenantCtx.ActiveTenant.TenantID == "" {
		return s.ListRouteRules()
	}
	rows, err := s.db.Query(
		`SELECT rr.id, rr.create_id, rr.owner_id, rr.priority, rr.match_type, rr.match_value, rr.action_type, COALESCE(rr.chain_id, ''), COALESCE(rr.destination_scope, ''), rr.enabled, trr.permission
		 FROM route_rules rr
		 JOIN tenant_route_rules trr ON trr.route_rule_id = rr.id
		 WHERE trr.tenant_id = ? AND trr.permission IN (?, ?)
		 ORDER BY rr.priority ASC`,
		tenantCtx.ActiveTenant.TenantID, domain.BindingPermissionUse, domain.BindingPermissionManage,
	)
	return s.scanRouteRules(rows, err)
}

func (s *MySQLStore) scanRouteRules(rows *sql.Rows, err error) []proxy.RouteRule {
	if err != nil {
		return nil
	}
	defer rows.Close()
	columns, _ := rows.Columns()
	hasPermission := len(columns) == 11
	items := make([]proxy.RouteRule, 0)
	for rows.Next() {
		var item proxy.RouteRule
		var enabled int
		if hasPermission {
			if err := rows.Scan(&item.ID, &item.CreateID, &item.OwnerID, &item.Priority, &item.MatchType, &item.MatchValue, &item.ActionType, &item.ChainID, &item.DestinationScope, &enabled, &item.Permission); err != nil {
				continue
			}
		} else {
			if err := rows.Scan(&item.ID, &item.CreateID, &item.OwnerID, &item.Priority, &item.MatchType, &item.MatchValue, &item.ActionType, &item.ChainID, &item.DestinationScope, &enabled); err != nil {
				continue
			}
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateRouteRule(input proxy.CreateRouteRuleInput) (proxy.RouteRule, error) {
	ruleID, err := s.nextID("route_rule")
	if err != nil {
		return proxy.RouteRule{}, err
	}
	item := proxy.RouteRule{
		ID:               ruleID,
		Priority:         input.Priority,
		MatchType:        input.MatchType,
		MatchValue:       input.MatchValue,
		ActionType:       input.ActionType,
		ChainID:          input.ChainID,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
	}
	now := nowRFC3339()
	_, err = s.db.Exec(
		`INSERT INTO route_rules (id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`,
		item.ID, item.Priority, item.MatchType, item.MatchValue, item.ActionType, item.ChainID, item.DestinationScope, 1, now, now,
	)
	return item, err
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
	now := nowRFC3339()
	tx, err := s.db.Begin()
	if err != nil {
		return proxy.RouteRule{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`INSERT INTO route_rules (id, create_id, owner_id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`,
		item.ID, item.CreateID, item.OwnerID, item.Priority, item.MatchType, item.MatchValue, item.ActionType, item.ChainID, item.DestinationScope, 1, now, now,
	); err != nil {
		return proxy.RouteRule{}, err
	}
	if tenantCtx.ActiveTenant.TenantID != "" {
		if err := bindTenantResource(tx, "tenant_route_rules", "route_rule_id", tenantCtx.ActiveTenant.TenantID, item.ID, tenantCtx.Account.ID); err != nil {
			return proxy.RouteRule{}, err
		}
	}
	return item, tx.Commit()
}

func (s *MySQLStore) UpdateRouteRule(ruleID string, input proxy.UpdateRouteRuleInput) (proxy.RouteRule, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE route_rules
		 SET priority = ?, match_type = ?, match_value = ?, action_type = ?, chain_id = NULLIF(?, ''), destination_scope = NULLIF(?, ''), enabled = ?, updated_at = ?
		 WHERE id = ?`,
		input.Priority, input.MatchType, input.MatchValue, input.ActionType, input.ChainID, input.DestinationScope, boolToInt(input.Enabled), now, ruleID,
	)
	if err != nil {
		return proxy.RouteRule{}, err
	}
	for _, item := range s.ListRouteRules() {
		if item.ID == ruleID {
			return item, nil
		}
	}
	return proxy.RouteRule{}, sql.ErrNoRows
}

func (s *MySQLStore) DeleteRouteRule(ruleID string) error {
	_, err := s.db.Exec("DELETE FROM route_rules WHERE id = ?", ruleID)
	return err
}

func (s *MySQLStore) RouteRuleBindingPermission(tenantCtx domain.TenantAuthContext, ruleID string) (domain.BindingPermission, bool) {
	return s.tenantResourcePermission(tenantCtx, "tenant_route_rules", "route_rule_id", ruleID)
}

func (s *MySQLStore) CountRouteRuleBindings(ruleID string) int {
	return s.countTenantResourceBindings("tenant_route_rules", "route_rule_id", ruleID)
}
