import { execFile, spawn } from 'node:child_process';
import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import { promisify } from 'node:util';
import type { CliContext } from './main.ts';

const execFileAsync = promisify(execFile);
const monitorIdleTimeoutSeconds = 300;

type ProcessInfo = {
  pid: number;
  parentPid: number;
  name: string;
};

type Endpoint = {
  address: string;
  port: number | null;
};

type MonitorLogEvent = {
  timestamp: string;
  source: 'netstat' | 'procfs' | 'lsof';
  process: string;
  pid: number;
  protocol: 'tcp' | 'udp';
  localAddress: string;
  localPort: number | null;
  remoteAddress: string;
  remotePort: number | null;
  remoteHost: string;
  domain: string | null;
  domainSource: null;
  state: string | null;
};

type ConnectionSampler = (rootPid: number, trackedPids: Set<number>, seenEvents: Set<string>, logPath: string) => Promise<number>;

const linuxTcpStates: Record<string, string> = {
  '01': 'ESTABLISHED',
  '02': 'SYN_SENT',
  '03': 'SYN_RECV',
  '04': 'FIN_WAIT1',
  '05': 'FIN_WAIT2',
  '06': 'TIME_WAIT',
  '07': 'CLOSE',
  '08': 'CLOSE_WAIT',
  '09': 'LAST_ACK',
  '0A': 'LISTEN',
  '0B': 'CLOSING'
};

function monitorLogName(executable: string, now = new Date()) {
  const stamp = now.toISOString().replace(/[-:]/g, '').replace(/\..+$/, '').replace('T', '-');
  const app = path.basename(executable).replace(/[^a-zA-Z0-9._-]+/g, '_') || 'app';
  return `${stamp}-${app}.log`;
}

async function appendMonitorEvent(logPath: string, event: MonitorLogEvent) {
  await fs.appendFile(logPath, `${JSON.stringify(event)}\n`, { mode: 0o600 });
  process.stderr.write(`onep monitor: ${event.process}[${event.pid}] ${event.protocol} ${event.remoteHost}:${event.remotePort ?? '*'} ${event.state ?? ''}\n`);
}

function quoteWindowsCommand(value: string): string {
  return `"${value.replace(/"/g, '""')}"`;
}

function spawnMonitoredCommand(executable: string, args: string[]) {
  if (process.platform === 'win32') {
    const shell = process.env.ComSpec || 'cmd.exe';
    const command = ['start', '""', '/wait', quoteWindowsCommand(executable), ...args.map(quoteWindowsCommand)].join(' ');
    return spawn(shell, ['/d', '/s', '/c', command], {
      stdio: 'inherit',
      windowsHide: false
    });
  }
  return spawn(executable, args, { stdio: 'inherit' });
}

function parseEndpoint(value: string): Endpoint {
  if (value === '*:*' || value === '*') {
    return { address: '*', port: null };
  }
  if (value.startsWith('[')) {
    const end = value.lastIndexOf(']:');
    if (end >= 0) {
      const port = Number(value.slice(end + 2));
      return { address: value.slice(1, end), port: Number.isInteger(port) ? port : null };
    }
  }
  const index = value.lastIndexOf(':');
  if (index < 0) {
    return { address: value, port: null };
  }
  const port = Number(value.slice(index + 1));
  return {
    address: value.slice(0, index),
    port: Number.isInteger(port) ? port : null
  };
}

function parseWindowsNetstat(output: string): MonitorLogEvent[] {
  const events: MonitorLogEvent[] = [];
  for (const line of output.split(/\r?\n/)) {
    const parts = line.trim().split(/\s+/);
    const protocol = parts[0]?.toLowerCase();
    if (protocol !== 'tcp' && protocol !== 'udp') {
      continue;
    }
    const local = parseEndpoint(parts[1] ?? '');
    const remote = parseEndpoint(parts[2] ?? '');
    const state = protocol === 'tcp' ? parts[3] ?? null : null;
    const pidText = protocol === 'tcp' ? parts[4] : parts[3];
    const pid = Number(pidText);
    if (!Number.isInteger(pid)) {
      continue;
    }
    events.push(baseEvent('netstat', pid, protocol, local, remote, state));
  }
  return events;
}

