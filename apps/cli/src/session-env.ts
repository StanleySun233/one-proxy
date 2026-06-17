import type { CliContext } from './main.ts';
import { isProxyIsolationUnavailable, proxyIsolationHelp, runProxyOnlyBestEffortCommand, runProxyOnlyIsolatedCommand } from './run-isolation.ts';
import { detectShellFamily } from './shell-detect.ts';
import {
  readConfig,
  readState,
  type DaemonBindings
} from './storage.ts';

const preservedProxyVariables = [
  'HTTP_PROXY',
  'HTTPS_PROXY',
  'ALL_PROXY',
  'NO_PROXY',
  'http_proxy',
  'https_proxy',
  'all_proxy',
  'no_proxy'
];

const unsetMarker = '__ONEPROXY_UNSET__';

type LifecycleModule = {
  ensureDaemon?: () => Promise<{ metadata: { bindings: DaemonBindings } }>;
  startDaemonSession?: () => Promise<{ metadata: { bindings: DaemonBindings }; end: () => Promise<void> }>;
};

async function lifecycleBindings(): Promise<DaemonBindings | null> {
  const lifecycle = (await import('./daemon/lifecycle.ts')) as LifecycleModule;
  const result = await lifecycle.ensureDaemon?.();
  if (!result) {
    throw Object.assign(new Error('Daemon lifecycle is unavailable'), { code: 'DAEMON_UNAVAILABLE' });
  }
  return result.metadata.bindings;
}

export async function ensureSessionProxyBindings(): Promise<DaemonBindings> {
  const lifecycle = await lifecycleBindings();
  if (!lifecycle?.httpPort || !lifecycle.httpsPort) {
    throw Object.assign(new Error('Daemon did not return proxy bindings'), { code: 'DAEMON_UNAVAILABLE' });
  }
  return lifecycle;
}

export function proxyEnv(bindings: DaemonBindings, extraNoProxyHosts: string[] = []): Record<string, string> {
  const httpProxy = `http://${bindings.host}:${bindings.httpPort}`;
  const httpsProxy = `http://${bindings.host}:${bindings.httpsPort}`;
  const noProxy = noProxyValue(extraNoProxyHosts);
  return {
    HTTP_PROXY: httpProxy,
    HTTPS_PROXY: httpsProxy,
    ALL_PROXY: httpProxy,
    NO_PROXY: noProxy,
    http_proxy: httpProxy,
    https_proxy: httpsProxy,
    all_proxy: httpProxy,
    no_proxy: noProxy,
    ONEPROXY_ACTIVE: '1',
    ONEPROXY_HTTP_PORT: String(bindings.httpPort),
    ONEPROXY_HTTPS_PORT: String(bindings.httpsPort)
  };
}

export function proxyOnlyEnv(bindings: DaemonBindings): Record<string, string> {
  if (!bindings.proxyOnlyPort) {
    throw Object.assign(new Error('Daemon did not return proxy-only binding'), { code: 'DAEMON_UNAVAILABLE' });
  }
  const proxy = `http://${bindings.host}:${bindings.proxyOnlyPort}`;
  return {
    HTTP_PROXY: proxy,
    HTTPS_PROXY: proxy,
    ALL_PROXY: proxy,
    NO_PROXY: '',
    http_proxy: proxy,
    https_proxy: proxy,
    all_proxy: proxy,
    no_proxy: '',
    ONEPROXY_ACTIVE: '1',
    ONEPROXY_PROXY_ONLY: '1',
    ONEPROXY_HTTP_PORT: String(bindings.proxyOnlyPort),
    ONEPROXY_HTTPS_PORT: String(bindings.proxyOnlyPort)
  };
}

export async function sessionProxyEnv(bindings?: DaemonBindings): Promise<Record<string, string>> {
  const resolvedBindings = bindings ?? await ensureSessionProxyBindings();
  const env = proxyEnv(resolvedBindings);
  const noProxy = noProxyValue(await proxyBypassHosts());
  return {
    ...env,
    NO_PROXY: noProxy,
    no_proxy: noProxy
  };
}

function noProxyValue(extraHosts: string[] = []): string {
  return [...new Set(['localhost', '127.0.0.1', '::1', ...extraHosts].map((item) => item.trim()).filter(Boolean))].join(',');
}

async function proxyBypassHosts(): Promise<string[]> {
  const [config, state] = await Promise.all([readConfig(), readState()]);
  return [
    hostnameFromUrl(config.controlPlaneUrl),
    ...(state.accessPaths ?? []).map((accessPath) => accessPath.listenHost)
  ].filter(Boolean);
}

