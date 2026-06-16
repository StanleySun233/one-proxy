# Product Requirements: Backend Modernization

**Date:** 20260616
**Status:** planned

## Goal

Modernize the panel API backend around a maintainable long-term stack:

- HTTP: `net/http` first, optional `chi` if route grouping and path parameters become painful.
- DB access: Bun.
- Migration: goose.
- Complex SQL: Bun query builder or explicit raw SQL.
- API contract: OpenAPI plus `oapi-codegen` later.
- Deletion: self-owned DeletePlan, not ORM cascade magic.

## Non-Goals

- Do not keep long-term compatibility with the current mixed GORM plus `database/sql` store implementation.
- Do not build a dual data-access layer that keeps old and new repositories alive indefinitely.
- Do not hide destructive relationship rules behind ORM cascade behavior.
- Do not migrate the whole backend in one unreviewable rewrite.

## Requirements

1. Replace the current DB foundation with a Bun-backed store connection while retaining access to raw `database/sql` when needed.
2. Introduce goose migrations as the source of truth for schema creation and schema evolution.
3. Move delete preview and delete execution onto a shared DeletePlan model so the UI impact preview and transaction behavior come from the same rule set.
4. Migrate the first high-risk resource group first: chains, route rules, node access paths, chain probes, and tenant bindings.
5. Keep complex SQL explicit where clarity matters more than abstraction.
6. Remove GORM as a production dependency when the Bun foundation is in place.
7. Add focused tests proving generated delete plans match executed delete transactions.

## Acceptance Criteria

- `apps/panel/api` builds and tests pass.
- New schema changes are represented as goose migrations.
- Chain and node access path delete preview and delete execution share DeletePlan definitions.
- The first migrated repository group no longer uses ad hoc SQL scattered across unrelated files.
- GORM is not required by the panel API runtime after the foundation task is complete.
- Documentation states when to use Bun query builder, raw SQL, and DeletePlan.