function parseWindowsProcesses(output: string): ProcessInfo[] {
  const processes: ProcessInfo[] = [];
  for (const line of output.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('Node,')) {
      continue;
    }
    const parts = trimmed.split(',');
    const name = parts[1];
    const parentPid = Number(parts[2]);
    const pid = Number(parts[3]);
    if (!name || !Number.isInteger(parentPid) || !Number.isInteger(pid)) {
      continue;
    }
    processes.push({ name, parentPid, pid });
  }
  return processes;
}

function watchedProcesses(rootPid: number, previousPids: Set<number>, processes: ProcessInfo[]) {
  const watched = new Set<number>([rootPid, ...previousPids]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const processInfo of processes) {
      if (watched.has(processInfo.parentPid) && !watched.has(processInfo.pid)) {
        watched.add(processInfo.pid);
        changed = true;
      }
    }
  }
  return watched;
}

async function windowsProcesses(): Promise<ProcessInfo[]> {
  const { stdout } = await execFileAsync('wmic', ['process', 'get', 'Name,ParentProcessId,ProcessId', '/format:csv'], { windowsHide: true, maxBuffer: 1024 * 1024 * 8 });
  return parseWindowsProcesses(stdout);
}

async function windowsConnections(): Promise<MonitorLogEvent[]> {
  const { stdout } = await execFileAsync('netstat', ['-ano'], { windowsHide: true, maxBuffer: 1024 * 1024 * 16 });
  return parseWindowsNetstat(stdout);
}

function baseEvent(source: MonitorLogEvent['source'], pid: number, protocol: 'tcp' | 'udp', local: Endpoint, remote: Endpoint, state: string | null): MonitorLogEvent {
  return {
    timestamp: new Date().toISOString(),
    source,
    process: '',
    pid,
    protocol,
    localAddress: local.address,
    localPort: local.port,
    remoteAddress: remote.address,
    remotePort: remote.port,
    remoteHost: remote.address,
    domain: null,
    domainSource: null,
    state
  };
}

function eventKey(event: MonitorLogEvent) {
  return [
    event.pid,
    event.protocol,
    event.localAddress,
    event.localPort ?? '*',
    event.remoteAddress,
    event.remotePort ?? '*',
    event.state ?? ''
  ].join('|');
}

async function sampleWindowsConnections(rootPid: number, trackedPids: Set<number>, seenEvents: Set<string>, logPath: string) {
  const processes = await windowsProcesses();
  const watched = watchedProcesses(rootPid, trackedPids, processes);
  for (const pid of watched) {
    trackedPids.add(pid);
  }
  const processNames = new Map(processes.map((processInfo) => [processInfo.pid, processInfo.name]));
  return writeNewEvents(await windowsConnections(), watched, processNames, seenEvents, logPath);
}

async function linuxProcesses(): Promise<ProcessInfo[]> {
  const processes: ProcessInfo[] = [];
  for (const entry of await fs.readdir('/proc')) {
    const pid = Number(entry);
    if (!Number.isInteger(pid)) {
      continue;
    }
    const stat = await fs.readFile(`/proc/${pid}/stat`, 'utf8').catch(() => '');
    const end = stat.lastIndexOf(')');
    if (end < 0) {
      continue;
    }
    const name = stat.slice(stat.indexOf('(') + 1, end);
    const fields = stat.slice(end + 2).split(/\s+/);
    const parentPid = Number(fields[1]);
    if (Number.isInteger(parentPid)) {
      processes.push({ pid, parentPid, name });
    }
  }
  return processes;
}

function parseLinuxIpv4(hex: string) {
  const parts = hex.match(/../g) ?? [];
  return parts.reverse().map((part) => Number.parseInt(part, 16)).join('.');
}

function parseLinuxIpv6(hex: string) {
  const groups = hex.match(/.{8}/g) ?? [];
  return groups.map((group) => {
    const bytes = group.match(/../g) ?? [];
    return [
      `${bytes[1] ?? '00'}${bytes[0] ?? '00'}`,
      `${bytes[3] ?? '00'}${bytes[2] ?? '00'}`
    ].join(':');
  }).join(':').replace(/(^|:)0(:0)+(:|$)/, '::');
}