function hostnameFromUrl(value: string | undefined): string {
  if (!value) {
    return '';
  }
  try {
    return new URL(value).hostname;
  } catch {
    return '';
  }
}

function quotePosix(value: string): string {
  return `'${value.replace(/'/g, "'\\''")}'`;
}

function quoteFish(value: string): string {
  return `'${value.replace(/\\/g, '\\\\').replace(/'/g, "\\'")}'`;
}

function posixOn(env: Record<string, string>): string {
  const lines: string[] = [];
  for (const key of preservedProxyVariables) {
    lines.push(`if [ -z "\${${key}+x}" ]; then export ONEPROXY_PREV_${key}=${quotePosix(unsetMarker)}; else export ONEPROXY_PREV_${key}="\$${key}"; fi`);
  }
  for (const [key, value] of Object.entries(env)) {
    lines.push(`export ${key}=${quotePosix(value)}`);
  }
  return `${lines.join('\n')}\n`;
}

function posixOff(): string {
  const lines: string[] = [];
  for (const key of preservedProxyVariables) {
    lines.push(`if [ "\${ONEPROXY_PREV_${key}-}" = ${quotePosix(unsetMarker)} ]; then unset ${key}; elif [ -n "\${ONEPROXY_PREV_${key}+x}" ]; then export ${key}="\$ONEPROXY_PREV_${key}"; fi`);
  }
  lines.push('unset ONEPROXY_ACTIVE ONEPROXY_HTTP_PORT ONEPROXY_HTTPS_PORT');
  for (const key of preservedProxyVariables) {
    lines.push(`unset ONEPROXY_PREV_${key}`);
  }
  return `${lines.join('\n')}\n`;
}

function fishOn(env: Record<string, string>): string {
  const lines: string[] = [];
  for (const key of preservedProxyVariables) {
    lines.push(`if set -q ${key}; set -gx ONEPROXY_PREV_${key} $${key}; else; set -gx ONEPROXY_PREV_${key} ${quoteFish(unsetMarker)}; end`);
  }
  for (const [key, value] of Object.entries(env)) {
    lines.push(`set -gx ${key} ${quoteFish(value)}`);
  }
  return `${lines.join('\n')}\n`;
}

function fishOff(): string {
  const lines: string[] = [];
  for (const key of preservedProxyVariables) {
    lines.push(`if test "$ONEPROXY_PREV_${key}" = ${quoteFish(unsetMarker)}; set -e ${key}; else if set -q ONEPROXY_PREV_${key}; set -gx ${key} "$ONEPROXY_PREV_${key}"; end`);
  }
  lines.push('set -e ONEPROXY_ACTIVE ONEPROXY_HTTP_PORT ONEPROXY_HTTPS_PORT');
  for (const key of preservedProxyVariables) {
    lines.push(`set -e ONEPROXY_PREV_${key}`);
  }
  return `${lines.join('\n')}\n`;
}

function powershellOn(env: Record<string, string>): string {
  const lines: string[] = [];
  for (const key of preservedProxyVariables) {
    lines.push(`if (Test-Path Env:${key}) { Set-Item Env:ONEPROXY_PREV_${key} (Get-Item Env:${key}).Value } else { Set-Item Env:ONEPROXY_PREV_${key} ${JSON.stringify(unsetMarker)} }`);
  }
  for (const [key, value] of Object.entries(env)) {
    lines.push(`Set-Item Env:${key} ${JSON.stringify(value)}`);
  }
  return `${lines.join('\n')}\n`;
}

function powershellOff(): string {
  const lines: string[] = [];
  for (const key of preservedProxyVariables) {
    lines.push(`if ((Get-Item Env:ONEPROXY_PREV_${key} -ErrorAction SilentlyContinue).Value -eq ${JSON.stringify(unsetMarker)}) { Remove-Item Env:${key} -ErrorAction SilentlyContinue } elseif (Test-Path Env:ONEPROXY_PREV_${key}) { Set-Item Env:${key} (Get-Item Env:ONEPROXY_PREV_${key}).Value }`);
  }
  lines.push('Remove-Item Env:ONEPROXY_ACTIVE,Env:ONEPROXY_HTTP_PORT,Env:ONEPROXY_HTTPS_PORT -ErrorAction SilentlyContinue');
  for (const key of preservedProxyVariables) {
    lines.push(`Remove-Item Env:ONEPROXY_PREV_${key} -ErrorAction SilentlyContinue`);
  }
  return `${lines.join('\n')}\n`;
}

