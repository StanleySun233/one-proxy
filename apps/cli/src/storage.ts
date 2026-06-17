import * as fs from 'node:fs/promises';
import * as fsSync from 'node:fs';
import * as net from 'node:net';
import * as os from 'node:os';
import * as path from 'node:path';

export type LocalOverrides = {
  direct: string[];
  proxy: string[];
};

export type OneProxyConfig = {
  schemaVersion: number;
  profileName?: string;
  controlPlaneUrl?: string;
  activeTenantId?: string;
  activeAccessPathId?: string;
  ignoredCliVersion?: string;
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

export type BootstrapNode = {
  id: string;
  name: string;
  mode: string;
  scopeKey: string;
  parentNodeId: string;
  enabled: boolean;
  status: string;
  publicHost?: string;
  publicPort?: number;
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

export type AccessPathSnapshot = {
  id: string;
  name: string;
  chainId: string;
  mode: string;
  protocol: string;
  serviceType: string;
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string[];
  listenHost: string;
  listenPort: number;
  targetProtocol: string;
  targetHost: string;
  targetPort: number;
  targetSni: string;
  tlsMode: string;
  authMode: 'proxy_token';
  enabled: boolean;
  options: Record<string, string>;
  topology: TopologyHop[];
  health: {
    status: string;
    reason: string;
    checkedAt: string;
  };
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

export type RouteEvaluationContract = {
  defaultClientMode: 'direct';
  defaultNodeMode: 'deny';
  ruleOrder: 'priority_asc_then_id_asc';
  noMatchNodeDenyReason: 'route_not_found';
  supportedMatchTypes: RouteSnapshot['matchType'][];
  supportedActions: RouteSnapshot['actionType'][];
};

export type OneProxyState = {
  schemaVersion: number;
  bootstrap?: {
    tenantId?: string;
    accessPathId?: string;
    entryNodes?: EntryNode[];
  };
  policyRevision?: string;
  fetchedAt?: string;
  nodes?: BootstrapNode[];
  accessPaths?: AccessPathSnapshot[];
  routes?: RouteSnapshot[];
  routeEvaluation?: RouteEvaluationContract;
};

export type DaemonBindings = {
  host: string;
  httpPort: number;
  httpsPort: number;
  ipcPort?: number;
  proxyOnlyPort?: number;
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
  portSelection?: {
    candidatePorts: number[];
    selectedPair: [number, number];
    excludedCommonPorts: number[];
  };
  idleTimeoutSeconds?: number;
  daemonSecret: string;
};

export const loopbackHost = '127.0.0.1';

export type ProfileRecord = {
  name: string;
  controlPlaneUrl: string;
};

export type ProfilesIndex = {
  schemaVersion: number;
  activeProfile?: string;
  profiles: Record<string, ProfileRecord>;
};

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

export function profilesFile(): string {
  return path.join(oneProxyHome(), 'profiles.json');
}

export function activeProfileName(): string {
  const envProfile = process.env.ONEPROXY_PROFILE || process.env.ONEPROXY_ACTIVE_PROFILE;
  if (envProfile) {
    return profileKey(envProfile);
  }
  if (fsSync.existsSync(profilesFile())) {
    const index = JSON.parse(fsSync.readFileSync(profilesFile(), 'utf8')) as ProfilesIndex;
    if (index.activeProfile) {
      return profileKey(index.activeProfile);
    }
  }
  return 'default';
}

export function profileRoot(name = activeProfileName()): string {
  return path.join(oneProxyHome(), 'profiles', profileKey(name));
}

export function storageFile(name: 'config' | 'state' | 'tokens' | 'daemon' | 'log'): string {
  const names = {
    config: 'config.json',
    state: 'state.json',
    tokens: 'tokens.json',
    daemon: 'daemon.json',
    log: 'onep.log'
  };
  return path.join(profileRoot(), names[name]);
}

export async function ensureStorageRoot(): Promise<void> {
  await fs.mkdir(oneProxyHome(), { recursive: true, mode: 0o700 });
}

export async function ensureProfileRoot(): Promise<void> {
  await fs.mkdir(profileRoot(), { recursive: true, mode: 0o700 });
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
  await fs.mkdir(path.dirname(file), { recursive: true, mode: 0o700 });
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
    profileName: activeProfileName(),
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
    profileName: config.profileName || activeProfileName(),
    overrides: {
      direct: uniqueHosts(config.overrides.direct),
      proxy: uniqueHosts(config.overrides.proxy)
    }
  });
}

export async function readProfilesIndex(): Promise<ProfilesIndex> {
  const index = await readJson<Partial<ProfilesIndex>>(profilesFile());
  return {
    schemaVersion: 1,
    activeProfile: index?.activeProfile,
    profiles: index?.profiles ?? {}
  };
}

export async function writeProfilesIndex(index: ProfilesIndex): Promise<void> {
  await writeJson(profilesFile(), {
    schemaVersion: 1,
    activeProfile: index.activeProfile,
    profiles: index.profiles
  });
}

export function profileKey(name: string): string {
  const value = name.trim().toLowerCase();
  if (!/^[a-z0-9][a-z0-9._-]*$/.test(value)) {
    throw Object.assign(new Error(`Invalid profile name: ${name}`), { code: 'SYNTAX_ERROR', exitCode: 2 });
  }
  return value;
}

export async function addProfile(name: string, controlPlaneUrl: string): Promise<ProfileRecord> {
  const key = profileKey(name);
  const index = await readProfilesIndex();
  const profile = { name: key, controlPlaneUrl };
  index.profiles[key] = profile;
  index.activeProfile = key;
  await writeProfilesIndex(index);
  await writeConfig({ ...(await readConfig()), profileName: key, controlPlaneUrl });
  return profile;
}

export async function useProfile(name: string): Promise<ProfileRecord> {
  const key = profileKey(name);
  const index = await readProfilesIndex();
  const profile = index.profiles[key];
  if (!profile) {
    throw Object.assign(new Error(`Profile not found: ${name}`), { code: 'PROFILE_REQUIRED' });
  }
  index.activeProfile = key;
  await writeProfilesIndex(index);
  process.env.ONEPROXY_PROFILE = key;
  await writeConfig({ ...(await readConfig()), profileName: key, controlPlaneUrl: profile.controlPlaneUrl });
  return profile;
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
    ...state
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
