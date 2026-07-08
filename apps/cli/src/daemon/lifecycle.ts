import * as fs from 'node:fs/promises';
import { spawn } from 'node:child_process';
import { randomBytes, timingSafeEqual } from 'node:crypto';
import * as http from 'node:http';
import * as net from 'node:net';
import * as path from 'node:path';
import { type PortSelection, selectProxyPorts } from './port-selection.ts';
import { profileRoot } from '../storage.ts';

export type LocalOverrides = {
  direct: string[];
  proxy: string[];
};

export type TopologyHop = {
  nodeId: string;
  nodeName: string;
  mode: string;
  scopeKey: string;
  publicHost?: string;
  publicPort?: number;
  transport: string;
};

export type OneProxyConfig = {
  schemaVersion: number;
  controlPlaneUrl?: string;
  activeTenantId?: string;
  activeAccessPathId?: string;
  overrides?: Partial<LocalOverrides>;
};

export type AccessPathSnapshot = {
  id: string;
  name?: string;
  chainId?: string;
  protocol?: string;
  entryNodeId?: string;
  listenHost: string;
  listenPort: number;
  enabled?: boolean;
  topology: TopologyHop[];
};

export type RouteSnapshot = {
  id: string;
  priority: number;
  matchType: 'domain' | 'domain_suffix' | 'ip' | 'ip_cidr' | 'protocol' | 'default';
  matchValue: string;
  actionType: 'chain' | 'direct' | 'deny';
  chainId: string;
  accessPathId: string;
  destinationScope: string;
  enabled: boolean;
  topology: TopologyHop[];
};

export type OneProxyState = {
  schemaVersion: number;
  bootstrap?: {
    tenantId?: string;
    accessPathId?: string;
  };
  policyRevision?: string;
  fetchedAt?: string;
  accessPaths?: AccessPathSnapshot[];
  routes?: RouteSnapshot[];
};

export type OneProxyTokens = {
  schemaVersion: number;
  account?: {
    id: string;
    email: string;
  };
  accessToken?: string;
  refreshToken?: string;
  proxyToken?: string;
  accessTokenExpiresAt?: string;
  refreshTokenExpiresAt?: string;
  proxyTokenExpiresAt?: string;
};

export type DaemonBindings = {
  host: string;
  httpPort: number;
  httpsPort: number;
  ipcPort: number;
  proxyOnlyPort: number;
};

export type DaemonMetadata = {
  schemaVersion: number;
  pid: number;
  startedAt: string;
  lastHeartbeatAt: string;
  controlPlaneUrl?: string;
  tenantId?: string;
  accessPathId?: string;
  policyRevision?: string;
  bindings: DaemonBindings;
  portSelection: PortSelection;
  idleTimeoutSeconds: number;
  persistent: boolean;
  daemonSecret: string;
};

export type DaemonHealth = {
  ok: boolean;
  pid: number;
  startedAt: string;
  lastHeartbeatAt: string;
  bindings: DaemonBindings;
  portSelection: PortSelection;
  policyRevision?: string;
  persistent: boolean;
};

export type DaemonRuntime = {
  metadata: DaemonMetadata;
  ipcServer: http.Server;
  proxyServers?: { close: () => Promise<void> };
  close: () => Promise<void>;
};

export type DaemonRuntimeOptions = {
  persistent?: boolean;
};

export const loopbackHost = '127.0.0.1';
export const envIdleTimeoutSeconds = 600;
export const runIdleTimeoutSeconds = 300;

export function dataRoot() {
  return profileRoot();
}

export function storagePath(name: 'config' | 'state' | 'tokens' | 'daemon' | 'log') {
  const filenames = {
    config: 'config.json',
    state: 'state.json',
    tokens: 'tokens.json',
    daemon: 'daemon.json',
    log: 'onep.log'
  };
  return path.join(dataRoot(), filenames[name]);
}

export async function ensureDataRoot() {
  await fs.mkdir(dataRoot(), { recursive: true, mode: 0o700 });
}

