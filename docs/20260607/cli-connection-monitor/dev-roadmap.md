# Dev Roadmap: CLI Connection Monitor

**Date:** 20260607
**Status:** completed
**Product document:** docs/20260607/cli-connection-monitor/product-requirements.md

## Summary

`onep monitor` is a connection audit command for launched applications and their child processes. The command should record real process connection information for whitelist discovery rather than relying on HTTP proxy environment variables.

## Team

| Role | Agent Name | Progress File |
|------|------------|---------------|
| Backend | cli-monitor | [->](./backend-cli-monitor.md) |

## Tasks

### API Contract
- [x] cli-monitor: write api-contract.md covering monitor log event shape

### Backend Tasks
- [x] cli-monitor: implement cross-platform connection monitor in apps/cli/src/monitor.ts (depends: api-contract.md)
- [x] cli-monitor: update monitor tests in apps/cli/test/daemon.test.mjs (depends: apps/cli/src/monitor.ts)

### Testing
- [x] test-engineer: run CLI tests and TypeScript check (depends: backend tasks)

## Blocker Log

| Date | Task | Engineer | Blocker Description | Resolution |
|------|------|----------|---------------------|------------|
