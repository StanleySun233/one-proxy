CREATE TABLE IF NOT EXISTS roles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS accounts (
  id TEXT PRIMARY KEY,
  account TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role_id TEXT NOT NULL,
  status TEXT NOT NULL,
  must_rotate_password INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (role_id) REFERENCES roles(id)
);

CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY,
  account_id TEXT NOT NULL,
  access_token_hash TEXT NOT NULL UNIQUE,
  refresh_token_hash TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  mode TEXT NOT NULL,
  public_host TEXT,
  public_port INTEGER,
  scope_key TEXT NOT NULL,
  parent_node_id TEXT,
  enabled INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (parent_node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS node_links (
  id TEXT PRIMARY KEY,
  source_node_id TEXT NOT NULL,
  target_node_id TEXT NOT NULL,
  link_type TEXT NOT NULL,
  trust_state TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (source_node_id) REFERENCES nodes(id),
  FOREIGN KEY (target_node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS chains (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  destination_scope TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chain_hops (
  chain_id TEXT NOT NULL,
  hop_index INTEGER NOT NULL,
  node_id TEXT NOT NULL,
  PRIMARY KEY (chain_id, hop_index),
  FOREIGN KEY (chain_id) REFERENCES chains(id),
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS route_rules (
  id TEXT PRIMARY KEY,
  priority INTEGER NOT NULL,
  match_type TEXT NOT NULL,
  match_value TEXT NOT NULL,
  action_type TEXT NOT NULL,
  chain_id TEXT,
  destination_scope TEXT,
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (chain_id) REFERENCES chains(id)
);

CREATE TABLE IF NOT EXISTS policy_revisions (
  id TEXT PRIMARY KEY,
  version TEXT NOT NULL UNIQUE,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL,
  created_by_account_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (created_by_account_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS node_policy_assignments (
  node_id TEXT PRIMARY KEY,
  policy_revision_id TEXT NOT NULL,
  assigned_at TEXT NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id),
  FOREIGN KEY (policy_revision_id) REFERENCES policy_revisions(id)
);

CREATE TABLE IF NOT EXISTS bootstrap_tokens (
  id TEXT PRIMARY KEY,
  token_hash TEXT NOT NULL UNIQUE,
  target_type TEXT NOT NULL,
  target_id TEXT,
  expires_at TEXT NOT NULL,
  consumed_at TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS certificates (
  id TEXT PRIMARY KEY,
  owner_type TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  cert_type TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT 'manual',
  status TEXT NOT NULL,
  not_before TEXT,
  not_after TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS node_health_snapshots (
  node_id TEXT PRIMARY KEY,
  heartbeat_at TEXT NOT NULL,
  policy_revision_id TEXT,
  listener_status_json TEXT NOT NULL,
  cert_status_json TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id),
  FOREIGN KEY (policy_revision_id) REFERENCES policy_revisions(id)
);

CREATE TABLE IF NOT EXISTS node_api_tokens (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);

CREATE TABLE IF NOT EXISTS node_trust_materials (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  material_type TEXT NOT NULL,
  material_value TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (node_id) REFERENCES nodes(id)
);
