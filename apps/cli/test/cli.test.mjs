import assert from 'node:assert/strict';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

import {
  readConfig,
  storageFile,
  writeConfig,
  writeState,
  writeTokens
} from '../src/storage.ts';
import { resolveRoute } from '../src/daemon/router.ts';

const repoRoot = path.resolve(import.meta.dirname, '../../..');
const mainEntrypoint = path.join(repoRoot, 'apps/cli/src/main.ts');

async function withHome(fn) {
  const previous = process.env.ONEPROXY_HOME;
  const home = await mkdtemp(path.join(tmpdir(), 'oneproxy-cli-test-'));
  process.env.ONEPROXY_HOME = home;
  try {
    return await fn(home);
  } finally {
    if (previous === undefined) {
      delete process.env.ONEPROXY_HOME;
    } else {
      process.env.ONEPROXY_HOME = previous;
    }
    await rm(home, { recursive: true, force: true });
  }
}

function runCli(args, home, extraEnv = {}) {
  return spawnSync(process.execPath, [mainEntrypoint, ...args], {
    cwd: repoRoot,
    env: {
      ...process.env,
      ONEPROXY_HOME: home,
      ...extraEnv
    },
    encoding: 'utf8'
  });
}

test('storage normalizes defaults and override hosts', async () => {
  await withHome(async () => {
    assert.deepEqual(await readConfig(), {
      schemaVersion: 1,
      localPorts: { http: 0, https: 0, ipc: 0 },
      overrides: { direct: [], proxy: [] }
    });

    await writeConfig({
      schemaVersion: 99,
      controlPlaneUrl: 'https://control.example.com',
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      localPorts: { http: 34120, https: 34121, ipc: 0 },
      overrides: {
        direct: [' Example.COM ', 'example.com', 'LOCALHOST'],
        proxy: ['Proxy.Example', '', 'proxy.example']
      }
    });

    const config = await readConfig();
    assert.equal(config.schemaVersion, 1);
    assert.deepEqual(config.localPorts, { http: 34120, https: 34121, ipc: 0 });
    assert.deepEqual(config.overrides.direct, ['example.com', 'localhost']);
    assert.deepEqual(config.overrides.proxy, ['proxy.example']);

    const stored = JSON.parse(await readFile(storageFile('config'), 'utf8'));
    assert.deepEqual(stored.overrides.direct, ['example.com', 'localhost']);
  });
});

test('route matching applies direct override before proxy override and policy', () => {
  const route = resolveRoute({
    config: {
      schemaVersion: 1,
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      overrides: {
        direct: ['example.com'],
        proxy: ['example.com', 'proxy.local']
      }
    },
    state: {
      schemaVersion: 1,
      bootstrap: {
        entryNodes: [{ id: 'entry_1', host: 'edge.example.com', port: 443, protocol: 'connect' }]
      },
      routeGroups: [
        {
          id: 'group_1',
          tenantId: 'tenant_1',
          name: 'Default',
          rules: [{ id: 'rule_1', type: 'suffix', pattern: 'example.com', mode: 'proxy' }]
        }
      ]
    },
    target: 'https://example.com/path'
  });

  assert.equal(route.mode, 'direct');
  assert.equal(route.matched.source, 'local_override_direct');
  assert.equal(route.tenant.id, 'tenant_1');
  assert.equal(route.group.id, 'group_1');
  assert.equal(route.topology, null);
});

test('route matching returns proxied topology for suffix policy', () => {
  const route = resolveRoute({
    config: {
      schemaVersion: 1,
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      overrides: { direct: [], proxy: [] }
    },
    state: {
      schemaVersion: 1,
      bootstrap: {
        entryNodes: [{ id: 'entry_1', host: 'edge.example.com', port: 443, protocol: 'connect' }]
      },
      routeGroups: [
        {
          id: 'group_1',
          tenantId: 'tenant_1',
          name: 'Default',
          rules: [{ id: 'rule_1', type: 'suffix', pattern: 'example.com', mode: 'proxy' }]
        }
      ]
    },
    target: 'api.example.com',
    protocol: 'https'
  });

  assert.equal(route.mode, 'proxy');
  assert.equal(route.port, 443);
  assert.deepEqual(route.topology, {
    entryNodeId: 'entry_1',
    entryHost: 'edge.example.com',
    entryPort: 443,
    protocol: 'connect'
  });
});

test('env off prints shell restoration output', async () => {
  await withHome(async (home) => {
    const result = runCli(['env', 'off'], home, { ONEPROXY_SHELL: '/bin/bash' });
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, /unset ONEPROXY_ACTIVE ONEPROXY_HTTP_PORT ONEPROXY_HTTPS_PORT/);
    assert.equal(result.stderr, '');
  });
});

test('command parser handles help and unknown commands', async () => {
  await withHome(async (home) => {
    const help = runCli(['help'], home);
    assert.equal(help.status, 0, help.stderr);
    assert.match(help.stdout, /Usage: onep <command>/);

    const unknown = runCli(['missing-command'], home);
    assert.equal(unknown.status, 2);
    assert.match(unknown.stderr, /Unknown command: missing-command/);
  });
});

test('status --json output matches contract shape', async () => {
  await withHome(async (home) => {
    await writeConfig({
      schemaVersion: 1,
      controlPlaneUrl: 'https://control.example.com',
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      localPorts: { http: 34120, https: 34121, ipc: 34122 },
      overrides: { direct: ['direct.example'], proxy: ['proxy.example'] }
    });
    await writeState({
      schemaVersion: 1,
      bootstrap: { groupId: 'group_1', entryNodes: [] },
      policyRevision: 'rev_1',
      fetchedAt: '2026-06-06T06:00:00.000Z',
      routeGroups: [{ id: 'group_1', tenantId: 'tenant_1', name: 'Default', rules: [] }]
    });
    await writeTokens({
      schemaVersion: 1,
      account: { id: 'user_1', email: 'user@example.com' },
      accessTokenExpiresAt: '2026-06-06T07:00:00.000Z',
      refreshTokenExpiresAt: '2026-07-06T06:00:00.000Z',
      proxyTokenExpiresAt: '2026-06-06T07:00:00.000Z'
    });

    const result = runCli(['status', '--json'], home);
    assert.equal(result.status, 0, result.stderr);
    const status = JSON.parse(result.stdout);
    assert.deepEqual(Object.keys(status).sort(), [
      'account',
      'controlPlane',
      'daemon',
      'group',
      'localPorts',
      'overrides',
      'policyRevision',
      'portSelection',
      'tenant',
      'tokens'
    ]);
    assert.equal(status.account.email, 'user@example.com');
    assert.equal(status.localPorts.http, 34120);
    assert.equal(status.overrides.directCount, 1);
  });
});
