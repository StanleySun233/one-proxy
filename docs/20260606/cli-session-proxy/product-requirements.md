# Product Requirements: CLI Session Proxy

## Problem

Users need a cross-platform command-line OneProxy client that can apply panel-managed proxy rules to selected command-line workflows without taking over the whole machine. The client must work on Windows, Linux, and macOS, coexist with other proxy tools, and support rootless environments where the user cannot modify system routing, firewall, service, or global proxy settings.

## Goal

Build a Node.js-based `onep` CLI package that provides session-scoped and process-scoped proxy usage. The CLI must support initialization, panel profiles, login, tenant selection, group selection, environment activation, command wrapping, local overrides, route testing, SSH access, status reporting, and diagnostics while storing local state under `~/.oneproxy`.

## Core Requirements

1. Provide a Node.js package with a `onep` executable that works on Windows, Linux, and macOS.
2. Ignore backward compatibility with the previous CLI command surface.
3. Store local config, state, logs, and tokens under `~/.oneproxy`.
4. Store tokens in local files with user-only permissions where the platform supports them.
5. Support interactive initialization:
   - `onep init`
   - Prompt for panel URL.
   - Test panel reachability.
   - Prompt for account and password.
   - List tenants with keyboard selection.
   - Ask whether to enable OneProxy for the current shell after setup.
6. Support panel profiles:
   - `onep profile add <name> --control-plane <url>`
   - `onep profile use <name>`
   - `onep profile list`
   - `onep profile current`
7. Support login through the active panel profile:
   - `onep login`
   - `onep logout`
8. Support tenant management:
   - `onep tenant list`
   - `onep tenant use <name-or-id>`
9. Support proxy group management:
   - `onep group list`
   - `onep group use <name-or-id>`
10. Support manual config sync:
   - `onep sync`
11. Provide process-scoped proxy execution:
   - `onep run <command...>`
   - The wrapped command and its children receive proxy environment variables.
   - Other processes and other shells are not affected.
12. Provide shell-session proxy activation:
    - `onep env`
    - `onep env on`
    - `onep env off`
    - `onep env` defaults to `on`.
13. Do not implement `onep proxy on` or `onep proxy off`.
14. `onep env on` must print shell code that the user can evaluate in the current shell.
15. `onep env off` must print shell code that restores previous proxy environment variables captured by `onep env on`.
16. The CLI must not modify system proxy settings in V1.
17. The CLI must not require root privileges in V1.
18. Provide a local helper daemon started on demand by `onep run`, `onep env`, `onep test`, `onep ssh`, or `onep doctor`.
19. The daemon must listen only on loopback by default.
20. The daemon must expose two loopback HTTP CONNECT proxy endpoints for `HTTP_PROXY` and `HTTPS_PROXY`.
21. The daemon must apply panel-managed rules and local overrides before deciding whether traffic uses OneProxy or direct connection.
22. The daemon may exit after an idle timeout when no client is using it.
23. Support local overrides:
    - `onep override list`
    - `onep override direct add <host>`
    - `onep override proxy add <host>`
    - `onep override remove <host>`
    - `onep override clear`
24. Support route explanation:
    - `onep route <url-or-host>`
    - The output must show direct or proxy mode, matched source, active tenant, active group, and topology when proxied.
25. Support active probing:
    - `onep test <url-or-host>`
    - The output must include route explanation and protocol probe results where supported.
26. Support SSH through OneProxy:
    - `onep ssh <host>`
    - `onep ssh <user>@<host> [-p <port>]`
    - The command should route SSH traffic through OneProxy when rules require proxying.
27. Support diagnostics:
    - `onep doctor`
    - Diagnostics must check config, token readability, control-plane health, bootstrap sync, daemon status, local ports, route calculation, and entry node reachability.
28. Support machine-readable output for automation:
    - `--json` for `status`, `route`, `test`, and `doctor`.

## Environment Behavior

`onep run <command...>` must inject at least these variables into the child process:

```text
HTTP_PROXY=http://127.0.0.1:<http-port>
HTTPS_PROXY=http://127.0.0.1:<https-port>
ALL_PROXY=http://127.0.0.1:<http-port>
NO_PROXY=localhost,127.0.0.1,::1
ONEPROXY_ACTIVE=1
```

The daemon must not use fixed default proxy ports. Before selecting proxy ports, it must scan loopback ports, exclude occupied ports and common system ports, record the available candidate ports, and randomly choose one consecutive two-port pair. The first selected port is `httpPort`; the second selected port is `httpsPort`. Both ports provide HTTP proxy and CONNECT behavior; `httpsPort` is not a TLS server.

`onep env on` must print shell code for the current shell family. The printed code must preserve existing proxy variables before replacing them. `onep env off` must restore preserved values when they exist and unset OneProxy variables afterward.

## Local Storage

The CLI must use this directory layout:

```text
~/.oneproxy/profiles.json
~/.oneproxy/profiles/<profile>/config.json
~/.oneproxy/profiles/<profile>/state.json
~/.oneproxy/profiles/<profile>/tokens.json
~/.oneproxy/profiles/<profile>/daemon.json
~/.oneproxy/profiles/<profile>/onep.log
```

`profiles.json` stores the active profile and profile-to-panel URL mapping. Each profile directory stores its own config, state, tokens, daemon metadata, and logs. `config.json` stores profile name, control-plane URL, active tenant, active group, and overrides. `state.json` stores bootstrap cache, policy revision, fetched time, and route groups. `tokens.json` stores access token, refresh token, proxy token, and token expiry timestamps. `daemon.json` stores daemon runtime metadata such as PID, port bindings, and last heartbeat.

## Non-Goals

1. Do not modify system proxy settings in V1.
2. Do not implement transparent proxying, TUN mode, firewall rules, or route-table changes in V1.
3. Do not implement persistent global proxy state.
4. Do not require root or administrator privileges.
5. Do not make `proxy on/off` command aliases in V1.
6. Do not store passwords in local files.

## Acceptance Criteria

1. `onep init` creates or updates a panel profile, verifies panel reachability, logs in, selects a tenant, syncs bootstrap state, and offers shell activation.
2. `onep profile add/use/list/current` manages panel URL profiles.
3. `onep login` stores session tokens under the active profile.
4. `onep tenant list/use` works from stored login state and switches the active tenant.
5. `onep group list/use` selects a panel-provided proxy group.
6. `onep env` and `onep env on` print activation shell code without changing other shells.
7. `onep env off` prints restoration shell code.
8. `onep run <command...>` runs only that command with OneProxy proxy environment variables.
9. `onep status` reports account, control plane, tenant, group, daemon, local ports, policy revision, token expiry, and override counts.
10. `onep route <url>` explains rule matching.
11. `onep test <url>` reports route and probe results.
12. `onep ssh <target>` can open SSH through a proxied route.
13. `onep doctor` reports actionable diagnostics and exits non-zero only for failed checks.
14. The implementation runs without root privileges on supported platforms.
15. Daemon startup records the scanned unused proxy port candidates and the randomly selected consecutive `httpPort`/`httpsPort` pair.
