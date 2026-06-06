# API Contract: CLI Session Proxy

## Scope

This contract defines the V1 command surface, local file formats, daemon metadata, daemon IPC, JSON output shapes, error format, shell environment output, `onep run`, and `onep ssh` behavior for the OneProxy CLI Session Proxy.

The CLI must not modify system proxy settings, routing tables, firewalls, services, or global shell state. All proxy activation is process-scoped or shell-session-scoped through printed shell code.

## Common CLI Rules

- Executable name: `onep`
- Supported platforms: Windows, Linux, macOS
- Local data root: `~/.oneproxy`
- Machine-readable output flag: `--json` for `status`, `route`, `test`, and `doctor`
- Default human output: concise text intended for terminals
- Default daemon binding: loopback only
- Exit code `0`: command completed successfully
- Exit code `1`: command failed because of user-visible runtime or validation error
- Exit code `2`: command syntax error
- Exit code `3`: doctor completed with one or more failed checks

Commands that require route calculation or local proxy endpoints must ensure the daemon is running on demand: `run`, `env`, `route`, `test`, `ssh`, and `doctor`.

## CLI Command Contract

### `onep login`

Starts control-plane login and stores tokens under `~/.oneproxy/tokens.json`.

Required behavior:

- Creates `~/.oneproxy` when missing.
- Stores token files with user-only permissions where supported.
- Selects a tenant and group only when the control plane returns an unambiguous default.
- Prints the authenticated account and next required action when tenant or group selection is needed.

### `onep logout`

Clears local tokens and leaves non-secret config, state, daemon metadata, logs, tenant, group, and overrides intact.

### `onep tenant list`

Lists tenants available to the authenticated account.

### `onep tenant use <name-or-id>`

Sets `config.activeTenantId` and clears `config.activeGroupId` when the current group does not belong to the selected tenant.

### `onep group list`

Lists proxy groups for the active tenant.

### `onep group use <name-or-id>`

Sets `config.activeGroupId` for the active tenant.

### `onep sync`

Refreshes bootstrap state and route groups from the control plane, writing `state.json`.

### `onep status [--json]`

Reports account, control plane, active tenant, active group, daemon, local ports, policy revision, token expiry, and override counts.

### `onep env`

Alias for `onep env on`.

### `onep env on`

Starts the daemon if needed and prints shell code for the detected shell family. The command itself must not modify the parent shell.

### `onep env off`

Prints shell code that restores proxy variables captured by `onep env on` and unsets OneProxy session markers.

### `onep run <command...>`

Starts the daemon if needed, injects proxy environment variables into the child process, and exits with the child process exit code.

### `onep override list`

Lists local direct and proxy host overrides.

### `onep override direct add <host>`

Adds a host override that forces direct routing.

### `onep override proxy add <host>`

Adds a host override that forces OneProxy routing.

### `onep override remove <host>`

Removes the host from both direct and proxy override lists.

### `onep override clear`

Clears all local overrides.

### `onep route <url-or-host> [--json]`

Explains the route decision for a URL or host.

### `onep test <url-or-host> [--json]`

Returns route explanation plus supported protocol probes.

### `onep ssh <host>`

Routes SSH traffic to `<host>` through OneProxy when route rules require proxying.

### `onep ssh <user>@<host> [-p <port>]`

Routes SSH traffic for the supplied target and port. The default SSH port is `22`.

### `onep doctor [--json]`

Runs diagnostics for config, token readability, control-plane health, token refresh, bootstrap sync, daemon status, local ports, route calculation, entry node reachability, and proxy-token acceptance.

## Local Storage Schemas

### `~/.oneproxy/config.json`

```json
{
  "schemaVersion": 1,
  "controlPlaneUrl": "https://control.example.com",
  "activeTenantId": "tenant_123",
  "activeGroupId": "group_123",
  "localPorts": {
    "http": 0,
    "https": 0,
    "ipc": 0
  },
  "overrides": {
    "direct": ["localhost"],
    "proxy": ["example.com"]
  }
}
```

Rules:

