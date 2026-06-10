import * as http from 'node:http';
import * as net from 'node:net';
import { closeServer, listenHttpServer, readConfig, readState, readTokens } from './lifecycle.ts';
import type { DaemonBindings } from './lifecycle.ts';
import { resolveRoute } from './router.ts';
import type { RouteResolverInput } from './router.ts';

export type ProxyRouteContext = Omit<RouteResolverInput, 'target' | 'protocol'>;

export type ProxyServers = {
  httpServer: http.Server;
  httpsServer: http.Server;
  close: () => Promise<void>;
};

const proxyRetryBackoffs = [100, 250];

export async function startHttpProxyListeners(input: ProxyRouteContext, bindings: DaemonBindings, liveState = false, onProxyActivity?: () => void): Promise<ProxyServers> {
  const httpServer = createHttpProxyServer(input, liveState, onProxyActivity);
  const httpsServer = createHttpProxyServer(input, liveState, onProxyActivity);
  await Promise.all([
    listenHttpServer(httpServer, bindings.httpPort),
    listenHttpServer(httpsServer, bindings.httpsPort)
  ]);
  return {
    httpServer,
    httpsServer,
    close: async () => {
      await Promise.all([closeServer(httpServer), closeServer(httpsServer)]);
    }
  };
}

export function createHttpProxyServer(input: ProxyRouteContext, liveState = false, onProxyActivity?: () => void) {
  const server = http.createServer((request, response) => {
    onProxyActivity?.();
    proxyHttpRequest(input, liveState, request, response).catch(() => {
      if (!response.headersSent) {
        response.writeHead(502);
      }
      response.end();
    });
  });

  server.on('connect', async (request, clientSocket, head) => {
    onProxyActivity?.();
    const target = parseConnectTarget(request.url ?? '');
    if (!target) {
      clientSocket.end('HTTP/1.1 400 Bad Request\r\n\r\n');
      return;
    }
    const context = await proxyContext(input, liveState);
    const route = resolveRoute({ ...context, target: `${target.host}:${target.port}`, protocol: 'connect' });
    if (route.mode === 'proxy' && !route.topology) {
      clientSocket.end('HTTP/1.1 502 Bad Gateway\r\n\r\n');
      return;
    }
    const token = route.mode === 'proxy' ? await proxyToken() : undefined;
    const upstream = route.mode === 'proxy' && route.topology ? {
      host: route.topology.entryHost,
      port: route.topology.entryPort
    } : target;
    const upstreamSocket = net.connect(upstream.port, upstream.host, () => {
      if (route.mode === 'proxy') {
        upstreamSocket.write(connectRequest(target.host, target.port, token));
      } else {
        clientSocket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
      }
      if (head.length > 0) {
        upstreamSocket.write(head);
      }
      clientSocket.pipe(upstreamSocket);
      upstreamSocket.pipe(clientSocket);
    });
    upstreamSocket.on('error', () => clientSocket.destroy());
    clientSocket.on('error', () => upstreamSocket.destroy());
  });

  return server;
}

async function proxyHttpRequest(input: ProxyRouteContext, liveState: boolean, request: http.IncomingMessage, response: http.ServerResponse) {
  const target = parseHttpTarget(request);
  if (!target) {
    response.writeHead(400);
    response.end();
    return;
  }
  const context = await proxyContext(input, liveState);
  const route = resolveRoute({ ...context, target: target.url, protocol: target.protocol });
  if (route.mode === 'proxy' && !route.topology) {
    response.writeHead(502);
    response.end();
    return;
  }
  const headers = { ...request.headers };
  delete headers['proxy-connection'];
  if (route.mode === 'proxy') {
    const token = await proxyToken();
    if (token) {
      headers['proxy-authorization'] = `Bearer ${token}`;
    }
  }
  const upstreamOptions = {
    host: route.mode === 'proxy' && route.topology ? route.topology.entryHost : target.host,
    port: route.mode === 'proxy' && route.topology ? route.topology.entryPort : target.port,
    method: request.method,
    path: route.mode === 'proxy' ? target.url : target.path,
    headers
  };
  try {
    const body = await readRequestBody(request);
    await forwardHttpWithRetry(request.method || 'GET', body, response, upstreamOptions);
  } catch {
    if (!response.headersSent) {
      response.writeHead(502);
    }
    response.end();
  }
}

