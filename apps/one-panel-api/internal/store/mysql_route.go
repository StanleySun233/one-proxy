package store

import (
	"database/sql"

	"github.com/StanleySun233/python-proxy/apps/one-panel-api/internal/domain"
)

func (s *MySQLStore) ListRouteRules() []domain.RouteRule {
	rows, err := s.db.Query(
		`SELECT id, priority, match_type, match_value, action_type, COALESCE(chain_id, ''), COALESCE(destination_scope, ''), enabled
		 FROM route_rules ORDER BY priority ASC`,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()
	items := make([]domain.RouteRule, 0)
	for rows.Next() {
		var item domain.RouteRule
		var enabled int
		if err := rows.Scan(&item.ID, &item.Priority, &item.MatchType, &item.MatchValue, &item.ActionType, &item.ChainID, &item.DestinationScope, &enabled); err != nil {
			continue
		}
		item.Enabled = enabled == 1
		items = append(items, item)
	}
	return items
}

func (s *MySQLStore) CreateRouteRule(input domain.CreateRouteRuleInput) (domain.RouteRule, error) {
	item := domain.RouteRule{
		ID:               newID("rule"),
		Priority:         input.Priority,
		MatchType:        input.MatchType,
		MatchValue:       input.MatchValue,
		ActionType:       input.ActionType,
		ChainID:          input.ChainID,
		DestinationScope: input.DestinationScope,
		Enabled:          true,
	}
	now := nowRFC3339()
	_, err := s.db.Exec(
		`INSERT INTO route_rules (id, priority, match_type, match_value, action_type, chain_id, destination_scope, enabled, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`,
		item.ID, item.Priority, item.MatchType, item.MatchValue, item.ActionType, item.ChainID, item.DestinationScope, 1, now, now,
	)
	return item, err
}

func (s *MySQLStore) UpdateRouteRule(ruleID string, input domain.UpdateRouteRuleInput) (domain.RouteRule, error) {
	now := nowRFC3339()
	_, err := s.db.Exec(
		`UPDATE route_rules
		 SET priority = ?, match_type = ?, match_value = ?, action_type = ?, chain_id = NULLIF(?, ''), destination_scope = NULLIF(?, ''), enabled = ?, updated_at = ?
		 WHERE id = ?`,
		input.Priority, input.MatchType, input.MatchValue, input.ActionType, input.ChainID, input.DestinationScope, boolToInt(input.Enabled), now, ruleID,
	)
	if err != nil {
		return domain.RouteRule{}, err
	}
	for _, item := range s.ListRouteRules() {
		if item.ID == ruleID {
			return item, nil
		}
	}
	return domain.RouteRule{}, sql.ErrNoRows
}

func (s *MySQLStore) DeleteRouteRule(ruleID string) error {
	_, err := s.db.Exec("DELETE FROM route_rules WHERE id = ?", ruleID)
	return err
}
