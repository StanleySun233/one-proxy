# Test Progress: test-release

**Engineer:** test-release
**Scope:** Compile checks, local Docker scenario, camelbot isolated scenario, GitHub Actions image verification, final-schema evidence, replacement deployment decision, database-backed user-flow tests, and release tag.

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
- [x] Verify final-schema-only panel delivery with no goose runtime and no numbered SQL migration chain
  - Commit: 65411e7
  - Evidence:
    - `apps/panel/api/schema/final.sql` is the only panel API schema file.
    - `apps/panel/api/migrations` has been removed.
    - Panel API initializes the schema only when the target database has zero tables.
    - A source scan found no panel API/web/Docker references to `goose`, `goose_db_version`, numbered SQL migrations, removed old access-path enums, old SOCKS-style aliases, or old raw TCP/UDP service names.
- [x] Add final-schema deployment guard to the camelbot panel replacement script
  - Evidence:
    - `bash -n scripts/deploy-v210-release-images.sh`: pass
    - `scripts/deploy-v210-release-images.sh dry-run camelbot-panel v2.1.0-rc.65411e7`: pass; prints required final DB and final volume
    - `scripts/deploy-v210-release-images.sh deploy camelbot-panel v2.1.0-rc.65411e7`: rejects without `ONEPROXY_FINAL_PANEL_DB_NAME`
    - `scripts/deploy-v210-release-images.sh check camelbot-panel v2.1.0-rc.65411e7`: pass; standing DB evidence `tables=40 goose_tables=1 access_paths=0`
- [x] Add final-schema standing cutover script for fresh panel DB and fresh node runtime bindings
  - Evidence:
    - `bash -n scripts/deploy-v210-final-cutover.sh`: pass
    - `scripts/deploy-v210-final-cutover.sh dry-run v2.1.0-rc.65411e7`: pass; prints final panel DB, fresh panel volume, fresh remote node volume, fresh local node volume, and target node names
    - `scripts/deploy-v210-final-cutover.sh dry-run latest`: rejects mutable image tag
    - `scripts/deploy-v210-final-cutover.sh check`: pass; local node, camelbot panel, and camelbot node are still on `v2.1.0-rc.38d3a66`; target final DB `one_proxy_v210_final` has `tables=0`
    - `scripts/deploy-v210-final-cutover.sh run v2.1.0-rc.65411e7`: rejects without `ONEPROXY_FINAL_SCHEMA_CONFIRM=deploy-final-schema`
    - Cutover rollback logic restores renamed backup containers if panel, remote-node, local-node, or final control-plane provisioning fails before completion
- [x] Run compile, unit, extension smoke, local Docker scenario, camelbot isolated scenario, image workflows, and isolated DB evidence
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
    - Final runtime commit: `65411e7`
    - `cd apps/panel/api && go test ./...`: pass after final schema baseline
    - Panel web source TypeScript check excluding stale `.next`: pass after final schema baseline
    - `node --test apps/cli/test/*.mjs`: pass, 50 tests
    - `node apps/extension/chrome/tools/domain_suffix_test.mjs`: pass
    - `node apps/extension/chrome/tools/validate_extension.mjs`: pass, `chrome_extension_static_ok`
    - `node apps/extension/chrome/tools/build_background_bundle.mjs`: pass
    - GitHub Actions panel image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27671854441
    - GitHub Actions node image workflow: pass, https://github.com/StanleySun233/one-proxy/actions/runs/27671854468
    - Immutable final-schema pre-tag image: `v2.1.0-rc.65411e7`
    - `ghcr.io/stanleysun233/oneproxy-panel:v2.1.0-rc.65411e7`: `sha256:4803a7bf3a1b94be9a0e951c4ff3c13a596d222ce5ab73d0f49fd30c7c9e7d4a`
    - `ghcr.io/stanleysun233/oneproxy-panel-base:v2.1.0-rc.65411e7`: `sha256:dfa63195389d6c82e9152121d43a69d76eacec55525463ff0de592ea2aa00d32`
    - `ghcr.io/stanleysun233/oneproxy-node:v2.1.0-rc.65411e7`: `sha256:0a19998dd5011f0a065889fad1a7dba96bbb1542778514e136af6ffdb369f567`
    - `ghcr.io/stanleysun233/oneproxy-node-base:v2.1.0-rc.65411e7`: `sha256:62fd4e5b070b1fbc2391dd05eb61b0844ded749d2d60b2b7dd727e5a707fe8ce`
    - Local isolated scenario with `ONEPROXY_IMAGE_TAG=v2.1.0-rc.65411e7 scripts/test-v210-docker-scenario.sh run`: pass; bootstrap `schema=v2.1.0 access_paths=1 routes=1`; proxy-token validation ok; session, node-token, and bootstrap-token hashes verified
    - Camelbot isolated scenario with `ONEPROXY_IMAGE_TAG=v2.1.0-rc.65411e7 ONEPROXY_MYSQL_IMAGE=mysql:8.0 ONEPROXY_REDIS_IMAGE=redis:7-alpine scripts/test-camelbot-v210-scenario.sh run`: pass; bootstrap `schema=v2.1.0 access_paths=1 routes=1`; proxy-token validation ok; session, node-token, and bootstrap-token hashes verified
    - Standing replacement deployment against an old non-empty panel database: intentionally not performed after final-schema-only change because the final release does not include old-version migration compatibility
- [ ] Create and push tag `v2.1.0` after all final deployment gates pass
  - Evidence:

## Blockers

| Date | Blocker | Status |
|------|---------|--------|
| 2026-06-17 | Raw panel web `tsc --noEmit` includes stale `.next/types/validator.ts` generated output that references a removed duplicate audit route. | Source TypeScript check passes with `.next` excluded; generated cache must be refreshed before using raw `.next` as a release artifact. |
| 2026-06-17 | Existing standing panel database is non-empty, has old migration history, and does not contain final access-path rows. | Do not deploy the final-schema-only image as an automatic upgrade. Recreate or directly provision final schema state. |
| 2026-06-17 | Standing replacement still needs final panel secrets, `ONEPROXY_FINAL_LOCAL_NODE_PARENT_URL`, and explicit destructive-operation authorization. | `scripts/deploy-v210-final-cutover.sh` is ready; target final DB is empty; run mode is gated by confirmation. |
