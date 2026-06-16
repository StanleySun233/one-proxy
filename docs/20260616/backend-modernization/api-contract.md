# Backend Modernization API Contract

**Date:** 20260616
**Status:** draft contract for implementation
**Product document:** `docs/20260616/backend-modernization/product-requirements.md`

## Purpose

This document defines the backend modernization contract for the first migration slice: chains, route rules, node access paths, chain probes, and tenant bindings. It is the compatibility boundary for implementation and testing while the panel API moves from the current mixed GORM plus `database/sql` store to Bun, goose, and explicit DeletePlan rules.

The modernization does not create a long-term compatibility layer between old and new repositories. Compatibility means preserving the externally observable HTTP behavior that existing panel clients depend on, while replacing the internal persistence foundation.

## HTTP Compatibility Expectations

The first backend slice must preserve the current HTTP contract for existing panel clients unless a product requirement explicitly changes it.

Compatibility requirements:

- Existing route paths, HTTP methods, authentication requirements, and authorization checks remain unchanged.
- Successful responses keep the same status codes and JSON field names.
- Error responses keep the same status code class and must not turn known client errors into `500` responses.
- Empty-list responses remain empty JSON arrays or the existing response wrapper shape, not `null`.
- Pagination, sorting, filtering, and search parameters retain their current names and semantics.
- Create and update requests continue to accept the existing JSON payload shape.
- Delete preview endpoints and delete execution endpoints must report impact from the same DeletePlan rules.

Implementation may change internal repository names, SQL shape, transaction boundaries, and migration ownership as long as the HTTP behavior above remains stable.

Any intentional HTTP behavior change must be documented before implementation and must name the affected route, old behavior, new behavior, and test coverage.

## Internal Repository Contracts

Bun-backed repositories are the primary data-access boundary for the migrated resource group. HTTP handlers and service logic must not scatter SQL for these resources across unrelated files.

Repository requirements:

- Repositories own database reads and writes for their resource group.
- Repository methods accept `context.Context`.
- Repository methods return domain/store types, not transport-only response objects.
- Repository methods must not depend on HTTP request or response types.
- Repository methods that mutate more than one table must either own the transaction or accept an explicit transaction handle from the caller.
- Complex raw SQL is allowed when it is clearer than query-builder code, but it must live behind a repository method or DeletePlan builder.
- Bun query builder is preferred for ordinary CRUD, simple joins, and queries where model mapping improves readability.
- Direct `database/sql` access remains available through the store foundation for driver-specific operations, migrations, and explicit raw queries.

Transaction requirements:

- Multi-step destructive operations execute in one SQL transaction.
- Delete preview does not mutate data.
- Delete execution must use the same ordered DeletePlan step set that preview used to derive impact.
- A failed step rolls back the full delete execution.
- Affected-row counts are reported per executed step.

The repository contract is intentionally internal. It should be stable enough for the first migration slice, but it is not an API promise to external clients.

## DeletePlan Shape

DeletePlan is the shared contract between impact preview and destructive execution. It must be explicit enough for code review to see what data can be deleted before the executor runs it.

Required logical shape:

```go
type DeletePlan struct {
    ResourceType string
    ResourceID   string
    Summary      []DeleteImpactItem
    Steps        []DeletePlanStep
}

type DeleteImpactItem struct {
    ResourceType string
    ResourceID   string
    DisplayName  string
    Count        int64
}

type DeletePlanStep struct {
    Name           string
    Table          string
    Operation      string
    WhereSQL       string
    Args           []any
    ExpectedImpact []DeleteImpactItem
}

type DeleteExecutionResult struct {
    PlanResourceType string
    PlanResourceID   string
    Steps            []DeleteStepResult
}

type DeleteStepResult struct {
    Name         string
    Table        string
    RowsAffected int64
}
```

Field rules:

- `ResourceType` identifies the root resource being deleted, such as `chain` or `node_access_path`.
- `ResourceID` identifies the root resource instance.
- `Summary` is the user-visible impact preview derived from the same relationships represented by `Steps`.
- `Steps` is ordered. Execution must run steps in the order provided.
- `Name` is a stable identifier used in logs, tests, and result reporting.
- `Table` names the table affected by the step.
- `Operation` is initially `delete`; other operations require a contract update.
- `WhereSQL` contains only the predicate fragment for the step, not a full ad hoc SQL script.
- `Args` contains bound parameters for `WhereSQL`; values must not be string-interpolated into SQL.
- `ExpectedImpact` describes the impact represented by that step and is used by preview tests.

DeletePlan builders must be deterministic for the same database state. Preview and execution tests should compare step names, table names, operations, predicates, and bound arguments, not only final counts.

## Migration Rules

Goose migrations are the source of truth for schema creation and schema evolution after the foundation task lands.

Migration requirements:

- New schema changes must be represented as goose migrations.
- Ad hoc schema bootstrap functions and ensure-column functions must not own schema evolution after their equivalent migrations exist.
- The initial migration must represent the current schema baseline needed by the panel API.
- Migrations must be ordered, reviewable, and safe to run on an empty database.
- Migration files must avoid hiding destructive relationship behavior behind ORM cascade rules.
- Foreign keys may protect integrity, but DeletePlan remains responsible for application-visible destructive relationship rules.
- Runtime repository code must not silently create, alter, or repair tables.

Rollback support should be explicit per migration. If a down migration cannot safely restore data, it must still make the limitation visible in the migration file instead of silently pretending that rollback is lossless.

## Test Acceptance

The modernization slice is accepted only when tests prove the contract, not just the implementation shape.

Required acceptance:

- `apps/panel/api` builds and tests pass.
- Migration bootstrap succeeds against an empty MySQL-compatible test database or a documented local test database.
- Chain delete preview and chain delete execution use the same DeletePlan step set.
- Node access path delete preview and delete execution use the same DeletePlan step set.
- DeletePlan executor tests cover ordered execution, rollback on failure, and per-step affected-row reporting.
- Repository tests or integration tests cover the migrated chain, route rule, node access path, probe, and tenant binding paths touched by the first slice.
- HTTP compatibility tests or handler-level tests cover unchanged success and error response shapes for the migrated endpoints.

Evidence expected in test output:

- The command or test target that was run.
- The database target type used for migration bootstrap tests.
- The DeletePlan step comparison used for preview versus execution.
- Any intentionally skipped test with the reason and follow-up owner.

## Out of Scope

This contract does not define the final OpenAPI document, generated server types, or long-term public SDK behavior. Those may be added after the first backend slice is stable. Until then, the existing HTTP behavior is the external compatibility source, and this document defines the internal modernization boundary.
