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
- [x] Update node proxy fail-closed auth tests in `apps/node/api/internal/proxy/server_test.go` and `apps/node/api/internal/proxy/reverse_test.go`
  - Commit: 4e03ebe4581af324ccf91dbf02422eb9410ba795
- [x] Verify direct QUIC identity in node-to-node stream clients in `apps/node/api/internal/direct/quic_stream.go`
  - Commit: cac1d22c8c9b72702c7a7c947d21880ef63452de
- [x] Pass direct QUIC peer identity through node candidates and link plans in `apps/node/api/internal/domain/direct.go`, `apps/node/api/internal/direct/**`, and `apps/panel/api/internal/{domain,store}/direct_transport.go`
  - Commit: fb5fd71

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
| 2026-06-17 | Node-to-node direct link plans do not currently pass DirectNodeIdentity trust material or certificate fingerprints; the node direct client now fails closed instead of falling back to insecure TLS. | Resolved by 5e57334 and fb5fd71 |

## Verification

- `cd apps/node/api && go test ./internal/direct`: pass
- `cd apps/panel/api && go test ./internal/service ./internal/store ./internal/httpapi`: pass
