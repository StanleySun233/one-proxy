# API Contract: Node Local Console

**Status:** contract ready
**Scope:** first-phase node-local console APIs, node web routes, and extension panel API path migration.

## Principles

- The node-local console is read-only. This contract does not define configuration editing, restart, reload, certificate renewal, or other state-changing management endpoints.
- Node-local API routes live under `/api/local/*`.
- Web page routes are direct browser routes served by the node HTTP listener.
- Existing node proxy/runtime traffic remains available for non-console traffic.
- Extension panel API paths migrate from `/api/v1/*` to `/api/*` with no old-path compatibility branches.
- Panel-side API changes are deferred until user approval and are listed separately below.

## Common API Format

All `/api/local/*` responses use JSON.

Success envelope:

```json
{
  "ok": true,
  "data": {}
}
```

Error envelope:

```json
{
  "ok": false,
  "error": {
    "code": "unauthorized",
    "message": "Unauthorized"
  }
}
```

Common status codes:

| Status | Meaning |
|--------|---------|
| 200 | Request succeeded |
| 400 | Invalid query or request payload |
| 401 | Missing or invalid node-console session |
| 403 | Authenticated account is not allowed to view this node |
| 404 | Route or requested record not found |
| 405 | Method not allowed |
| 502 | Control plane request failed |
| 503 | Required node runtime data is unavailable |

## Node Web Routes

These routes serve the node console web app shell or static assets. Browser navigation should work on every listed route.

| Method | Path | Behavior |
|--------|------|----------|
| GET | `/` | Redirect to `/overview` when a node-console session exists; otherwise redirect to `/login`. |
| GET | `/login` | Serve the login view. |
| GET | `/overview` | Serve the overview view shell. |
| GET | `/health` | Serve the health view shell. |
| GET | `/audit` | Serve the audit view shell. |
| GET | `/policy` | Serve the policy view shell. |
| GET | `/diagnostics` | Serve the diagnostics view shell. |

Only these direct console routes are reserved. Non-console requests continue through the existing node runtime/proxy handling path.

## Node-Local Authentication

### `POST /api/local/login`

First-phase behavior: the node forwards the submitted credentials to its configured upstream control plane URL. For middle nodes this must be the upstream node/control-plane path already configured for the node, not a browser-supplied panel URL. If the control plane authenticates the account, the node then validates manage access for its own node ID before creating a local console session cookie for the browser.

Request:

```json
{
  "username": "operator@example.com",
  "password": "secret"
}
```

Response:

```json
{
  "ok": true,
  "data": {
    "user": {
      "id": "user-123",
      "name": "Operator",
      "email": "operator@example.com"
    },
    "authorization": {
      "mode": "panel_manage_access",
      "manageAccessChecked": true
    }
  }
}
```

Session cookie:

| Name | Attributes |
|------|------------|
| `one_proxy_node_console` | `HttpOnly`, `SameSite=Lax`, `Path=/` |

### `POST /api/local/logout`

Clears the node-console session cookie.

Response:

```json
{
  "ok": true,
  "data": {
    "loggedOut": true
  }
}
```

### `GET /api/local/session`

Returns the current node-console session state.

Response:

```json
{
  "ok": true,
  "data": {
    "authenticated": true,
    "user": {
      "id": "user-123",
      "name": "Operator",
      "email": "operator@example.com"
    },
    "authorization": {
      "mode": "panel_manage_access",
      "manageAccessChecked": true
    }
  }
}
```

## Read-Only Node-Local APIs

All endpoints in this section require a valid node-console session.

### `GET /api/local/status`

Returns node identity, listener state, control-plane binding, and runtime counters.

Response:

```json
{
  "ok": true,
  "data": {
    "node": {
      "id": "node-001",
      "name": "edge-sg-1",
      "role": "child",
      "version": "1.0.0",
      "startedAt": "2026-06-04T08:00:00Z",
      "uptimeSeconds": 3600
    },
    "controlPlane": {
      "url": "https://panel.example.com",
      "bound": true,
      "lastSyncAt": "2026-06-04T08:55:00Z"
    },
    "listeners": {
      "http": "0.0.0.0:8080",
      "https": "",
      "tcpAccess": "",
      "udpAccess": ""
    },
    "runtime": {
      "activeProxySessions": 12,
      "activeTunnels": 2,
      "policyRevision": "rev-42"
    }
  }
}
```

