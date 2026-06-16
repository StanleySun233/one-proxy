-- +goose Up
CREATE TABLE IF NOT EXISTS route_rule_groups (
  id VARCHAR(191) PRIMARY KEY,
  name VARCHAR(191) NOT NULL,
  description TEXT NOT NULL,
  enabled TINYINT(1) NOT NULL DEFAULT 1,
  create_id VARCHAR(191) NOT NULL,
  owner_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  CONSTRAINT fk_route_rule_groups_create_id FOREIGN KEY (create_id) REFERENCES accounts(id),
  CONSTRAINT fk_route_rule_groups_owner_id FOREIGN KEY (owner_id) REFERENCES accounts(id)
);

CREATE TABLE IF NOT EXISTS tenant_route_rule_groups (
  tenant_id VARCHAR(191) NOT NULL,
  route_rule_group_id VARCHAR(191) NOT NULL,
  permission VARCHAR(64) NOT NULL,
  create_id VARCHAR(191) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  PRIMARY KEY (tenant_id, route_rule_group_id),
  CONSTRAINT fk_tenant_route_rule_groups_tenant_id FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_route_rule_groups_group_id FOREIGN KEY (route_rule_group_id) REFERENCES route_rule_groups(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant_route_rule_groups_create_id FOREIGN KEY (create_id) REFERENCES accounts(id)
);

ALTER TABLE route_rules ADD COLUMN group_id VARCHAR(191) NULL AFTER id;

CREATE TEMPORARY TABLE route_rule_group_signatures AS
SELECT
  rr.id AS route_rule_id,
  CONCAT('route-rule-group-', LEFT(MD5(COALESCE(GROUP_CONCAT(CONCAT(trr.tenant_id, ':', trr.permission) ORDER BY trr.tenant_id, trr.permission SEPARATOR '|'), '')), 16)) AS group_id,
  LEFT(MD5(COALESCE(GROUP_CONCAT(CONCAT(trr.tenant_id, ':', trr.permission) ORDER BY trr.tenant_id, trr.permission SEPARATOR '|'), '')), 8) AS group_hash,
  rr.create_id,
  rr.owner_id,
  rr.created_at,
  rr.updated_at
FROM route_rules rr
LEFT JOIN tenant_route_rules trr ON trr.route_rule_id = rr.id
GROUP BY rr.id, rr.create_id, rr.owner_id, rr.created_at, rr.updated_at;

INSERT INTO route_rule_groups (id, name, description, enabled, create_id, owner_id, created_at, updated_at)
SELECT
  group_id,
  CONCAT('Migrated route group ', group_hash),
  '',
  1,
  MIN(create_id),
  MIN(owner_id),
  MIN(created_at),
  MAX(updated_at)
FROM route_rule_group_signatures
GROUP BY group_id, group_hash;

UPDATE route_rules rr
JOIN route_rule_group_signatures sig ON sig.route_rule_id = rr.id
SET rr.group_id = sig.group_id;

INSERT INTO tenant_route_rule_groups (tenant_id, route_rule_group_id, permission, create_id, created_at)
SELECT DISTINCT trr.tenant_id, rr.group_id, trr.permission, trr.create_id, trr.created_at
FROM tenant_route_rules trr
JOIN route_rules rr ON rr.id = trr.route_rule_id;

DROP TEMPORARY TABLE route_rule_group_signatures;

ALTER TABLE route_rules MODIFY group_id VARCHAR(191) NOT NULL;
ALTER TABLE route_rules ADD CONSTRAINT fk_route_rules_group_id FOREIGN KEY (group_id) REFERENCES route_rule_groups(id);

DROP TABLE tenant_route_rules;

-- +goose Down
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

INSERT INTO tenant_route_rules (tenant_id, route_rule_id, permission, create_id, created_at)
SELECT trg.tenant_id, rr.id, trg.permission, trg.create_id, trg.created_at
FROM tenant_route_rule_groups trg
JOIN route_rules rr ON rr.group_id = trg.route_rule_group_id;

ALTER TABLE route_rules DROP FOREIGN KEY fk_route_rules_group_id;
ALTER TABLE route_rules DROP COLUMN group_id;
DROP TABLE tenant_route_rule_groups;
DROP TABLE route_rule_groups;
