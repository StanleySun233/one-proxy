import * as http from 'node:http';
import type { CliContext } from './main.ts';
import { ensureSessionProxyBindings } from './session-env.ts';
import {
  processIsRunning,
  readConfig,
  readDaemonMetadata,
  readState,
  readTokens,
  writeConfig,
  type DaemonMetadata,
} from './storage.ts';

type ErrorResult = {
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
};

type RouteResult = {
  target: string;
  host: string;
  port: number;
  mode: 'direct' | 'proxy';
  matched: {
    source: 'local_override_direct' | 'local_override_proxy' | 'policy' | 'default_direct' | 'proxy_only';
    ruleId?: string;
    ruleType?: string;
    pattern?: string;
  };
  tenant: {
    id?: string;
    name?: string;
  };
  group: {
    id?: string;
    name?: string;
  };
  topology: null | {
    entryNodeId: string;
    entryHost: string;
    entryPort: number;
    protocol: string;
  };
};

type DoctorResult = {
  summary: {
    status: 'pass' | 'warn' | 'fail';
    passed: number;
    warned: number;
    failed: number;
  };
  checks: Array<{
    name: string;
    status: 'pass' | 'warn' | 'fail';
    message: string;
    action?: string;
  }>;
};

function write(value: unknown, context: CliContext): void {
  if (context.json) {
    process.stdout.write(`${JSON.stringify(value, null, 2)}\n`);
    return;
  }
  process.stdout.write(`${String(value)}\n`);
}

export function writeError(error: ErrorResult['error'], context: CliContext): void {
  if (context.json) {
    process.stderr.write(`${JSON.stringify({ error }, null, 2)}\n`);
    return;
  }
  process.stderr.write(`Error: ${error.message}\n`);
}

function normalizeHost(host: string): string {
  return host.trim().toLowerCase();
}

function parseTarget(raw: string): { target: string; host: string; port: number; protocol: string } {
  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(raw) ? raw : `http://${raw}`;
  const url = new URL(withScheme);
  if (!url.hostname) {
    throw Object.assign(new Error(`Invalid target: ${raw}`), { code: 'INVALID_TARGET' });
  }
  const protocol = url.protocol.replace(':', '');
  const defaultPort = protocol === 'https' ? 443 : protocol === 'ssh' ? 22 : 80;
  return {
    target: raw,
    host: normalizeHost(url.hostname),
    port: url.port ? Number(url.port) : defaultPort,
    protocol
  };
}

async function postIpc<T>(daemon: DaemonMetadata | null, path: string, body: unknown): Promise<T> {
  if (!daemon?.bindings.ipcPort || !processIsRunning(daemon.pid)) {
    throw Object.assign(new Error('Daemon is unavailable'), { code: 'DAEMON_UNAVAILABLE' });
  }
  const payload = JSON.stringify(body);
  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        host: daemon.bindings.host,
        port: daemon.bindings.ipcPort,
        path,
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(payload)
        },
        timeout: 1000
      },
      (res) => {
        const chunks: Buffer[] = [];
        res.on('data', (chunk: Buffer) => chunks.push(chunk));
        res.on('end', () => {
          resolve(JSON.parse(Buffer.concat(chunks).toString('utf8')) as T);
        });
      }
    );
    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(Object.assign(new Error('Daemon IPC request timed out'), { code: 'DAEMON_UNAVAILABLE' }));
    });
    req.end(payload);
  });
}

function routeText(route: RouteResult): string {
  const lines = [
    `Target: ${route.target}`,
    `Host: ${route.host}:${route.port}`,
    `Mode: ${route.mode}`,
    `Matched: ${route.matched.source}${route.matched.pattern ? ` ${route.matched.pattern}` : ''}`,
    `Tenant: ${route.tenant.name || route.tenant.id || 'none'}`,
    `Group: ${route.group.name || route.group.id || 'none'}`
  ];
  if (route.topology) {
    lines.push(`Entry: ${route.topology.entryHost}:${route.topology.entryPort} (${route.topology.protocol})`);
  }
  return lines.join('\n');
}

