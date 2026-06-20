# Backend Progress: backend-panel-runtime

**Engineer:** backend-panel-runtime
**Scope:** Panel API, storage, maintenance, and automatic access-path selection.

## Tasks

- [ ] Define connection and runtime snapshot API models in `apps/panel/api/internal/domain/connection_state.go`
  - Commit:
- [ ] Persist latest node runtime snapshots and recent snapshot history in `apps/panel/api/internal/store/mysql_runtime_snapshot.go`
  - Commit:
- [ ] Expose runtime snapshot service methods in `apps/panel/api/internal/service/runtime_snapshot.go`
  - Commit:
- [ ] Serve connection diagnostics endpoints in `apps/panel/api/internal/httpapi/handler_connection_state.go`
  - Commit:
- [ ] Wire diagnostics routes in `apps/panel/api/internal/httpapi/router.go`
  - Commit:
- [ ] Apply stale snapshot maintenance in `apps/panel/api/internal/store/mysql_maintenance.go`
  - Commit:
- [ ] Implement automatic access-path candidate selection in `apps/panel/api/internal/features/proxy/service/access_path_autoselect.go`
  - Commit:
- [ ] Use automatic access-path selection from route creation in `apps/panel/api/internal/features/proxy/service/route.go`
  - Commit:
- [ ] Use automatic relay and public endpoint defaults in `apps/panel/api/internal/features/proxy/service/access_path.go`
  - Commit:
- [ ] Add panel runtime snapshot and autoselect tests in `apps/panel/api/internal/service/connection_resilience_test.go`
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
