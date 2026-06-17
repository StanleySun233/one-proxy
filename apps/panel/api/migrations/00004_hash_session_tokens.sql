-- +goose Up
UPDATE sessions
SET access_token_hash = LOWER(SHA2(access_token_hash, 256))
WHERE access_token_hash NOT REGEXP '^[0-9a-f]{64}$';

UPDATE sessions
SET refresh_token_hash = LOWER(SHA2(refresh_token_hash, 256))
WHERE refresh_token_hash NOT REGEXP '^[0-9a-f]{64}$';

-- +goose Down
SELECT 1;
