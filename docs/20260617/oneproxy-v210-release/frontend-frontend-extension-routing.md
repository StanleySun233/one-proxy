# Frontend Progress: frontend-extension-routing

**Engineer:** frontend-extension-routing
**Scope:** Chrome extension latest route/access-path state, PAC behavior, runtime message safety, permissions, and extension UI.

## Tasks

- [x] Replace legacy group state with latest route/access-path state in `apps/extension/chrome/tools/background-source/state.js`
  - Commit: `31c9896`
- [x] Sync latest bootstrap contract in `apps/extension/chrome/tools/background-source/api.js`
  - Commit: `01493f5`
- [x] Compile latest route rules into PAC behavior in `apps/extension/chrome/tools/background-source/pac.js`
  - Commit: `dafee2c`
- [x] Make route preview share the same evaluator assumptions as PAC in `apps/extension/chrome/tools/background-source/routing.js`
  - Commit: `75c9424`
- [x] Restrict runtime message responses and session exposure in `apps/extension/chrome/tools/background-source/messages.js`
  - Commit: `453cc9d`
- [x] Minimize Chrome extension permissions in `apps/extension/chrome/manifest.json`
  - Commit: `6df9897`
- [x] Update popup route/group display for latest access-path state in `apps/extension/chrome/popup/runtime.js`
  - Commit: `4169d52`
- [x] Update options route/group display for latest access-path state in `apps/extension/chrome/options/runtime.js`
  - Commit: `dc8bff3`
- [x] Authorize multiple access-path proxy challenges in `apps/extension/chrome/tools/background-source/proxy-auth.js`
  - Commit: `c1dd049`
- [x] Sync Chrome page-source popup/options with latest access-path runtime in `apps/extension/chrome/tools/page-source/popup/index.js` and `apps/extension/chrome/tools/page-source/options/index.js`
  - Commit: `6cc26db`
- [x] Regenerate tracked Chrome extension bundles in `apps/extension/chrome/background/one-proxy-worker.js`, `apps/extension/chrome/popup/runtime.js`, and `apps/extension/chrome/options/runtime.js`
  - Commit: `6ff1552`
- [x] Replace status bubble and monitor legacy group probes with latest access-path route topology in `apps/extension/chrome/tools/background-source/status-bubble.js` and `apps/extension/chrome/tools/background-source/monitor.js`
  - Commit: `5539c4d`
- [x] Update shared page-source contracts and extension smoke fixtures to latest access-path bootstrap state in `apps/extension/chrome/tools/page-source/shared/contracts.js`, `apps/extension/chrome/tools/domain_suffix_test.mjs`, and `apps/extension/chrome/tools/service_worker_smoke.mjs`
  - Commit: `5539c4d`
- [x] Regenerate tracked Chrome extension bundles after monitor/status/test fixture updates in `apps/extension/chrome/background/one-proxy-worker.js`
  - Commit: `5539c4d`
- [x] Rename visible extension group terminology to access-path terminology in popup, options, status bubble, locales, and generated runtime assets
  - Commit: `13b6358`

## Verification

- `node --check apps/extension/chrome/tools/background-source/status-bubble.js`: pass
- `node --check apps/extension/chrome/tools/background-source/monitor.js`: pass
- `node --check apps/extension/chrome/tools/page-source/shared/contracts.js`: pass
- `node --check apps/extension/chrome/tools/domain_suffix_test.mjs`: pass
- `node --check apps/extension/chrome/tools/service_worker_smoke.mjs`: pass
- `node --test apps/extension/chrome/tools/domain_suffix_test.mjs`: pass, 3 tests
- `node apps/extension/chrome/tools/build_background_bundle.mjs`: pass
- `node apps/extension/chrome/tools/validate_extension.mjs`: pass, `chrome_extension_static_ok`
- `rg -n "activeGroup|noGroup|statusNoGroups|\bgroups\b|groupDetail|Group default|Remote groups|remote groups|远程分组|当前分组|无分组|分组详情|分组默认|panel-groups" apps/extension/chrome/tools apps/extension/chrome/_locales apps/extension/chrome/popup apps/extension/chrome/options apps/extension/chrome/content apps/extension/chrome/background`: pass, no matches

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
| 2026-06-17 | `apps/extension/chrome/tools/background-source/proxy-auth.js` is outside this assignment and still caches one proxy auth target while latest PAC routes can emit multiple access-path proxy targets. | Resolved by `c1dd049` |
