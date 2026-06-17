import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const expectedVersion = String(process.env.ONEPROXY_RELEASE_VERSION || '').trim();

test('extension artifacts match release version', () => {
  if (!expectedVersion) {
    return;
  }

  const manifest = JSON.parse(readFileSync('apps/extension/chrome/manifest.json', 'utf8'));
  const vscodePackage = JSON.parse(readFileSync('apps/extension/vscode/package.json', 'utf8'));
  const cliMain = readFileSync('apps/extension/cli/cmd/oneproxy/main.go', 'utf8');

  assert.equal(manifest.version, expectedVersion);
  assert.equal(vscodePackage.version, expectedVersion);
  assert.ok(cliMain.includes(`const version = "${expectedVersion}"`));
});
