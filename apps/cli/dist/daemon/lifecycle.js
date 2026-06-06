import * as fs from 'node:fs/promises';
import { spawn } from 'node:child_process';
import * as http from 'node:http';
import * as net from 'node:net';
import * as path from 'node:path';
import { selectProxyPorts } from "./port-selection.js";
import { profileRoot } from "../storage.js";
export const loopbackHost = '127.0.0.1';
export const envIdleTimeoutSeconds = 600;
export const runIdleTimeoutSeconds = 300;
export function dataRoot() {
    return profileRoot();
}
export function storagePath(name) {
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
export async function readJsonFile(filePath) {
    try {
        return JSON.parse(await fs.readFile(filePath, 'utf8'));
    }
    catch (error) {
        if (error.code === 'ENOENT') {
            return null;
        }
        throw error;
    }
}
export async function writeJsonFile(filePath, value, mode = 0o600) {
    await ensureDataRoot();
    await fs.writeFile(filePath, `${JSON.stringify(value, null, 2)}\n`, { mode });
}
export function normalizeConfig(config) {
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
    return normalizeConfig(await readJsonFile(storagePath('config')));
}
export async function readState() {
    return (await readJsonFile(storagePath('state'))) ?? { schemaVersion: 1 };
}
export async function readTokens() {
    return await readJsonFile(storagePath('tokens'));
}
export async function readDaemonMetadata() {
    return await readJsonFile(storagePath('daemon'));
}
export async function writeDaemonMetadata(metadata) {
    await writeJsonFile(storagePath('daemon'), metadata);
}
export async function appendLog(message) {
    await ensureDataRoot();
    await fs.appendFile(storagePath('log'), `${new Date().toISOString()} ${message}\n`, { mode: 0o600 });
}
export async function allocateLoopbackPort(requestedPort = 0) {
    const server = net.createServer();
    await new Promise((resolve, reject) => {
        server.once('error', reject);
        server.listen(requestedPort, loopbackHost, resolve);
    });
    const address = server.address();
    await new Promise((resolve, reject) => {
        server.close((error) => (error ? reject(error) : resolve()));
    });
    if (!address || typeof address === 'string') {
        throw new Error('Unable to allocate loopback port');
    }
    return address.port;
}
export async function resolveBindings(config) {
    const portSelection = await selectProxyPorts();
    return {
        bindings: {
            host: loopbackHost,
            httpPort: portSelection.selectedPair[0],
            httpsPort: portSelection.selectedPair[1],
            ipcPort: await allocateLoopbackPort()
        },
        portSelection
    };
}
export async function buildDaemonMetadata(resolved) {
    const [config, state] = await Promise.all([readConfig(), readState()]);
    const now = new Date().toISOString();
    return {
        schemaVersion: 1,
        pid: process.pid,
        startedAt: now,
        lastHeartbeatAt: now,
        controlPlaneUrl: config.controlPlaneUrl,
        tenantId: config.activeTenantId,
        groupId: config.activeGroupId,
        policyRevision: state.policyRevision,
        bindings: resolved.bindings,
        portSelection: resolved.portSelection,
        idleTimeoutSeconds: envIdleTimeoutSeconds
    };
}
export function healthFromMetadata(metadata) {
    return {
        ok: true,
        pid: metadata.pid,
        startedAt: metadata.startedAt,
        lastHeartbeatAt: metadata.lastHeartbeatAt,
        bindings: metadata.bindings,
        portSelection: metadata.portSelection,
        policyRevision: metadata.policyRevision
    };
}
export async function listenHttpServer(server, port) {
    await new Promise((resolve, reject) => {
        server.once('error', reject);
        server.listen(port, loopbackHost, resolve);
    });
}
export async function closeServer(server) {
    await new Promise((resolve, reject) => {
        server.close((error) => (error ? reject(error) : resolve()));
    });
}
export function createIpcServer(metadata, handlers) {
    return http.createServer(async (request, response) => {
        const url = new URL(request.url ?? '/', `http://${loopbackHost}`);
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
export async function startDaemonRuntime(handlers) {
    const resolved = await resolveBindings();
    const metadata = await buildDaemonMetadata(resolved);
    const [config, state] = await Promise.all([readConfig(), readState()]);
    const { startHttpProxyListeners } = await import("./http-proxy.js");
    let activeSessions = 0;
    let idleTimeoutSeconds = envIdleTimeoutSeconds;
    let lastProxyActivity = Date.now();
    let runtime;
    let shutdownTimer;
    const scheduleIdleCheck = () => {
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
    const runtimeHandlers = {
        ...defaultDaemonHandlers(),
        ...handlers,
        '/v1/session/start': async () => {
            activeSessions += 1;
            return { activeSessions };
        },
        '/v1/session/end': async () => {
            activeSessions = Math.max(0, activeSessions - 1);
            idleTimeoutSeconds = runIdleTimeoutSeconds;
            metadata.idleTimeoutSeconds = idleTimeoutSeconds;
            await writeDaemonMetadata(metadata);
            scheduleIdleCheck();
            return { activeSessions, idleTimeoutSeconds };
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
export function defaultDaemonHandlers() {
    return {
        '/v1/route': async (body) => {
            const request = body;
            const [{ resolveRoute }, config, state] = await Promise.all([
                import("./router.js"),
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
            const request = body;
            const { probeTarget } = await import("../doctor.js");
            return await probeTarget(request.target ?? '');
        },
        '/v1/shutdown-if-idle': async () => ({ accepted: true })
    };
}
export async function startDaemonSession() {
    const { metadata } = await ensureDaemon();
    await postDaemon(metadata, '/v1/session/start');
    return {
        metadata,
        end: async () => {
            await postDaemon(metadata, '/v1/session/end');
        }
    };
}
export async function ensureDaemon() {
    const existing = await readDaemonMetadata();
    const health = await probeDaemon(existing);
    if (existing && health) {
        return { metadata: existing };
    }
    if (process.env.ONEPROXY_DAEMON_CHILD === '1') {
        return { metadata: (await startDaemonRuntime()).metadata };
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
            ONEPROXY_DAEMON_CHILD: '1'
        }
    });
    await logFile.close();
    let childExit = null;
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
export async function serveDaemon() {
    await startDaemonRuntime();
    await new Promise(() => undefined);
}
export async function probeDaemon(metadata) {
    const targetMetadata = metadata ?? await readDaemonMetadata();
    if (!targetMetadata) {
        return null;
    }
    return await new Promise((resolve) => {
        const request = http.get({
            host: targetMetadata.bindings.host,
            port: targetMetadata.bindings.ipcPort,
            path: '/v1/health',
            timeout: 1000
        }, (response) => {
            let body = '';
            response.setEncoding('utf8');
            response.on('data', (chunk) => {
                body += chunk;
            });
            response.on('end', () => {
                resolve(response.statusCode === 200 ? JSON.parse(body) : null);
            });
        });
        request.on('error', () => resolve(null));
        request.on('timeout', () => {
            request.destroy();
            resolve(null);
        });
    });
}
async function postDaemon(metadata, path) {
    await new Promise((resolve, reject) => {
        const request = http.request({
            host: metadata.bindings.host,
            port: metadata.bindings.ipcPort,
            path,
            method: 'POST',
            timeout: 1000
        }, (response) => {
            response.resume();
            response.on('end', () => resolve());
        });
        request.on('error', reject);
        request.on('timeout', () => {
            request.destroy();
            reject(Object.assign(new Error('Daemon IPC timeout'), { code: 'DAEMON_UNAVAILABLE' }));
        });
        request.end('{}');
    });
}
async function readRequestJson(request) {
    const chunks = [];
    for await (const chunk of request) {
        chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk));
    }
    if (chunks.length === 0) {
        return {};
    }
    return JSON.parse(Buffer.concat(chunks).toString('utf8'));
}
//# sourceMappingURL=lifecycle.js.map