function parseLinuxEndpoint(value: string, family: 'ipv4' | 'ipv6'): Endpoint {
  const [addressHex, portHex] = value.split(':');
  const port = Number.parseInt(portHex ?? '', 16);
  return {
    address: family === 'ipv4' ? parseLinuxIpv4(addressHex ?? '') : parseLinuxIpv6(addressHex ?? ''),
    port: Number.isInteger(port) ? port : null
  };
}

async function linuxSocketInodes(pids: Set<number>): Promise<Map<string, number>> {
  const inodes = new Map<string, number>();
  for (const pid of pids) {
    const fdDir = `/proc/${pid}/fd`;
    const entries = await fs.readdir(fdDir).catch(() => []);
    for (const entry of entries) {
      const target = await fs.readlink(path.join(fdDir, entry)).catch(() => '');
      const match = /^socket:\[(\d+)\]$/.exec(target);
      if (match) {
        inodes.set(match[1], pid);
      }
    }
  }
  return inodes;
}

async function parseLinuxConnectionFile(filePath: string, protocol: 'tcp' | 'udp', family: 'ipv4' | 'ipv6', inodes: Map<string, number>): Promise<MonitorLogEvent[]> {
  const content = await fs.readFile(filePath, 'utf8').catch(() => '');
  const events: MonitorLogEvent[] = [];
  for (const line of content.split(/\r?\n/).slice(1)) {
    const parts = line.trim().split(/\s+/);
    if (parts.length < 10) {
      continue;
    }
    const pid = inodes.get(parts[9]);
    if (!pid) {
      continue;
    }
    const local = parseLinuxEndpoint(parts[1], family);
    const remote = parseLinuxEndpoint(parts[2], family);
    const state = protocol === 'tcp' ? linuxTcpStates[(parts[3] ?? '').toUpperCase()] ?? parts[3] : null;
    if ((remote.address === '0.0.0.0' || remote.address === '::') && remote.port === 0) {
      continue;
    }
    events.push(baseEvent('procfs', pid, protocol, local, remote, state));
  }
  return events;
}

async function linuxConnections(pids: Set<number>): Promise<MonitorLogEvent[]> {
  const inodes = await linuxSocketInodes(pids);
  return [
    ...await parseLinuxConnectionFile('/proc/net/tcp', 'tcp', 'ipv4', inodes),
    ...await parseLinuxConnectionFile('/proc/net/udp', 'udp', 'ipv4', inodes),
    ...await parseLinuxConnectionFile('/proc/net/tcp6', 'tcp', 'ipv6', inodes),
    ...await parseLinuxConnectionFile('/proc/net/udp6', 'udp', 'ipv6', inodes)
  ];
}

async function sampleLinuxConnections(rootPid: number, trackedPids: Set<number>, seenEvents: Set<string>, logPath: string) {
  const processes = await linuxProcesses();
  const watched = watchedProcesses(rootPid, trackedPids, processes);
  for (const pid of watched) {
    trackedPids.add(pid);
  }
  const processNames = new Map(processes.map((processInfo) => [processInfo.pid, processInfo.name]));
  return writeNewEvents(await linuxConnections(watched), watched, processNames, seenEvents, logPath);
}

async function darwinProcesses(): Promise<ProcessInfo[]> {
  const { stdout } = await execFileAsync('ps', ['-axo', 'pid=,ppid=,comm='], { maxBuffer: 1024 * 1024 * 8 });
  const processes: ProcessInfo[] = [];
  for (const line of stdout.split(/\r?\n/)) {
    const match = /^\s*(\d+)\s+(\d+)\s+(.+)$/.exec(line);
    if (match) {
      processes.push({ pid: Number(match[1]), parentPid: Number(match[2]), name: path.basename(match[3]) });
    }
  }
  return processes;
}

function parseLsofEndpoint(value: string): { local: Endpoint; remote: Endpoint; state: string | null } | null {
  const stateMatch = /\(([^)]+)\)$/.exec(value);
  const clean = value.replace(/\s+\([^)]+\)$/, '');
  const [localText, remoteText] = clean.split('->');
  if (!remoteText) {
    return null;
  }
  return {
    local: parseEndpoint(localText),
    remote: parseEndpoint(remoteText),
    state: stateMatch?.[1] ?? null
  };
}

