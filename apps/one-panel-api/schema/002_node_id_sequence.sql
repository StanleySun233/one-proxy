CREATE TABLE IF NOT EXISTS id_sequences (
  name VARCHAR(64) PRIMARY KEY,
  current_value BIGINT NOT NULL DEFAULT 0,
  updated_at VARCHAR(64) NOT NULL
);

INSERT INTO id_sequences (name, current_value, updated_at)
VALUES ('node_id', 0, UTC_TIMESTAMP())
ON DUPLICATE KEY UPDATE name = name;
