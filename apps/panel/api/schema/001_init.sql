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

CREATE TABLE IF NOT EXISTS tenants (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL UNIQUE,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL
);

CREATE TABLE IF NOT EXISTS tenant_memberships (
  tenant_id VARCHAR(191) NOT NULL,
  account_id VARCHAR(191) NOT NULL,
  role VARCHAR(64) NOT NULL,
  create_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  PRIMARY KEY (tenant_id, account_id),
  CONSTRAINT fk_tenant_memberships_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_memberships_account_id FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_memberships_create_id FOREIGN KEY (create_id) REFERENCES accounts(id)
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
  create_id VARCHAR(191) NOT NULL,
  owner_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_nodes_parent_node_id FOREIGN KEY (parent_node_id) REFERENCES nodes(id),
  CONSTRAINT fk_nodes_create_id FOREIGN KEY (create_id) REFERENCES accounts(id),
  CONSTRAINT fk_nodes_owner_id FOREIGN KEY (owner_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS node_links (
  id VARCHAR(191) PRIMARY KEY,
  source_node_id VARCHAR(191) NOT NULL,
  target_node_id VARCHAR(191) NOT NULL,
  link_type VARCHAR(64) NOT NULL,
  trust_state VARCHAR(64) NOT NULL,
  create_id VARCHAR(191) NOT NULL,
  owner_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_links_source_node_id FOREIGN KEY (source_node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_links_target_node_id FOREIGN KEY (target_node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_links_create_id FOREIGN KEY (create_id) REFERENCES accounts(id),
  CONSTRAINT fk_node_links_owner_id FOREIGN KEY (owner_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS scopes (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL,
  description TEXT,
  create_id VARCHAR(191) NOT NULL,
  owner_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_scopes_create_id FOREIGN KEY (create_id) REFERENCES accounts(id),
  CONSTRAINT fk_scopes_owner_id FOREIGN KEY (owner_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS chains (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL UNIQUE,
  destination_scope VARCHAR(191) NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  create_id VARCHAR(191) NOT NULL,
  owner_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_chains_create_id FOREIGN KEY (create_id) REFERENCES accounts(id),
  CONSTRAINT fk_chains_owner_id FOREIGN KEY (owner_id) REFERENCES accounts(id)
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
  create_id VARCHAR(191) NOT NULL,
  owner_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_route_rules_chain_id FOREIGN KEY (chain_id) REFERENCES chains(id),
  CONSTRAINT fk_route_rules_create_id FOREIGN KEY (create_id) REFERENCES accounts(id),
  CONSTRAINT fk_route_rules_owner_id FOREIGN KEY (owner_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS policy_revisions (
  id VARCHAR(191) PRIMARY KEY,
  tenant_id VARCHAR(191) NOT NULL,
  version VARCHAR(191) NOT NULL UNIQUE,
  payload_json LONGTEXT NOT NULL,
  status VARCHAR(64) NOT NULL,
  created_by_account_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_policy_revisions_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_policy_revisions_created_by_account_id FOREIGN KEY (created_by_account_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS node_policy_assignments (
  tenant_id VARCHAR(191) NOT NULL,
  node_id VARCHAR(191) NOT NULL,
  policy_revision_id VARCHAR(191) NOT NULL,
  snapshot_json LONGTEXT NOT NULL,
  assigned_at VARCHAR(64) NOT NULL,
  PRIMARY KEY (tenant_id, node_id),
  CONSTRAINT fk_node_policy_assignments_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_node_policy_assignments_node_id FOREIGN KEY (node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_policy_assignments_policy_revision_id FOREIGN KEY (policy_revision_id) REFERENCES policy_revisions(id)
);

CREATE TABLE IF NOT EXISTS bootstrap_tokens (
  id VARCHAR(191) PRIMARY KEY,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  target_type VARCHAR(64) NOT NULL,
  target_id VARCHAR(191),
  node_name VARCHAR(255),
  node_mode VARCHAR(64),
  scope_key VARCHAR(191),
  parent_node_id VARCHAR(191),
  public_host VARCHAR(255),
  public_port INT,
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
  chain_id VARCHAR(191),
  name VARCHAR(191) NOT NULL,
  mode VARCHAR(64) NOT NULL,
  protocol VARCHAR(64) NOT NULL DEFAULT 'http',
  service_type VARCHAR(64) NOT NULL DEFAULT 'http',
  target_node_id VARCHAR(191),
  entry_node_id VARCHAR(191),
  relay_node_ids_json LONGTEXT NOT NULL,
  listen_host VARCHAR(255),
  listen_port INT NOT NULL DEFAULT 0,
  target_protocol VARCHAR(64) NOT NULL DEFAULT 'http',
  target_host VARCHAR(255),
  target_port INT NOT NULL DEFAULT 0,
  target_sni VARCHAR(255),
  tls_mode VARCHAR(64) NOT NULL DEFAULT 'none',
  auth_mode VARCHAR(64) NOT NULL DEFAULT 'proxy_token',
  options_json LONGTEXT NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  create_id VARCHAR(191) NOT NULL,
  owner_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_node_access_paths_target_node_id FOREIGN KEY (target_node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_access_paths_entry_node_id FOREIGN KEY (entry_node_id) REFERENCES nodes(id),
  CONSTRAINT fk_node_access_paths_chain_id FOREIGN KEY (chain_id) REFERENCES chains(id),
  CONSTRAINT fk_node_access_paths_create_id FOREIGN KEY (create_id) REFERENCES accounts(id),
  CONSTRAINT fk_node_access_paths_owner_id FOREIGN KEY (owner_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS tenant_nodes (
  tenant_id VARCHAR(191) NOT NULL,
  node_id VARCHAR(191) NOT NULL,
  permission VARCHAR(64) NOT NULL,
  create_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  PRIMARY KEY (tenant_id, node_id),
  CONSTRAINT fk_tenant_nodes_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_nodes_node_id FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_nodes_create_id FOREIGN KEY (create_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS tenant_node_links (
  tenant_id VARCHAR(191) NOT NULL,
  node_link_id VARCHAR(191) NOT NULL,
  permission VARCHAR(64) NOT NULL,
  create_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  PRIMARY KEY (tenant_id, node_link_id),
  CONSTRAINT fk_tenant_node_links_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_node_links_node_link_id FOREIGN KEY (node_link_id) REFERENCES node_links(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_node_links_create_id FOREIGN KEY (create_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS tenant_chains (
  tenant_id VARCHAR(191) NOT NULL,
  chain_id VARCHAR(191) NOT NULL,
  permission VARCHAR(64) NOT NULL,
  create_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  PRIMARY KEY (tenant_id, chain_id),
  CONSTRAINT fk_tenant_chains_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_chains_chain_id FOREIGN KEY (chain_id) REFERENCES chains(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_chains_create_id FOREIGN KEY (create_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS tenant_route_rules (
  tenant_id VARCHAR(191) NOT NULL,
  route_rule_id VARCHAR(191) NOT NULL,
  permission VARCHAR(64) NOT NULL,
  create_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  PRIMARY KEY (tenant_id, route_rule_id),
  CONSTRAINT fk_tenant_route_rules_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_route_rules_route_rule_id FOREIGN KEY (route_rule_id) REFERENCES route_rules(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_route_rules_create_id FOREIGN KEY (create_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS tenant_scopes (
  tenant_id VARCHAR(191) NOT NULL,
  scope_id VARCHAR(191) NOT NULL,
  permission VARCHAR(64) NOT NULL,
  create_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  PRIMARY KEY (tenant_id, scope_id),
  CONSTRAINT fk_tenant_scopes_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_scopes_scope_id FOREIGN KEY (scope_id) REFERENCES scopes(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_scopes_create_id FOREIGN KEY (create_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS tenant_access_paths (
  tenant_id VARCHAR(191) NOT NULL,
  access_path_id VARCHAR(191) NOT NULL,
  permission VARCHAR(64) NOT NULL,
  create_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  PRIMARY KEY (tenant_id, access_path_id),
  CONSTRAINT fk_tenant_access_paths_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_access_paths_access_path_id FOREIGN KEY (access_path_id) REFERENCES node_access_paths(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_access_paths_create_id FOREIGN KEY (create_id) REFERENCES accounts(id)
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

CREATE TABLE IF NOT EXISTS business_audit_events (
  id VARCHAR(191) PRIMARY KEY,
  tenant_id VARCHAR(191) NOT NULL,
  occurred_at VARCHAR(64) NOT NULL,
  actor_type VARCHAR(64) NOT NULL,
  actor_id VARCHAR(191) NOT NULL,
  actor_name VARCHAR(191) NOT NULL,
  actor_ip VARCHAR(191) NOT NULL,
  actor_agent VARCHAR(512) NOT NULL,
  action VARCHAR(191) NOT NULL,
  resource_type VARCHAR(191) NOT NULL,
  resource_id VARCHAR(191) NOT NULL,
  resource_name VARCHAR(255) NOT NULL,
  outcome VARCHAR(64) NOT NULL,
  reason VARCHAR(255) NOT NULL,
  request_id VARCHAR(191) NOT NULL,
  before_json LONGTEXT NOT NULL,
  after_json LONGTEXT NOT NULL,
  metadata_json LONGTEXT NOT NULL,
  INDEX idx_business_audit_tenant_time (tenant_id, occurred_at),
  INDEX idx_business_audit_actor_time (actor_id, occurred_at),
  INDEX idx_business_audit_resource_time (resource_type, resource_id, occurred_at),
  INDEX idx_business_audit_action_time (action, occurred_at),
  INDEX idx_business_audit_outcome_time (outcome, occurred_at)
);

CREATE TABLE IF NOT EXISTS network_audit_sessions (
  id VARCHAR(191) PRIMARY KEY,
  tenant_id VARCHAR(191) NOT NULL,
  started_at VARCHAR(64) NOT NULL,
  ended_at VARCHAR(64) NOT NULL,
  actor_type VARCHAR(64) NOT NULL,
  actor_id VARCHAR(191) NOT NULL,
  token_id VARCHAR(191) NOT NULL,
  source_ip VARCHAR(191) NOT NULL,
  entry_node_id VARCHAR(191) NOT NULL,
  exit_node_id VARCHAR(191) NOT NULL,
  target_host VARCHAR(255) NOT NULL,
  target_port INT NOT NULL DEFAULT 0,
  scheme VARCHAR(64) NOT NULL,
  method VARCHAR(64) NOT NULL,
  route_id VARCHAR(191) NOT NULL,
  scope_id VARCHAR(191) NOT NULL,
  chain_id VARCHAR(191) NOT NULL,
  decision VARCHAR(64) NOT NULL,
  deny_reason VARCHAR(255) NOT NULL,
  bytes_in BIGINT NOT NULL DEFAULT 0,
  bytes_out BIGINT NOT NULL DEFAULT 0,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  status_code INT NOT NULL DEFAULT 0,
  error_code VARCHAR(191) NOT NULL,
  received_at VARCHAR(64) NOT NULL,
  metadata_json LONGTEXT NOT NULL,
  INDEX idx_network_audit_tenant_time (tenant_id, ended_at),
  INDEX idx_network_audit_actor_time (actor_id, ended_at),
  INDEX idx_network_audit_token_time (token_id, ended_at),
  INDEX idx_network_audit_entry_node_time (entry_node_id, ended_at),
  INDEX idx_network_audit_exit_node_time (exit_node_id, ended_at),
  INDEX idx_network_audit_target_time (target_host, ended_at),
  INDEX idx_network_audit_route_time (route_id, ended_at),
  INDEX idx_network_audit_scope_time (scope_id, ended_at),
  INDEX idx_network_audit_chain_time (chain_id, ended_at),
  INDEX idx_network_audit_decision_time (decision, ended_at)
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

INSERT IGNORE INTO field_enum (id, field, value, name, meta) VALUES
('enum-node_mode-edge', 'node_mode', 'edge', 'Edge', '{}'),
('enum-node_mode-relay', 'node_mode', 'relay', 'Relay', '{}'),
('enum-node_status-healthy', 'node_status', 'healthy', 'Healthy', '{"color":"#22c55e","className":"is-good"}'),
('enum-node_status-degraded', 'node_status', 'degraded', 'Degraded', '{"color":"#f59e0b","className":"is-warn"}'),
('enum-node_status-pending', 'node_status', 'pending', 'Pending', '{"color":"#6b7280","className":"is-muted"}'),
('enum-node_status-inactive', 'node_status', 'inactive', 'Inactive', '{"color":"#9ca3af","className":"is-muted"}'),
('enum-account_role-super_admin', 'account_role', 'super_admin', 'Super Admin', '{}'),
('enum-account_role-user', 'account_role', 'user', 'User', '{}'),
('enum-tenant_role-tenant_admin', 'tenant_role', 'tenant_admin', 'Tenant Admin', '{}'),
('enum-tenant_role-user', 'tenant_role', 'user', 'User', '{}'),
('enum-binding_permission-manage', 'binding_permission', 'manage', 'Manage', '{}'),
('enum-binding_permission-use', 'binding_permission', 'use', 'Use', '{}'),
('enum-binding_permission-view', 'binding_permission', 'view', 'View', '{}'),
('enum-account_status-active', 'account_status', 'active', 'Active', '{}'),
('enum-account_status-disabled', 'account_status', 'disabled', 'Disabled', '{}'),
('enum-path_mode-direct', 'path_mode', 'direct', 'Direct', '{}'),
('enum-path_mode-relay_chain', 'path_mode', 'relay_chain', 'Relay Chain', '{}'),
('enum-path_mode-upstream_pull', 'path_mode', 'upstream_pull', 'Upstream Pull', '{}'),
('enum-access_protocol-http', 'access_protocol', 'http', 'HTTP', '{}'),
('enum-access_protocol-tcp', 'access_protocol', 'tcp', 'TCP', '{}'),
('enum-access_protocol-tls', 'access_protocol', 'tls', 'TLS', '{}'),
('enum-access_protocol-ssh', 'access_protocol', 'ssh', 'SSH', '{}'),
('enum-access_protocol-rdp', 'access_protocol', 'rdp', 'RDP', '{}'),
('enum-access_protocol-socks5', 'access_protocol', 'socks5', 'SOCKS5', '{}'),
('enum-access_protocol-ss5', 'access_protocol', 'ss5', 'SS5', '{}'),
('enum-access_protocol-udp', 'access_protocol', 'udp', 'UDP', '{}'),
('enum-access_service_type-http', 'access_service_type', 'http', 'HTTP', '{}'),
('enum-access_service_type-https', 'access_service_type', 'https', 'HTTPS', '{}'),
('enum-access_service_type-raw_tcp', 'access_service_type', 'raw_tcp', 'Raw TCP', '{}'),
('enum-access_service_type-raw_udp', 'access_service_type', 'raw_udp', 'Raw UDP', '{}'),
('enum-access_service_type-ssh', 'access_service_type', 'ssh', 'SSH', '{}'),
('enum-access_service_type-rdp', 'access_service_type', 'rdp', 'RDP', '{}'),
('enum-access_service_type-socks5', 'access_service_type', 'socks5', 'SOCKS5', '{}'),
('enum-access_service_type-ss5', 'access_service_type', 'ss5', 'SS5', '{}'),
('enum-access_service_type-tls_passthrough', 'access_service_type', 'tls_passthrough', 'TLS Passthrough', '{}'),
('enum-tls_mode-none', 'tls_mode', 'none', 'None', '{}'),
('enum-tls_mode-passthrough', 'tls_mode', 'passthrough', 'Passthrough', '{}'),
('enum-tls_mode-terminate', 'tls_mode', 'terminate', 'Terminate', '{}'),
('enum-access_auth_mode-proxy_token', 'access_auth_mode', 'proxy_token', 'Proxy Token', '{}'),
('enum-task_status-planned', 'task_status', 'planned', 'Planned', '{"color":"#6b7280","className":"is-muted"}'),
('enum-task_status-pending', 'task_status', 'pending', 'Pending', '{"color":"#3b82f6","className":"is-info"}'),
('enum-task_status-connected', 'task_status', 'connected', 'Connected', '{"color":"#22c55e","className":"is-good"}'),
('enum-task_status-failed', 'task_status', 'failed', 'Failed', '{"color":"#ef4444","className":"is-bad"}'),
('enum-task_status-cancelled', 'task_status', 'cancelled', 'Cancelled', '{"color":"#9ca3af","className":"is-muted"}'),
('enum-action_type-chain', 'action_type', 'chain', 'Chain', '{}'),
('enum-action_type-direct', 'action_type', 'direct', 'Direct', '{}'),
('enum-link_type-parent_child', 'link_type', 'parent_child', 'Parent-Child', '{}'),
('enum-link_type-relay', 'link_type', 'relay', 'Relay', '{}'),
('enum-link_type-managed', 'link_type', 'managed', 'Managed', '{}'),
('enum-trust_state-trusted', 'trust_state', 'trusted', 'Trusted', '{"color":"#22c55e","className":"is-good"}'),
('enum-trust_state-active', 'trust_state', 'active', 'Active', '{"color":"#3b82f6","className":"is-info"}'),
('enum-transport_type-public_http', 'transport_type', 'public_http', 'Public HTTP', '{}'),
('enum-transport_type-public_https', 'transport_type', 'public_https', 'Public HTTPS', '{}'),
('enum-transport_type-reverse_ws_parent', 'transport_type', 'reverse_ws_parent', 'Reverse WS Parent', '{}'),
('enum-transport_type-direct_udp_candidate', 'transport_type', 'direct_udp_candidate', 'Direct UDP Candidate', '{}'),
('enum-transport_type-direct_quic', 'transport_type', 'direct_quic', 'Direct QUIC', '{}'),
('enum-transport_type-child_ws', 'transport_type', 'child_ws', 'Child WS', '{}'),
('enum-transport_type-reverse_ws', 'transport_type', 'reverse_ws', 'Reverse WS', '{}'),
('enum-transport_status-connected', 'transport_status', 'connected', 'Connected', '{"color":"#22c55e","className":"is-good"}'),
('enum-transport_status-available', 'transport_status', 'available', 'Available', '{"color":"#3b82f6","className":"is-info"}'),
('enum-transport_status-degraded', 'transport_status', 'degraded', 'Degraded', '{"color":"#f59e0b","className":"is-warn"}'),
('enum-transport_status-failed', 'transport_status', 'failed', 'Failed', '{"color":"#ef4444","className":"is-bad"}'),
('enum-transport_status-pending', 'transport_status', 'pending', 'Pending', '{"color":"#6b7280","className":"is-muted"}'),
('enum-cert_status-healthy', 'cert_status', 'healthy', 'Healthy', '{"color":"#22c55e","className":"is-good"}'),
('enum-cert_status-degraded', 'cert_status', 'degraded', 'Degraded', '{"color":"#f59e0b","className":"is-warn"}'),
('enum-cert_status-renew_soon', 'cert_status', 'renew-soon', 'Renew Soon', '{"color":"#f59e0b","className":"is-warn"}'),
('enum-cert_status-expired', 'cert_status', 'expired', 'Expired', '{"color":"#dc2626","className":"is-bad"}'),
('enum-cert_status-renewed', 'cert_status', 'renewed', 'Renewed', '{"color":"#22c55e","className":"is-good"}'),
('enum-cert_type-public', 'cert_type', 'public', 'Public', '{}'),
('enum-cert_type-internal', 'cert_type', 'internal', 'Internal', '{}'),
('enum-bootstrap_target_type-node', 'bootstrap_target_type', 'node', 'Node', '{}'),
('enum-trust_material_status-active', 'trust_material_status', 'active', 'Active', '{"color":"#22c55e","className":"is-good"}'),
('enum-trust_material_status-rotated', 'trust_material_status', 'rotated', 'Rotated', '{"color":"#3b82f6","className":"is-info"}'),
('enum-trust_material_status-pending', 'trust_material_status', 'pending', 'Pending', '{"color":"#f59e0b","className":"is-warn"}'),
('enum-trust_material_status-consumed', 'trust_material_status', 'consumed', 'Consumed', '{"color":"#6b7280","className":"is-muted"}'),
('enum-probe_result_status-connected', 'probe_result_status', 'connected', 'Connected', '{"color":"#22c55e","className":"is-good"}'),
('enum-probe_result_status-failed', 'probe_result_status', 'failed', 'Failed', '{"color":"#ef4444","className":"is-bad"}'),
('enum-policy_status-published', 'policy_status', 'published', 'Published', '{"color":"#22c55e","className":"is-good"}'),
('enum-listener_status-up', 'listener_status', 'up', 'Up', '{"color":"#22c55e","className":"is-good"}'),
('enum-listener_status-degraded', 'listener_status', 'degraded', 'Degraded', '{"color":"#f59e0b","className":"is-warn"}'),
('enum-approval_state-pending', 'approval_state', 'pending', 'Pending', '{"color":"#f59e0b","className":"is-warn"}'),
('enum-approval_state-approved', 'approval_state', 'approved', 'Approved', '{"color":"#22c55e","className":"is-good"}'),
('enum-approval_state-rejected', 'approval_state', 'rejected', 'Rejected', '{"color":"#ef4444","className":"is-bad"}'),
('enum-match_type-domain', 'match_type', 'domain', 'Domain', '{"placeholder":"example.com","validationRegex":"^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?(\\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)*\\.[a-zA-Z]{2,}$"}'),
('enum-match_type-domain_suffix', 'match_type', 'domain_suffix', 'Domain Suffix', '{"placeholder":".example.com","validationRegex":"^\\*?(\\.[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?)+\\.[a-zA-Z]{2,}$"}'),
('enum-match_type-ip_cidr', 'match_type', 'ip_cidr', 'IP CIDR', '{"placeholder":"10.0.0.0/24","validationRegex":"^([0-9]{1,3}\\.){3}[0-9]{1,3}/[0-9]{1,2}$"}'),
('enum-match_type-ip_range', 'match_type', 'ip_range', 'IP Range', '{"placeholder":"10.0.0.1-10.0.0.255","validationRegex":"^([0-9]{1,3}\\.){3}[0-9]{1,3}-([0-9]{1,3}\\.){3}[0-9]{1,3}$"}'),
('enum-match_type-port', 'match_type', 'port', 'Port', '{"placeholder":"8080","validationRegex":"^[0-9]{1,5}$"}'),
('enum-match_type-url_regex', 'match_type', 'url_regex', 'URL Regex', '{"placeholder":"^https://.*\\\\.example\\\\.com/.*","validationRegex":""}'),
('enum-match_type-default', 'match_type', 'default', 'Default (Catch-all)', '{"placeholder":"*","validationRegex":""}');
