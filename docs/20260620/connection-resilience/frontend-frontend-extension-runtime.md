# Frontend Progress: frontend-extension-runtime

**Engineer:** frontend-extension-runtime
**Scope:** Chrome extension connection state, reconnect scheduling, diagnostics payload, status bubble rendering, and extension tests.

## Tasks

- [ ] Add extension connection-state reducer in `apps/extension/chrome/tools/background-source/connection-state.js`
  - Commit:
- [ ] Persist connection state and retry metadata in `apps/extension/chrome/tools/background-source/state.js`
  - Commit:
- [ ] Sync connection diagnostics from panel bootstrap and status APIs in `apps/extension/chrome/tools/background-source/api.js`
  - Commit:
- [ ] Implement bounded reconnect scheduling in `apps/extension/chrome/tools/background-source/monitor.js`
  - Commit:
- [ ] Detect proxy setting drift and proxy auth failures in `apps/extension/chrome/tools/background-source/proxy-auth.js`
  - Commit:
- [ ] Expose route, phase, retry, selected access path, and hop diagnostics in `apps/extension/chrome/tools/background-source/status-bubble.js`
  - Commit:
- [ ] Render diagnostic details in `apps/extension/chrome/tools/content-source/status-bubble.js`
  - Commit:
- [ ] Style diagnostic details in `apps/extension/chrome/content/status-bubble.css`
  - Commit:
- [ ] Update extension localization keys in `apps/extension/chrome/_locales/zh_CN/messages.json` and `apps/extension/chrome/_locales/en/messages.json`
  - Commit:
- [ ] Regenerate Chrome extension runtime bundle in `apps/extension/chrome/background/one-proxy-worker.js`
  - Commit:
- [ ] Add extension connection-state and reconnect tests in `apps/extension/chrome/test/connection_state_test.mjs`
  - Commit:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
