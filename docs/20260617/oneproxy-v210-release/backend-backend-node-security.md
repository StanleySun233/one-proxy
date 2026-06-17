# Backend Progress: backend-node-security

**Engineer:** backend-node-security
**Scope:** Node authorization, bootstrap safety, proxy forwarding, TCP/UDP access, and chain behavior.

## Tasks

- [x] Make proxy token authorization fail closed in `apps/node/api/internal/proxy/auth.go`
  - Commit: 4536c1ca81bca99ecd14e357a830d5cdfe5d1f2b
- [x] Remove default join password and require explicit bootstrap secret in `apps/node/api/internal/agentconfig/config.go`
  - Commit: 8f62f83d511d6ef61d9f5d5151df94d0cb78ec6a
- [x] Harden node attach and password rotation behavior in `apps/node/api/internal/bootstrap/handler.go`
  - Commit: b526127a6ba294da577e813df6389af9a19c00d1
- [x] Prevent unbound node proxy exposure in `apps/node/api/cmd/one-proxy-node/main.go`
  - Commit: 3acdfa9eb72f05f7bd86d70b8a93aee9aff84a0d
- [x] Require closed authorization semantics in `apps/node/api/internal/tcpaccess/server.go`
  - Commit: 4c3e597dd57939354ad6648a38c3fe13a9443e4b
- [x] Require closed authorization semantics and packet limits in `apps/node/api/internal/udpaccess/server.go`
  - Commit: f20399ddc475a90a19b40830c3c48daa2834809c
- [x] Authenticate public next-hop CONNECT forwarding in `apps/node/api/internal/proxy/connect_tunnel.go`
  - Commit: 8b74303d46c4d29b0ba05543e0273d4dde8f8186
- [x] Stream HTTP forwarding, add timeouts, and limit unsafe retries in `apps/node/api/internal/proxy/forward_http.go`
  - Commit: 10a8f53e91a79652a9eff009f40d52738fce2031

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
