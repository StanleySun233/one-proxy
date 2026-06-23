import * as dns from 'node:dns/promises';
import * as http from 'node:http';
import * as https from 'node:https';
import * as net from 'node:net';
import { loopbackHost, probeDaemon, readConfig, readDaemonMetadata, readState, readTokens, startDaemonRuntime, storagePath } from "./daemon/lifecycle.js";
import { isUsablePort } from "./daemon/port-selection.js";
import { resolveRoute } from "./daemon/router.js";
export async function probeTarget(target) {
    const [config, state] = await Promise.all([readConfig(), readState()]);
    const route = resolveRoute({ config, state, target });
    const probes = [await dnsProbe(route.host)];
    if (route.mode === 'direct') {
        probes.push(await tcpProbe('direct_connect', route.host, route.port));
    }
    else if (route.topology) {
        probes.push(await tcpProbe('proxy_connect', route.topology.entryHost, route.topology.entryPort));
    }
    const metadata = await readDaemonMetadata();
    if (metadata) {
        probes.push(await tcpProbe('http_proxy', metadata.bindings.host, metadata.bindings.httpPort));
        probes.push(await tcpProbe('https_proxy', metadata.bindings.host, metadata.bindings.httpsPort));
    }
    return { route, probes };
}
export async function runDoctor(routeTarget = 'https://example.com') {
    const checks = [];
    const config = await readConfig().then((value) => {
        checks.push({ name: 'config', status: 'pass', message: 'Config file is readable' });
        return value;
    }, () => {
        checks.push({ name: 'config', status: 'fail', message: 'Config file is not readable', action: `Check ${storagePath('config')}` });
        return null;
    });
    const tokens = await readTokens();
    checks.push(tokens ? { name: 'token_readability', status: 'pass', message: 'Token file is readable' } : {
        name: 'token_readability',
        status: 'fail',
        message: 'Token file is missing',
        action: 'Run onep login'
    });
    const state = await readState();
    checks.push(await controlPlaneHealthCheck(config?.controlPlaneUrl));
    checks.push(bootstrapSyncCheck(state.fetchedAt, state.accessPaths?.length ?? 0, state.routes?.length ?? 0));
    const runtime = await startDaemonRuntime();
    const metadata = runtime.metadata;
    const health = await probeDaemon(metadata);
    checks.push(health ? { name: 'daemon_status', status: 'pass', message: 'Daemon health endpoint is reachable' } : {
        name: 'daemon_status',
        status: 'fail',
        message: 'Daemon health endpoint is not reachable',
        action: 'Run a command that starts the daemon, such as onep env on'
    });
    checks.push(await localPortsCheck(metadata));
    if (config) {
        const route = resolveRoute({ config, state, target: routeTarget });
        checks.push({ name: 'route_calculation', status: 'pass', message: `Route calculation returned ${route.mode}` });
        checks.push(await entryNodeReachabilityCheck(route));
    }
    else {
        checks.push({ name: 'route_calculation', status: 'fail', message: 'Route calculation requires readable config' });
        checks.push({ name: 'entry_node_reachability', status: 'fail', message: 'Entry node reachability requires route calculation' });
    }
    await runtime.close();
    return summarize(checks);
}
async function controlPlaneHealthCheck(controlPlaneUrl) {
    if (!controlPlaneUrl) {
        return { name: 'control_plane_health', status: 'fail', message: 'Control plane URL is not configured', action: 'Run onep login' };
    }
    const reachable = await httpHealth(controlPlaneUrl);
    return reachable ? { name: 'control_plane_health', status: 'pass', message: 'Control plane is reachable' } : {
        name: 'control_plane_health',
        status: 'warn',
        message: 'Control plane health endpoint is not reachable'
    };
}
function bootstrapSyncCheck(fetchedAt, accessPathCount, routeCount) {
    if (!fetchedAt || accessPathCount === 0) {
        return { name: 'bootstrap_sync', status: 'fail', message: 'Bootstrap state is missing access paths', action: 'Run onep sync' };
    }
    return { name: 'bootstrap_sync', status: 'pass', message: `Bootstrap state has ${accessPathCount} access path(s) and ${routeCount} route(s)` };
}
async function localPortsCheck(metadata) {
    if (!metadata) {
        return { name: 'local_ports', status: 'fail', message: 'Daemon metadata is missing' };
    }
    const { httpPort, httpsPort } = metadata.bindings;
    if (httpsPort !== httpPort + 1) {
        return { name: 'local_ports', status: 'fail', message: 'HTTP and HTTPS proxy ports are not consecutive' };
    }
    const [httpOpen, httpsOpen] = await Promise.all([
        tcpOpen(loopbackHost, httpPort),
        tcpOpen(loopbackHost, httpsPort)
    ]);
    if (!httpOpen || !httpsOpen || await isUsablePort(httpPort) || await isUsablePort(httpsPort)) {
        return { name: 'local_ports', status: 'fail', message: 'Local proxy ports are not both listening' };
    }
    return { name: 'local_ports', status: 'pass', message: 'Local HTTP and HTTPS proxy ports are listening' };
}
async function entryNodeReachabilityCheck(route) {
    if (route.mode === 'direct') {
        return { name: 'entry_node_reachability', status: 'pass', message: 'Route is direct; no entry node required' };
    }
    if (!route.topology) {
        return { name: 'entry_node_reachability', status: 'fail', message: 'Proxied route has no entry node', action: 'Run onep sync' };
    }
    const open = await tcpOpen(route.topology.entryHost, route.topology.entryPort);
    return open ? { name: 'entry_node_reachability', status: 'pass', message: 'Entry node is reachable' } : {
        name: 'entry_node_reachability',
        status: 'fail',
        message: 'Entry node is not reachable'
    };
}
async function dnsProbe(host) {
    const started = Date.now();
    try {
        await dns.lookup(host);
        return { name: 'dns', status: 'pass', latencyMs: Date.now() - started, message: `Resolved ${host}` };
    }
    catch {
        return { name: 'dns', status: 'fail', latencyMs: Date.now() - started, message: `Could not resolve ${host}` };
    }
}
async function tcpProbe(name, host, port) {
    const started = Date.now();
    const open = await tcpOpen(host, port);
    return {
        name,
        status: open ? 'pass' : 'fail',
        latencyMs: Date.now() - started,
        message: open ? `Connected to ${host}:${port}` : `Could not connect to ${host}:${port}`
    };
}
async function tcpOpen(host, port) {
    const socket = net.connect({ host, port });
    socket.setTimeout(1000);
    return await new Promise((resolve) => {
        socket.once('connect', () => {
            socket.destroy();
            resolve(true);
        });
        socket.once('error', () => resolve(false));
        socket.once('timeout', () => {
            socket.destroy();
            resolve(false);
        });
    });
}
async function httpHealth(controlPlaneUrl) {
    const url = new URL('/healthz', controlPlaneUrl);
    const client = url.protocol === 'https:' ? https : http;
    return await new Promise((resolve) => {
        const request = client.get(url, (response) => {
            response.resume();
            resolve((response.statusCode ?? 500) < 500);
        });
        request.setTimeout(1500, () => {
            request.destroy();
            resolve(false);
        });
        request.on('error', () => resolve(false));
    });
}
function summarize(checks) {
    const passed = checks.filter((check) => check.status === 'pass').length;
    const warned = checks.filter((check) => check.status === 'warn').length;
    const failed = checks.filter((check) => check.status === 'fail').length;
    return {
        summary: {
            status: failed > 0 ? 'fail' : warned > 0 ? 'warn' : 'pass',
            passed,
            warned,
            failed
        },
        checks
    };
}
//# sourceMappingURL=doctor.js.map