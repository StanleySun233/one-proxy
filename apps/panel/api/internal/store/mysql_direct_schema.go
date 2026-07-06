package store

import "context"

func (s *MySQLStore) initDirectSchema(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS direct_link_attempts (
  link_id VARCHAR(191) PRIMARY KEY,
  punch_token VARCHAR(191) NOT NULL,
  expires_at VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_direct_link_attempts_link_id FOREIGN KEY (link_id) REFERENCES node_links(id) ON DELETE CASCADE
)`); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT IGNORE INTO field_enum (id, field, value, name, meta) VALUES
('enum-transport_type-direct_relay', 'transport_type', 'direct_relay', 'Direct Relay', '{}')`)
	return err
}
