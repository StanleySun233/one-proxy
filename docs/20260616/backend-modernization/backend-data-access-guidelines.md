# Backend Data Access Guidelines

**Date:** 20260616
**Status:** implementation guidance
**API contract:** `docs/20260616/backend-modernization/api-contract.md`

## Purpose

These guidelines define when backend modernization work should use Bun query builder, explicit raw SQL, and DeletePlan. They apply to the first migrated resource group: chains, route rules, node access paths, chain probes, and tenant bindings.

The goal is not to maximize abstraction. The goal is to keep persistence code reviewable, testable, and located behind clear repository or DeletePlan boundaries.

## Default Boundary

All new data access for the migrated resource group should enter through repository methods or DeletePlan builders.

Use this boundary:

- HTTP handlers call service or store interfaces, not SQL directly.
- Store interfaces expose domain operations, not query fragments.
- Repositories own ordinary reads and writes.
- DeletePlan builders own destructive relationship rules.
- Goose migrations own schema creation and schema evolution.

Do not add ad hoc SQL in handlers, route registration, response mapping, or unrelated helper files.

## Use Bun Query Builder

Use Bun query builder for ordinary repository work where model mapping and structured composition make the query easier to read.

Good Bun query builder cases:

- Create, read, update, and list operations for one primary table.
- Simple joins where the selected model shape is obvious.
- Filtering by request parameters such as tenant, chain, status, name, or enabled state.
- Pagination and ordering that follow existing HTTP behavior.
- Inserts or updates where Bun model tags keep column mapping clearer than manual SQL.
- Queries that are expected to evolve with fields on the same model.

Expected shape:

- Keep query construction inside the repository method.
- Accept `context.Context`.
- Return domain or store types.
- Preserve existing HTTP response semantics through the caller.
- Keep transactions explicit when a method mutates more than one table.

Avoid using Bun query builder when the resulting query hides the actual SQL relationship being expressed. If a reviewer has to mentally decompile the builder chain to find the join or delete rule, raw SQL or DeletePlan is the better choice.

## Use Raw SQL

Use raw SQL when explicit SQL is the clearest representation of the operation.

Good raw SQL cases:

- Complex relationship reads across chains, route rules, node access paths, probes, and tenant bindings.
- Aggregate impact counts used to construct DeletePlan summaries.
- Queries that rely on database-specific behavior.
- Bulk operations where the SQL predicate is the important review surface.
- Migration support or low-level store foundation work that needs direct `database/sql` access.

Raw SQL rules:

- Keep raw SQL behind repository methods, DeletePlan builders, or migration code.
- Bind parameters; do not interpolate values into SQL strings.
- Keep predicates stable enough for tests to compare behavior.
- Prefer one readable SQL statement over a heavily abstracted helper that hides the table relationships.
- Name repository methods by the domain operation, not by the SQL implementation.

Raw SQL is allowed because some backend rules are easier to audit as SQL. It is not a license to scatter SQL across the codebase.

## Use DeletePlan

Use DeletePlan for destructive operations where deleting one resource can affect related data or where the UI needs an impact preview.

Required DeletePlan cases:

- Chain deletion.
- Node access path deletion.
- Any deletion touching route rules, probes, tenant bindings, or other dependent proxy data.
- Any operation where preview and execution must show the same relationship rules.
- Any destructive operation requiring multiple ordered delete steps.

DeletePlan builder rules:

- Build the plan from current database state without mutating data.
- Make each step ordered and named.
- Represent destructive predicates through bound SQL predicates and arguments.
- Include impact items that can be shown in preview and tested against execution.
- Keep the same step set for preview and execution.

DeletePlan executor rules:

- Execute all steps in one transaction.
- Run steps in plan order.
- Roll back the full execution if any step fails.
- Return affected-row counts per step.
- Do not add hidden cascade behavior outside the plan.

Do not use ORM cascade behavior as the product rule for destructive relationships. Foreign keys may protect database integrity, but DeletePlan owns the application-visible deletion contract.

## Choosing Between Options

Use this order of questions:

1. Is the operation destructive and relationship-aware?
   Use DeletePlan.
2. Is the operation ordinary CRUD or a simple list/read that maps cleanly to Bun models?
   Use Bun query builder.
3. Is the operation clearer as explicit SQL because the relationship, aggregate, predicate, or database behavior is the important part?
   Use raw SQL behind a repository or DeletePlan builder.
4. Does the operation change schema?
   Use goose migration, not runtime repository code.

When the choice is close, choose the version that makes code review more likely to catch a wrong table, wrong predicate, missing tenant boundary, or hidden destructive effect.

## Review Checklist

Before merging data-access work for the migrated resource group, verify:

- SQL or query-builder code lives behind repository methods or DeletePlan builders.
- HTTP handlers do not gain persistence details.
- Multi-table mutations have explicit transaction boundaries.
- Delete preview and execution share the same DeletePlan rules.
- New schema changes are goose migrations.
- Raw SQL uses bound parameters.
- Tests cover behavior at the same boundary where the rule lives.

## Test Expectations

Focused tests should prove the behavior that the chosen access pattern owns.

For Bun query builder repositories:

- Test the domain operation and returned data shape.
- Cover relevant filters, ordering, pagination, and tenant boundaries.

For raw SQL repositories:

- Test the relationship or aggregate result that made raw SQL necessary.
- Include a case that would fail if a join or predicate is missing.

For DeletePlan:

- Compare preview and execution step sets.
- Cover ordered execution, rollback on failure, and per-step affected rows.
- Verify that impact summaries are derived from the same plan rules used for execution.

Tests do not need to prove Bun itself works. They need to prove that the backend rule encoded through Bun, raw SQL, or DeletePlan is correct.