function cmdOn(env: Record<string, string>): string {
  const lines: string[] = [];
  for (const key of preservedProxyVariables) {
    lines.push(`if defined ${key} (set "ONEPROXY_PREV_${key}=%${key}%") else (set "ONEPROXY_PREV_${key}=${unsetMarker}")`);
  }
  for (const [key, value] of Object.entries(env)) {
    lines.push(`set "${key}=${value}"`);
  }
  return `${lines.join('\r\n')}\r\n`;
}

function cmdOff(): string {
  const lines: string[] = [];
  for (const key of preservedProxyVariables) {
    lines.push(`if "%ONEPROXY_PREV_${key}%"=="${unsetMarker}" (set "${key}=") else (set "${key}=%ONEPROXY_PREV_${key}%")`);
  }
  lines.push('set "ONEPROXY_ACTIVE="');
  lines.push('set "ONEPROXY_HTTP_PORT="');
  lines.push('set "ONEPROXY_HTTPS_PORT="');
  for (const key of preservedProxyVariables) {
    lines.push(`set "ONEPROXY_PREV_${key}="`);
  }
  return `${lines.join('\r\n')}\r\n`;
}

function activationScript(env: Record<string, string>): string {
  switch (detectShellFamily()) {
    case 'fish':
      return fishOn(env);
    case 'powershell':
      return powershellOn(env);
    case 'cmd':
      return cmdOn(env);
    default:
      return posixOn(env);
  }
}

function deactivationScript(): string {
  switch (detectShellFamily()) {
    case 'fish':
      return fishOff();
    case 'powershell':
      return powershellOff();
    case 'cmd':
      return cmdOff();
    default:
      return posixOff();
  }
}

export function parseEnvCommandArgs(argv: string[]) {
  for (const value of argv) {
    throw Object.assign(new Error(`Unknown env option: ${value}`), { code: 'SYNTAX_ERROR', exitCode: 2 });
  }
  return {};
}

export async function envOn(args: string[] = []): Promise<void> {
  parseEnvCommandArgs(args);
  process.stdout.write(activationScript(await sessionProxyEnv()));
}

export async function envOff(args: string[] = []): Promise<void> {
  parseEnvCommandArgs(args);
  process.stdout.write(deactivationScript());
}

export async function runCommand(args: string[], context: CliContext): Promise<number> {
  const parsed = parseRunCommandArgs(args);
  const executable = parsed.args[0];
  if (!executable) {
    throw Object.assign(new Error('run requires a command.'), { code: 'COMMAND_NOT_FOUND', exitCode: 2 });
  }
  const lifecycle = (await import('./daemon/lifecycle.ts')) as LifecycleModule;
  const session = await lifecycle.startDaemonSession?.();
  if (!session) {
    throw Object.assign(new Error('Daemon lifecycle is unavailable'), { code: 'DAEMON_UNAVAILABLE' });
  }
  try {
    const proxyOnlyPort = session.metadata.bindings.proxyOnlyPort;
    if (!proxyOnlyPort) {
      throw Object.assign(new Error('Daemon did not return proxy-only binding'), { code: 'DAEMON_UNAVAILABLE' });
    }
    const runInput = {
      executable,
      args: parsed.args.slice(1),
      env: {
        ...process.env,
        ...proxyOnlyEnv(session.metadata.bindings)
      },
      proxyPort: proxyOnlyPort
    };
    try {
      return await runProxyOnlyIsolatedCommand(runInput);
    } catch (error) {
      if (!isProxyIsolationUnavailable(error)) {
        throw error;
      }
      if (!context.json) {
        process.stderr.write(proxyIsolationFallbackMessage(error));
      }
      return await runProxyOnlyBestEffortCommand(runInput);
    }
  } finally {
    await session.end();
  }
}

function proxyIsolationFallbackMessage(error: unknown): string {
  const lines = [
    'onep run: strict proxy isolation is unavailable; falling back to proxy environment mode.',
    'Programs that ignore HTTP_PROXY/HTTPS_PROXY/ALL_PROXY may bypass OneProxy.',
    ...proxyIsolationHelp(error)
  ];
  return `${lines.join('\n')}\n`;
}

export function parseRunCommandArgs(argv: string[]) {
  return stripTuiFlag(argv);
}

function stripTuiFlag(argv: string[]) {
  const args: string[] = [];
  let tui = false;
  for (const value of argv) {
    if (value === '--tui') {
      tui = true;
    } else {
      args.push(value);
    }
  }
  return { args, tui };
}
