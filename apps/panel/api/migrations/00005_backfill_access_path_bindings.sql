-- +goose Up
INSERT IGNORE INTO tenant_access_paths (tenant_id, access_path_id, permission, create_id, created_at)
SELECT tc.tenant_id, nap.id, 'manage', nap.create_id, UTC_TIMESTAMP()
FROM node_access_paths nap
JOIN tenant_chains tc ON tc.chain_id = nap.chain_id
WHERE tc.permission IN ('use', 'manage');

-- +goose Down
SELECT 1;
