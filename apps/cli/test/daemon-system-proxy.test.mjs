import assert from 'node:assert/strict';
import http from 'node:http';
import net from 'node:net';
import test from 'node:test';
import { systemProxyForRoute } from '../src/daemon/system-proxy.ts';

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

function restoreEnv(snapshot) {
  for (const [key, value] of Object.entries(snapshot)) {
    if (value === undefined) {
      delete process.env[key];
    } else {
      process.env[key] = value;
    }
  }
}

function proxyBindings() {
  return {
    host: '127.0.0.1',
    httpPort: 0,
    httpsPort: 0,
    ipcPort: 0
  };
}

function defaultDirectRoute(overrides = {}) {
  return {
    target: 'https://unmatched.example',
    host: 'unmatched.example',
    port: 443,
    targetHost: 'unmatched.example',
    targetPort: 443,
    protocol: 'https',
    mode: 'direct',
    source: 'default_direct',
    routeId: '',
    chainId: '',
    accessPathId: '',
    denyReason: '',
    matched: { source: 'default_direct' },
    tenant: {},
    accessPath: {},
    topology: null,
    ...overrides
  };
}

test('system proxy fallback uses preserved proxy env while onep is active', () => {
  assert.deepEqual(systemProxyForRoute(defaultDirectRoute(), {
    host: '127.0.0.1',
    httpPort: 12000,
    httpsPort: 12001,
    ipcPort: 12002,
    proxyOnlyPort: 12003
  }, {
    ONEPROXY_ACTIVE: '1',
    HTTPS_PROXY: 'http://127.0.0.1:12001',
    ONEPROXY_PREV_HTTPS_PROXY: 'http://proxy.example:18080'
  }), {
    host: 'proxy.example',
    port: 18080
  });
});

test('system proxy fallback ignores onep loopback bindings', () => {
  assert.equal(systemProxyForRoute(defaultDirectRoute(), {
    host: '127.0.0.1',
    httpPort: 12000,
    httpsPort: 12001,
    ipcPort: 12002,
    proxyOnlyPort: 12003
  }, {
    HTTPS_PROXY: 'http://127.0.0.1:12001'
  }), null);
});

test('HTTP proxy listener falls back to system proxy for unmatched HTTP routes', async () => {
  const { startHttpProxyListeners } = await import('../src/daemon/http-proxy.ts');
  const envSnapshot = {
    HTTP_PROXY: process.env.HTTP_PROXY,
    http_proxy: process.env.http_proxy,
    ALL_PROXY: process.env.ALL_PROXY,
    all_proxy: process.env.all_proxy,
    NO_PROXY: process.env.NO_PROXY,
    no_proxy: process.env.no_proxy
  };
  let upstreamPath = '';
  let upstreamHost = '';
  const systemProxy = http.createServer((request, response) => {
    upstreamPath = request.url;
    upstreamHost = request.headers.host;
    response.end('via-system');
  });
  const systemProxyPort = await listen(systemProxy);
  process.env.HTTP_PROXY = `http://127.0.0.1:${systemProxyPort}`;
  delete process.env.http_proxy;
  delete process.env.ALL_PROXY;
  delete process.env.all_proxy;
  delete process.env.NO_PROXY;
  delete process.env.no_proxy;
  const proxy = await startHttpProxyListeners({
    config: { schemaVersion: 1, overrides: { direct: [], proxy: [] } },
    state: { schemaVersion: 1 }
  }, proxyBindings());
  const httpPort = proxy.httpServer.address().port;

  try {
    const body = await new Promise((resolve, reject) => {
      const request = http.get({
        host: '127.0.0.1',
        port: httpPort,
        path: 'http://unmatched.example/path?q=1',
        headers: { host: 'unmatched.example' }
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

    assert.equal(body, 'via-system');
    assert.equal(upstreamPath, 'http://unmatched.example/path?q=1');
    assert.equal(upstreamHost, 'unmatched.example');
  } finally {
    await proxy.close();
    await closeServer(systemProxy);
    restoreEnv(envSnapshot);
  }
});

test('HTTP CONNECT listener falls back to system proxy for unmatched tunnels', async () => {
  const { startHttpProxyListeners } = await import('../src/daemon/http-proxy.ts');
  const envSnapshot = {
    HTTPS_PROXY: process.env.HTTPS_PROXY,
    https_proxy: process.env.https_proxy,
    HTTP_PROXY: process.env.HTTP_PROXY,
    http_proxy: process.env.http_proxy,
    ALL_PROXY: process.env.ALL_PROXY,
    all_proxy: process.env.all_proxy,
    NO_PROXY: process.env.NO_PROXY,
    no_proxy: process.env.no_proxy
  };
  let connectTarget = '';
  const systemProxy = net.createServer((socket) => {
    socket.once('data', (chunk) => {
      const request = chunk.toString('utf8');
      connectTarget = request.split('\r\n', 1)[0];
      socket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
      socket.once('data', (payload) => {
        socket.write(payload);
      });
    });
  });
  const systemProxyPort = await listen(systemProxy);
  process.env.HTTPS_PROXY = `http://127.0.0.1:${systemProxyPort}`;
  delete process.env.https_proxy;
  delete process.env.HTTP_PROXY;
  delete process.env.http_proxy;
  delete process.env.ALL_PROXY;
  delete process.env.all_proxy;
  delete process.env.NO_PROXY;
  delete process.env.no_proxy;
  const proxy = await startHttpProxyListeners({
    config: { schemaVersion: 1, overrides: { direct: [], proxy: [] } },
    state: { schemaVersion: 1 }
  }, proxyBindings());
  const httpPort = proxy.httpServer.address().port;

  try {
    const socket = net.connect(httpPort, '127.0.0.1');
    await new Promise((resolve, reject) => {
      socket.once('connect', resolve);
      socket.once('error', reject);
    });
    socket.write('CONNECT unmatched.example:443 HTTP/1.1\r\nHost: unmatched.example:443\r\n\r\n');
    const response = await new Promise((resolve) => socket.once('data', (chunk) => resolve(chunk.toString('utf8'))));
    assert.match(response, /^HTTP\/1\.1 200 Connection Established/);
    socket.write('ping');
    const echo = await new Promise((resolve) => socket.once('data', (chunk) => resolve(chunk.toString('utf8'))));
    assert.equal(echo, 'ping');
    assert.equal(connectTarget, 'CONNECT unmatched.example:443 HTTP/1.1');
    socket.destroy();
  } finally {
    await proxy.close();
    await closeServer(systemProxy);
    restoreEnv(envSnapshot);
  }
});