- `localPorts.http = 0` and `localPorts.https = 0` means auto-select a random available consecutive two-port pair on loopback.
- When both `http` and `https` are configured, they must be consecutive. If they are not consecutive, daemon startup fails with `INVALID_PORT_PAIR`.
- Auto-selection must exclude occupied ports and common system ports before random choice.
- Override hosts are stored lowercase.
- `overrides.direct` has precedence over `overrides.proxy` when the same host appears in both lists.

### `~/.oneproxy/state.json`

```json
{
  "schemaVersion": 1,
  "bootstrap": {
    "tenantId": "tenant_123",
    "groupId": "group_123",
    "entryNodes": [
      {
        "id": "entry_1",
        "host": "edge.example.com",
        "port": 443,
        "protocol": "connect"
      }
    ]
  },
  "policyRevision": "rev_20260606_001",
  "fetchedAt": "2026-06-06T06:00:00.000Z",
  "routeGroups": [
    {
      "id": "group_123",
      "tenantId": "tenant_123",
      "name": "Default",
      "rules": [
        {
          "id": "rule_1",
          "type": "domain",
          "pattern": "example.com",
          "mode": "proxy"
        }
      ]
    }
  ]
}
```

Allowed rule `type` values: `domain`, `suffix`, `cidr`, `wildcard`.

Allowed rule `mode` values: `direct`, `proxy`.

### `~/.oneproxy/tokens.json`

```json
{
  "schemaVersion": 1,
  "account": {
    "id": "user_123",
    "email": "user@example.com"
  },
  "accessToken": "opaque-access-token",
  "refreshToken": "opaque-refresh-token",
  "proxyToken": "opaque-proxy-token",
  "accessTokenExpiresAt": "2026-06-06T07:00:00.000Z",
  "refreshTokenExpiresAt": "2026-07-06T06:00:00.000Z",
  "proxyTokenExpiresAt": "2026-06-06T07:00:00.000Z"
}
```

The CLI must never store passwords locally.

### `~/.oneproxy/daemon.json`

```json
{
  "schemaVersion": 1,
  "pid": 12345,
  "startedAt": "2026-06-06T06:00:00.000Z",
  "lastHeartbeatAt": "2026-06-06T06:10:00.000Z",
  "controlPlaneUrl": "https://control.example.com",
  "tenantId": "tenant_123",
  "groupId": "group_123",
  "policyRevision": "rev_20260606_001",
  "bindings": {
    "host": "127.0.0.1",
    "httpPort": 21432,
    "httpsPort": 21433,
    "ipcPort": 18082
  },
  "portSelection": {
    "candidatePorts": [21430, 21431, 21432, 21433, 21434],
    "selectedPair": [21432, 21433],
    "excludedCommonPorts": [20, 21, 22, 25, 53, 80, 110, 143, 443, 3306, 5432, 6379, 8080]
  },
  "idleTimeoutSeconds": 300
}
```

### `~/.oneproxy/onep.log`

Line-oriented local log file. Logs must not contain access tokens, refresh tokens, proxy tokens, or passwords.

## Daemon Metadata and IPC Contracts

The daemon exposes two local HTTP proxy endpoints plus an IPC HTTP endpoint on loopback. The `httpPort` endpoint is used by `HTTP_PROXY`; the `httpsPort` endpoint is used by `HTTPS_PROXY`. Both proxy endpoints support HTTP requests and CONNECT tunnels. `httpsPort` is not a TLS listener.

Before choosing proxy ports, the daemon must scan loopback ports, exclude occupied ports and common system ports, record available candidate ports, and randomly select one consecutive two-port pair. Commands that need to show this information must use daemon metadata or `status --json`; `onep env` must keep stdout reserved for shell code and may report candidate information on stderr or in logs only.

### `GET /v1/health`

Response:

```json
{
  "ok": true,
  "pid": 12345,
  "startedAt": "2026-06-06T06:00:00.000Z",
  "lastHeartbeatAt": "2026-06-06T06:10:00.000Z",
  "bindings": {
    "host": "127.0.0.1",
    "httpPort": 21432,
    "httpsPort": 21433,
    "ipcPort": 18082
  },
  "portSelection": {
    "candidatePorts": [21430, 21431, 21432, 21433, 21434],
    "selectedPair": [21432, 21433],
    "excludedCommonPorts": [20, 21, 22, 25, 53, 80, 110, 143, 443, 3306, 5432, 6379, 8080]
  },
  "policyRevision": "rev_20260606_001"
}
```

