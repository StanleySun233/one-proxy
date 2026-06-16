package store

import (
	"fmt"
	"time"
)

func (s *MySQLStore) nextID(name string) (string, error) {
	tables := map[string]string{
		"account":          "accounts",
		"session":          "sessions",
		"node":             "nodes",
		"node_link":        "node_links",
		"chain":            "chains",
		"route_rule":       "route_rules",
		"route_rule_group": "route_rule_groups",
		"policy_revision":  "policy_revisions",
		"bootstrap_token":  "bootstrap_tokens",
		"certificate":      "certificates",
		"node_api_token":   "node_api_tokens",
		"trust_material":   "node_trust_materials",
		"node_access_path": "node_access_paths",
		"onboarding_task":  "node_onboarding_tasks",
		"node_transport":   "node_transports",
		"group":            "`groups`",
		"scope":            "scopes",
		"audit_event":      "business_audit_events",
		"audit_session":    "network_audit_sessions",
	}
	table, ok := tables[name]
	if !ok {
		return "", fmt.Errorf("unknown id sequence %q", name)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	now := nowRFC3339()
	var maxExisting int64
	if err := tx.QueryRow(fmt.Sprintf("SELECT COALESCE(MAX(CAST(id AS UNSIGNED)), 0) FROM %s WHERE id REGEXP '^[0-9]+$'", table)).Scan(&maxExisting); err != nil {
		return "", err
	}
	_, err = tx.Exec(
		`INSERT INTO id_sequences (name, current_value, updated_at)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE current_value = GREATEST(current_value + 1, VALUES(current_value)), updated_at = ?`,
		name, maxExisting+1, now, now,
	)
	if err != nil {
		return "", err
	}

	var nextID int64
	err = tx.QueryRow(`SELECT current_value FROM id_sequences WHERE name = ?`, name).Scan(&nextID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%d", nextID), nil
}

func (s *MySQLStore) nextNodeID() (string, error) {
	return s.nextID("node")
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
