CREATE TABLE roles (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL UNIQUE,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);

CREATE TABLE accounts (
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

CREATE TABLE sessions (
  id VARCHAR(191) PRIMARY KEY,
  account_id VARCHAR(191) NOT NULL,
  access_token_hash VARCHAR(255) NOT NULL UNIQUE,
  refresh_token_hash VARCHAR(255) NOT NULL,
  expires_at VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_sessions_account_id FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE TABLE nodes (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL,
  mode VARCHAR(64) NOT NULL,
  public_host VARCHAR(255),
  public_port INT,
  scope_key VARCHAR(191) NOT NULL,
  parent_node_id VARCHAR(191),
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  status VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_nodes_parent_node_id FOREIGN KEY (parent_node_id) REFERENCES nodes(id)
);

CREATE TABLE node_links (
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

CREATE TABLE chains (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL UNIQUE,
  destination_scope VARCHAR(191) NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);

CREATE TABLE chain_hops (
  chain_id VARCHAR(191) NOT NULL,
  hop_index INT NOT NULL,
  node_id VARCHAR(191) NOT NULL,
  PRIMARY KEY (chain_id, hop_index),
  CONSTRAINT fk_chain_hops_chain_id FOREIGN KEY (chain_id) REFERENCES chains(id),
  CONSTRAINT fk_chain_hops_node_id FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE route_rules (
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

CREATE TABLE policy_revisions (
  id VARCHAR(191) PRIMARY KEY,
  version VARCHAR(191) NOT NULL UNIQUE,
  payload_json LONGTEXT NOT NULL,
  status VARCHAR(64) NOT NULL,
  created_by_account_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_policy_revisions_created_by_account_id FOREIGN KEY (created_by_account_id) REFERENCES accounts(id)
);

CREATE TABLE node_policy_assignments (
  node_id VARCHAR(191) PRIMARY KEY,
  policy_revision_id VARCHAR(191) NOT NULL,
  snapshot_json LONGTEXT NOT NULL,
  assigned_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_policy_assignments_node_id FOREIGN KEY (node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_policy_assignments_policy_revision_id FOREIGN KEY (policy_revision_id) REFERENCES policy_revisions(id)
);

CREATE TABLE bootstrap_tokens (
  id VARCHAR(191) PRIMARY KEY,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  target_type VARCHAR(64) NOT NULL,
  target_id VARCHAR(191),
  expires_at VARCHAR(64) NOT NULL,
  consumed_at VARCHAR(64),
  created_at VARCHAR(64) NOT NULL
);

CREATE TABLE certificates (
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

CREATE TABLE node_health_snapshots (
  node_id VARCHAR(191) PRIMARY KEY,
  heartbeat_at VARCHAR(64) NOT NULL,
  policy_revision_id VARCHAR(191),
  listener_status_json LONGTEXT NOT NULL,
  cert_status_json LONGTEXT NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_health_snapshots_node_id FOREIGN KEY (node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_health_snapshots_policy_revision_id FOREIGN KEY (policy_revision_id) REFERENCES policy_revisions(id)
);

CREATE TABLE node_api_tokens (
  id VARCHAR(191) PRIMARY KEY,
  node_id VARCHAR(191) NOT NULL,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  expires_at VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_api_tokens_node_id FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE node_trust_materials (
  id VARCHAR(191) PRIMARY KEY,
  node_id VARCHAR(191) NOT NULL,
  material_type VARCHAR(64) NOT NULL,
  material_value LONGTEXT NOT NULL,
  status VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_trust_materials_node_id FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE node_access_paths (
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

CREATE TABLE node_onboarding_tasks (
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

CREATE TABLE node_transports (
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

CREATE TABLE chain_probe_results (
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

CREATE TABLE id_sequences (
  name VARCHAR(64) PRIMARY KEY,
  current_value BIGINT NOT NULL DEFAULT 0,
  updated_at VARCHAR(64) NOT NULL
);

INSERT INTO id_sequences (name, current_value, updated_at)
VALUES ('node_id', 0, UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE name = name;

CREATE TABLE node_enrollment_approvals (
  id VARCHAR(191) PRIMARY KEY,
  bootstrap_token_id VARCHAR(191) NOT NULL,
  node_name VARCHAR(191),
  node_mode VARCHAR(64) NOT NULL,
  scope_key VARCHAR(191) NOT NULL,
  parent_node_id VARCHAR(191),
  public_host VARCHAR(255),
  public_port INT,
  status VARCHAR(64) NOT NULL,
  reviewed_by VARCHAR(191),
  reviewed_at VARCHAR(64),
  reject_reason TEXT,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_enrollment_approvals_bootstrap_token_id FOREIGN KEY (bootstrap_token_id) REFERENCES bootstrap_tokens(id),
  CONSTRAINT fk_node_enrollment_approvals_parent_node_id FOREIGN KEY (parent_node_id) REFERENCES nodes(id)
);

CREATE INDEX idx_node_enrollment_approvals_status
  ON node_enrollment_approvals (status, created_at);

CREATE INDEX idx_node_enrollment_approvals_bootstrap_token_id
  ON node_enrollment_approvals (bootstrap_token_id);

CREATE TABLE node_health_history (
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

CREATE TABLE groups (
  id VARCHAR(191) NOT NULL PRIMARY KEY,
  name VARCHAR(191) NOT NULL,
  description TEXT,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);

CREATE TABLE account_groups (
  account_id VARCHAR(191) NOT NULL,
  group_id VARCHAR(191) NOT NULL,
  PRIMARY KEY (account_id, group_id),
  FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

CREATE TABLE group_scopes (
  group_id VARCHAR(191) NOT NULL,
  scope_key VARCHAR(191) NOT NULL,
  PRIMARY KEY (group_id, scope_key),
  FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

CREATE TABLE config (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  value TEXT NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);