### `POST /v1/route`

Request:

```json
{
  "target": "https://example.com/path",
  "protocol": "https"
}
```

Response: `RouteResult`

### `POST /v1/probe`

Request:

```json
{
  "target": "https://example.com/path"
}
```

Response: `TestResult`

### `POST /v1/shutdown-if-idle`

Request:

```json
{
  "clientId": "cli-123"
}
```

Response:

```json
{
  "accepted": true
}
```

## JSON Schemas

### Shared Types

`RouteMode` is `"direct"` or `"proxy"`.

`CheckStatus` is `"pass"`, `"warn"`, or `"fail"`.

### `ErrorResult`

```json
{
  "error": {
    "code": "TOKEN_EXPIRED",
    "message": "Token expired. Run onep login.",
    "details": {
      "command": "status"
    }
  }
}
```

Rules:

- `code` is stable and uppercase snake case.
- `message` is human-readable and actionable.
- `details` is optional and must not contain secrets.

Common error codes:

- `AUTH_REQUIRED`
- `TOKEN_EXPIRED`
- `TENANT_REQUIRED`
- `GROUP_REQUIRED`
- `DAEMON_UNAVAILABLE`
- `CONTROL_PLANE_UNAVAILABLE`
- `INVALID_TARGET`
- `COMMAND_NOT_FOUND`
- `ROUTE_UNAVAILABLE`
- `PROXY_TOKEN_REJECTED`

### `StatusResult`

```json
{
  "account": {
    "id": "user_123",
    "email": "user@example.com"
  },
  "controlPlane": {
    "url": "https://control.example.com",
    "reachable": true
  },
  "tenant": {
    "id": "tenant_123",
    "name": "Acme"
  },
  "group": {
    "id": "group_123",
    "name": "Default"
  },
  "daemon": {
    "running": true,
    "pid": 12345,
    "startedAt": "2026-06-06T06:00:00.000Z",
    "lastHeartbeatAt": "2026-06-06T06:10:00.000Z"
  },
  "localPorts": {
    "http": 21432,
    "https": 21433,
    "ipc": 18082
  },
  "portSelection": {
    "candidatePorts": [21430, 21431, 21432, 21433, 21434],
    "selectedPair": [21432, 21433]
  },
  "policyRevision": "rev_20260606_001",
  "tokens": {
    "accessTokenExpiresAt": "2026-06-06T07:00:00.000Z",
    "refreshTokenExpiresAt": "2026-07-06T06:00:00.000Z",
    "proxyTokenExpiresAt": "2026-06-06T07:00:00.000Z"
  },
  "overrides": {
    "directCount": 1,
    "proxyCount": 1
  }
}
```

### `RouteResult`

```json
{
  "target": "https://example.com/path",
  "host": "example.com",
  "port": 443,
  "mode": "proxy",
  "matched": {
    "source": "policy",
    "ruleId": "rule_1",
    "ruleType": "domain",
    "pattern": "example.com"
  },
  "tenant": {
    "id": "tenant_123",
    "name": "Acme"
  },
  "group": {
    "id": "group_123",
    "name": "Default"
  },
  "topology": {
    "entryNodeId": "entry_1",
    "entryHost": "edge.example.com",
    "entryPort": 443,
    "protocol": "connect"
  }
}
```

Allowed `matched.source` values: `local_override_direct`, `local_override_proxy`, `policy`, `default_direct`.

`topology` is `null` when `mode` is `direct`.

### `TestResult`

```json
{
  "route": {
    "target": "https://example.com/path",
    "host": "example.com",
    "port": 443,
    "mode": "proxy",
    "matched": {
      "source": "policy",
      "ruleId": "rule_1",
      "ruleType": "domain",
      "pattern": "example.com"
    },
    "tenant": {
      "id": "tenant_123",
      "name": "Acme"
    },
    "group": {
      "id": "group_123",
      "name": "Default"
    },
    "topology": {
      "entryNodeId": "entry_1",
      "entryHost": "edge.example.com",
      "entryPort": 443,
      "protocol": "connect"
    }
  },
  "probes": [
    {
      "name": "dns",
      "status": "pass",
      "latencyMs": 12,
      "message": "Resolved example.com"
    },
    {
      "name": "connect",
      "status": "pass",
      "latencyMs": 48,
      "message": "Connected through entry node"
    }
  ]
}
```

