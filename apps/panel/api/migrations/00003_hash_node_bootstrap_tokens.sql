-- +goose Up
UPDATE bootstrap_tokens
SET token_hash = LOWER(SHA2(token_hash, 256))
WHERE token_hash NOT REGEXP '^[0-9a-f]{64}$';

UPDATE node_api_tokens
SET token_hash = LOWER(SHA2(token_hash, 256))
WHERE token_hash NOT REGEXP '^[0-9a-f]{64}$';

-- +goose Down
SELECT 1;
