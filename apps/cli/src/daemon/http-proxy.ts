import * as http from 'node:http';
import * as net from 'node:net';
import type { Duplex } from 'node:stream';
import { closeServer, listenHttpServer, readConfig, readState, readTokens } from './lifecycle.ts';
import type { DaemonBindings } from './lifecycle.ts';
import { resolveRoute } from './router.ts';
import type { RouteResolverInput, RouteResult } from './router.ts';

export type ProxyRouteContext = Omit<RouteResolverInput, 'target' | 'protocol'>;

export type ProxyServers = {
  httpServer: http.Server;
  httpsServer: http.Server;
  proxyOnlyServer?: http.Server;
  close: () => Promise<void>;
};

export async function startHttpProxyListeners(input: ProxyRouteContext, bindings: DaemonBindings, liveState = false, onProxyActivity?: () => void): Promise<ProxyServers> {
  const httpServer = createHttpProxyServer(input, liveState, onProxyActivity);
  const httpsServer = createHttpProxyServer(input, liveState, onProxyActivity);
  const proxyOnlyServer = bindings.proxyOnlyPort !== undefined ? createHttpProxyServer(input, liveState, onProxyActivity, true) : undefined;
  const listeners = [
    listenHttpServer(httpServer, bindings.httpPort),
    listenHttpServer(httpsServer, bindings.httpsPort)
  ];
  if (proxyOnlyServer) {
    listeners.push(listenHttpServer(proxyOnlyServer, bindings.proxyOnlyPort ?? 0));
  }
  await Promise.all(listeners);
  return {
    httpServer,
    httpsServer,
    proxyOnlyServer,
    close: async () => {
      await Promise.all([closeServer(httpServer), closeServer(httpsServer), proxyOnlyServer ? closeServer(proxyOnlyServer) : Promise.resolve()]);
    }
  };
}