export async function readJsonFile<T>(filePath: string): Promise<T | null> {
  try {
    return JSON.parse(await fs.readFile(filePath, 'utf8')) as T;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
      return null;
    }
    throw error;
  }
}

export async function writeJsonFile(filePath: string, value: unknown, mode = 0o600) {
  await ensureDataRoot();
  await fs.writeFile(filePath, `${JSON.stringify(value, null, 2)}\n`, { mode });
}

export function normalizeConfig(config: OneProxyConfig | null): OneProxyConfig {
  return {
    schemaVersion: 1,
    ...config,
    overrides: {
      direct: (config?.overrides?.direct ?? []).map((host) => host.toLowerCase()),
      proxy: (config?.overrides?.proxy ?? []).map((host) => host.toLowerCase())
    }
  };
}

export async function readConfig() {
  return normalizeConfig(await readJsonFile<OneProxyConfig>(storagePath('config')));
}

export async function readState() {
  return (await readJsonFile<OneProxyState>(storagePath('state'))) ?? { schemaVersion: 1 };
}

export async function readTokens() {
  return await readJsonFile<OneProxyTokens>(storagePath('tokens'));
}

export async function readDaemonMetadata() {
  return await readJsonFile<DaemonMetadata>(storagePath('daemon'));
}

export async function writeDaemonMetadata(metadata: DaemonMetadata) {
  await writeJsonFile(storagePath('daemon'), metadata);
}

export async function appendLog(message: string) {
  await ensureDataRoot();
  await fs.appendFile(storagePath('log'), `${new Date().toISOString()} ${message}\n`, { mode: 0o600 });
}

export async function allocateLoopbackPort(requestedPort = 0): Promise<number> {
  const server = net.createServer();
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(requestedPort, loopbackHost, resolve);
  });
  const address = server.address();
  await new Promise<void>((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()));
  });
  if (!address || typeof address === 'string') {
    throw new Error('Unable to allocate loopback port');
  }
  return address.port;
}

export type ResolvedBindings = {
  bindings: DaemonBindings;
  portSelection: PortSelection;
};

export async function resolveBindings(config?: OneProxyConfig): Promise<ResolvedBindings> {
  const portSelection = await selectProxyPorts();
  return {
    bindings: {
      host: loopbackHost,
      httpPort: portSelection.selectedPair[0],
      httpsPort: portSelection.selectedPair[1],
      ipcPort: await allocateLoopbackPort(),
      proxyOnlyPort: await allocateLoopbackPort()
    },
    portSelection
  };
}

export async function buildDaemonMetadata(resolved: ResolvedBindings, options: DaemonRuntimeOptions = {}): Promise<DaemonMetadata> {
  const [config, state] = await Promise.all([readConfig(), readState()]);
  const now = new Date().toISOString();
  const persistent = options.persistent === true;
  return {
    schemaVersion: 1,
    pid: process.pid,
    startedAt: now,
    lastHeartbeatAt: now,
    controlPlaneUrl: config.controlPlaneUrl,
    tenantId: config.activeTenantId,
    accessPathId: config.activeAccessPathId,
    policyRevision: state.policyRevision,
    bindings: resolved.bindings,
    portSelection: resolved.portSelection,
    idleTimeoutSeconds: persistent ? 0 : envIdleTimeoutSeconds,
    persistent,
    daemonSecret: randomBytes(32).toString('hex')
  };
}

export function healthFromMetadata(metadata: DaemonMetadata): DaemonHealth {
  return {
    ok: true,
    pid: metadata.pid,
    startedAt: metadata.startedAt,
    lastHeartbeatAt: metadata.lastHeartbeatAt,
    bindings: metadata.bindings,
    portSelection: metadata.portSelection,
    policyRevision: metadata.policyRevision,
    persistent: metadata.persistent === true
  };
}

export async function listenHttpServer(server: http.Server, port: number) {
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, loopbackHost, resolve);
  });
}

