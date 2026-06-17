# Frontend Progress: frontend-panel-ops

**Engineer:** frontend-panel-ops
**Scope:** Panel authentication client behavior, access-path UX, security headers, and operational visual density.

## Tasks

- [ ] Move production panel web auth away from localStorage token persistence in `apps/panel/web/components/auth-provider.tsx`
  - Commit:
- [ ] Align panel API client auth behavior with the latest session contract in `apps/panel/web/lib/api/client.ts`
  - Commit:
- [ ] Remove duplicate audit proxy fallback in `apps/panel/web/app/api/audit/[...path]/route.ts`
  - Commit:
- [ ] Add production security headers in `apps/panel/web/next.config.mjs`
  - Commit:
- [ ] Make access-path editor reject unusable listeners and show route health in `apps/panel/web/app/[locale]/(console)/proxy/studio/_components/access-path-panel.tsx`
  - Commit:
- [ ] Revise operational panel visual tokens in `apps/panel/web/app/styles/tokens.css`
  - Commit:
- [ ] Reduce nested card chrome and improve console density in `apps/panel/web/app/styles/layout.css`
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
