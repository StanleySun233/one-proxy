CREATE TABLE IF NOT EXISTS node_enrollment_approvals (
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

CREATE INDEX IF NOT EXISTS idx_node_enrollment_approvals_status
  ON node_enrollment_approvals (status, created_at);

CREATE INDEX IF NOT EXISTS idx_node_enrollment_approvals_bootstrap_token_id
  ON node_enrollment_approvals (bootstrap_token_id);