export async function closeServer(server: http.Server | net.Server) {
  await new Promise<void>((resolve, reject) => {
    server.close((error) => (error ? reject(error) : resolve()));
  });
}

export function createIpcServer(metadata: DaemonMetadata, handlers: Record<string, (body: unknown) => Promise<unknown>>) {
  return http.createServer(async (request, response) => {
    const url = new URL(request.url ?? '/', `http://${loopbackHost}`);
    if (!authorizedDaemonRequest(request, metadata)) {
      writeDaemonAuthRequired(response);
      return;
    }
    if (request.method === 'GET' && url.pathname === '/v1/health') {
      response.setHeader('content-type', 'application/json');
      response.end(JSON.stringify(healthFromMetadata(metadata)));
      return;
    }
    if (request.method === 'POST' && handlers[url.pathname]) {
      const body = await readRequestJson(request);
      const result = await handlers[url.pathname](body);
      response.setHeader('content-type', 'application/json');
      response.end(JSON.stringify(result));
      return;
    }
    response.statusCode = 404;
    response.end();
  });
}

export async function startDaemonRuntime(handlers?: Record<string, (body: unknown) => Promise<unknown>>, options: DaemonRuntimeOptions = {}): Promise<DaemonRuntime> {
  const resolved = await resolveBindings();
  const metadata = await buildDaemonMetadata(resolved, options);
  const [config, state] = await Promise.all([readConfig(), readState()]);
  const { startHttpProxyListeners } = await import('./http-proxy.ts');
  let activeSessions = 0;
  let idleTimeoutSeconds = metadata.idleTimeoutSeconds;
  let lastProxyActivity = Date.now();
  let runtime: DaemonRuntime;
  let shutdownTimer: NodeJS.Timeout | undefined;
  const scheduleIdleCheck = () => {
    if (metadata.persistent || idleTimeoutSeconds <= 0) {
      if (shutdownTimer) {
        clearTimeout(shutdownTimer);
        shutdownTimer = undefined;
      }
      return;
    }
    if (shutdownTimer) {
      clearTimeout(shutdownTimer);
    }
    shutdownTimer = setTimeout(async () => {
      if (activeSessions > 0) {
        scheduleIdleCheck();
        return;
      }
      const idleFor = Date.now() - lastProxyActivity;
      if (idleFor < idleTimeoutSeconds * 1000) {
        scheduleIdleCheck();
        return;
      }
      await runtime.close();
      process.exit(0);
    }, idleTimeoutSeconds * 1000);
    shutdownTimer.unref();
  };
  const proxyServers = await startHttpProxyListeners({ config, state }, metadata.bindings, true, () => {
    lastProxyActivity = Date.now();
    scheduleIdleCheck();
  });
  const enablePersistent = async () => {
    metadata.persistent = true;
    metadata.idleTimeoutSeconds = 0;
    idleTimeoutSeconds = 0;
    if (shutdownTimer) {
      clearTimeout(shutdownTimer);
      shutdownTimer = undefined;
    }
    await writeDaemonMetadata(metadata);
    return { persistent: metadata.persistent, idleTimeoutSeconds: metadata.idleTimeoutSeconds };
  };
  const runtimeHandlers = {
    ...defaultDaemonHandlers(),
    ...handlers,
    '/v1/session/start': async () => {
      activeSessions += 1;
      return { activeSessions };
    },
    '/v1/session/end': async () => {
      activeSessions = Math.max(0, activeSessions - 1);
      if (!metadata.persistent) {
        idleTimeoutSeconds = runIdleTimeoutSeconds;
        metadata.idleTimeoutSeconds = idleTimeoutSeconds;
        await writeDaemonMetadata(metadata);
        scheduleIdleCheck();
      }
      return { activeSessions, idleTimeoutSeconds: metadata.idleTimeoutSeconds, persistent: metadata.persistent };
    },
    '/v1/persistent/on': enablePersistent,
    '/v1/shutdown': async () => {
      setTimeout(() => {
        void runtime.close().finally(() => process.exit(0));
      }, 10).unref();
      return { accepted: true };
    }
  };
  const ipcServer = createIpcServer(metadata, runtimeHandlers);
  await listenHttpServer(ipcServer, metadata.bindings.ipcPort);
  await writeDaemonMetadata(metadata);
  runtime = {
    metadata,
    ipcServer,
    proxyServers,
    close: async () => {
      if (shutdownTimer) {
        clearTimeout(shutdownTimer);
      }
      await Promise.all([closeServer(ipcServer), proxyServers.close()]);
    }
  };
  scheduleIdleCheck();
  return runtime;
}

