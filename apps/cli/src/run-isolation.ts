import { execFile } from 'node:child_process';
import { spawn } from 'node:child_process';
import * as fs from 'node:fs/promises';
import * as fsSync from 'node:fs';
import * as path from 'node:path';
import { promisify } from 'node:util';

const execFileAsync = promisify(execFile);
const cgroupRoot = '/sys/fs/cgroup';

export type IsolatedRunInput = {
  executable: string;
  args: string[];
  env: NodeJS.ProcessEnv;
  proxyPort: number;
};

type FirewallRule = {
  command: 'iptables' | 'ip6tables';
  add: string[];
  remove: string[];
};

type ProxyIsolationReason =
  | 'non_linux'
  | 'not_root'
  | 'missing_cgroup_v2'
  | 'missing_iptables'
  | 'cgroup_create_failed'
  | 'firewall_install_failed';

export async function runProxyOnlyIsolatedCommand(input: IsolatedRunInput): Promise<number> {
  await requireProxyIsolationSupport();
  const cgroup = await createRunCgroup().catch((error) => {
    throw proxyIsolationUnavailable('cgroup_create_failed', 'onep run could not create cgroup isolation.', error);
  });
  const rules = firewallRules(input.proxyPort, cgroup.relativePath);
  const installed: FirewallRule[] = [];
  try {
    for (const rule of [...rules].reverse()) {
      await execFileAsync(rule.command, rule.add).catch((error) => {
        throw proxyIsolationUnavailable('firewall_install_failed', 'onep run could not install proxy isolation firewall rules.', error);
      });
      installed.push(rule);
    }
    return await spawnInCgroup(input, cgroup.path);
  } finally {
    await killCgroupProcesses(cgroup.path);
    for (const rule of installed) {
      await execFileAsync(rule.command, rule.remove).catch(() => {});
    }
    await fs.rm(cgroup.path, { recursive: true, force: true });
  }
}

export async function runProxyOnlyBestEffortCommand(input: IsolatedRunInput): Promise<number> {
  const child = spawn(input.executable, input.args, {
    stdio: 'inherit',
    env: input.env
  });
  if (!child.pid) {
    throw Object.assign(new Error('Unable to start command.'), { code: 'COMMAND_NOT_FOUND' });
  }
  const signalHandlers = installSignalHandlers(child.pid);
  try {
    return await waitForChild(child);
  } finally {
    for (const [signal, handler] of signalHandlers) {
      process.off(signal, handler);
    }
  }
}

export function isProxyIsolationUnavailable(error: unknown): boolean {
  return typeof error === 'object' && error !== null && 'code' in error && (error as { code?: unknown }).code === 'PROXY_ISOLATION_REQUIRED';
}

export function proxyIsolationHelp(error: unknown): string[] {
  if (!isProxyIsolationUnavailable(error)) {
    return [];
  }
  const reason = (error as { reason?: ProxyIsolationReason }).reason;
  switch (reason) {
    case 'not_root':
      return ['For strict isolation, run: sudo onep run <command...>'];
    case 'missing_iptables':
      return [
        'Install iptables/ip6tables for strict isolation:',
        '  Debian/Ubuntu: sudo apt-get install iptables',
        '  Fedora/RHEL: sudo dnf install iptables iptables-nft',
        '  Arch: sudo pacman -S iptables'
      ];
    case 'missing_cgroup_v2':
      return [
        'Enable cgroup v2 at /sys/fs/cgroup for strict isolation.',
        'cgroup v2 is a Linux host capability, not a OneProxy package dependency.',
        'On systemd Linux, /sys/fs/cgroup should be mounted as cgroup2fs.'
      ];
    case 'cgroup_create_failed':
      return ['Check that cgroup v2 is mounted at /sys/fs/cgroup and writable by root.'];
    case 'firewall_install_failed':
      return [
        'Check that iptables/ip6tables and the cgroup match module are available.',
        '  Debian/Ubuntu: sudo apt-get install iptables',
        '  Fedora/RHEL: sudo dnf install iptables iptables-nft',
        '  Arch: sudo pacman -S iptables'
      ];
    case 'non_linux':
      return ['Strict isolation is only available on Linux.'];
    default:
      return [];
  }
}

async function requireProxyIsolationSupport(): Promise<void> {
  if (process.platform !== 'linux') {
    throw proxyIsolationUnavailable('non_linux', 'onep run requires Linux proxy isolation.');
  }
  if (typeof process.getuid !== 'function' || process.getuid() !== 0) {
    throw proxyIsolationUnavailable('not_root', 'onep run requires root so direct network egress can be blocked.');
  }
  if (!fsSync.existsSync(path.join(cgroupRoot, 'cgroup.controllers'))) {
    throw proxyIsolationUnavailable('missing_cgroup_v2', 'onep run requires cgroup v2 at /sys/fs/cgroup.');
  }
  await Promise.all([
    execFileAsync('iptables', ['--version']),
    execFileAsync('ip6tables', ['--version'])
  ]).catch((error) => {
    throw proxyIsolationUnavailable('missing_iptables', 'onep run requires iptables and ip6tables for proxy isolation.', error);
  });
}

