import { spawn } from 'node:child_process';
import type { CliContext } from './main.ts';
import {
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

function proxyEnv(bindings: DaemonBindings): Record<string, string> {
  const httpProxy = `http://${bindings.host}:${bindings.httpPort}`;
  const httpsProxy = `http://${bindings.host}:${bindings.httpsPort}`;
  return {
    HTTP_PROXY: httpProxy,
    HTTPS_PROXY: httpsProxy,
    ALL_PROXY: httpProxy,
    NO_PROXY: 'localhost,127.0.0.1,::1',
    ONEPROXY_ACTIVE: '1',
    ONEPROXY_HTTP_PORT: String(bindings.httpPort),
    ONEPROXY_HTTPS_PORT: String(bindings.httpsPort)
  };
}

function shellFamily(): 'posix' | 'fish' | 'powershell' | 'cmd' {
  const shell = `${process.env.ONEPROXY_SHELL || process.env.SHELL || process.env.ComSpec || ''}`.toLowerCase();
  if (shell.includes('fish')) {
    return 'fish';
  }
  if (shell.includes('powershell') || shell.includes('pwsh')) {
    return 'powershell';
  }
  if (process.platform === 'win32' && shell.includes('cmd')) {
    return 'cmd';
  }
  return 'posix';
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

function activationScript(bindings: DaemonBindings): string {
  const env = proxyEnv(bindings);
  switch (shellFamily()) {
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
  switch (shellFamily()) {
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

export async function envOn(): Promise<void> {
  process.stdout.write(activationScript(await ensureSessionProxyBindings()));
}

export async function envOff(): Promise<void> {
  process.stdout.write(deactivationScript());
}

export async function runCommand(args: string[], _context: CliContext): Promise<number> {
  const executable = args[0];
  if (!executable) {
    throw Object.assign(new Error('run requires a command.'), { code: 'COMMAND_NOT_FOUND', exitCode: 2 });
  }
  const bindings = await ensureSessionProxyBindings();
  const child = spawn(executable, args.slice(1), {
    stdio: 'inherit',
    env: {
      ...process.env,
      ...proxyEnv(bindings)
    }
  });
  return new Promise((resolve, reject) => {
    child.once('error', (error: NodeJS.ErrnoException) => {
      if (error.code === 'ENOENT') {
        reject(Object.assign(new Error(`Command not found: ${executable}`), { code: 'COMMAND_NOT_FOUND' }));
        return;
      }
      reject(error);
    });
    child.once('exit', (code, signal) => {
      if (signal) {
        resolve(1);
        return;
      }
      resolve(code ?? 0);
    });
  });
}
