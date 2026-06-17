import assert from 'node:assert/strict';
import test from 'node:test';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

function createStorageArea() {
  const values = new Map();
  return {
    async get(keys) {
      if (Array.isArray(keys)) {
        return Object.fromEntries(keys.map((key) => [key, values.get(key)]));
      }
      if (typeof keys === 'string') {
        return { [keys]: values.get(keys) };
      }
      return Object.fromEntries(values.entries());
    },
    async set(items) {
      for (const [key, value] of Object.entries(items)) {
        values.set(key, value);
      }
    },
    async clear() {
      values.clear();
    }
  };
}

async function loadStateModule() {
  const file = path.resolve('apps/extension/chrome/tools/background-source/state.js');
  return import(`${pathToFileURL(file).href}?case=${Date.now()}-${Math.random()}`);
}

test('session secrets survive chrome.storage.session reset', async () => {
  const local = createStorageArea();
  const session = createStorageArea();
  globalThis.chrome = { storage: { local, session } };

  const firstWorker = await loadStateModule();
  await firstWorker.persistState({
    ...firstWorker.DEFAULT_STATE,
    controlPlaneUrl: 'https://panel.oneproxy.test',
    session: {
      account: 'admin',
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      expiresAt: '2026-07-04T00:00:00Z',
      proxyToken: 'proxy-token',
      proxyTokenExpiresAt: '2026-07-04T00:00:00Z',
      mustRotatePassword: false,
      tenantMemberships: [{ tenantId: 'tenant-1', tenantName: 'Default' }],
      activeTenantId: 'tenant-1'
    }
  });

  await session.clear();

  const restartedWorker = await loadStateModule();
  const restored = await restartedWorker.getState();

  assert.equal(restored.session.accessToken, 'access-token');
  assert.equal(restored.session.refreshToken, 'refresh-token');
  assert.equal(restored.session.proxyToken, 'proxy-token');
  assert.equal(restored.session.activeTenantId, 'tenant-1');
});