async function proxyContext(input: ProxyRouteContext, liveState: boolean): Promise<ProxyRouteContext> {
  return liveState ? { config: await readConfig(), state: await readState() } : input;
}

function parseConnectTarget(value: string) {
  const index = value.lastIndexOf(':');
  if (index <= 0) {
    return null;
  }
  const host = value.slice(0, index).replace(/^\[|\]$/g, '').toLowerCase();
  const port = Number(value.slice(index + 1));
  if (!host || !Number.isInteger(port) || port < 1 || port > 65535) {
    return null;
  }
  return { host, port };
}

function parseHttpTarget(request: http.IncomingMessage) {
  const rawUrl = request.url ?? '';
  const hostHeader = Array.isArray(request.headers.host) ? request.headers.host[0] : request.headers.host;
  const url = /^[a-z][a-z0-9+.-]*:\/\//i.test(rawUrl)
    ? new URL(rawUrl)
    : new URL(rawUrl || '/', `http://${hostHeader ?? ''}`);
  const host = url.hostname.toLowerCase();
  const port = url.port ? Number(url.port) : url.protocol === 'https:' ? 443 : 80;
  if (!host || !Number.isInteger(port)) {
    return null;
  }
  return {
    url: url.toString(),
    protocol: url.protocol.replace(':', ''),
    host,
    port,
    path: `${url.pathname}${url.search}`
  };
}

function connectRequest(host: string, port: number, token: string | undefined) {
  const lines = [
    `CONNECT ${host}:${port} HTTP/1.1`,
    `Host: ${host}:${port}`
  ];
  if (token) {
    lines.push(`Proxy-Authorization: Bearer ${token}`);
  }
  return `${lines.join('\r\n')}\r\n\r\n`;
}

async function proxyToken() {
  return (await readTokens())?.proxyToken;
}

async function forwardHttpWithRetry(method: string, body: Buffer, response: http.ServerResponse, upstreamOptions: http.RequestOptions) {
  let attempt = 0;

  for (;;) {
    try {
      const upstreamResponse = await sendProxyRequest(method, body, upstreamOptions);
      if (attempt < proxyRetryBackoffs.length && retryableProxyStatus(upstreamResponse.statusCode)) {
        const delay = proxyRetryBackoffs[attempt];
        attempt += 1;
        await sleep(delay);
        continue;
      }
      response.writeHead(upstreamResponse.statusCode, upstreamResponse.headers);
      response.end(method === 'HEAD' ? undefined : upstreamResponse.body);
      return;
    } catch (error) {
      if (attempt < proxyRetryBackoffs.length) {
        const delay = proxyRetryBackoffs[attempt];
        attempt += 1;
        await sleep(delay);
        continue;
      }
      throw error;
    }
  }
}

function readRequestBody(request: http.IncomingMessage): Promise<Buffer> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    request.on('data', (chunk) => {
      chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
    });
    request.on('end', () => resolve(Buffer.concat(chunks)));
    request.on('error', reject);
    request.on('aborted', () => reject(new Error('request_aborted')));
  });
}

function sendProxyRequest(method: string, body: Buffer, upstreamOptions: http.RequestOptions): Promise<{statusCode: number; headers: http.IncomingHttpHeaders; body: Buffer}> {
  return new Promise((resolve, reject) => {
    const upstreamRequest = http.request(upstreamOptions, (upstreamResponse) => {
      const chunks: Buffer[] = [];
      upstreamResponse.on('data', (chunk) => {
        chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
      });
      upstreamResponse.on('end', () => {
        const responseBody = method === 'HEAD' ? Buffer.alloc(0) : Buffer.concat(chunks);
        const contentLength = upstreamResponse.headers['content-length'];
        if (method !== 'HEAD' && typeof contentLength === 'string' && Number(contentLength) !== responseBody.length) {
          reject(new Error('response_content_length_mismatch'));
          return;
        }
        resolve({
          statusCode: upstreamResponse.statusCode ?? 502,
          headers: upstreamResponse.headers,
          body: responseBody
        });
      });
      upstreamResponse.on('error', reject);
      upstreamResponse.on('aborted', () => reject(new Error('response_aborted')));
    });
    upstreamRequest.on('error', reject);
    if (body.length > 0) {
      upstreamRequest.end(body);
      return;
    }
    upstreamRequest.end();
  });
}

function retryableProxyStatus(statusCode: number) {
  return statusCode === 502 || statusCode === 503 || statusCode === 504;
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
