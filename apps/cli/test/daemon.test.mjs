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

function accessPath(upstreamPort, overrides = {}) {
  return {
    id: 'path_1',
    name: 'Default path',
    chainId: 'chain_1',
    protocol: 'http',
    entryNodeId: 'entry_1',
    listenHost: '127.0.0.1',
    listenPort: upstreamPort,
    enabled: true,
    topology: [],
    ...overrides
  };
}

function latestState(upstreamPort, routes = []) {
  return {
    schemaVersion: 1,
    bootstrap: { tenantId: 'tenant_1', accessPathId: 'path_1' },
    accessPaths: [accessPath(upstreamPort)],
    routes
  };
}

function routeSnapshot(overrides = {}) {
  return {
    id: 'route_1',
    priority: 100,
    matchType: 'domain',
    matchValue: 'example.com',
    actionType: 'direct',
    chainId: 'chain_1',
    accessPathId: 'path_1',
    enabled: true,
    topology: [],
    ...overrides
  };
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
  const proxy = await startHttpProxyListeners({
    config: { schemaVersion: 1, overrides: { direct: ['127.0.0.1'], proxy: [] } },
    state: { schemaVersion: 1 }
  }, {
    host: '127.0.0.1',
    httpPort: 0,
    httpsPort: 0,
    ipcPort: 0
  });
  const httpPort = proxy.httpServer.address().port;

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
      activeAccessPathId: 'path_1',
      overrides: { direct: ['example.com'], proxy: [] }
    }));
    await writeFile(path.join(root, 'state.json'), JSON.stringify(latestState(upstreamPort)));
    await writeFile(path.join(root, 'tokens.json'), JSON.stringify({
      schemaVersion: 1,
      proxyToken: 'proxy-token'
    }));
    const proxy = await startHttpProxyListeners({
      config: { schemaVersion: 1, activeTenantId: 'tenant_1', activeAccessPathId: 'path_1', overrides: { direct: ['example.com'], proxy: [] } },
      state: latestState(upstreamPort)
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
        activeAccessPathId: 'path_1',
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

test('proxy-only listener ignores direct routes and forwards through entry node', async () => {
  await withHome(async (home) => {
    const { startHttpProxyListeners } = await import('../src/daemon/http-proxy.ts');
    const root = profileRoot();
    await mkdir(root, { recursive: true });
    let upstreamPath = '';
    let upstreamToken = '';
    const upstream = http.createServer((request, response) => {
      upstreamPath = request.url;
      upstreamToken = request.headers['proxy-authorization'];
      response.end('proxied-only');
    });
    const upstreamPort = await listen(upstream);
    await writeFile(path.join(root, 'tokens.json'), JSON.stringify({
      schemaVersion: 1,
      proxyToken: 'proxy-token'
    }));
    const proxy = await startHttpProxyListeners({
      config: {
        schemaVersion: 1,
        activeTenantId: 'tenant_1',
        activeAccessPathId: 'path_1',
        overrides: { direct: ['example.com'], proxy: [] }
      },
      state: latestState(upstreamPort, [routeSnapshot({ id: 'direct_1' })])
    }, {
      host: '127.0.0.1',
      httpPort: 0,
      httpsPort: 0,
      ipcPort: 0,
      proxyOnlyPort: 0
    });
    const proxyOnlyPort = proxy.proxyOnlyServer.address().port;

    try {
      const body = await new Promise((resolve, reject) => {
        const request = http.get({
          host: '127.0.0.1',
          port: proxyOnlyPort,
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

      assert.equal(body, 'proxied-only');
      assert.equal(upstreamPath, 'http://example.com/path');
      assert.equal(upstreamToken, 'Bearer proxy-token');
    } finally {
      await proxy.close();
      await closeServer(upstream);
    }
  });
});

test('HTTP proxy listener retries transient static resource failures', async () => {
  const { startHttpProxyListeners } = await import('../src/daemon/http-proxy.ts');
  let attempts = 0;
  const upstream = http.createServer((request, response) => {
    attempts += 1;
    if (request.url !== '/static/js/app.js') {
      response.writeHead(404);
      response.end();
      return;
    }
    if (attempts === 1) {
      response.writeHead(502);
      response.end('bad_gateway');
      return;
    }
    response.end('loaded');
  });
  const upstreamPort = await listen(upstream);
  const proxy = await startHttpProxyListeners({
    config: { schemaVersion: 1, overrides: { direct: ['127.0.0.1'], proxy: [] } },
    state: { schemaVersion: 1 }
  }, {
    host: '127.0.0.1',
    httpPort: 0,
    httpsPort: 0,
    ipcPort: 0
  });
  const httpPort = proxy.httpServer.address().port;

  try {
    const body = await new Promise((resolve, reject) => {
      const request = http.get({
        host: '127.0.0.1',
        port: httpPort,
        path: `http://127.0.0.1:${upstreamPort}/static/js/app.js`,
        headers: { host: `127.0.0.1:${upstreamPort}` }
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

    assert.equal(body, 'loaded');
    assert.equal(attempts, 2);
  } finally {
    await proxy.close();
    await closeServer(upstream);
  }
});

test('HTTP proxy listener does not retry unsafe POST requests', async () => {
  const { startHttpProxyListeners } = await import('../src/daemon/http-proxy.ts');
  let attempts = 0;
  const bodies = [];
  const upstream = http.createServer((request, response) => {
    let body = '';
    request.setEncoding('utf8');
    request.on('data', (chunk) => {
      body += chunk;
    });
    request.on('end', () => {
      attempts += 1;
      bodies.push(body);
      if (body !== 'upload') {
        response.writeHead(400);
        response.end('bad_body');
        return;
      }
      if (attempts === 1) {
        response.writeHead(502);
        response.end('bad_gateway');
        return;
      }
      response.end('saved');
    });
  });
  const upstreamPort = await listen(upstream);
  const proxy = await startHttpProxyListeners({
    config: { schemaVersion: 1, overrides: { direct: ['127.0.0.1'], proxy: [] } },
    state: { schemaVersion: 1 }
  }, {
    host: '127.0.0.1',
    httpPort: 0,
    httpsPort: 0,
    ipcPort: 0
  });
  const httpPort = proxy.httpServer.address().port;

  try {
    const body = await new Promise((resolve, reject) => {
      const request = http.request({
        host: '127.0.0.1',
        port: httpPort,
        method: 'POST',
        path: `http://127.0.0.1:${upstreamPort}/api/save`,
        headers: {
          host: `127.0.0.1:${upstreamPort}`,
          'content-length': '6'
        }
      }, (response) => {
        let data = '';
        response.setEncoding('utf8');
        response.on('data', (chunk) => {
          data += chunk;
        });
        response.on('end', () => resolve(data));
      });
      request.on('error', (error) => reject(new Error(`${error.message}; attempts=${attempts}; bodies=${JSON.stringify(bodies)}`)));
      request.end('upload');
    });

    assert.equal(body, 'bad_gateway');
    assert.equal(attempts, 1);
    assert.deepEqual(bodies, ['upload']);
  } finally {
    await proxy.close();
    await closeServer(upstream);
  }
});

test('HTTP proxy listener retries content length mismatches before responding', async () => {
  const { startHttpProxyListeners } = await import('../src/daemon/http-proxy.ts');
  let attempts = 0;
  const upstream = http.createServer((request, response) => {
    attempts += 1;
    if (attempts === 1) {
      response.setHeader('content-length', '6');
      response.setHeader('connection', 'close');
      response.end('bad');
      return;
    }
    response.end('loaded');
  });
  const upstreamPort = await listen(upstream);
  const proxy = await startHttpProxyListeners({
    config: { schemaVersion: 1, overrides: { direct: ['127.0.0.1'], proxy: [] } },
    state: { schemaVersion: 1 }
  }, {
    host: '127.0.0.1',
    httpPort: 0,
    httpsPort: 0,
    ipcPort: 0
  });
  const httpPort = proxy.httpServer.address().port;

  try {
    const body = await new Promise((resolve, reject) => {
      const request = http.get({
        host: '127.0.0.1',
        port: httpPort,
        path: `http://127.0.0.1:${upstreamPort}/api/scenario/id/track`,
        headers: { host: `127.0.0.1:${upstreamPort}` }
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

    assert.equal(body, 'loaded');
    assert.equal(attempts, 2);
  } finally {
    await proxy.close();
    await closeServer(upstream);
  }
});

test('HTTP proxy listener streams large safe responses without retry buffering', async () => {
  const { startHttpProxyListeners } = await import('../src/daemon/http-proxy.ts');
  let attempts = 0;
  const payload = Buffer.alloc(9 * 1024 * 1024, 'x');
  const upstream = http.createServer((_request, response) => {
    attempts += 1;
    response.setHeader('content-length', String(payload.length));
    response.end(payload);
  });
  const upstreamPort = await listen(upstream);
  const proxy = await startHttpProxyListeners({
    config: { schemaVersion: 1, overrides: { direct: ['127.0.0.1'], proxy: [] } },
    state: { schemaVersion: 1 }
  }, {
    host: '127.0.0.1',
    httpPort: 0,
    httpsPort: 0,
    ipcPort: 0
  });
  const httpPort = proxy.httpServer.address().port;

  try {
    const bytes = await new Promise((resolve, reject) => {
      const request = http.get({
        host: '127.0.0.1',
        port: httpPort,
        path: `http://127.0.0.1:${upstreamPort}/download.bin`,
        headers: { host: `127.0.0.1:${upstreamPort}` }
      }, (response) => {
        let total = 0;
        response.on('data', (chunk) => {
          total += chunk.length;
        });
        response.on('end', () => resolve(total));
      });
      request.on('error', reject);
    });

    assert.equal(bytes, payload.length);
    assert.equal(attempts, 1);
  } finally {
    await proxy.close();
    await closeServer(upstream);
  }
});

test('monitor parser reads Windows netstat connection rows', async () => {
  const { monitorInternals } = await import('../src/monitor.ts');
  const entries = monitorInternals.parseWindowsNetstat(`
  Proto  Local Address          Foreign Address        State           PID
  TCP    192.168.1.2:50123      203.0.113.10:443      ESTABLISHED     4242
  UDP    0.0.0.0:5353           *:*                                    5353
`);
  assert.equal(entries.length, 2);
  assert.equal(entries[0].source, 'netstat');
  assert.equal(entries[0].pid, 4242);
  assert.equal(entries[0].protocol, 'tcp');
  assert.equal(entries[0].localAddress, '192.168.1.2');
  assert.equal(entries[0].localPort, 50123);
  assert.equal(entries[0].remoteAddress, '203.0.113.10');
  assert.equal(entries[0].remotePort, 443);
  assert.equal(entries[0].state, 'ESTABLISHED');
  assert.equal(entries[1].protocol, 'udp');
  assert.equal(entries[1].remoteAddress, '*');
  assert.equal(entries[1].remotePort, null);
});

test('monitor parser tracks process descendants and platform endpoints', async () => {
  const { monitorInternals } = await import('../src/monitor.ts');
  const watched = monitorInternals.watchedProcesses(10, new Set([13]), [
    { pid: 10, parentPid: 1, name: 'launcher' },
    { pid: 11, parentPid: 10, name: 'game' },
    { pid: 12, parentPid: 11, name: 'worker' },
    { pid: 13, parentPid: 999, name: 'previous' },
    { pid: 14, parentPid: 13, name: 'previous-child' }
  ]);
  assert.deepEqual([...watched].sort((a, b) => a - b), [10, 11, 12, 13, 14]);

  assert.deepEqual(monitorInternals.parseLinuxEndpoint('0100007F:1F90', 'ipv4'), {
    address: '127.0.0.1',
    port: 8080
  });

  const lsof = monitorInternals.parseLsof(`COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
Game     4242 main   10u  IPv4      0      0t0  TCP 192.168.1.2:50123->203.0.113.10:443 (ESTABLISHED)
`);
  assert.equal(lsof.length, 1);
  assert.equal(lsof[0].source, 'lsof');
  assert.equal(lsof[0].pid, 4242);
  assert.equal(lsof[0].protocol, 'tcp');
  assert.equal(lsof[0].remoteAddress, '203.0.113.10');
  assert.equal(lsof[0].remotePort, 443);
  assert.equal(lsof[0].state, 'ESTABLISHED');
});

test('lifecycle metadata and health expose contract shape', async () => {
  await withHome(async (home) => {
    const {
      buildDaemonMetadata,
      envIdleTimeoutSeconds,
      healthFromMetadata,
      resolveBindings
    } = await import('../src/daemon/lifecycle.ts');

    const root = profileRoot();
    await mkdir(root, { recursive: true });

    await writeFile(path.join(root, 'config.json'), JSON.stringify({
      schemaVersion: 1,
      controlPlaneUrl: 'https://control.example.com',
      activeTenantId: 'tenant_1',
      overrides: { direct: [], proxy: [] }
    }));
    await writeFile(path.join(root, 'state.json'), JSON.stringify({
      schemaVersion: 1,
      policyRevision: 'rev_1'
    }));

    const resolved = await resolveBindings();
    const metadata = await buildDaemonMetadata(resolved);
    const health = healthFromMetadata(metadata);
    assert.equal(metadata.schemaVersion, 1);
    assert.equal(metadata.idleTimeoutSeconds, envIdleTimeoutSeconds);
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

test('daemon session end switches idle timeout to run cleanup window', async () => {
  await withHome(async (home) => {
    const {
      readDaemonMetadata,
      runIdleTimeoutSeconds,
      startDaemonRuntime
    } = await import('../src/daemon/lifecycle.ts');

    const root = profileRoot();
    await mkdir(root, { recursive: true });
    await writeFile(path.join(root, 'config.json'), JSON.stringify({
      schemaVersion: 1,
      overrides: { direct: [], proxy: [] }
    }));
    await writeFile(path.join(root, 'state.json'), JSON.stringify({
      schemaVersion: 1
    }));

    const runtime = await startDaemonRuntime();
    try {
      await new Promise((resolve, reject) => {
        const request = http.request({
          host: runtime.metadata.bindings.host,
          port: runtime.metadata.bindings.ipcPort,
          path: '/v1/session/start',
          method: 'POST',
          headers: { 'X-One-Proxy-Daemon-Secret': runtime.metadata.daemonSecret }
        }, (response) => {
          response.resume();
          response.on('end', () => response.statusCode && response.statusCode >= 400 ? reject(new Error(`start rejected: ${response.statusCode}`)) : resolve());
        });
        request.on('error', reject);
        request.end('{}');
      });
      await new Promise((resolve, reject) => {
        const request = http.request({
          host: runtime.metadata.bindings.host,
          port: runtime.metadata.bindings.ipcPort,
          path: '/v1/session/end',
          method: 'POST',
          headers: { 'X-One-Proxy-Daemon-Secret': runtime.metadata.daemonSecret }
        }, (response) => {
          response.resume();
          response.on('end', () => response.statusCode && response.statusCode >= 400 ? reject(new Error(`end rejected: ${response.statusCode}`)) : resolve());
        });
        request.on('error', reject);
        request.end('{}');
      });
      const metadata = await readDaemonMetadata();
      assert.equal(metadata.idleTimeoutSeconds, runIdleTimeoutSeconds);
    } finally {
      await runtime.close();
    }
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