export function createHttpProxyServer(input: ProxyRouteContext, liveState = false, onProxyActivity?: () => void, proxyOnly = false) {
  const server = http.createServer((request, response) => {
    onProxyActivity?.();
    proxyHttpRequest(input, liveState, request, response, proxyOnly).catch(() => {
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
    const route = resolveRoute({ ...context, target: `${target.host}:${target.port}`, protocol: 'connect', proxyOnly });
    if (route.mode === 'deny') {
      clientSocket.end(connectErrorResponse(route));
      return;
    }
    if (route.mode === 'proxy' && !route.topology) {
      clientSocket.end('HTTP/1.1 502 Bad Gateway\r\n\r\n');
      return;
    }
    const token = route.mode === 'proxy' ? await proxyToken() : undefined;
    if (route.mode === 'proxy' && !token) {
      clientSocket.end('HTTP/1.1 407 Proxy Authentication Required\r\n\r\nproxy_auth_required');
      return;
    }
    const upstream = route.mode === 'proxy' && route.topology ? {
      host: route.topology.entryHost,
      port: route.topology.entryPort
    } : target;
    const upstreamSocket = net.connect(upstream.port, upstream.host, async () => {
      if (route.mode === 'proxy') {
        upstreamSocket.write(connectRequest(target.host, target.port, token));
        try {
          const proxyResponse = await readConnectProxyResponse(upstreamSocket);
          if (proxyResponse.statusCode !== 200) {
            clientSocket.end('HTTP/1.1 502 Bad Gateway\r\n\r\nbad_connect_response');
            upstreamSocket.destroy();
            return;
          }
          clientSocket.write(proxyResponse.header);
          pipeTunnel(clientSocket, upstreamSocket, head, proxyResponse.remaining);
        } catch {
          clientSocket.end('HTTP/1.1 502 Bad Gateway\r\n\r\nbad_connect_response');
          upstreamSocket.destroy();
        }
        return;
      } else {
        clientSocket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
      }
      pipeTunnel(clientSocket, upstreamSocket, head);
    });
    upstreamSocket.on('error', () => clientSocket.destroy());
    clientSocket.on('error', () => upstreamSocket.destroy());
  });

  return server;
}

async function proxyHttpRequest(input: ProxyRouteContext, liveState: boolean, request: http.IncomingMessage, response: http.ServerResponse, proxyOnly: boolean) {
  const target = parseHttpTarget(request);
  if (!target) {
    response.writeHead(400);
    response.end();
    return;
  }
  const context = await proxyContext(input, liveState);
  const route = resolveRoute({ ...context, target: target.url, protocol: target.protocol, proxyOnly });
  if (route.mode === 'deny') {
    writeRouteDenied(response, route);
    return;
  }
  if (route.mode === 'proxy' && !route.topology) {
    response.writeHead(502);
    response.end();
    return;
  }
  const headers = { ...request.headers };
  delete headers['proxy-connection'];
  delete headers['proxy-authorization'];
  if (route.mode === 'proxy') {
    const token = await proxyToken();
    if (!token) {
      response.writeHead(407, { 'content-type': 'text/plain' });
      response.end('proxy_auth_required');
      return;
    }
    headers['proxy-authorization'] = `Bearer ${token}`;
  }
  const upstreamOptions = {
    host: route.mode === 'proxy' && route.topology ? route.topology.entryHost : target.host,
    port: route.mode === 'proxy' && route.topology ? route.topology.entryPort : target.port,
    method: request.method,
    path: route.mode === 'proxy' ? target.url : target.path,
    headers
  };
  try {
    await streamHttpProxyRequest(request, response, upstreamOptions);
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

function pipeTunnel(clientSocket: Duplex, upstreamSocket: Duplex, clientHead: Uint8Array, upstreamHead: Uint8Array = Buffer.alloc(0)) {
  if (upstreamHead.length > 0) {
    clientSocket.write(upstreamHead);
  }
  if (clientHead.length > 0) {
    upstreamSocket.write(clientHead);
  }
  clientSocket.pipe(upstreamSocket);
  upstreamSocket.pipe(clientSocket);
}

function readConnectProxyResponse(socket: net.Socket): Promise<{ statusCode: number; header: Buffer; remaining: Buffer }> {
  return new Promise((resolve, reject) => {
    let buffered = Buffer.alloc(0);
    const cleanup = () => {
      socket.off('data', onData);
      socket.off('error', onError);
      socket.off('end', onEnd);
    };
    const onData = (chunk: Buffer) => {
      buffered = Buffer.concat([buffered, chunk]);
      const headerEnd = buffered.indexOf('\r\n\r\n');
      if (headerEnd < 0) {
        if (buffered.length > 65536) {
          cleanup();
          reject(new Error('connect_response_too_large'));
        }
        return;
      }
      cleanup();
      const header = buffered.subarray(0, headerEnd + 4);
      resolve({
        statusCode: connectStatusCode(header),
        header,
        remaining: buffered.subarray(headerEnd + 4)
      });
    };
    const onError = (error: Error) => {
      cleanup();
      reject(error);
    };
    const onEnd = () => {
      cleanup();
      reject(new Error('connect_response_closed'));
    };
    socket.on('data', onData);
    socket.once('error', onError);
    socket.once('end', onEnd);
  });
}

function connectStatusCode(header: Buffer) {
  const line = header.toString('ascii', 0, header.indexOf('\r\n')).trim();
  const match = /^HTTP\/\d(?:\.\d)?\s+(\d{3})/.exec(line);
  return match ? Number(match[1]) : 0;
}

function streamHttpProxyRequest(request: http.IncomingMessage, response: http.ServerResponse, upstreamOptions: http.RequestOptions): Promise<void> {
  return new Promise((resolve, reject) => {
    let settled = false;
    const settle = (error?: Error) => {
      if (settled) {
        return;
      }
      settled = true;
      error ? reject(error) : resolve();
    };
    const upstreamRequest = http.request(upstreamOptions, (upstreamResponse) => {
      response.writeHead(upstreamResponse.statusCode ?? 502, upstreamResponse.headers);
      if (request.method === 'HEAD') {
        upstreamResponse.resume();
        response.end();
        settle();
        return;
      }
      upstreamResponse.pipe(response);
      upstreamResponse.on('error', settle);
      upstreamResponse.on('end', () => settle());
    });
    upstreamRequest.on('error', settle);
    request.on('aborted', () => {
      upstreamRequest.destroy();
      settle(new Error('request_aborted'));
    });
    request.pipe(upstreamRequest);
  });
}

function connectErrorResponse(route: RouteResult) {
  if (route.denyReason === 'access_path_unavailable' || route.denyReason === 'node_unavailable') {
    return `HTTP/1.1 502 Bad Gateway\r\n\r\n${route.denyReason}`;
  }
  return `HTTP/1.1 403 Forbidden\r\n\r\n${route.denyReason || 'route_denied'}`;
}

function writeRouteDenied(response: http.ServerResponse, route: RouteResult) {
  if (route.denyReason === 'access_path_unavailable' || route.denyReason === 'node_unavailable') {
    response.writeHead(502, { 'content-type': 'text/plain' });
    response.end(route.denyReason);
    return;
  }
  response.writeHead(403, { 'content-type': 'text/plain' });
  response.end(route.denyReason || 'route_denied');
}
