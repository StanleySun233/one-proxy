import assert from 'node:assert/strict';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import http from 'node:http';
import net from 'node:net';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { mock } from 'node:test';
import test from 'node:test';

import {
  excludedCommonPorts,
  isUsablePort,
  scanAvailableCandidatePorts,
  selectProxyPorts
} from '../src/daemon/port-selection.ts';
import { profileRoot } from '../src/storage.ts';

async function withHome(fn) {
  const previous = process.env.ONEPROXY_HOME;
  const home = await mkdtemp(path.join(tmpdir(), 'oneproxy-daemon-test-'));
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

async function closeServer(server) {
  await new Promise((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()));
  });
}

async function listen(server, port = 0) {
  await new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, '127.0.0.1', resolve);
  });
  return server.address().port;
}

async function freeConsecutivePorts() {
  for (let port = 10000; port < 10150; port += 1) {
    if (await isUsablePort(port) && await isUsablePort(port + 1)) {
      return [port, port + 1];
    }
  }
  throw new Error('No free consecutive test ports');
}

test('candidate scanning excludes common and occupied ports', async () => {
  const server = net.createServer();
  const occupiedPort = await listen(server);
  try {
    assert.equal(await isUsablePort(80), false);
    assert.equal(await isUsablePort(occupiedPort), false);
    const candidates = await scanAvailableCandidatePorts(occupiedPort - 1, occupiedPort + 1);
    assert.equal(candidates.includes(occupiedPort), false);
    assert.equal(candidates.includes(80), false);
  } finally {
    await closeServer(server);
  }
});

test('proxy port selection always chooses a random consecutive candidate pair', async () => {
  const random = mock.method(Math, 'random', () => 0);
  try {
    const selection = await selectProxyPorts();
    assert.equal(selection.selectedPair[1], selection.selectedPair[0] + 1);
    assert.equal(selection.candidatePorts.includes(selection.selectedPair[0]), true);
    assert.equal(selection.candidatePorts.includes(selection.selectedPair[1]), true);
    assert.equal(selection.excludedCommonPorts.includes(8080), true);
  } finally {
    random.mock.restore();
  }
});

test('HTTP CONNECT listener establishes a direct tunnel', async () => {
  const { startHttpProxyListeners } = await import('../src/daemon/http-proxy.ts');
  const upstream = net.createServer((socket) => {
    socket.once('data', (chunk) => {
      socket.write(chunk);
    });
  });
  const upstreamPort = await listen(upstream);
  const [httpPort, httpsPort] = await freeConsecutivePorts();
  const proxy = await startHttpProxyListeners({
    config: { schemaVersion: 1, overrides: { direct: ['127.0.0.1'], proxy: [] } },
    state: { schemaVersion: 1, routeGroups: [] }
  }, {
    host: '127.0.0.1',
    httpPort,
    httpsPort,
    ipcPort: 0
  });

  try {
    const socket = net.connect(httpPort, '127.0.0.1');
    await new Promise((resolve, reject) => {
      socket.once('connect', resolve);
      socket.once('error', reject);
    });
    socket.write(`CONNECT 127.0.0.1:${upstreamPort} HTTP/1.1\r\nHost: 127.0.0.1:${upstreamPort}\r\n\r\n`);
    const response = await new Promise((resolve) => socket.once('data', (chunk) => resolve(chunk.toString('utf8'))));
    assert.match(response, /^HTTP\/1\.1 200 Connection Established/);
    socket.write('ping');
    const echo = await new Promise((resolve) => socket.once('data', (chunk) => resolve(chunk.toString('utf8'))));
    assert.equal(echo, 'ping');
    socket.destroy();
  } finally {
    await proxy.close();
    await closeServer(upstream);
  }
});

