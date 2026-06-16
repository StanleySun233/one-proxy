# Dev Roadmap: Backend Modernization

**Date:** 20260616
**Status:** in-progress
**Product document:** docs/20260616/backend-modernization/product-requirements.md

## Summary

Modernize the panel API backend around Bun, goose, and explicit DeletePlan rules. This plan does not preserve the current mixed GORM plus `database/sql` backend as a long-term compatibility layer; it replaces the foundation and migrates the riskiest relationship-heavy resources first.

## Team

| Role | Agent Name | Progress File |
|------|------------|---------------|
| Backend | db-foundation | [->](./backend-db-foundation.md) |
| Backend | delete-plan | [->](./backend-delete-plan.md) |
| Backend | proxy-repositories | [->](./backend-proxy-repositories.md) |
| Backend | contract-docs | [->](./backend-contract-docs.md) |
| Test | backend-modernization-test | [->](./test-backend-modernization.md) |
| Product | product-manager | [->](./product-manager.md) |

## Tasks

Each task represents one atomic, file-scoped unit of work. No two engineers may touch the same file.

### API Contract

- [x] contract-docs: write `docs/20260616/backend-modernization/api-contract.md` covering HTTP compatibility expectations, internal repository contracts, DeletePlan shape, migration rules, and test acceptance.

### Backend Tasks

- [x] db-foundation: update `apps/panel/api/go.mod` to replace GORM runtime dependencies with Bun and goose dependencies (depends: api-contract.md).
- [x] db-foundation: update `apps/panel/api/go.sum` for Bun and goose dependency resolution (depends: go.mod).
- [x] db-foundation: replace GORM-backed initialization with Bun-backed initialization in `apps/panel/api/internal/store/mysql.go` (depends: go.mod).
- [x] db-foundation: create `apps/panel/api/internal/store/bun_models.go` with Bun table models for the first migration group (depends: mysql.go).
- [x] db-foundation: create `apps/panel/api/internal/store/migrations.go` to run goose migrations during store initialization (depends: mysql.go).
- [x] db-foundation: create `apps/panel/api/migrations/00001_initial.sql` from the current schema baseline (depends: migrations.go).
- [x] db-foundation: delete schema bootstrap ownership from `apps/panel/api/internal/store/mysql_schema.go` after goose owns schema execution (depends: 00001_initial.sql).
- [x] db-foundation: update `apps/panel/api/internal/store/mysql_account_role_schema.go` into a goose migration or remove it if covered by migrations (depends: migrations.go).
- [x] db-foundation: update `apps/panel/api/internal/store/mysql_audit_schema.go` into a goose migration or remove it if covered by migrations (depends: migrations.go).
- [x] db-foundation: update `apps/panel/api/internal/store/mysql_node_access_path_schema.go` into a goose migration or remove it if covered by migrations (depends: migrations.go).
- [x] delete-plan: create `apps/panel/api/internal/store/deleteplan/plan.go` with DeletePlan, DeletePlanStep, DeleteImpactItem, and execution result types (depends: api-contract.md).
- [x] delete-plan: create `apps/panel/api/internal/store/deleteplan/mysql_executor.go` to execute DeletePlan steps inside one SQL transaction (depends: plan.go).
- [x] delete-plan: create `apps/panel/api/internal/store/deleteplan/mysql_executor_test.go` covering ordered execution, rollback, and affected-row reporting (depends: mysql_executor.go).
- [x] proxy-repositories: create `apps/panel/api/internal/store/proxy_repository.go` with Bun-backed chain, route rule, access path, probe, and tenant binding repository methods (depends: bun_models.go, deleteplan/plan.go).
- [x] proxy-repositories: replace chain CRUD and chain DeletePlan construction in `apps/panel/api/internal/store/mysql_chain.go` (depends: proxy_repository.go).
- [x] proxy-repositories: replace route rule CRUD and relationship cleanup in `apps/panel/api/internal/store/mysql_route.go` (depends: proxy_repository.go).
- [x] proxy-repositories: replace node access path CRUD and access path DeletePlan construction in `apps/panel/api/internal/store/mysql_node_access_path.go` (depends: proxy_repository.go).
- [x] proxy-repositories: replace chain and access path delete impact SQL with DeletePlan-derived output in `apps/panel/api/internal/store/mysql_delete_impact.go` (depends: proxy_repository.go).
- [x] proxy-repositories: update store interfaces in `apps/panel/api/internal/store/store.go` for DeletePlan-backed delete impact and execution boundaries (depends: proxy_repository.go).
- [x] contract-docs: create `docs/20260616/backend-modernization/backend-data-access-guidelines.md` documenting when to use Bun query builder, raw SQL, and DeletePlan (depends: implementation tasks).

### Testing

- [x] backend-modernization-test: run `go test ./...` in `apps/panel/api` (depends: all backend tasks).
- [ ] backend-modernization-test: run migration bootstrap test against an empty MySQL-compatible database or documented local test database (depends: migrations.go).
- [x] backend-modernization-test: verify chain delete preview and chain delete execution use the same DeletePlan step set (depends: proxy_repository.go).
- [x] backend-modernization-test: verify node access path delete preview and delete execution use the same DeletePlan step set (depends: proxy_repository.go).

### Product Verification

- [ ] product-manager: verify implementation against `docs/20260616/backend-modernization/product-requirements.md` after testing passes.

## Architecture Decisions

- Bun is the primary DB access layer for new repository code.
- Raw SQL remains allowed for high-signal complex queries, but it must live behind repository methods or DeletePlan builders.
- Goose migrations replace ad hoc schema bootstrap and ensure-column functions.
- Delete preview and delete execution must share DeletePlan definitions.
- No old GORM/data-access compatibility layer is kept after the foundation migration.
- The first implementation slice is proxy data: chains, route rules, node access paths, probe results, and tenant bindings.

## Blocker Log

| Date | Task | Engineer | Blocker Description | Resolution |
|------|------|----------|---------------------|------------|
| 2026-06-16 | migration bootstrap test against empty MySQL-compatible database | backend-modernization-test | No local MySQL-compatible test database or documented DSN was available without starting services; local evidence is limited to `runMigrations` calling goose and `00001_initial.sql` existing. Acceptance remains not fully passed until an empty database bootstrap is executed. | Open |
