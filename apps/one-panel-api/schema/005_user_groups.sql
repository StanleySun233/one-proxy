CREATE TABLE IF NOT EXISTS groups (
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
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS group_scopes (
    group_id VARCHAR(191) NOT NULL,
    scope_key VARCHAR(191) NOT NULL,
    PRIMARY KEY (group_id, scope_key),
    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);
