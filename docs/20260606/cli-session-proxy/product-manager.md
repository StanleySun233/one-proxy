# Product Progress: product-manager

**Engineer:** product-manager
**Scope:** Verify the delivered CLI behavior against the product requirements.

## Product Acceptance Report

**Status:** GAPS FOUND - not ready for delivery

**Verification performed:**

- Reviewed `product-requirements.md`, `dev-roadmap.md`, and `api-contract.md`.
- Reviewed `apps/cli/src/*`, daemon runtime files, and `apps/cli/test/*.mjs`.
- Ran `node --test apps/cli/test/*.mjs`.

**Test result:**

- Pass: 12
- Fail: 0

## Requirements Met

- CLI command surface includes the required V1 commands and excludes `proxy on/off`.
- Local storage uses `~/.oneproxy` paths for `config.json`, `state.json`, `tokens.json`, `daemon.json`, and `onep.log`.
- Token/config/state writes use user-only file modes on non-Windows platforms.
- `onep env` defaults to `env on`, and `env off` prints restoration shell code.
- `onep run` starts/reuses the daemon and injects process-scoped proxy environment variables into the child process.
- Daemon bindings use loopback host by default.
- Daemon exposes separate HTTP proxy listeners for `HTTP_PROXY` and `HTTPS_PROXY`, and both support CONNECT behavior.
- Auto proxy port selection scans loopback availability, excludes common ports, records candidates, and selects a random consecutive pair.
- Local overrides are normalized to lowercase, and direct overrides take precedence over proxy overrides.
- `route`, `test`, `status`, and `doctor` support `--json`.
- SSH command planning supports direct and proxied route modes.

## Gaps Found

1. `onep doctor` does not start the daemon on demand.
   - Requirement: commands including `onep doctor` must start the local helper daemon on demand.
   - Evidence: `runDoctor()` only reads daemon metadata and probes an existing daemon, then fails `daemon_status` when absent; it never calls `ensureDaemon()` (`apps/cli/src/doctor.ts:98`).
   - Impact: fresh installations fail doctor even though the required behavior is to start/reuse the helper daemon.

2. Doctor `token_refresh` does not verify token refresh.
   - Requirement: diagnostics must check token refresh.
   - Evidence: `tokenRefreshCheck()` only checks whether `refreshTokenExpiresAt` is present and in the future (`apps/cli/src/doctor.ts:133`).
   - Impact: an invalid refresh token can pass diagnostics if the timestamp is future-dated.

3. Doctor `proxy_token_acceptance` does not verify proxy-token acceptance.
   - Requirement: diagnostics must check proxy-token acceptance.
   - Evidence: `proxyTokenAcceptanceCheck()` only checks token presence (`apps/cli/src/doctor.ts:116`).
   - Impact: a rejected proxy token can pass diagnostics.

4. Fish shell activation does not correctly preserve previous proxy variables.
   - Requirement: `env on` must preserve previous proxy variables for supported shell families, including Fish.
   - Evidence: Fish activation emits `$$key` instead of the concrete variable value for each proxy variable (`apps/cli/src/session-env.ts:103`).
   - Impact: Fish users may lose previous proxy environment values on `env off`.

5. `tenant use` always clears the active group.
   - Contract: `tenant use` clears `config.activeGroupId` only when the current group does not belong to the selected tenant.
   - Evidence: `tenantUse()` writes `activeGroupId: undefined` unconditionally (`apps/cli/src/control-plane.ts:262`).
   - Impact: switching to the same tenant or a tenant that still owns the selected group unnecessarily loses group selection.

6. `status --json` includes a non-contract `localPorts.allProxy` field.
   - Contract: `localPorts` contains `http`, `https`, and `ipc`.
   - Evidence: `statusCommand()` adds `allProxy` under `localPorts` (`apps/cli/src/commands.ts:272`).
   - Impact: strict automation clients expecting the published schema may reject the response.

## Tasks

- [x] verify delivered behavior against product-requirements.md after tests pass
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
