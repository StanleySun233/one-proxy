# Backend Progress: backend-node-runtime

**Engineer:** backend-node-runtime
**Scope:** Node runtime snapshot reporting, upstream reachability, relay probe phases, and local diagnostics.

## Tasks

- [ ] Define node runtime snapshot collector in `apps/node/api/internal/runtime/snapshot.go`
  - Commit:
- [ ] Include runtime snapshot payloads in heartbeat loop in `apps/node/api/internal/heartbeat/loop.go`
  - Commit:
- [ ] Report upstream reachability and parent relay state in `apps/node/api/internal/controlplane/client.go`
  - Commit:
- [ ] Expose local runtime snapshot in `apps/node/api/internal/localconsole/handler.go`
  - Commit:
- [ ] Include relay probe phase and error codes in `apps/node/api/internal/controlrelay/probe.go`
  - Commit:
- [ ] Add node runtime snapshot tests in `apps/node/api/internal/runtime/snapshot_test.go`
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