export function defaultDaemonHandlers(): Record<string, (body: unknown) => Promise<unknown>> {
  return {
    '/v1/route': async (body) => {
      const request = body as { target?: string; protocol?: string };
      const [{ resolveRoute }, config, state] = await Promise.all([
        import('./router.ts'),
        readConfig(),
        readState()
      ]);
      return resolveRoute({
        config,
        state,
        target: request.target ?? '',
        protocol: request.protocol
      });
    },
    '/v1/probe': async (body) => {
      const request = body as { target?: string };
      const { probeTarget } = await import('../doctor.ts');
      return await probeTarget(request.target ?? '');
    },
    '/v1/shutdown-if-idle': async () => ({ accepted: true })
  };
}

export async function startDaemonSession(): Promise<{ metadata: DaemonMetadata; end: () => Promise<void> }> {
  const { metadata } = await ensureDaemon();
  await postDaemon(metadata, '/v1/session/start');
  return {
    metadata,
    end: async () => {
      await postDaemon(metadata, '/v1/session/end');
    }
  };
}

export async function ensureDaemon(options: DaemonRuntimeOptions = {}): Promise<{ metadata: DaemonMetadata }> {
  const existing = await readDaemonMetadata();
  const health = await probeDaemon(existing);
  if (existing && health) {
    if (options.persistent && !existing.persistent) {
      try {
        await postDaemon(existing, '/v1/persistent/on');
        return { metadata: await readDaemonMetadata() ?? { ...existing, persistent: true, idleTimeoutSeconds: 0 } };
      } catch (error) {
        await appendLog(`daemon persistent switch failed: ${(error as Error).message || String(error)}`);
      }
    }
    return { metadata: existing };
  }
  if (process.env.ONEPROXY_DAEMON_CHILD === '1') {
    return { metadata: (await startDaemonRuntime(undefined, { persistent: options.persistent || process.env.ONEPROXY_DAEMON_PERSISTENT === '1' })).metadata };
  }
  const entrypoint = process.argv[1];
  if (!entrypoint) {
    throw Object.assign(new Error('Cannot determine CLI entrypoint'), { code: 'DAEMON_UNAVAILABLE' });
  }
  await appendLog(`starting daemon child from ${entrypoint}`);
  const logFile = await fs.open(storagePath('log'), 'a');
  const child = spawn(process.execPath, [entrypoint, 'daemon', 'serve'], {
    detached: true,
    stdio: ['ignore', logFile.fd, logFile.fd],
    windowsHide: true,
    env: {
      ...process.env,
      ONEPROXY_DAEMON_CHILD: '1',
      ONEPROXY_DAEMON_PERSISTENT: options.persistent ? '1' : '0'
    }
  });
  await logFile.close();
  let childExit: string | null = null;
  child.once('error', (error) => {
    childExit = error.message;
    void appendLog(`daemon child spawn error: ${error.message}`);
  });
  child.once('exit', (code, signal) => {
    childExit = signal ? `signal ${signal}` : `exit ${code}`;
    void appendLog(`daemon child exited before ready: ${childExit}`);
  });
  child.unref();
  const deadline = Date.now() + (process.platform === 'win32' ? 10000 : 3000);
  while (Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, 100));
    const metadata = await readDaemonMetadata();
    if (metadata && await probeDaemon(metadata)) {
      return { metadata };
    }
    if (childExit) {
      throw Object.assign(new Error(`Daemon exited before ready: ${childExit}. See ${storagePath('log')}`), { code: 'DAEMON_UNAVAILABLE' });
    }
  }
  await appendLog('daemon readiness timeout');
  throw Object.assign(new Error(`Daemon did not become ready. See ${storagePath('log')}`), { code: 'DAEMON_UNAVAILABLE' });
}

