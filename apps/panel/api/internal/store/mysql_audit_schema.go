package store

import "context"

func (s *MySQLStore) ensureNetworkAuditGovernanceColumns(ctx context.Context) error {
	columns := []struct {
		name      string
		statement string
	}{
		{"governance_mode", "ALTER TABLE network_audit_sessions ADD COLUMN governance_mode VARCHAR(64) NOT NULL DEFAULT '' AFTER chain_id"},
		{"policy_revision", "ALTER TABLE network_audit_sessions ADD COLUMN policy_revision VARCHAR(191) NOT NULL DEFAULT '' AFTER governance_mode"},
		{"matched_rule_id", "ALTER TABLE network_audit_sessions ADD COLUMN matched_rule_id VARCHAR(191) NOT NULL DEFAULT '' AFTER policy_revision"},
		{"matched_rule_type", "ALTER TABLE network_audit_sessions ADD COLUMN matched_rule_type VARCHAR(64) NOT NULL DEFAULT '' AFTER matched_rule_id"},
		{"matched_rule_pattern", "ALTER TABLE network_audit_sessions ADD COLUMN matched_rule_pattern VARCHAR(255) NOT NULL DEFAULT '' AFTER matched_rule_type"},
		{"matched_action", "ALTER TABLE network_audit_sessions ADD COLUMN matched_action VARCHAR(64) NOT NULL DEFAULT '' AFTER matched_rule_pattern"},
		{"decision_source", "ALTER TABLE network_audit_sessions ADD COLUMN decision_source VARCHAR(64) NOT NULL DEFAULT '' AFTER matched_action"},
	}
	for _, column := range columns {
		exists, err := s.exists(ctx, "SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'network_audit_sessions' AND column_name = ?", column.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := s.db.ExecContext(ctx, column.statement); err != nil {
			return err
		}
	}
	indexes := []struct {
		name      string
		statement string
	}{
		{"idx_network_audit_policy_time", "CREATE INDEX idx_network_audit_policy_time ON network_audit_sessions (policy_revision, ended_at)"},
		{"idx_network_audit_matched_rule_time", "CREATE INDEX idx_network_audit_matched_rule_time ON network_audit_sessions (matched_rule_id, ended_at)"},
		{"idx_network_audit_deny_reason_time", "CREATE INDEX idx_network_audit_deny_reason_time ON network_audit_sessions (deny_reason, ended_at)"},
	}
	for _, index := range indexes {
		exists, err := s.exists(ctx, "SELECT 1 FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'network_audit_sessions' AND index_name = ?", index.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := s.db.ExecContext(ctx, index.statement); err != nil {
			return err
		}
	}
	return nil
}