test('HTTP proxy listener reads updated local state after startup', async () => {
  await withHome(async (home) => {
    const { startHttpProxyListeners } = await import('../src/daemon/http-proxy.ts');
    const root = profileRoot();
    await mkdir(root, { recursive: true });
    let upstreamPath = '';
    let upstreamToken = '';
    const upstream = http.createServer((request, response) => {
      upstreamPath = request.url;
      upstreamToken = request.headers['proxy-authorization'];
      response.end('proxied');
    });
    const upstreamPort = await listen(upstream);
    const [httpPort, httpsPort] = await freeConsecutivePorts();
    await writeFile(path.join(root, 'config.json'), JSON.stringify({
      schemaVersion: 1,
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      overrides: { direct: ['example.com'], proxy: [] }
    }));
    await writeFile(path.join(root, 'state.json'), JSON.stringify({
      schemaVersion: 1,
      bootstrap: {
        entryNodes: [{ id: 'entry_1', host: '127.0.0.1', port: upstreamPort, protocol: 'PROXY' }]
      },
      routeGroups: [{ id: 'group_1', tenantId: 'tenant_1', rules: [] }]
    }));
    await writeFile(path.join(root, 'tokens.json'), JSON.stringify({
      schemaVersion: 1,
      proxyToken: 'proxy-token'
    }));
    const proxy = await startHttpProxyListeners({
      config: { schemaVersion: 1, activeTenantId: 'tenant_1', activeGroupId: 'group_1', overrides: { direct: ['example.com'], proxy: [] } },
      state: {
        schemaVersion: 1,
        bootstrap: {
          entryNodes: [{ id: 'entry_1', host: '127.0.0.1', port: upstreamPort, protocol: 'PROXY' }]
        },
        routeGroups: [{ id: 'group_1', tenantId: 'tenant_1', rules: [] }]
      }
    }, {
      host: '127.0.0.1',
      httpPort,
      httpsPort,
      ipcPort: 0
    }, true);

    try {
      await writeFile(path.join(root, 'config.json'), JSON.stringify({
        schemaVersion: 1,
        activeTenantId: 'tenant_1',
        activeGroupId: 'group_1',
        overrides: { direct: [], proxy: ['example.com'] }
      }));
      const body = await new Promise((resolve, reject) => {
        const request = http.get({
          host: '127.0.0.1',
          port: httpPort,
          path: 'http://example.com/path',
          headers: { host: 'example.com' }
        }, (response) => {
          let data = '';
          response.setEncoding('utf8');
          response.on('data', (chunk) => {
            data += chunk;
          });
          response.on('end', () => resolve(data));
        });
        request.on('error', reject);
      });

      assert.equal(body, 'proxied');
      assert.equal(upstreamPath, 'http://example.com/path');
      assert.equal(upstreamToken, 'Bearer proxy-token');
    } finally {
      await proxy.close();
      await closeServer(upstream);
    }
  });
});

test('lifecycle metadata and health expose contract shape', async () => {
  await withHome(async (home) => {
    const {
      buildDaemonMetadata,
      healthFromMetadata,
      resolveBindings
    } = await import('../src/daemon/lifecycle.ts');

    const root = profileRoot();
    await mkdir(root, { recursive: true });

    await writeFile(path.join(root, 'config.json'), JSON.stringify({
      schemaVersion: 1,
      controlPlaneUrl: 'https://control.example.com',
      activeTenantId: 'tenant_1',
      activeGroupId: 'group_1',
      overrides: { direct: [], proxy: [] }
    }));
    await writeFile(path.join(root, 'state.json'), JSON.stringify({
      schemaVersion: 1,
      policyRevision: 'rev_1',
      routeGroups: []
    }));

    const resolved = await resolveBindings();
    const metadata = await buildDaemonMetadata(resolved);
    const health = healthFromMetadata(metadata);
    assert.equal(metadata.schemaVersion, 1);
    assert.equal(metadata.bindings.host, '127.0.0.1');
    assert.equal(metadata.bindings.httpsPort, metadata.bindings.httpPort + 1);
    assert.deepEqual(health, {
      ok: true,
      pid: metadata.pid,
      startedAt: metadata.startedAt,
      lastHeartbeatAt: metadata.lastHeartbeatAt,
      bindings: metadata.bindings,
      portSelection: metadata.portSelection,
      policyRevision: 'rev_1'
    });
  });
});

test('doctor reports actionable failures when local state is missing', async () => {
  await withHome(async () => {
    const { runDoctor } = await import('../src/doctor.ts');
    const result = await runDoctor('https://example.com');
    assert.equal(result.summary.status, 'fail');
    assert.equal(result.summary.failed > 0, true);
    assert.equal(result.checks.some((check) => check.name === 'token_readability' && check.status === 'fail' && check.action === 'Run onep login'), true);
    assert.equal(result.checks.some((check) => check.name === 'daemon_status' && check.status === 'pass'), true);
    assert.equal(result.checks.some((check) => check.name === 'local_ports' && check.status === 'pass'), true);
  });
});
