import * as http from 'node:http';
import * as net from 'node:net';
import { closeServer, listenHttpServer, readConfig, readState, readTokens } from "./lifecycle.js";
import { resolveRoute } from "./router.js";
export async function startHttpProxyListeners(input, bindings, liveState = false, onProxyActivity) {
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
export function createHttpProxyServer(input, liveState = false, onProxyActivity) {
    const server = http.createServer((request, response) => {
        onProxyActivity?.();
        proxyHttpRequest(input, liveState, request, response);
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
            }
            else {
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
async function proxyHttpRequest(input, liveState, request, response) {
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
    const upstreamRequest = http.request({
        host: route.mode === 'proxy' && route.topology ? route.topology.entryHost : target.host,
        port: route.mode === 'proxy' && route.topology ? route.topology.entryPort : target.port,
        method: request.method,
        path: route.mode === 'proxy' ? target.url : target.path,
        headers
    }, (upstreamResponse) => {
        response.writeHead(upstreamResponse.statusCode ?? 502, upstreamResponse.headers);
        upstreamResponse.pipe(response);
    });
    upstreamRequest.on('error', () => {
        if (!response.headersSent) {
            response.writeHead(502);
        }
        response.end();
    });
    request.pipe(upstreamRequest);
}
async function proxyContext(input, liveState) {
    return liveState ? { config: await readConfig(), state: await readState() } : input;
}
function parseConnectTarget(value) {
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
function parseHttpTarget(request) {
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
function connectRequest(host, port, token) {
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
//# sourceMappingURL=http-proxy.js.map