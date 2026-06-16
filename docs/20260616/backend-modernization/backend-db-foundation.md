# Backend Progress: db-foundation

**Engineer:** db-foundation
**Scope:** Replace the panel API database foundation with Bun and goose.

## Tasks

- [ ] Update `apps/panel/api/go.mod` to replace GORM runtime dependencies with Bun and goose dependencies.
  - Commit:
- [ ] Update `apps/panel/api/go.sum` for Bun and goose dependency resolution.
  - Commit:
- [ ] Replace GORM-backed initialization with Bun-backed initialization in `apps/panel/api/internal/store/mysql.go`.
  - Commit:
- [ ] Create `apps/panel/api/internal/store/bun_models.go` with Bun table models for the first migration group.
  - Commit:
- [ ] Create `apps/panel/api/internal/store/migrations.go` to run goose migrations during store initialization.
  - Commit:
- [ ] Create `apps/panel/api/migrations/00001_initial.sql` from the current schema baseline.
  - Commit:
- [ ] Delete schema bootstrap ownership from `apps/panel/api/internal/store/mysql_schema.go` after goose owns schema execution.
  - Commit:
- [ ] Update `apps/panel/api/internal/store/mysql_account_role_schema.go` into a goose migration or remove it if covered by migrations.
  - Commit:
- [ ] Update `apps/panel/api/internal/store/mysql_audit_schema.go` into a goose migration or remove it if covered by migrations.
  - Commit:
- [ ] Update `apps/panel/api/internal/store/mysql_node_access_path_schema.go` into a goose migration or remove it if covered by migrations.
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
