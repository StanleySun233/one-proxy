CREATE TABLE IF NOT EXISTS roles (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL UNIQUE,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);

CREATE TABLE IF NOT EXISTS accounts (
  id VARCHAR(191) PRIMARY KEY,
  account VARCHAR(191) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role_id VARCHAR(191) NOT NULL,
  status VARCHAR(64) NOT NULL,
  must_rotate_password TINYINT(1) NOT NULL DEFAULT 1,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_accounts_role_id FOREIGN KEY (role_id) REFERENCES roles(id)
);

CREATE TABLE IF NOT EXISTS sessions (
  id VARCHAR(191) PRIMARY KEY,
  account_id VARCHAR(191) NOT NULL,
  access_token_hash VARCHAR(255) NOT NULL UNIQUE,
  refresh_token_hash VARCHAR(255) NOT NULL,
  expires_at VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_sessions_account_id FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS nodes (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL,
  mode VARCHAR(64) NOT NULL,
  public_host VARCHAR(255),
  public_port INT,
  scope_key VARCHAR(191) NOT NULL,
  parent_node_id VARCHAR(191),
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  status VARCHAR(64) NOT NULL,
  reviewed_by VARCHAR(191),
  reviewed_at VARCHAR(64),
  reject_reason TEXT,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_nodes_parent_node_id FOREIGN KEY (parent_node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS node_links (
  id VARCHAR(191) PRIMARY KEY,
  source_node_id VARCHAR(191) NOT NULL,
  target_node_id VARCHAR(191) NOT NULL,
  link_type VARCHAR(64) NOT NULL,
  trust_state VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_links_source_node_id FOREIGN KEY (source_node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_links_target_node_id FOREIGN KEY (target_node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS chains (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL UNIQUE,
  destination_scope VARCHAR(191) NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);

CREATE TABLE IF NOT EXISTS chain_hops (
  chain_id VARCHAR(191) NOT NULL,
  hop_index INT NOT NULL,
  node_id VARCHAR(191) NOT NULL,
  PRIMARY KEY (chain_id, hop_index),
  CONSTRAINT fk_chain_hops_chain_id FOREIGN KEY (chain_id) REFERENCES chains(id),
  CONSTRAINT fk_chain_hops_node_id FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS route_rules (
  id VARCHAR(191) PRIMARY KEY,
  priority INT NOT NULL,
  match_type VARCHAR(64) NOT NULL,
  match_value VARCHAR(255) NOT NULL,
  action_type VARCHAR(64) NOT NULL,
  chain_id VARCHAR(191),
  destination_scope VARCHAR(191),
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_route_rules_chain_id FOREIGN KEY (chain_id) REFERENCES chains(id)
);

CREATE TABLE IF NOT EXISTS policy_revisions (
  id VARCHAR(191) PRIMARY KEY,
  version VARCHAR(191) NOT NULL UNIQUE,
  payload_json LONGTEXT NOT NULL,
  status VARCHAR(64) NOT NULL,
  created_by_account_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_policy_revisions_created_by_account_id FOREIGN KEY (created_by_account_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS node_policy_assignments (
  node_id VARCHAR(191) PRIMARY KEY,
  policy_revision_id VARCHAR(191) NOT NULL,
  snapshot_json LONGTEXT NOT NULL,
  assigned_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_policy_assignments_node_id FOREIGN KEY (node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_policy_assignments_policy_revision_id FOREIGN KEY (policy_revision_id) REFERENCES policy_revisions(id)
);

CREATE TABLE IF NOT EXISTS bootstrap_tokens (
  id VARCHAR(191) PRIMARY KEY,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  target_type VARCHAR(64) NOT NULL,
  target_id VARCHAR(191),
  node_name VARCHAR(255),
  expires_at VARCHAR(64) NOT NULL,
  consumed_at VARCHAR(64),
  created_at VARCHAR(64) NOT NULL
);

CREATE TABLE IF NOT EXISTS certificates (
  id VARCHAR(191) PRIMARY KEY,
  owner_type VARCHAR(64) NOT NULL,
  owner_id VARCHAR(191) NOT NULL,
  cert_type VARCHAR(64) NOT NULL,
  provider VARCHAR(64) NOT NULL DEFAULT 'manual',
  status VARCHAR(64) NOT NULL,
  not_before VARCHAR(64),
  not_after VARCHAR(64),
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);

CREATE TABLE IF NOT EXISTS node_health_snapshots (
  node_id VARCHAR(191) PRIMARY KEY,
  heartbeat_at VARCHAR(64) NOT NULL,
  policy_revision_id VARCHAR(191),
  listener_status_json LONGTEXT NOT NULL,
  cert_status_json LONGTEXT NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_health_snapshots_node_id FOREIGN KEY (node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_health_snapshots_policy_revision_id FOREIGN KEY (policy_revision_id) REFERENCES policy_revisions(id)
);

CREATE TABLE IF NOT EXISTS node_api_tokens (
  id VARCHAR(191) PRIMARY KEY,
  node_id VARCHAR(191) NOT NULL,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  expires_at VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_api_tokens_node_id FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS node_trust_materials (
  id VARCHAR(191) PRIMARY KEY,
  node_id VARCHAR(191) NOT NULL,
  material_type VARCHAR(64) NOT NULL,
  material_value LONGTEXT NOT NULL,
  status VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_trust_materials_node_id FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS node_access_paths (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL,
  mode VARCHAR(64) NOT NULL,
  target_node_id VARCHAR(191),
  entry_node_id VARCHAR(191),
  relay_node_ids_json LONGTEXT NOT NULL,
  target_host VARCHAR(255),
  target_port INT NOT NULL DEFAULT 0,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_access_paths_target_node_id FOREIGN KEY (target_node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_access_paths_entry_node_id FOREIGN KEY (entry_node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS node_onboarding_tasks (
  id VARCHAR(191) PRIMARY KEY,
  mode VARCHAR(64) NOT NULL,
  path_id VARCHAR(191),
  target_node_id VARCHAR(191),
  target_host VARCHAR(255),
  target_port INT NOT NULL DEFAULT 0,
  status VARCHAR(64) NOT NULL,
  status_message VARCHAR(255) NOT NULL,
  requested_by_account_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_onboarding_tasks_path_id FOREIGN KEY (path_id) REFERENCES node_access_paths(id),
  CONSTRAINT fk_node_onboarding_tasks_target_node_id FOREIGN KEY (target_node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_onboarding_tasks_requested_by_account_id FOREIGN KEY (requested_by_account_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS node_transports (
  id VARCHAR(191) PRIMARY KEY,
  node_id VARCHAR(191) NOT NULL,
  transport_type VARCHAR(64) NOT NULL,
  direction VARCHAR(32) NOT NULL,
  address VARCHAR(255) NOT NULL,
  status VARCHAR(64) NOT NULL,
  parent_node_id VARCHAR(191),
  connected_at VARCHAR(64),
  last_heartbeat_at VARCHAR(64),
  latency_ms INT NOT NULL DEFAULT 0,
  details_json LONGTEXT NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  UNIQUE KEY uniq_node_transports_node_type_address (node_id, transport_type, address),
  CONSTRAINT fk_node_transports_node_id FOREIGN KEY (node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_transports_parent_node_id FOREIGN KEY (parent_node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS chain_probe_results (
  chain_id VARCHAR(191) PRIMARY KEY,
  status VARCHAR(64) NOT NULL,
  message VARCHAR(255) NOT NULL,
  resolved_hops_json LONGTEXT NOT NULL,
  blocking_node_id VARCHAR(191),
  blocking_reason VARCHAR(255),
  target_host VARCHAR(255),
  target_port INT NOT NULL DEFAULT 0,
  probed_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_chain_probe_results_chain_id FOREIGN KEY (chain_id) REFERENCES chains(id),
  CONSTRAINT fk_chain_probe_results_blocking_node_id FOREIGN KEY (blocking_node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS id_sequences (
  name VARCHAR(64) PRIMARY KEY,
  current_value BIGINT NOT NULL DEFAULT 0,
  updated_at VARCHAR(64) NOT NULL
);

INSERT INTO id_sequences (name, current_value, updated_at)
VALUES ('node_id', 0, UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE name = name;

CREATE TABLE IF NOT EXISTS node_health_history (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  node_id VARCHAR(191) NOT NULL,
  heartbeat_at VARCHAR(64) NOT NULL,
  policy_revision_id VARCHAR(191) DEFAULT '',
  listener_status_json LONGTEXT NOT NULL,
  cert_status_json LONGTEXT NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  INDEX idx_history_node_time (node_id, heartbeat_at),
  INDEX idx_history_time (heartbeat_at)
);

CREATE TABLE IF NOT EXISTS `groups` (
  id VARCHAR(191) NOT NULL PRIMARY KEY,
  name VARCHAR(191) NOT NULL,
  description TEXT,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);

CREATE TABLE IF NOT EXISTS account_groups (
  account_id VARCHAR(191) NOT NULL,
  group_id VARCHAR(191) NOT NULL,
  PRIMARY KEY (account_id, group_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
  FOREIGN KEY (group_id) REFERENCES `groups`(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS group_scopes (
  group_id VARCHAR(191) NOT NULL,
  scope_key VARCHAR(191) NOT NULL,
  PRIMARY KEY (group_id, scope_key),
  FOREIGN KEY (group_id) REFERENCES `groups`(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS config (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  value TEXT NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);

CREATE TABLE IF NOT EXISTS field_enum (
  id VARCHAR(191) PRIMARY KEY,
  field VARCHAR(191) NOT NULL,
  value VARCHAR(191) NOT NULL,
  name VARCHAR(191) NOT NULL,
  meta TEXT,
  UNIQUE KEY uniq_field_enum_field_value (field, value)
);

INSERT IGNORE INTO field_enum (id, field, value, name) VALUES
('enum-node_mode-edge', 'node_mode', 'edge', 'Edge'),
('enum-node_mode-relay', 'node_mode', 'relay', 'Relay'),
('enum-node_status-healthy', 'node_status', 'healthy', 'Healthy'),
('enum-node_status-degraded', 'node_status', 'degraded', 'Degraded'),
('enum-node_status-pending', 'node_status', 'pending', 'Pending'),
('enum-node_status-inactive', 'node_status', 'inactive', 'Inactive'),
('enum-account_role-super_admin', 'account_role', 'super_admin', 'Super Admin'),
('enum-account_status-active', 'account_status', 'active', 'Active'),
('enum-account_status-disabled', 'account_status', 'disabled', 'Disabled'),
('enum-path_mode-direct', 'path_mode', 'direct', 'Direct'),
('enum-path_mode-relay_chain', 'path_mode', 'relay_chain', 'Relay Chain'),
('enum-path_mode-upstream_pull', 'path_mode', 'upstream_pull', 'Upstream Pull'),
('enum-task_status-planned', 'task_status', 'planned', 'Planned'),
('enum-task_status-pending', 'task_status', 'pending', 'Pending'),
('enum-task_status-connected', 'task_status', 'connected', 'Connected'),
('enum-task_status-failed', 'task_status', 'failed', 'Failed'),
('enum-task_status-cancelled', 'task_status', 'cancelled', 'Cancelled'),
('enum-action_type-chain', 'action_type', 'chain', 'Chain'),
('enum-action_type-direct', 'action_type', 'direct', 'Direct'),
('enum-link_type-parent_child', 'link_type', 'parent_child', 'Parent-Child'),
('enum-link_type-relay', 'link_type', 'relay', 'Relay'),
('enum-link_type-managed', 'link_type', 'managed', 'Managed'),
('enum-trust_state-trusted', 'trust_state', 'trusted', 'Trusted'),
('enum-trust_state-active', 'trust_state', 'active', 'Active'),
('enum-transport_type-public_http', 'transport_type', 'public_http', 'Public HTTP'),
('enum-transport_type-public_https', 'transport_type', 'public_https', 'Public HTTPS'),
('enum-transport_type-reverse_ws_parent', 'transport_type', 'reverse_ws_parent', 'Reverse WS Parent'),
('enum-transport_type-child_ws', 'transport_type', 'child_ws', 'Child WS'),
('enum-transport_type-reverse_ws', 'transport_type', 'reverse_ws', 'Reverse WS'),
('enum-transport_status-connected', 'transport_status', 'connected', 'Connected'),
('enum-transport_status-available', 'transport_status', 'available', 'Available'),
('enum-transport_status-degraded', 'transport_status', 'degraded', 'Degraded'),
('enum-transport_status-failed', 'transport_status', 'failed', 'Failed'),
('enum-transport_status-pending', 'transport_status', 'pending', 'Pending'),
('enum-cert_status-healthy', 'cert_status', 'healthy', 'Healthy'),
('enum-cert_status-renew_soon', 'cert_status', 'renew-soon', 'Renew Soon'),
('enum-cert_status-expired', 'cert_status', 'expired', 'Expired'),
('enum-cert_status-renewed', 'cert_status', 'renewed', 'Renewed'),
('enum-cert_type-public', 'cert_type', 'public', 'Public'),
('enum-cert_type-internal', 'cert_type', 'internal', 'Internal'),
('enum-bootstrap_target_type-node', 'bootstrap_target_type', 'node', 'Node'),
('enum-trust_material_status-active', 'trust_material_status', 'active', 'Active'),
('enum-trust_material_status-rotated', 'trust_material_status', 'rotated', 'Rotated'),
('enum-trust_material_status-pending', 'trust_material_status', 'pending', 'Pending'),
('enum-trust_material_status-consumed', 'trust_material_status', 'consumed', 'Consumed'),
('enum-probe_result_status-connected', 'probe_result_status', 'connected', 'Connected'),
('enum-probe_result_status-failed', 'probe_result_status', 'failed', 'Failed'),
('enum-policy_status-published', 'policy_status', 'published', 'Published'),
('enum-listener_status-up', 'listener_status', 'up', 'Up'),
('enum-listener_status-degraded', 'listener_status', 'degraded', 'Degraded'),
('enum-approval_state-pending', 'approval_state', 'pending', 'Pending'),
('enum-approval_state-approved', 'approval_state', 'approved', 'Approved'),
('enum-approval_state-rejected', 'approval_state', 'rejected', 'Rejected');

INSERT IGNORE INTO field_enum (id, field, value, name, meta) VALUES
('enum-match_type-domain', 'match_type', 'domain', 'Domain', '{"placeholder":"example.com","validationRegex":"^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*\\.[a-zA-Z]{2,}$"}'),
('enum-match_type-domain_suffix', 'match_type', 'domain_suffix', 'Domain Suffix', '{"placeholder":".example.com","validationRegex":"^\\*?(\\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+\\.[a-zA-Z]{2,}$"}'),
('enum-match_type-ip_cidr', 'match_type', 'ip_cidr', 'IP CIDR', '{"placeholder":"10.0.0.0/24","validationRegex":"^([0-9]{1,3}\\.){3}[0-9]{1,3}/[0-9]{1,2}$"}'),
('enum-match_type-ip_range', 'match_type', 'ip_range', 'IP Range', '{"placeholder":"10.0.0.1-10.0.0.255","validationRegex":"^([0-9]{1,3}\\.){3}[0-9]{1,3}-([0-9]{1,3}\\.){3}[0-9]{1,3}$"}'),
('enum-match_type-port', 'match_type', 'port', 'Port', '{"placeholder":"8080","validationRegex":"^[0-9]{1,5}$"}'),
('enum-match_type-url_regex', 'match_type', 'url_regex', 'URL Regex', '{"placeholder":"^https://.*\\\\.example\\\\.com/.*","validationRegex":""}'),
('enum-match_type-default', 'match_type', 'default', 'Default (Catch-all)', '{"placeholder":"*","validationRegex":""}');
