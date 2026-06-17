# Test Progress: test-release

**Engineer:** test-release
**Scope:** Compile checks, local Docker scenario, camelbot isolated scenario, GitHub Actions image verification, replacement deployment, database-backed user-flow tests, and release tag.

## Tasks

- [x] Add local v2.1.0 Docker scenario runner in `scripts/test-v210-docker-scenario.sh`
  - Commit: 3df6698
- [x] Add camelbot isolated scenario runner in `scripts/test-camelbot-v210-scenario.sh`
  - Commit: 3df6698
- [x] Upgrade local and camelbot scenario runners to create latest control-plane state, enroll and approve an edge node, validate latest extension bootstrap, validate hashed proxy-token authorization, and query hash-shaped token storage
  - Commit: 58ab36a
- [x] Extend GitHub Actions image workflows for immutable pre-tag test images in `.github/workflows/one-proxy-node-image.yml`
  - Commit: 3df6698
- [x] Extend GitHub Actions image workflows for immutable pre-tag test images in `.github/workflows/one-proxy-panel-image.yml`
  - Commit: 3df6698
- [x] Add release deployment script for local node, camelbot node, and camelbot panel in `scripts/deploy-v210-release-images.sh`
  - Commit: 3df6698
- [x] Document database-backed real-user verification in `docs/20260617/oneproxy-v210-release/release-test-plan.md`
  - Commit: 7679cab
- [ ] Run compile, unit, extension smoke, local Docker scenario, camelbot isolated scenario, replacement deployment, DB queries, and real-user functional tests
  - Evidence:
    - `bash -n scripts/test-v210-docker-scenario.sh scripts/test-camelbot-v210-scenario.sh scripts/deploy-v210-release-images.sh`: pass
    - `git diff --check`: pass
    - `scripts/test-v210-docker-scenario.sh check`: pass; no containers started
    - `scripts/test-camelbot-v210-scenario.sh check`: pass through SSH; no containers started; remote Docker, Compose, curl, and Python 3 available
    - `scripts/deploy-v210-release-images.sh dry-run all v2.1.0-rc.local`: pass; no deployment performed
    - `cd apps/node/api && go test ./...`: pass
    - `cd apps/panel/api && go test ./...`: pass after `58ab36a`
    - `cd apps/extension/cli && go test ./internal/proxycommand`: pass
    - `cd apps/cli && node ../panel/web/node_modules/typescript/bin/tsc -p tsconfig.json --noEmit`: pass
    - `node --test apps/extension/chrome/tools/domain_suffix_test.mjs`: pass, 3 tests
    - `node apps/extension/chrome/tools/validate_extension.mjs`: pass, `chrome_extension_static_ok`
    - `cd apps/panel/web && node node_modules/typescript/bin/tsc --noEmit --pretty false --incremental false`: fail; `.next/types/validator.ts` still references the removed duplicate audit route
    - Pushed commit: `e3de1ed56bef` to `origin/main`
    - GitHub Actions node image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27666696189
    - GitHub Actions panel image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27666696137
    - Immutable pre-tag image: `v2.1.0-rc.e3de1ed56bef`
    - `ghcr.io/stanleysun233/oneproxy-node-base:v2.1.0-rc.e3de1ed56bef`: `sha256:7bca41bcd0b672a0d2c29a388facbd6cb8800f91bc13bb664d06926a8d2c0936`
    - `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.e3de1ed56bef`: `sha256:d50310dca5c012123e8c6af82b3db3c0147a175abdc749801650806cd45908c9`
    - `ghcr.io/stanleysun233/oneproxy-panel-base:v2.1.0-rc.e3de1ed56bef`: `sha256:c5e7acbc1bf418de69d583a57d51a77c639bced5fc935a24af0a803f6e7d19bf`
    - `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.e3de1ed56bef`: `sha256:323fe5dbb95d579b0be01562cee91c8c652deda1e0e61e323fc98d2a74cc21d0`
    - `shellcheck`: not run; command not installed
    - Local Docker `build/run`, camelbot isolated `build/run`, replacement deployment, DB queries, real-user tests, and tag gate: not run yet
- [ ] Create and push tag `v2.1.0` after all gates pass
  - Evidence:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
