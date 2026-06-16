# Backend Progress: db-foundation

**Engineer:** db-foundation
**Scope:** Replace the panel API database foundation with Bun and goose.

## Tasks

- [x] Update `apps/panel/api/go.mod` to replace GORM runtime dependencies with Bun and goose dependencies.
  - Commit: 15b3f4f
- [x] Update `apps/panel/api/go.sum` for Bun and goose dependency resolution.
  - Commit: 15b3f4f
- [x] Replace GORM-backed initialization with Bun-backed initialization in `apps/panel/api/internal/store/mysql.go`.
  - Commit: 15b3f4f
- [x] Create `apps/panel/api/internal/store/bun_models.go` with Bun table models for the first migration group.
  - Commit: 15b3f4f
- [x] Create `apps/panel/api/internal/store/migrations.go` to run goose migrations during store initialization.
  - Commit: 15b3f4f
- [x] Create `apps/panel/api/migrations/00001_initial.sql` from the current schema baseline.
  - Commit: 15b3f4f
- [x] Delete schema bootstrap ownership from `apps/panel/api/internal/store/mysql_schema.go` after goose owns schema execution.
  - Commit: 15b3f4f
- [x] Update `apps/panel/api/internal/store/mysql_account_role_schema.go` into a goose migration or remove it if covered by migrations.
  - Commit: 15b3f4f
- [x] Update `apps/panel/api/internal/store/mysql_audit_schema.go` into a goose migration or remove it if covered by migrations.
  - Commit: 15b3f4f
- [x] Update `apps/panel/api/internal/store/mysql_node_access_path_schema.go` into a goose migration or remove it if covered by migrations.
  - Commit: 15b3f4f

## Verification

- `go test ./internal/store`
- `go test ./...`

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