export async function statusCommand(_args: string[], context: CliContext): Promise<void> {
  const [config, tokens, state, daemon] = await Promise.all([readConfig(), readTokens(), readState(), readDaemonMetadata()]);
  const running = Boolean(daemon?.pid && processIsRunning(daemon.pid));
  const ports = running && daemon ? daemon.bindings : null;
  const result = {
    account: tokens?.account ?? null,
    controlPlane: {
      url: config.controlPlaneUrl ?? null,
      reachable: Boolean(config.controlPlaneUrl)
    },
    tenant: {
      id: config.activeTenantId ?? null,
      name: null
    },
    group: {
      id: config.activeGroupId ?? state.bootstrap?.groupId ?? null,
      name: state.routeGroups.find((group) => group.id === config.activeGroupId)?.name ?? null
    },
    daemon: {
      running,
      pid: running ? daemon?.pid : null,
      startedAt: running ? daemon?.startedAt : null,
      lastHeartbeatAt: running ? daemon?.lastHeartbeatAt : null
    },
    localPorts: {
      http: ports?.httpPort ?? null,
      https: ports?.httpsPort ?? null,
      proxyOnly: ports?.proxyOnlyPort ?? null,
      ipc: ports?.ipcPort ?? null
    },
    portSelection: daemon && running ? daemon.portSelection ?? null : null,
    policyRevision: state.policyRevision ?? daemon?.policyRevision ?? null,
    tokens: {
      accessTokenExpiresAt: tokens?.accessTokenExpiresAt ?? null,
      refreshTokenExpiresAt: tokens?.refreshTokenExpiresAt ?? null,
      proxyTokenExpiresAt: tokens?.proxyTokenExpiresAt ?? null
    },
    overrides: {
      directCount: config.overrides.direct.length,
      proxyCount: config.overrides.proxy.length
    }
  };
  if (context.json) {
    write(result, context);
    return;
  }
  write(
    [
      `Account: ${tokens?.account?.email || tokens?.account?.account || tokens?.account?.id || 'not logged in'}`,
      `Control plane: ${config.controlPlaneUrl || 'not configured'}`,
      `Tenant: ${result.tenant.id || 'none'}`,
      `Group: ${result.group.name || result.group.id || 'none'}`,
      `Daemon: ${running ? `running pid ${daemon?.pid}` : 'not running'}`,
      `Ports: http=${result.localPorts.http || 'unset'} https=${result.localPorts.https || 'unset'}`,
      `Policy: ${result.policyRevision || 'none'}`,
      `Overrides: direct=${result.overrides.directCount} proxy=${result.overrides.proxyCount}`
    ].join('\n'),
    context
  );
}

export async function overrideCommand(args: string[], context: CliContext): Promise<void> {
  const config = await readConfig();
  const action = args[0];
  if (action === 'list') {
    write(context.json ? { overrides: config.overrides } : `Direct:\n${config.overrides.direct.join('\n')}\nProxy:\n${config.overrides.proxy.join('\n')}`, context);
    return;
  }
  if (action === 'clear') {
    await writeConfig({ ...config, overrides: { direct: [], proxy: [] } });
    write(context.json ? { cleared: true } : 'Overrides cleared.', context);
    return;
  }
  if (action === 'remove') {
    const host = normalizeHost(args[1] || '');
    if (!host) {
      throw Object.assign(new Error('override remove requires a host.'), { code: 'COMMAND_NOT_FOUND', exitCode: 2 });
    }
    await writeConfig({
      ...config,
      overrides: {
        direct: config.overrides.direct.filter((item) => item !== host),
        proxy: config.overrides.proxy.filter((item) => item !== host)
      }
    });
    write(context.json ? { removed: host } : `Removed override: ${host}`, context);
    return;
  }
  const mode = action;
  const subaction = args[1];
  const host = normalizeHost(args[2] || '');
  if ((mode !== 'direct' && mode !== 'proxy') || subaction !== 'add' || !host) {
    throw Object.assign(new Error('override requires list, clear, remove <host>, direct add <host>, or proxy add <host>.'), {
      code: 'COMMAND_NOT_FOUND',
      exitCode: 2
    });
  }
  const next = {
    direct: config.overrides.direct.filter((item) => item !== host),
    proxy: config.overrides.proxy.filter((item) => item !== host)
  };
  next[mode] = [...next[mode], host];
  await writeConfig({ ...config, overrides: next });
  write(context.json ? { added: { mode, host } } : `Added ${mode} override: ${host}`, context);
}

export async function routeCommand(args: string[], context: CliContext): Promise<void> {
  const target = args[0];
  if (!target) {
    throw Object.assign(new Error('route requires a URL or host.'), { code: 'INVALID_TARGET', exitCode: 2 });
  }
  await ensureSessionProxyBindings();
  const daemon = await readDaemonMetadata();
  const route = await postIpc<RouteResult>(daemon, '/v1/route', { target, protocol: parseTarget(target).protocol });
  write(context.json ? route : routeText(route), context);
}

export async function testCommand(args: string[], context: CliContext): Promise<void> {
  const target = args[0];
  if (!target) {
    throw Object.assign(new Error('test requires a URL or host.'), { code: 'INVALID_TARGET', exitCode: 2 });
  }
  await ensureSessionProxyBindings();
  const daemon = await readDaemonMetadata();
  const result = await postIpc<{ route: RouteResult; probes: Array<{ name: string; status: string; message: string }> }>(daemon, '/v1/probe', { target });
  if (context.json) {
    write(result, context);
    return;
  }
  write(`${routeText(result.route)}\nProbes:\n${result.probes.map((probe) => `${probe.name}: ${probe.status} ${probe.message}`).join('\n')}`, context);
}

export async function doctor(_args: string[], context: CliContext): Promise<number> {
  const { runDoctor } = await import('./doctor.ts');
  const result = await runDoctor() as DoctorResult;
  if (context.json) {
    write(result, context);
  } else {
    write(result.checks.map((check) => `${check.status}\t${check.name}\t${check.message}`).join('\n'), context);
  }
  return result.summary.failed ? 3 : 0;
}
