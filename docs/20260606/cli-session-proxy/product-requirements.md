# Product Requirements: CLI Session Proxy

## Problem

Users need a cross-platform command-line OneProxy client that can apply panel-managed proxy rules to selected command-line workflows without taking over the whole machine. The client must work on Windows, Linux, and macOS, coexist with other proxy tools, and support rootless environments where the user cannot modify system routing, firewall, service, or global proxy settings.

## Goal

Build a Node.js-based `onep` CLI package that provides session-scoped and process-scoped proxy usage. The CLI must support login, tenant selection, group selection, environment activation, command wrapping, local overrides, route testing, SSH access, status reporting, and diagnostics while storing local state under `~/.oneproxy`.

## Core Requirements

1. Provide a Node.js package with a `onep` executable that works on Windows, Linux, and macOS.
2. Ignore backward compatibility with the previous CLI command surface.
3. Store local config, state, logs, and tokens under `~/.oneproxy`.
4. Store tokens in local files with user-only permissions where the platform supports them.
5. Support login through the control plane:
   - `onep login`
   - `onep logout`
6. Support tenant management without profiles:
   - `onep tenant list`
   - `onep tenant use <name-or-id>`
7. Support proxy group management:
   - `onep group list`
   - `onep group use <name-or-id>`
8. Support manual config sync:
   - `onep sync`
9. Provide process-scoped proxy execution:
   - `onep run <command...>`
   - The wrapped command and its children receive proxy environment variables.
   - Other processes and other shells are not affected.
10. Provide shell-session proxy activation:
    - `onep env`
    - `onep env on`
    - `onep env off`
    - `onep env` defaults to `on`.
11. Do not implement `onep proxy on` or `onep proxy off`.
12. `onep env on` must print shell code that the user can evaluate in the current shell.
13. `onep env off` must print shell code that restores previous proxy environment variables captured by `onep env on`.
14. The CLI must not modify system proxy settings in V1.
15. The CLI must not require root privileges in V1.
16. Provide a local helper daemon started on demand by `onep run`, `onep env`, `onep test`, `onep ssh`, or `onep doctor`.
17. The daemon must listen only on loopback by default.
18. The daemon must expose local HTTP CONNECT and SOCKS5 proxy endpoints.
19. The daemon must apply panel-managed rules and local overrides before deciding whether traffic uses OneProxy or direct connection.
20. The daemon may exit after an idle timeout when no client is using it.
21. Support local overrides:
    - `onep override list`
    - `onep override direct add <host>`
    - `onep override proxy add <host>`
    - `onep override remove <host>`
    - `onep override clear`
22. Support route explanation:
    - `onep route <url-or-host>`
    - The output must show direct or proxy mode, matched source, active tenant, active group, and topology when proxied.
23. Support active probing:
    - `onep test <url-or-host>`
    - The output must include route explanation and protocol probe results where supported.
24. Support SSH through OneProxy:
    - `onep ssh <host>`
    - `onep ssh <user>@<host> [-p <port>]`
    - The command should route SSH traffic through OneProxy when rules require proxying.
25. Support diagnostics:
    - `onep doctor`
    - Diagnostics must check config, token readability, control-plane health, token refresh, bootstrap sync, daemon status, local ports, route calculation, entry node reachability, and proxy-token acceptance.
26. Support machine-readable output for automation:
    - `--json` for `status`, `route`, `test`, and `doctor`.

## Environment Behavior

`onep run <command...>` must inject at least these variables into the child process:

```text
HTTP_PROXY=http://127.0.0.1:<http-port>
HTTPS_PROXY=http://127.0.0.1:<http-port>
ALL_PROXY=socks5://127.0.0.1:<socks-port>
NO_PROXY=localhost,127.0.0.1,::1
ONEPROXY_ACTIVE=1
```

`onep env on` must print shell code for the current shell family. The printed code must preserve existing proxy variables before replacing them. `onep env off` must restore preserved values when they exist and unset OneProxy variables afterward.

## Local Storage

The CLI must use this directory layout:

```text
~/.oneproxy/config.json
~/.oneproxy/state.json
~/.oneproxy/tokens.json
~/.oneproxy/daemon.json
~/.oneproxy/onep.log
```

`config.json` stores control-plane URL, active tenant, active group, local ports, and overrides. `state.json` stores bootstrap cache, policy revision, fetched time, and route groups. `tokens.json` stores access token, refresh token, proxy token, and token expiry timestamps. `daemon.json` stores daemon runtime metadata such as PID, port bindings, and last heartbeat.

## Non-Goals

1. Do not modify system proxy settings in V1.
2. Do not implement transparent proxying, TUN mode, firewall rules, or route-table changes in V1.
3. Do not implement persistent global proxy state.
4. Do not implement multiple profiles in V1.
5. Do not require root or administrator privileges.
6. Do not make `proxy on/off` command aliases in V1.
7. Do not store passwords in local files.

## Acceptance Criteria

1. `onep login` stores session tokens under `~/.oneproxy`.
2. `onep tenant list/use` works from stored login state and switches the active tenant without profile support.
3. `onep group list/use` selects a panel-provided proxy group.
4. `onep env` and `onep env on` print activation shell code without changing other shells.
5. `onep env off` prints restoration shell code.
6. `onep run <command...>` runs only that command with OneProxy proxy environment variables.
7. `onep status` reports account, control plane, tenant, group, daemon, local ports, policy revision, token expiry, and override counts.
8. `onep route <url>` explains rule matching.
9. `onep test <url>` reports route and probe results.
10. `onep ssh <target>` can open SSH through a proxied route.
11. `onep doctor` reports actionable diagnostics and exits non-zero only for failed checks.
12. The implementation runs without root privileges on supported platforms.