function proxyIsolationUnavailable(reason: ProxyIsolationReason, message: string, cause?: unknown): Error {
  return Object.assign(new Error(message), { code: 'PROXY_ISOLATION_REQUIRED', exitCode: 2, reason, cause });
}

async function createRunCgroup(): Promise<{ path: string; relativePath: string }> {
  const relativePath = `oneproxy-run-${process.pid}-${Date.now()}`;
  const cgroupPath = path.join(cgroupRoot, relativePath);
  await fs.mkdir(cgroupPath, { mode: 0o700 });
  return { path: cgroupPath, relativePath };
}

function firewallRules(proxyPort: number, cgroupPath: string): FirewallRule[] {
  return [
    rule('iptables', [
      '-p', 'tcp',
      '-m', 'cgroup', '--path', cgroupPath,
      '-d', '127.0.0.1/32',
      '--dport', String(proxyPort),
      '-j', 'ACCEPT'
    ]),
    rule('iptables', [
      '-m', 'cgroup', '--path', cgroupPath,
      '-j', 'REJECT'
    ]),
    rule('ip6tables', [
      '-m', 'cgroup', '--path', cgroupPath,
      '-j', 'REJECT'
    ])
  ];
}

function rule(command: FirewallRule['command'], match: string[]): FirewallRule {
  return {
    command,
    add: ['-I', 'OUTPUT', '1', ...match],
    remove: ['-D', 'OUTPUT', ...match]
  };
}

async function spawnInCgroup(input: IsolatedRunInput, cgroupPath: string): Promise<number> {
  const child = spawn('/bin/sh', ['-c', 'kill -STOP $$; exec "$@"', 'onep-run-child', input.executable, ...input.args], {
    stdio: 'inherit',
    env: input.env,
    ...originalUserIdentity()
  });
  if (!child.pid) {
    throw Object.assign(new Error('Unable to start isolated command.'), { code: 'COMMAND_NOT_FOUND' });
  }
  const signalHandlers = installSignalHandlers(child.pid);
  try {
    await fs.writeFile(path.join(cgroupPath, 'cgroup.procs'), String(child.pid));
    process.kill(child.pid, 'SIGCONT');
    return await waitForChild(child);
  } catch (error) {
    try {
      process.kill(child.pid, 'SIGKILL');
    } catch {}
    throw error;
  } finally {
    for (const [signal, handler] of signalHandlers) {
      process.off(signal, handler);
    }
  }
}

function installSignalHandlers(childPid: number): Array<[NodeJS.Signals, () => void]> {
  const signals: NodeJS.Signals[] = ['SIGINT', 'SIGTERM', 'SIGHUP'];
  return signals.map((signal) => {
    const handler = () => {
      try {
        process.kill(childPid, signal);
      } catch {}
    };
    process.on(signal, handler);
    return [signal, handler];
  });
}

function waitForChild(child: ReturnType<typeof spawn>): Promise<number> {
  return new Promise((resolve, reject) => {
    child.once('error', (error: NodeJS.ErrnoException) => {
      if (error.code === 'ENOENT') {
        reject(Object.assign(new Error(`Command not found: ${child.spawnfile}`), { code: 'COMMAND_NOT_FOUND' }));
        return;
      }
      reject(error);
    });
    child.once('exit', (code, signal) => resolve(signal ? 1 : code ?? 0));
  });
}

async function killCgroupProcesses(cgroupPath: string): Promise<void> {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    const pids = await readCgroupPids(cgroupPath);
    if (pids.length === 0) {
      return;
    }
    for (const pid of pids) {
      try {
        process.kill(pid, 'SIGKILL');
      } catch {}
    }
    await new Promise((resolve) => setTimeout(resolve, 25));
  }
  const pids = await readCgroupPids(cgroupPath);
  if (pids.length > 0) {
    throw Object.assign(new Error('Unable to stop isolated child processes; firewall rules were left installed to prevent direct egress.'), { code: 'PROXY_ISOLATION_CLEANUP_FAILED' });
  }
}

async function readCgroupPids(cgroupPath: string): Promise<number[]> {
  try {
    const body = await fs.readFile(path.join(cgroupPath, 'cgroup.procs'), 'utf8');
    return body
      .split(/\s+/)
      .map((value) => Number(value))
      .filter((pid) => Number.isInteger(pid) && pid > 0);
  } catch {
    return [];
  }
}

function originalUserIdentity(): { uid?: number; gid?: number } {
  if (typeof process.getuid !== 'function' || process.getuid() !== 0) {
    return {};
  }
  const uid = numericEnv('SUDO_UID');
  const gid = numericEnv('SUDO_GID');
  if (uid === undefined || gid === undefined || uid === 0 || gid === 0) {
    return {};
  }
  return { uid, gid };
}

function numericEnv(key: string): number | undefined {
  const value = process.env[key];
  if (!value || !/^\d+$/.test(value)) {
    return undefined;
  }
  return Number(value);
}

export const runIsolationInternals = {
  isProxyIsolationUnavailable,
  proxyIsolationHelp,
  firewallRules
};