Probe names: `dns`, `direct_connect`, `proxy_connect`, `http_proxy`, `https_proxy`, `ssh`.

### `DoctorResult`

```json
{
  "summary": {
    "status": "fail",
    "passed": 8,
    "warned": 1,
    "failed": 1
  },
  "checks": [
    {
      "name": "config",
      "status": "pass",
      "message": "Config file is readable"
    },
    {
      "name": "proxy_token_acceptance",
      "status": "fail",
      "message": "Proxy token was rejected by entry node",
      "action": "Run onep login, then onep sync"
    }
  ]
}
```

Required check names:

- `config`
- `token_readability`
- `control_plane_health`
- `token_refresh`
- `bootstrap_sync`
- `daemon_status`
- `local_ports`
- `route_calculation`
- `entry_node_reachability`
- `proxy_token_acceptance`

`doctor --json` exits `3` when any check has `status = "fail"`.

## Shell Env Output Contract

`onep env on` prints shell code for the current shell family. Supported families:

- POSIX shells: `sh`, `bash`, `zsh`
- Fish
- PowerShell
- Windows CMD

Activation must preserve previous values for:

- `HTTP_PROXY`
- `HTTPS_PROXY`
- `ALL_PROXY`
- `NO_PROXY`
- lowercase equivalents: `http_proxy`, `https_proxy`, `all_proxy`, `no_proxy`

Activation must set:

```text
HTTP_PROXY=http://127.0.0.1:<http-port>
HTTPS_PROXY=http://127.0.0.1:<https-port>
ALL_PROXY=http://127.0.0.1:<http-port>
NO_PROXY=localhost,127.0.0.1,::1
ONEPROXY_ACTIVE=1
ONEPROXY_HTTP_PORT=<http-port>
ONEPROXY_HTTPS_PORT=<https-port>
```

Preserved values must be stored in shell variables prefixed with `ONEPROXY_PREV_`.

`onep env off` restores preserved values when present. When a preserved marker says the original variable was unset, `env off` unsets that variable. `env off` must unset all `ONEPROXY_*` session variables after restoration.

## `onep run` Behavior

`onep run <command...>`:

- Starts or reuses the loopback daemon.
- Runs exactly one child command.
- Injects the same proxy variables as `env on` into the child environment.
- Does not mutate the parent process environment.
- Streams child stdout and stderr without changing content.
- Forwards stdin to the child.
- Exits with the child exit code.
- Returns `COMMAND_NOT_FOUND` with exit code `1` when the executable cannot be found.

Child processes inherit:

```text
HTTP_PROXY=http://127.0.0.1:<http-port>
HTTPS_PROXY=http://127.0.0.1:<https-port>
ALL_PROXY=http://127.0.0.1:<http-port>
NO_PROXY=localhost,127.0.0.1,::1
ONEPROXY_ACTIVE=1
ONEPROXY_HTTP_PORT=<http-port>
ONEPROXY_HTTPS_PORT=<https-port>
```

## `onep ssh` Behavior

`onep ssh` accepts:

- `onep ssh <host>`
- `onep ssh <user>@<host>`
- `onep ssh <host> -p <port>`
- `onep ssh <user>@<host> -p <port>`

Behavior:

- Default port is `22`.
- The daemon is started or reused before route calculation.
- The route target is `ssh://<host>:<port>`.
- If route mode is `direct`, the CLI executes the platform `ssh` command normally.
- If route mode is `proxy`, the CLI executes `ssh` with a proxy command that connects through the local OneProxy HTTP CONNECT endpoint.
- The CLI exits with the `ssh` process exit code.
- The CLI must not store SSH passwords or private keys.

When proxied, the implementation contract is equivalent to:

```text
ssh -o ProxyCommand=<oneproxy-connect-helper> [-p <port>] [user@]host
```

The helper connects to `127.0.0.1:<http-port>` and issues HTTP CONNECT for `<host>:<port>`.
