import * as fs from 'node:fs/promises';
import * as net from 'node:net';
import * as os from 'node:os';
import * as path from 'node:path';

export type LocalOverrides = {
  direct: string[];
  proxy: string[];
};

export type OneProxyConfig = {
  schemaVersion: number;
  controlPlaneUrl?: string;
  activeTenantId?: string;
  activeGroupId?: string;
  overrides: LocalOverrides;
};

export type Account = {
  id: string;
  email?: string;
  account?: string;
};

export type OneProxyTokens = {
  schemaVersion: number;
  account?: Account;
  accessToken?: string;
  refreshToken?: string;
  proxyToken?: string;
  accessTokenExpiresAt?: string;
  refreshTokenExpiresAt?: string;
  proxyTokenExpiresAt?: string;
};

export type EntryNode = {
  id: string;
  host: string;
  port: number;
  protocol: string;
};

export type RouteRule = {
  id: string;
  type: 'domain' | 'suffix' | 'cidr' | 'wildcard';
  pattern: string;
  mode: 'direct' | 'proxy';
};

export type RouteGroup = {
  id: string;
  tenantId: string;
  name?: string;
  rules: RouteRule[];
};

export type OneProxyState = {
  schemaVersion: number;
  bootstrap?: {
    tenantId?: string;
    groupId?: string;
    entryNodes?: EntryNode[];
  };
  policyRevision?: string;
  fetchedAt?: string;
  routeGroups: RouteGroup[];
};

export type DaemonBindings = {
  host: string;
  httpPort: number;
  httpsPort: number;
  ipcPort?: number;
};

export type DaemonMetadata = {
  schemaVersion: number;
  pid: number;
  startedAt: string;
  lastHeartbeatAt: string;
  controlPlaneUrl?: string;
  tenantId?: string;
  groupId?: string;
  policyRevision?: string;
  bindings: DaemonBindings;
  portSelection?: {
    candidatePorts: number[];
    selectedPair: [number, number];
    excludedCommonPorts: number[];
  };
  idleTimeoutSeconds?: number;
};

export const loopbackHost = '127.0.0.1';

const portRangeStart = 10000;
const portRangeEnd = 60999;
const commonPorts = new Set([
  20, 21, 22, 25, 53, 67, 68, 80, 110, 123, 143, 161, 389, 443, 445, 465, 587, 631, 993, 995,
  1433, 1521, 2049, 2375, 2376, 3000, 3306, 3389, 5000, 5432, 5601, 5672, 5900, 6379, 8000,
  8080, 8443, 9000, 9200, 9300, 11211, 27017
]);

export function oneProxyHome(): string {
  return process.env.ONEPROXY_HOME || path.join(os.homedir(), '.oneproxy');
}

export function storageFile(name: 'config' | 'state' | 'tokens' | 'daemon' | 'log'): string {
  const names = {
    config: 'config.json',
    state: 'state.json',
    tokens: 'tokens.json',
    daemon: 'daemon.json',
    log: 'onep.log'
  };
  return path.join(oneProxyHome(), names[name]);
}

export async function ensureStorageRoot(): Promise<void> {
  await fs.mkdir(oneProxyHome(), { recursive: true, mode: 0o700 });
}

async function readJson<T>(file: string): Promise<T | null> {
  try {
    return JSON.parse(await fs.readFile(file, 'utf8')) as T;
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === 'ENOENT') {
      return null;
    }
    throw error;
  }
}

async function writeJson(file: string, value: unknown, mode = 0o600): Promise<void> {
  await ensureStorageRoot();
  await fs.writeFile(file, `${JSON.stringify(value, null, 2)}\n`, { mode });
  if (process.platform !== 'win32') {
    await fs.chmod(file, mode);
  }
}

function uniqueHosts(items: string[] | undefined): string[] {
  return [...new Set((items ?? []).map((item) => item.trim().toLowerCase()).filter(Boolean))].sort();
}

export function defaultConfig(): OneProxyConfig {
  return {
    schemaVersion: 1,
    overrides: { direct: [], proxy: [] }
  };
}

export async function readConfig(): Promise<OneProxyConfig> {
  const config = await readJson<Partial<OneProxyConfig>>(storageFile('config'));
  return {
    ...defaultConfig(),
    ...config,
    overrides: {
      direct: uniqueHosts(config?.overrides?.direct),
      proxy: uniqueHosts(config?.overrides?.proxy)
    }
  };
}

export async function writeConfig(config: OneProxyConfig): Promise<void> {
  await writeJson(storageFile('config'), {
    ...config,
    schemaVersion: 1,
    overrides: {
      direct: uniqueHosts(config.overrides.direct),
      proxy: uniqueHosts(config.overrides.proxy)
    }
  });
}

export async function readTokens(): Promise<OneProxyTokens | null> {
  return readJson<OneProxyTokens>(storageFile('tokens'));
}

export async function writeTokens(tokens: OneProxyTokens): Promise<void> {
  await writeJson(storageFile('tokens'), { ...tokens, schemaVersion: 1 });
}

export async function clearTokens(): Promise<void> {
  try {
    await fs.rm(storageFile('tokens'));
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== 'ENOENT') {
      throw error;
    }
  }
}

export async function readState(): Promise<OneProxyState> {
  const state = await readJson<Partial<OneProxyState>>(storageFile('state'));
  return {
    schemaVersion: 1,
    ...state,
    routeGroups: state?.routeGroups ?? []
  };
}

export async function writeState(state: OneProxyState): Promise<void> {
  await writeJson(storageFile('state'), { ...state, schemaVersion: 1 });
}

export async function readDaemonMetadata(): Promise<DaemonMetadata | null> {
  return readJson<DaemonMetadata>(storageFile('daemon'));
}

export async function writeDaemonMetadata(metadata: DaemonMetadata): Promise<void> {
  await writeJson(storageFile('daemon'), { ...metadata, schemaVersion: 1 });
}

export async function appendLog(message: string): Promise<void> {
  await ensureStorageRoot();
  await fs.appendFile(storageFile('log'), `${new Date().toISOString()} ${message}\n`, { mode: 0o600 });
}

function isExcludedPort(port: number): boolean {
  return commonPorts.has(port) || port < portRangeStart || port > portRangeEnd;
}

export async function isLoopbackPortAvailable(port: number): Promise<boolean> {
  if (isExcludedPort(port)) {
    return false;
  }
  const server = net.createServer();
  return new Promise((resolve) => {
    server.once('error', () => resolve(false));
    server.listen(port, loopbackHost, () => {
      server.close(() => resolve(true));
    });
  });
}

export async function scanAvailableProxyPortPairs(): Promise<Array<[number, number]>> {
  const pairs: Array<[number, number]> = [];
  for (let port = portRangeStart; port < portRangeEnd; port += 1) {
    if (isExcludedPort(port) || isExcludedPort(port + 1)) {
      continue;
    }
    const [httpAvailable, httpsAvailable] = await Promise.all([
      isLoopbackPortAvailable(port),
      isLoopbackPortAvailable(port + 1)
    ]);
    if (httpAvailable && httpsAvailable) {
      pairs.push([port, port + 1]);
    }
  }
  return pairs;
}

export function processIsRunning(pid: number): boolean {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}