### `GET /api/local/health`

Returns current node health checks.

Response:

```json
{
  "ok": true,
  "data": {
    "status": "healthy",
    "checkedAt": "2026-06-04T09:00:00Z",
    "checks": [
      {
        "name": "control_plane",
        "status": "healthy",
        "message": "connected",
        "lastCheckedAt": "2026-06-04T09:00:00Z"
      },
      {
        "name": "policy_store",
        "status": "healthy",
        "message": "revision rev-42 loaded",
        "lastCheckedAt": "2026-06-04T09:00:00Z"
      }
    ]
  }
}
```

`status` values: `healthy`, `degraded`, `unhealthy`.

### `GET /api/local/audit`

Returns recent read-only audit entries from the current node. Query parameters are optional.

| Query | Type | Meaning |
|-------|------|---------|
| `limit` | integer | Maximum records to return. Default `50`, maximum `200`. |
| `cursor` | string | Cursor from a previous response. |
| `type` | string | Optional event type filter. |

Response:

```json
{
  "ok": true,
  "data": {
    "items": [
      {
        "id": "audit-001",
        "timestamp": "2026-06-04T09:00:00Z",
        "type": "proxy_session_started",
        "severity": "info",
        "subject": "node-001",
        "message": "Proxy session started",
        "metadata": {
          "sessionId": "session-123",
          "routeId": "route-456"
        }
      }
    ],
    "nextCursor": "cursor-002"
  }
}
```

### `GET /api/local/policy`

Returns the current local policy snapshot loaded by the node.

Response:

```json
{
  "ok": true,
  "data": {
    "revision": "rev-42",
    "loadedAt": "2026-06-04T08:55:00Z",
    "source": "control_plane",
    "nodes": [
      {
        "id": "node-001",
        "name": "edge-sg-1",
        "role": "child"
      }
    ],
    "routes": [
      {
        "id": "route-456",
        "name": "default-web",
        "action": "direct",
        "enabled": true,
        "match": {
          "host": "example.com",
          "protocol": "http"
        }
      }
    ]
  }
}
```

### `GET /api/local/diagnostics`

Returns diagnostic information for operator inspection. The response must not include secrets, bearer tokens, session cookies, passwords, or private keys.

Response:

```json
{
  "ok": true,
  "data": {
    "generatedAt": "2026-06-04T09:00:00Z",
    "environment": {
      "version": "1.0.0",
      "goVersion": "go1.23.12",
      "os": "linux",
      "arch": "amd64"
    },
    "network": {
      "localAddresses": ["192.0.2.10"],
      "natType": "unknown"
    },
    "controlPlane": {
      "configured": true,
      "reachable": true,
      "lastError": ""
    },
    "recentErrors": [
      {
        "timestamp": "2026-06-04T08:58:00Z",
        "component": "control_plane",
        "message": "temporary timeout"
      }
    ]
  }
}
```

## Extension Panel API Path Migration

The extension connects to the panel by default. Update panel-facing extension calls from `/api/v1/*` to `/api/*` and do not keep old-path fallback branches.

| Current path | New path |
|--------------|----------|
| `/api/v1/auth/login` | `/api/auth/login` |
| `/api/v1/auth/refresh` | `/api/auth/refresh` |
| `/api/v1/auth/logout` | `/api/auth/logout` |
| `/api/v1/extension/bootstrap` | `/api/extension/bootstrap` |
| `/api/v1/proxy/extension/page-status` | `/api/proxy/extension/page-status` |

The extension task should update both the source file and generated runtime worker listed in the roadmap. No compatibility code should call `/api/v1/*` after migration.

## Panel API Requirements

| Requirement | Target shape |
|-------------|--------------|
| Panel API route migration | Move approved panel routes from `/api/v1/*` to `/api/*`. |
| Node manage-access validation | Add a panel endpoint that validates whether the authenticated account can manage a specific node. |
| Full node login authorization | Node login must require both successful control-plane authentication and positive node manage-access validation. |
| Panel web API client migration | Deferred to the panel-web phase. |

Manage-access endpoint shape:

```text
GET /api/nodes/{nodeId}/manage-access
```

Proposed response:

```json
{
  "ok": true,
  "data": {
    "nodeId": "node-001",
    "allowed": true,
    "reason": "",
    "permission": "manage"
  }
}
```