export async function serveDaemon(): Promise<void> {
  await startDaemonRuntime(undefined, { persistent: process.env.ONEPROXY_DAEMON_PERSISTENT === '1' });
  await new Promise(() => undefined);
}

export async function shutdownDaemon(): Promise<boolean> {
  const metadata = await readDaemonMetadata();
  const health = metadata ? await probeDaemon(metadata) : null;
  if (!metadata || !health) {
    return false;
  }
  try {
    await postDaemon(metadata, '/v1/shutdown');
    return true;
  } catch (error) {
    await appendLog(`daemon shutdown failed: ${(error as Error).message || String(error)}`);
    return false;
  }
}

export async function probeDaemon(metadata?: DaemonMetadata | null): Promise<DaemonHealth | null> {
  const targetMetadata = metadata ?? await readDaemonMetadata();
  if (!targetMetadata?.daemonSecret) {
    return null;
  }
  return await new Promise((resolve) => {
    const request = http.get(
      {
        host: targetMetadata.bindings.host,
        port: targetMetadata.bindings.ipcPort,
        path: '/v1/health',
        headers: daemonSecretHeader(targetMetadata),
        timeout: 1000
      },
      (response) => {
        let body = '';
        response.setEncoding('utf8');
        response.on('data', (chunk) => {
          body += chunk;
        });
        response.on('end', () => {
          resolve(response.statusCode === 200 ? (JSON.parse(body) as DaemonHealth) : null);
        });
      }
    );
    request.on('error', () => resolve(null));
    request.on('timeout', () => {
      request.destroy();
      resolve(null);
    });
  });
}

async function postDaemon(metadata: DaemonMetadata, path: string): Promise<void> {
  await new Promise<void>((resolve, reject) => {
    const request = http.request({
      host: metadata.bindings.host,
      port: metadata.bindings.ipcPort,
      path,
      method: 'POST',
      headers: daemonSecretHeader(metadata),
      timeout: 1000
    }, (response) => {
      response.resume();
      response.on('end', () => {
        if (response.statusCode && response.statusCode >= 400) {
          reject(Object.assign(new Error('Daemon IPC request was rejected'), { code: 'DAEMON_UNAVAILABLE' }));
          return;
        }
        resolve();
      });
    });
    request.on('error', reject);
    request.on('timeout', () => {
      request.destroy();
      reject(Object.assign(new Error('Daemon IPC timeout'), { code: 'DAEMON_UNAVAILABLE' }));
    });
    request.end('{}');
  });
}

function daemonSecretHeader(metadata: DaemonMetadata) {
  return { 'X-One-Proxy-Daemon-Secret': metadata.daemonSecret };
}

function authorizedDaemonRequest(request: http.IncomingMessage, metadata: DaemonMetadata) {
  const header = request.headers['x-one-proxy-daemon-secret'];
  const actual = Array.isArray(header) ? header[0] : header;
  if (!actual) {
    return false;
  }
  const expectedBytes = Buffer.from(metadata.daemonSecret);
  const actualBytes = Buffer.from(actual);
  return actualBytes.length === expectedBytes.length && timingSafeEqual(actualBytes, expectedBytes);
}

function writeDaemonAuthRequired(response: http.ServerResponse) {
  response.writeHead(401, { 'content-type': 'application/json' });
  response.end(JSON.stringify({ code: 401, message: 'daemon_auth_required', data: null }));
}

async function readRequestJson(request: http.IncomingMessage) {
  const chunks: Buffer[] = [];
  for await (const chunk of request) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
  }
  if (chunks.length === 0) {
    return {};
  }
  return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}
