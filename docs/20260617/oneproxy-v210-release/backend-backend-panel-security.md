# Backend Progress: backend-panel-security

**Engineer:** backend-panel-security
**Scope:** Panel API session security, setup safety, proxy-token validation, access-path validation, and repository semantics.

## Tasks

- [ ] Hash panel access and refresh session tokens in `apps/panel/api/internal/store/mysql_account.go`
  - Commit:
- [ ] Sanitize service error responses in `apps/panel/api/internal/httpapi/response.go`
  - Commit:
- [ ] Remove raw proxy-token validation fallback and enforce latest scope fields in `apps/panel/api/internal/httpapi/proxy_token.go`
  - Commit:
- [ ] Harden setup DB test/init behavior in `apps/panel/api/internal/setup/handler.go`
  - Commit:
- [ ] Enforce listener port and latest access-path validation in `apps/panel/api/internal/features/proxy/service/access_path.go`
  - Commit:
- [ ] Make access-path update semantics latest-contract-only in `apps/panel/api/internal/store/proxy_repository.go`
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
