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
