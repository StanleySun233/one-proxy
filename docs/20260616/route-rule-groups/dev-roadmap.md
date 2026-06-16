# Dev Roadmap: Route Rule Groups

**Date:** 20260616
**Status:** completed
**Product document:** docs/20260616/route-rule-groups/product-requirements.md

## Summary

Route rules will be moved under route rule groups, and tenant grants will be issued on the group instead of individual rules. The old route-rule grant model will be removed from runtime code; production data on camelbot will be converted with SQL after code verification.

## Team

| Role | Agent Name | Progress File |
|------|------------|---------------|
| Backend | route-groups-api | [->](./backend-route-groups-api.md) |
| Frontend | route-groups-ui | [->](./frontend-route-groups-ui.md) |
| Test | route-groups-test | [->](./test-route-groups.md) |
| Product | route-groups-product | [->](./product-route-groups.md) |

## Business Construction

- Data model:
  - `route_rule_groups`: `id`, `name`, `description`, `enabled`, `create_id`, `owner_id`, `created_at`, `updated_at`.
  - `route_rules.group_id`: required foreign key to `route_rule_groups`.
  - `tenant_route_rule_groups`: `tenant_id`, `route_rule_group_id`, `permission`, `create_id`, `created_at`.
  - `tenant_route_rules`: removed from runtime and dropped from the target schema.
- Authorization:
  - Creating a route rule group binds the active tenant with `manage`.
  - Listing route rules returns rules whose group is bound to the active tenant with `use` or `manage`.
  - Creating, editing, deleting a rule requires `manage` on its group.
  - Grant UI operates on `route_rule_group`; rule-level grant buttons are removed.
- Policy:
  - Policy publication and extension bootstrap use route rules visible through group bindings.
  - Disabled groups exclude their rules from policy compilation while still allowing management.
- Delete behavior:
  - Deleting a route rule group shows impacted route rules and tenant group bindings, then deletes group bindings, contained route rules, and the group through DeletePlan.
- Production SQL:
  - Create new group tables.
  - Add nullable `group_id` to `route_rules`.
  - Generate initial groups from current `tenant_route_rules` permission signatures so tenant visibility is not broadened.
  - Assign each existing rule to its generated group.
  - Make `route_rules.group_id` required.
  - Drop old `tenant_route_rules`.

## Tasks

Each task represents one atomic, file-scoped unit of work. No two engineers may touch the same file.

### API Contract
- [x] route-groups-api: write api-contract.md covering route rule group endpoints, route rule payload changes, grant resource type changes, and camelbot SQL migration.

### Backend Tasks
- [x] route-groups-api: update schema and migration files in `apps/panel/api/migrations/00002_route_rule_groups.sql` and `apps/panel/api/schema/001_init.sql` (depends: api-contract.md)
- [x] route-groups-api: add route rule group domain models in `apps/panel/api/internal/features/proxy/domain/route.go` (depends: api-contract.md)
- [x] route-groups-api: add Bun/store models in `apps/panel/api/internal/store/bun_models.go` (depends: api-contract.md)
- [x] route-groups-api: update route repository and group repository methods in `apps/panel/api/internal/store/proxy_repository.go` and `apps/panel/api/internal/store/mysql_route.go` (depends: api-contract.md)
- [x] route-groups-api: update store interfaces in `apps/panel/api/internal/store/store.go` (depends: api-contract.md)
- [x] route-groups-api: update resource binding mapping in `apps/panel/api/internal/store/mysql_tenant_resource_binding.go` and `apps/panel/api/internal/domain/tenant.go` (depends: api-contract.md)
- [x] route-groups-api: implement route group service and route rule permission changes in `apps/panel/api/internal/features/proxy/service/route.go` (depends: api-contract.md)
- [x] route-groups-api: add HTTP handlers/routes in `apps/panel/api/internal/features/proxy/httpapi/handler_route.go` and `apps/panel/api/internal/features/proxy/httpapi/router.go` (depends: api-contract.md)
- [x] route-groups-api: update delete impact and DeletePlan behavior in `apps/panel/api/internal/store/mysql_delete_impact.go` and `apps/panel/api/internal/features/proxy/domain/delete_impact.go` (depends: api-contract.md)
- [x] route-groups-api: update policy and bootstrap callers if route group filtering changes method names in `apps/panel/api/internal/store/mysql_policy.go` and `apps/panel/api/internal/service/policy.go` (depends: api-contract.md)

### Frontend Tasks
- [x] route-groups-ui: update TypeScript API/types in `apps/panel/web/lib/types/proxy.ts`, `apps/panel/web/lib/types/grants.ts`, and `apps/panel/web/lib/api/proxy.ts` (depends: api-contract.md)
- [x] route-groups-ui: update route form state and validation payload in `apps/panel/web/app/[locale]/(console)/proxy/routes/_lib/form.ts` and `apps/panel/web/app/[locale]/(console)/proxy/routes/_hooks/use-route-rule-validation.ts` (depends: api-contract.md)
- [x] route-groups-ui: update route form and table components in `apps/panel/web/app/[locale]/(console)/proxy/routes/_components/route-rule-form.tsx` and `apps/panel/web/app/[locale]/(console)/proxy/routes/_components/route-rule-table.tsx` (depends: api-contract.md)
- [x] route-groups-ui: rebuild `/proxy/routes` group-first page workflow in `apps/panel/web/app/[locale]/(console)/proxy/routes/page.tsx` (depends: api-contract.md)
- [x] route-groups-ui: add route group i18n strings in `apps/panel/web/messages/zh/proxyRoutes.json` and `apps/panel/web/messages/en/proxyRoutes.json` (depends: api-contract.md)

### Testing
- [x] route-groups-test: compile backend with `go test ./...` from `apps/panel/api` and build panel web with `npm run build` from `apps/panel/web` (depends: all backend and frontend tasks)
- [x] route-groups-test: verify route group CRUD, group grants, route rule CRUD under group permissions, policy publish, extension bootstrap route filtering, delete impact, and camelbot SQL migration plan (depends: all backend and frontend tasks)

## Blocker Log

| Date | Task | Engineer | Blocker Description | Resolution |
|------|------|----------|---------------------|------------|
