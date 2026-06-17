# Backend Progress: backend-node-security

**Engineer:** backend-node-security
**Scope:** Node authorization, bootstrap safety, proxy forwarding, TCP/UDP access, and chain behavior.

## Tasks

- [ ] Make proxy token authorization fail closed in `apps/node/api/internal/proxy/auth.go`
  - Commit:
- [ ] Remove default join password and require explicit bootstrap secret in `apps/node/api/internal/agentconfig/config.go`
  - Commit:
- [ ] Harden node attach and password rotation behavior in `apps/node/api/internal/bootstrap/handler.go`
  - Commit:
- [ ] Prevent unbound node proxy exposure in `apps/node/api/cmd/one-proxy-node/main.go`
  - Commit:
- [ ] Require closed authorization semantics in `apps/node/api/internal/tcpaccess/server.go`
  - Commit:
- [ ] Require closed authorization semantics and packet limits in `apps/node/api/internal/udpaccess/server.go`
  - Commit:
- [ ] Authenticate public next-hop CONNECT forwarding in `apps/node/api/internal/proxy/connect_tunnel.go`
  - Commit:
- [ ] Stream HTTP forwarding, add timeouts, and limit unsafe retries in `apps/node/api/internal/proxy/forward_http.go`
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