function parseLsof(output: string): MonitorLogEvent[] {
  const events: MonitorLogEvent[] = [];
  for (const line of output.split(/\r?\n/).slice(1)) {
    const parts = line.trim().split(/\s+/);
    const pid = Number(parts[1]);
    if (!Number.isInteger(pid)) {
      continue;
    }
    const node = parts[7]?.toLowerCase();
    const protocol = node === 'tcp' || node === 'udp' ? node : null;
    if (!protocol) {
      continue;
    }
    const endpoint = parseLsofEndpoint(parts.slice(8).join(' '));
    if (!endpoint) {
      continue;
    }
    const event = baseEvent('lsof', pid, protocol, endpoint.local, endpoint.remote, endpoint.state);
    event.process = parts[0] ?? '';
    events.push(event);
  }
  return events;
}

async function sampleDarwinConnections(rootPid: number, trackedPids: Set<number>, seenEvents: Set<string>, logPath: string) {
  const processes = await darwinProcesses();
  const watched = watchedProcesses(rootPid, trackedPids, processes);
  for (const pid of watched) {
    trackedPids.add(pid);
  }
  const { stdout } = await execFileAsync('lsof', ['-nP', '-iTCP', '-iUDP'], { maxBuffer: 1024 * 1024 * 16 });
  const processNames = new Map(processes.map((processInfo) => [processInfo.pid, processInfo.name]));
  return writeNewEvents(parseLsof(stdout), watched, processNames, seenEvents, logPath);
}

async function writeNewEvents(events: MonitorLogEvent[], watched: Set<number>, processNames: Map<number, string>, seenEvents: Set<string>, logPath: string) {
  let eventCount = 0;
  for (const event of events) {
    if (!watched.has(event.pid)) {
      continue;
    }
    const key = eventKey(event);
    if (seenEvents.has(key)) {
      continue;
    }
    seenEvents.add(key);
    event.process = processNames.get(event.pid) ?? (event.process || `pid-${event.pid}`);
    await appendMonitorEvent(logPath, event);
    eventCount += 1;
  }
  return eventCount;
}

function connectionSampler(): ConnectionSampler {
  if (process.platform === 'win32') {
    return sampleWindowsConnections;
  }
  if (process.platform === 'linux') {
    return sampleLinuxConnections;
  }
  if (process.platform === 'darwin') {
    return sampleDarwinConnections;
  }
  throw Object.assign(new Error(`onep monitor does not support ${process.platform}.`), { code: 'PLATFORM_UNSUPPORTED', exitCode: 2 });
}

async function waitForMonitor(rootPid: number, logPath: string, childExit: () => boolean) {
  const trackedPids = new Set<number>();
  const seenEvents = new Set<string>();
  const sampleConnections = connectionSampler();
  let lastActivity = Date.now();
  while (!childExit() || Date.now() - lastActivity < monitorIdleTimeoutSeconds * 1000) {
    const count = await sampleConnections(rootPid, trackedPids, seenEvents, logPath);
    if (count > 0) {
      lastActivity = Date.now();
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
}

export async function monitorCommand(args: string[], _context: CliContext): Promise<number> {
  const executable = args[0];
  if (!executable) {
    throw Object.assign(new Error('monitor requires a command.'), { code: 'COMMAND_NOT_FOUND', exitCode: 2 });
  }
  const logPath = path.resolve(process.cwd(), monitorLogName(executable));
  await fs.writeFile(logPath, '', { flag: 'a', mode: 0o600 });
  process.stderr.write(`onep monitor: writing ${logPath}\n`);
  const child = spawnMonitoredCommand(executable, args.slice(1));
  if (!child.pid) {
    throw Object.assign(new Error('Unable to determine monitor launcher pid.'), { code: 'MONITOR_UNAVAILABLE' });
  }
  let exited = false;
  return new Promise((resolve, reject) => {
    child.once('error', reject);
    child.once('exit', () => {
      exited = true;
      process.stderr.write(`onep monitor: command exited, stopping after ${monitorIdleTimeoutSeconds}s without new connections\n`);
    });
    waitForMonitor(child.pid as number, logPath, () => exited).then(() => {
      resolve(child.exitCode ?? 0);
    }, reject);
  });
}

export const monitorInternals = {
  monitorLogName,
  parseEndpoint,
  parseLsof,
  parseLinuxEndpoint,
  parseWindowsNetstat,
  parseWindowsProcesses,
  watchedProcesses
};
