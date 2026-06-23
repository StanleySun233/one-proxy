import * as http from 'node:http';
import * as net from 'node:net';
import { closeServer, listenHttpServer, readConfig, readState, readTokens } from "./lifecycle.js";
import { resolveRoute } from "./router.js";
const maxRetryBufferBytes = 8 * 1024 * 1024;
export async function startHttpProxyListeners(input, bindings, liveState = false, onProxyActivity) {
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
export function createHttpProxyServer(input, liveState = false, onProxyActivity, proxyOnly = false) {
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
                }
                catch {
                    clientSocket.end('HTTP/1.1 502 Bad Gateway\r\n\r\nbad_connect_response');
                    upstreamSocket.destroy();
                }
                return;
            }
            else {
                clientSocket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
            }
            pipeTunnel(clientSocket, upstreamSocket, head);
        });
        upstreamSocket.on('error', () => clientSocket.destroy());
        clientSocket.on('error', () => upstreamSocket.destroy());
    });
    return server;
}
async function proxyHttpRequest(input, liveState, request, response, proxyOnly) {
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
        if (isRetryableProxyRequest(request.method)) {
            await retryingBufferedHttpProxyRequest(request, response, upstreamOptions);
            return;
        }
        await streamHttpProxyRequest(request, response, upstreamOptions);
    }
    catch {
        if (!response.headersSent) {
            response.writeHead(502);
        }
        response.end();
    }
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
function pipeTunnel(clientSocket, upstreamSocket, clientHead, upstreamHead = Buffer.alloc(0)) {
    if (upstreamHead.length > 0) {
        clientSocket.write(upstreamHead);
    }
    if (clientHead.length > 0) {
        upstreamSocket.write(clientHead);
    }
    clientSocket.pipe(upstreamSocket);
    upstreamSocket.pipe(clientSocket);
}
function readConnectProxyResponse(socket) {
    return new Promise((resolve, reject) => {
        let buffered = Buffer.alloc(0);
        const cleanup = () => {
            socket.off('data', onData);
            socket.off('error', onError);
            socket.off('end', onEnd);
        };
        const onData = (chunk) => {
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
        const onError = (error) => {
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
function connectStatusCode(header) {
    const line = header.toString('ascii', 0, header.indexOf('\r\n')).trim();
    const match = /^HTTP\/\d(?:\.\d)?\s+(\d{3})/.exec(line);
    return match ? Number(match[1]) : 0;
}
function streamHttpProxyRequest(request, response, upstreamOptions) {
    return new Promise((resolve, reject) => {
        let settled = false;
        const settle = (error) => {
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
async function retryingBufferedHttpProxyRequest(request, response, upstreamOptions) {
    const body = await readRequestBody(request);
    const method = (request.method || '').toUpperCase();
    const isHead = method === 'HEAD';
    let lastError = null;
    for (let attempt = 1; attempt <= 2; attempt += 1) {
        try {
            const result = await collectOrStreamHttpProxyResponse(upstreamOptions, body, response, isHead, attempt === 1);
            if (result.streamed) {
                return;
            }
            if (attempt === 1 && isRetryableProxyStatus(result.statusCode)) {
                continue;
            }
            const headers = bufferedResponseHeaders(result.headers, result.body.length, isHead);
            response.writeHead(result.statusCode, headers);
            if (isHead) {
                response.end();
            }
            else {
                response.end(result.body);
            }
            return;
        }
        catch (error) {
            lastError = error;
            if (attempt === 2) {
                throw lastError;
            }
        }
    }
    throw lastError ?? new Error('proxy_request_failed');
}
function collectOrStreamHttpProxyResponse(upstreamOptions, body, response, skipLengthCheck, allowRetry) {
    return new Promise((resolve, reject) => {
        let settled = false;
        const settle = (error, result) => {
            if (settled) {
                return;
            }
            settled = true;
            error ? reject(error) : resolve(result ?? { streamed: true });
        };
        const upstreamRequest = http.request(upstreamOptions, (upstreamResponse) => {
            const chunks = [];
            let totalBytes = 0;
            const statusCode = upstreamResponse.statusCode ?? 502;
            const cleanupBufferedHandlers = () => {
                upstreamResponse.off('aborted', onAborted);
                upstreamResponse.off('error', onError);
                upstreamResponse.off('end', onEnd);
                upstreamResponse.off('data', onData);
            };
            const onAborted = () => {
                cleanupBufferedHandlers();
                settle(new Error('upstream_response_aborted'));
            };
            const onError = (error) => {
                cleanupBufferedHandlers();
                settle(error);
            };
            const onEnd = () => {
                cleanupBufferedHandlers();
                const responseBody = Buffer.concat(chunks);
                const expectedLength = expectedContentLength(upstreamResponse.headers);
                if (!skipLengthCheck && expectedLength !== null && expectedLength !== responseBody.length) {
                    settle(new Error('upstream_content_length_mismatch'));
                    return;
                }
                settle(null, {
                    streamed: false,
                    statusCode,
                    headers: upstreamResponse.headers,
                    body: responseBody
                });
            };
            const streamOverflow = () => {
                cleanupBufferedHandlers();
                response.writeHead(statusCode, upstreamResponse.headers);
                for (const chunk of chunks) {
                    response.write(chunk);
                }
                upstreamResponse.once('aborted', () => {
                    response.destroy(new Error('upstream_response_aborted'));
                    settle(new Error('upstream_response_aborted'));
                });
                upstreamResponse.once('error', (error) => {
                    response.destroy(error);
                    settle(error);
                });
                upstreamResponse.once('end', () => settle(null, { streamed: true }));
                upstreamResponse.pipe(response);
            };
            const onData = (chunk) => {
                chunks.push(Buffer.from(chunk));
                totalBytes += chunk.length;
                if (totalBytes <= maxRetryBufferBytes || skipLengthCheck) {
                    return;
                }
                if (allowRetry && isRetryableProxyStatus(statusCode)) {
                    cleanupBufferedHandlers();
                    upstreamResponse.destroy();
                    settle(null, {
                        streamed: false,
                        statusCode,
                        headers: upstreamResponse.headers,
                        body: Buffer.alloc(0)
                    });
                    return;
                }
                streamOverflow();
            };
            upstreamResponse.on('data', onData);
            upstreamResponse.once('aborted', onAborted);
            upstreamResponse.once('error', onError);
            upstreamResponse.once('end', onEnd);
        });
        upstreamRequest.once('error', (error) => settle(error));
        upstreamRequest.end(body);
    });
}
function readRequestBody(request) {
    return new Promise((resolve, reject) => {
        const chunks = [];
        request.on('data', (chunk) => chunks.push(Buffer.from(chunk)));
        request.once('aborted', () => reject(new Error('request_aborted')));
        request.once('error', reject);
        request.once('end', () => resolve(Buffer.concat(chunks)));
    });
}
function isRetryableProxyRequest(method) {
    return new Set(['GET', 'HEAD', 'OPTIONS']).has((method || 'GET').toUpperCase());
}
function isRetryableProxyStatus(statusCode) {
    return statusCode === 502 || statusCode === 503 || statusCode === 504;
}
function expectedContentLength(headers) {
    const value = headers['content-length'];
    if (typeof value !== 'string') {
        return null;
    }
    const parsed = Number(value);
    return Number.isInteger(parsed) && parsed >= 0 ? parsed : null;
}
function bufferedResponseHeaders(headers, bodyLength, preserveContentLength) {
    const next = { ...headers };
    delete next['transfer-encoding'];
    delete next.connection;
    if (!preserveContentLength) {
        next['content-length'] = String(bodyLength);
    }
    return next;
}
function connectErrorResponse(route) {
    if (route.denyReason === 'access_path_unavailable' || route.denyReason === 'node_unavailable') {
        return `HTTP/1.1 502 Bad Gateway\r\n\r\n${route.denyReason}`;
    }
    return `HTTP/1.1 403 Forbidden\r\n\r\n${route.denyReason || 'route_denied'}`;
}
function writeRouteDenied(response, route) {
    if (route.denyReason === 'access_path_unavailable' || route.denyReason === 'node_unavailable') {
        response.writeHead(502, { 'content-type': 'text/plain' });
        response.end(route.denyReason);
        return;
    }
    response.writeHead(403, { 'content-type': 'text/plain' });
    response.end(route.denyReason || 'route_denied');
}
//# sourceMappingURL=http-proxy.js.map