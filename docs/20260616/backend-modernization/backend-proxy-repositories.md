# Backend Progress: proxy-repositories

**Engineer:** proxy-repositories
**Scope:** Migrate the first proxy repository group to Bun and DeletePlan.

## Tasks

- [ ] Create `apps/panel/api/internal/store/proxy_repository.go` with Bun-backed chain, route rule, access path, probe, and tenant binding repository methods.
  - Commit:
- [ ] Replace chain CRUD and chain DeletePlan construction in `apps/panel/api/internal/store/mysql_chain.go`.
  - Commit:
- [ ] Replace route rule CRUD and relationship cleanup in `apps/panel/api/internal/store/mysql_route.go`.
  - Commit:
- [ ] Replace node access path CRUD and access path DeletePlan construction in `apps/panel/api/internal/store/mysql_node_access_path.go`.
  - Commit:
- [ ] Replace chain and access path delete impact SQL with DeletePlan-derived output in `apps/panel/api/internal/store/mysql_delete_impact.go`.
  - Commit:
- [ ] Update store interfaces in `apps/panel/api/internal/store/store.go` for DeletePlan-backed delete impact and execution boundaries.
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
