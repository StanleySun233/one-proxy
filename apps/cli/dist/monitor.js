import { spawn } from 'node:child_process';
import * as fs from 'node:fs/promises';
import * as http from 'node:http';
import * as net from 'node:net';
import * as path from 'node:path';
import { allocateLoopbackPort, closeServer, listenHttpServer, loopbackHost } from "./daemon/lifecycle.js";
function monitorLogName(executable, now = new Date()) {
    const stamp = now.toISOString().replace(/[-:]/g, '').replace(/\..+$/, '').replace('T', '-');
    const app = path.basename(executable).replace(/[^a-zA-Z0-9._-]+/g, '_') || 'app';
    return `${stamp}-${app}.log`;
}
async function appendMonitorEvent(logPath, event) {
    await fs.appendFile(logPath, `${JSON.stringify(event)}\n`, { mode: 0o600 });
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
        host,
        port,
        path: `${url.pathname}${url.search}`,
        target: url.toString()
    };
}
async function startMonitorProxy(logPath) {
    const server = http.createServer((request, response) => {
        void proxyHttpRequest(logPath, request, response);
    });
    server.on('connect', (request, clientSocket, head) => {
        void proxyConnect(logPath, request, clientSocket, head);
    });
    const port = await allocateLoopbackPort();
    await listenHttpServer(server, port);
    return {
        port,
        close: () => closeServer(server)
    };
}
async function proxyConnect(logPath, request, clientSocket, head) {
    const target = parseConnectTarget(request.url ?? '');
    if (!target) {
        clientSocket.end('HTTP/1.1 400 Bad Request\r\n\r\n');
        return;
    }
    const upstreamSocket = net.connect(target.port, target.host, () => {
        clientSocket.write('HTTP/1.1 200 Connection Established\r\n\r\n');
        if (head.length > 0) {
            upstreamSocket.write(head);
        }
        clientSocket.pipe(upstreamSocket);
        upstreamSocket.pipe(clientSocket);
        void appendMonitorEvent(logPath, {
            timestamp: new Date().toISOString(),
            protocol: 'connect',
            method: 'CONNECT',
            host: target.host,
            port: target.port,
            target: `${target.host}:${target.port}`,
            status: 200
        });
    });
    upstreamSocket.on('error', () => {
        void appendMonitorEvent(logPath, {
            timestamp: new Date().toISOString(),
            protocol: 'connect',
            method: 'CONNECT',
            host: target.host,
            port: target.port,
            target: `${target.host}:${target.port}`,
            status: 502
        });
        clientSocket.destroy();
    });
    clientSocket.on('error', () => upstreamSocket.destroy());
}
async function proxyHttpRequest(logPath, request, response) {
    const target = parseHttpTarget(request);
    if (!target) {
        response.writeHead(400);
        response.end();
        return;
    }
    const headers = { ...request.headers };
    delete headers['proxy-connection'];
    const upstreamRequest = http.request({
        host: target.host,
        port: target.port,
        method: request.method,
        path: target.path,
        headers
    }, (upstreamResponse) => {
        const status = upstreamResponse.statusCode ?? 502;
        response.writeHead(status, upstreamResponse.headers);
        upstreamResponse.pipe(response);
        upstreamResponse.once('end', () => {
            void appendMonitorEvent(logPath, {
                timestamp: new Date().toISOString(),
                protocol: 'http',
                method: request.method ?? 'GET',
                host: target.host,
                port: target.port,
                target: target.target,
                status
            });
        });
    });
    upstreamRequest.on('error', () => {
        response.writeHead(502);
        response.end();
        void appendMonitorEvent(logPath, {
            timestamp: new Date().toISOString(),
            protocol: 'http',
            method: request.method ?? 'GET',
            host: target.host,
            port: target.port,
            target: target.target,
            status: 502
        });
    });
    request.pipe(upstreamRequest);
}
function monitorEnv(port) {
    const proxy = `http://${loopbackHost}:${port}`;
    return {
        HTTP_PROXY: proxy,
        HTTPS_PROXY: proxy,
        ALL_PROXY: proxy,
        http_proxy: proxy,
        https_proxy: proxy,
        all_proxy: proxy,
        NO_PROXY: 'localhost,127.0.0.1,::1',
        no_proxy: 'localhost,127.0.0.1,::1',
        ONEPROXY_MONITOR_ACTIVE: '1',
        ONEPROXY_MONITOR_PORT: String(port)
    };
}
export async function monitorCommand(args, _context) {
    const executable = args[0];
    if (!executable) {
        throw Object.assign(new Error('monitor requires a command.'), { code: 'COMMAND_NOT_FOUND', exitCode: 2 });
    }
    const logPath = path.resolve(process.cwd(), monitorLogName(executable));
    await fs.writeFile(logPath, '', { flag: 'a', mode: 0o600 });
    const proxy = await startMonitorProxy(logPath);
    process.stderr.write(`onep monitor: writing ${logPath}\n`);
    const child = spawn(executable, args.slice(1), {
        shell: process.platform === 'win32',
        stdio: 'inherit',
        windowsHide: false,
        env: {
            ...process.env,
            ...monitorEnv(proxy.port)
        }
    });
    return new Promise((resolve, reject) => {
        child.once('error', (error) => {
            proxy.close().then(() => {
                if (error.code === 'ENOENT') {
                    reject(Object.assign(new Error(`Command not found: ${executable}`), { code: 'COMMAND_NOT_FOUND' }));
                    return;
                }
                reject(error);
            }, reject);
        });
        child.once('exit', (code, signal) => {
            proxy.close().then(() => {
                if (signal) {
                    resolve(1);
                    return;
                }
                resolve(code ?? 0);
            }, reject);
        });
    });
}
export const monitorInternals = {
    monitorLogName,
    startMonitorProxy
};
//# sourceMappingURL=monitor.js.map