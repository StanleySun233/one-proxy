package store

import "context"

func (s *MySQLStore) ensureNodeAccessPathProtocolColumns(ctx context.Context) error {
	columns := []struct {
		name      string
		statement string
	}{
		{"protocol", "ALTER TABLE node_access_paths ADD COLUMN protocol VARCHAR(64) NOT NULL DEFAULT 'http' AFTER mode"},
		{"service_type", "ALTER TABLE node_access_paths ADD COLUMN service_type VARCHAR(64) NOT NULL DEFAULT 'http' AFTER protocol"},
		{"listen_host", "ALTER TABLE node_access_paths ADD COLUMN listen_host VARCHAR(255) AFTER relay_node_ids_json"},
		{"listen_port", "ALTER TABLE node_access_paths ADD COLUMN listen_port INT NOT NULL DEFAULT 0 AFTER listen_host"},
		{"target_protocol", "ALTER TABLE node_access_paths ADD COLUMN target_protocol VARCHAR(64) NOT NULL DEFAULT 'http' AFTER listen_port"},
		{"target_sni", "ALTER TABLE node_access_paths ADD COLUMN target_sni VARCHAR(255) AFTER target_port"},
		{"tls_mode", "ALTER TABLE node_access_paths ADD COLUMN tls_mode VARCHAR(64) NOT NULL DEFAULT 'none' AFTER target_sni"},
		{"auth_mode", "ALTER TABLE node_access_paths ADD COLUMN auth_mode VARCHAR(64) NOT NULL DEFAULT 'proxy_token' AFTER tls_mode"},
		{"options_json", "ALTER TABLE node_access_paths ADD COLUMN options_json LONGTEXT AFTER auth_mode"},
	}
	for _, column := range columns {
		exists, err := s.exists(ctx, "SELECT 1 FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'node_access_paths' AND column_name = ?", column.name)
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
	_, err := s.db.ExecContext(ctx, "UPDATE node_access_paths SET options_json = '{}' WHERE options_json IS NULL OR options_json = ''")
	return err
}
