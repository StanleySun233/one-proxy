import { spawn } from 'node:child_process';
import * as fs from 'node:fs/promises';
import * as http from 'node:http';
import * as net from 'node:net';
import * as path from 'node:path';
import type { CliContext } from './main.ts';
import { allocateLoopbackPort, closeServer, listenHttpServer, loopbackHost } from './daemon/lifecycle.ts';

type MonitorProxy = {
  port: number;
  close: () => Promise<void>;
};

const monitorIdleTimeoutSeconds = 300;

type MonitorLogEvent = {
  timestamp: string;
  protocol: 'http' | 'connect';
  method: string;
  host: string;
  port: number;
  target: string;
  status: number;
};

function monitorLogName(executable: string, now = new Date()) {
  const stamp = now.toISOString().replace(/[-:]/g, '').replace(/\..+$/, '').replace('T', '-');
  const app = path.basename(executable).replace(/[^a-zA-Z0-9._-]+/g, '_') || 'app';
  return `${stamp}-${app}.log`;
}

async function appendMonitorEvent(logPath: string, event: MonitorLogEvent) {
  await fs.appendFile(logPath, `${JSON.stringify(event)}\n`, { mode: 0o600 });
  process.stderr.write(`onep monitor: ${event.protocol} ${event.method} ${event.host}:${event.port} ${event.status} ${event.target}\n`);
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
    host,
    port,
    path: `${url.pathname}${url.search}`,
    target: url.toString()
  };
}

async function startMonitorProxy(logPath: string, onActivity?: () => void): Promise<MonitorProxy> {
  const server = http.createServer((request, response) => {
    onActivity?.();
    void proxyHttpRequest(logPath, request, response);
  });
  server.on('connect', (request, clientSocket, head) => {
    onActivity?.();
    void proxyConnect(logPath, request, clientSocket as net.Socket, head);
  });
  const port = await allocateLoopbackPort();
  await listenHttpServer(server, port);
  return {
    port,
    close: () => closeServer(server)
  };
}

async function proxyConnect(logPath: string, request: http.IncomingMessage, clientSocket: net.Socket, head: Buffer) {
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

async function proxyHttpRequest(logPath: string, request: http.IncomingMessage, response: http.ServerResponse) {
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

function monitorEnv(port: number): Record<string, string> {
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

function quoteWindowsCommand(value: string): string {
  return `"${value.replace(/"/g, '""')}"`;
}

function spawnWindowsCommand(executable: string, args: string[], env: NodeJS.ProcessEnv) {
  const shell = process.env.ComSpec || 'cmd.exe';
  const command = ['start', '""', '/wait', quoteWindowsCommand(executable), ...args.map(quoteWindowsCommand)].join(' ');
  return spawn(shell, ['/d', '/s', '/c', command], {
    stdio: 'inherit',
    windowsHide: false,
    env
  });
}

async function waitForMonitorIdle(lastActivity: () => number) {
  while (Date.now() - lastActivity() < monitorIdleTimeoutSeconds * 1000) {
    const remaining = monitorIdleTimeoutSeconds * 1000 - (Date.now() - lastActivity());
    await new Promise((resolve) => setTimeout(resolve, Math.min(remaining, 1000)));
  }
}

export async function monitorCommand(args: string[], _context: CliContext): Promise<number> {
  const executable = args[0];
  if (!executable) {
    throw Object.assign(new Error('monitor requires a command.'), { code: 'COMMAND_NOT_FOUND', exitCode: 2 });
  }
  const logPath = path.resolve(process.cwd(), monitorLogName(executable));
  await fs.writeFile(logPath, '', { flag: 'a', mode: 0o600 });
  let lastActivity = Date.now();
  const proxy = await startMonitorProxy(logPath, () => {
    lastActivity = Date.now();
  });
  process.stderr.write(`onep monitor: writing ${logPath}\n`);
  const env = {
    ...process.env,
    ...monitorEnv(proxy.port)
  };
  const child = process.platform === 'win32'
    ? spawnWindowsCommand(executable, args.slice(1), env)
    : spawn(executable, args.slice(1), {
      stdio: 'inherit',
      env
    });
  return new Promise((resolve, reject) => {
    child.once('error', (error: NodeJS.ErrnoException) => {
      proxy.close().then(() => {
        if (error.code === 'ENOENT') {
          reject(Object.assign(new Error(`Command not found: ${executable}`), { code: 'COMMAND_NOT_FOUND' }));
          return;
        }
        reject(error);
      }, reject);
    });
    child.once('exit', (code, signal) => {
      process.stderr.write(`onep monitor: command exited, stopping after ${monitorIdleTimeoutSeconds}s without requests\n`);
      waitForMonitorIdle(() => lastActivity).then(() => proxy.close()).then(() => {
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
