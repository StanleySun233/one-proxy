import type { CliContext } from './main.ts';
import { promptPassword, promptText } from './prompt.ts';
import {
  clearTokens,
  activeProfileName,
  addProfile,
  appendLog,
  readConfig,
  readState,
  readTokens,
  writeConfig,
  writeState,
  writeTokens,
  useProfile,
  type Account,
  type OneProxyConfig,
  type OneProxyTokens
} from './storage.ts';

type RequestOptions = {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  accessToken?: string;
  refreshToken?: string;
  tenantId?: string;
  body?: unknown;
};

type Envelope<T> = {
  code: number;
  message?: string;
  data: T;
};

type Tenant = {
  id?: string;
  tenantId?: string;
  name?: string;
  tenantName?: string;
};

type LoginResult = {
  account: Account;
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  mustRotatePassword: boolean;
  tenantMemberships?: Tenant[];
  activeTenantId?: string | null;
};

type TopologyHop = {
  nodeId: string;
  nodeName: string;
  mode: string;
  scopeKey: string;
  publicHost?: string;
  publicPort?: number;
  transport: string;
};

type BootstrapNode = {
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

type AccessPathSnapshot = {
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

type RouteSnapshot = {
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

type RouteEvaluationContract = {
  defaultClientMode: 'direct';
  defaultNodeMode: 'deny';
  ruleOrder: 'priority_asc_then_id_asc';
  noMatchNodeDenyReason: 'route_not_found';
  supportedMatchTypes: RouteSnapshot['matchType'][];
  supportedActions: RouteSnapshot['actionType'][];
};

type ExtensionBootstrap = {
  schemaVersion: 'v2.1.0';
  account: Account;
  tenant: Tenant;
  policyRevision: string;
  fetchedAt: string;
  proxyToken: string;
  proxyTokenExpiresAt: string;
  nodes: BootstrapNode[];
  accessPaths: AccessPathSnapshot[];
  routes: RouteSnapshot[];
  routeEvaluation: RouteEvaluationContract;
};

type SyncResult = {
  policyRevision?: string;
  accessPathCount: number;
};

type LatestConfig = OneProxyConfig & {
  activeAccessPathId?: string;
};

type LatestState = Awaited<ReturnType<typeof readState>> & {
  nodes?: BootstrapNode[];
  accessPaths?: AccessPathSnapshot[];
  routes?: RouteSnapshot[];
  routeEvaluation?: RouteEvaluationContract;
  bootstrap?: {
    tenantId?: string;
    accessPathId?: string;
  };
};

function print(value: unknown, context: CliContext): void {
  if (context.json) {
    process.stdout.write(`${JSON.stringify(value, null, 2)}\n`);
    return;
  }
  if (Array.isArray(value)) {
    process.stdout.write(value.map((item) => JSON.stringify(item)).join('\n') + (value.length ? '\n' : ''));
    return;
  }
  process.stdout.write(`${String(value)}\n`);
}

function endpoint(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/\/+$/, '')}/api${path}`;
}

async function request<T>(config: OneProxyConfig, path: string, options: RequestOptions = {}): Promise<T> {
  if (!config.controlPlaneUrl) {
    throw Object.assign(new Error('Control plane URL is not configured. Run onep login --control-plane <url>.'), {
      code: 'AUTH_REQUIRED'
    });
  }
  const headers = new Headers();
  if (options.accessToken) {
    headers.set('X-One-Proxy-Access-Token', options.accessToken);
  }
  if (options.refreshToken) {
    headers.set('X-One-Proxy-Refresh-Token', options.refreshToken);
  }
  if (options.tenantId) {
    headers.set('X-One-Proxy-Tenant-ID', options.tenantId);
  }
  if (options.body !== undefined) {
    headers.set('Content-Type', 'application/json');
  }
  const response = await fetch(endpoint(config.controlPlaneUrl, path), {
    method: options.method ?? 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  });
  const raw = await response.text();
  const envelope = raw ? (JSON.parse(raw) as Envelope<T>) : null;
  if (!response.ok || !envelope || envelope.code !== 0) {
    const code = response.status === 401 ? 'AUTH_REQUIRED' : 'CONTROL_PLANE_UNAVAILABLE';
    throw Object.assign(new Error(envelope?.message || `Control plane request failed with HTTP ${response.status}.`), { code });
  }
  return envelope.data;
}

function optionValue(args: string[], name: string): string | undefined {
  const index = args.indexOf(name);
  return index >= 0 ? args[index + 1] : undefined;
}

async function promptMissing(value: string | undefined, label: string): Promise<string> {
  if (value) {
    return value;
  }
  return promptText(`${label}: `);
}

async function promptMissingPassword(value: string | undefined): Promise<string> {
  if (value) {
    return value;
  }
  return promptPassword('Password: ');
}

function tokenFromLogin(result: LoginResult): OneProxyTokens {
  return {
    schemaVersion: 1,
    account: result.account,
    accessToken: result.accessToken,
    refreshToken: result.refreshToken,
    proxyToken: undefined,
    accessTokenExpiresAt: result.expiresAt,
    refreshTokenExpiresAt: undefined,
    proxyTokenExpiresAt: undefined
  };
}

function tenantIdOf(tenant: Tenant): string {
  return tenant.id || tenant.tenantId || '';
}

function tenantNameOf(tenant: Tenant): string {
  return tenant.name || tenant.tenantName || tenantIdOf(tenant);
}

async function requireTokens(): Promise<OneProxyTokens> {
  const tokens = await readTokens();
  if (!tokens?.accessToken) {
    throw Object.assign(new Error('Authentication is required. Run onep login.'), { code: 'AUTH_REQUIRED' });
  }
  return tokens;
}

export async function login(args: string[], context: CliContext): Promise<void> {
  const requestedProfile = optionValue(args, '--profile') || process.env.ONEPROXY_PROFILE;
  const requestedControlPlaneUrl = optionValue(args, '--control-plane') || process.env.ONEPROXY_CONTROL_PLANE_URL;
  if (requestedProfile && requestedControlPlaneUrl) {
    process.env.ONEPROXY_PROFILE = requestedProfile;
    await addProfile(requestedProfile, requestedControlPlaneUrl);
  } else if (requestedProfile) {
    await useProfile(requestedProfile);
  } else if (requestedControlPlaneUrl) {
    await addProfile(activeProfileName(), requestedControlPlaneUrl);
  }
  const config = await readConfig();
  const controlPlaneUrl = requestedControlPlaneUrl || config.controlPlaneUrl;
  const account = await promptMissing(optionValue(args, '--account') || process.env.ONEPROXY_ACCOUNT, 'Account');
  const password = await promptMissingPassword(process.env.ONEPROXY_PASSWORD);
  const nextConfig = { ...config, controlPlaneUrl };
  if (!nextConfig.controlPlaneUrl) {
    throw Object.assign(new Error('login requires --control-plane <url> or ONEPROXY_CONTROL_PLANE_URL.'), { code: 'AUTH_REQUIRED' });
  }
  const result = await request<LoginResult>(nextConfig, '/auth/login', {
    method: 'POST',
    body: { account, password }
  });
  const tokens = tokenFromLogin(result);
  const memberships = result.tenantMemberships ?? [];
  const defaultTenantId = result.activeTenantId || (memberships.length === 1 ? tenantIdOf(memberships[0]) : undefined);
  const loginConfig: LatestConfig = {
    ...nextConfig,
    activeTenantId: defaultTenantId || config.activeTenantId,
    activeAccessPathId: undefined
  };
  await writeConfig(loginConfig);
  await writeTokens(tokens);
  if (defaultTenantId) {
    await syncRemoteStateWithRefresh(loginConfig, tokens);
  }
  print(
    context.json
      ? { account: tokens.account, activeTenantId: defaultTenantId ?? null, tenantSelectionRequired: !defaultTenantId }
      : `Logged in as ${tokens.account?.email || tokens.account?.account || tokens.account?.id}.` +
          (defaultTenantId ? ` Active tenant: ${defaultTenantId}` : ' Run onep tenant list, then onep tenant use <name-or-id>.'),
    context
  );
}

export async function logout(_args: string[], context: CliContext): Promise<void> {
  const config = await readConfig();
  const tokens = await readTokens();
  if (tokens?.accessToken) {
    await request(config, '/auth/logout', { method: 'POST', accessToken: tokens.accessToken });
  }
  await clearTokens();
  print(context.json ? { loggedOut: true } : 'Logged out.', context);
}

export async function refreshSession(): Promise<OneProxyTokens> {
  const config = await readConfig();
  const tokens = await readTokens();
  if (!tokens?.refreshToken) {
    throw Object.assign(new Error('Refresh token is missing. Run onep login.'), { code: 'AUTH_REQUIRED' });
  }
  const result = await request<LoginResult>(config, '/auth/refresh', {
    method: 'POST',
    refreshToken: tokens.refreshToken
  });
  const nextTokens = { ...tokens, ...tokenFromLogin(result) };
  await writeTokens(nextTokens);
  return nextTokens;
}

async function authenticatedRequest<T>(path: string, options: Omit<RequestOptions, 'accessToken'> = {}): Promise<T> {
  const config = await readConfig();
  const tokens = await requireTokens();
  return request<T>(config, path, { ...options, accessToken: tokens.accessToken });
}

export async function tenantList(_args: string[], context: CliContext): Promise<void> {
  const tenants = await authenticatedRequest<{ tenants: Tenant[] }>('/tenants').then((result) => result.tenants);
  if (context.json) {
    print({ tenants }, context);
    return;
  }
  print(tenants.map((tenant) => `${tenantIdOf(tenant)}\t${tenantNameOf(tenant)}`).join('\n'), context);
}

export async function tenantUse(args: string[], context: CliContext): Promise<void> {
  const target = args[0].toLowerCase();
  const tenants = await authenticatedRequest<{ tenants: Tenant[] }>('/tenants').then((result) => result.tenants);
  const tenant = tenants.find((item) => tenantIdOf(item).toLowerCase() === target || tenantNameOf(item).toLowerCase() === target);
  if (!tenant) {
    throw Object.assign(new Error(`Tenant not found: ${args[0]}`), { code: 'TENANT_REQUIRED' });
  }
  const config = await readConfig();
  const nextTenantId = tenantIdOf(tenant);
  const latestConfig = config as LatestConfig;
  const nextConfig: LatestConfig = {
    ...config,
    activeTenantId: nextTenantId,
    activeAccessPathId: latestConfig.activeTenantId === nextTenantId ? latestConfig.activeAccessPathId : undefined
  };
  await writeConfig(nextConfig);
  await syncRemoteStateWithRefresh(nextConfig, await requireTokens());
  print(context.json ? { activeTenantId: tenantIdOf(tenant) } : `Active tenant: ${tenantNameOf(tenant)}`, context);
}

export async function accessPathList(_args: string[], context: CliContext): Promise<void> {
  const config = await readConfig();
  if (!config.activeTenantId) {
    throw Object.assign(new Error('Active tenant is required. Run onep tenant use <name-or-id>.'), { code: 'TENANT_REQUIRED' });
  }
  const state = (await readState()) as LatestState;
  const accessPaths = state.accessPaths ?? [];
  if (context.json) {
    print({ accessPaths }, context);
    return;
  }
  print(accessPaths.map((accessPath) => `${accessPath.id}\t${accessPath.name || accessPath.id}`).join('\n'), context);
}

export async function accessPathUse(args: string[], context: CliContext): Promise<void> {
  const config = (await readConfig()) as LatestConfig;
  if (!config.activeTenantId) {
    throw Object.assign(new Error('Active tenant is required. Run onep tenant use <name-or-id>.'), { code: 'TENANT_REQUIRED' });
  }
  const target = args[0].toLowerCase();
  const state = (await readState()) as LatestState;
  const accessPath = (state.accessPaths ?? []).find((item) => item.id.toLowerCase() === target || (item.name || '').toLowerCase() === target);
  if (!accessPath) {
    throw Object.assign(new Error(`Access path not found in synced state: ${args[0]}. Run onep sync first.`), { code: 'ACCESS_PATH_REQUIRED' });
  }
  const nextConfig: LatestConfig = { ...config, activeAccessPathId: accessPath.id };
  await writeConfig(nextConfig);
  print(context.json ? { activeAccessPathId: accessPath.id } : `Active access path: ${accessPath.name || accessPath.id}`, context);
}

async function syncRemoteState(config: OneProxyConfig, tokens: OneProxyTokens): Promise<SyncResult> {
  if (!config.activeTenantId) {
    throw Object.assign(new Error('Active tenant is required. Run onep tenant use <name-or-id>.'), { code: 'TENANT_REQUIRED' });
  }
  if (!tokens.accessToken) {
    throw Object.assign(new Error('Authentication is required. Run onep login.'), { code: 'AUTH_REQUIRED' });
  }
  const bootstrap = await request<ExtensionBootstrap>(config, '/proxy/extension/bootstrap', {
    accessToken: tokens.accessToken,
    tenantId: config.activeTenantId
  });
  if (bootstrap.schemaVersion !== 'v2.1.0') {
    throw Object.assign(new Error(`Unsupported bootstrap schema version: ${bootstrap.schemaVersion}`), { code: 'UNSUPPORTED_BOOTSTRAP_CONTRACT' });
  }
  const latestConfig = config as LatestConfig;
  const activeAccessPath =
    bootstrap.accessPaths.find((accessPath) => accessPath.enabled && accessPath.id === latestConfig.activeAccessPathId) ||
    bootstrap.accessPaths.find((accessPath) => accessPath.enabled);
  const state = {
    ...(await readState()),
    schemaVersion: 1,
    bootstrap: {
      tenantId: config.activeTenantId,
      accessPathId: activeAccessPath?.id
    },
    policyRevision: bootstrap.policyRevision,
    fetchedAt: bootstrap.fetchedAt,
    nodes: bootstrap.nodes,
    accessPaths: bootstrap.accessPaths,
    routes: bootstrap.routes,
    routeEvaluation: bootstrap.routeEvaluation
  };
  await writeState(state);
  await writeTokens({
    ...tokens,
    proxyToken: bootstrap.proxyToken,
    proxyTokenExpiresAt: bootstrap.proxyTokenExpiresAt
  });
  if (activeAccessPath?.id && activeAccessPath.id !== latestConfig.activeAccessPathId) {
    const nextConfig: LatestConfig = { ...config, activeAccessPathId: activeAccessPath.id };
    await writeConfig(nextConfig);
  }
  return { policyRevision: state.policyRevision, accessPathCount: bootstrap.accessPaths.length };
}

async function syncRemoteStateWithRefresh(config: OneProxyConfig, tokens: OneProxyTokens): Promise<SyncResult> {
  try {
    return await syncRemoteState(config, tokens);
  } catch (error) {
    if ((error as { code?: string }).code !== 'AUTH_REQUIRED' || !tokens.refreshToken) {
      throw error;
    }
    const refreshed = await refreshSession();
    return syncRemoteState(await readConfig(), refreshed);
  }
}

export async function autoSyncRemoteState(): Promise<boolean> {
  const [config, tokens] = await Promise.all([readConfig(), readTokens()]);
  if (!config.controlPlaneUrl || !config.activeTenantId || !tokens?.accessToken) {
    return false;
  }
  try {
    await syncRemoteStateWithRefresh(config, tokens);
    return true;
  } catch (error) {
    await appendLog(`auto sync failed: ${(error as Error).message || String(error)}`).catch(() => {});
    return false;
  }
}

export async function sync(_args: string[], context: CliContext): Promise<void> {
  const result = await syncRemoteStateWithRefresh(await readConfig(), await requireTokens());
  print(
    context.json
      ? { synced: true, policyRevision: result.policyRevision, accessPathCount: result.accessPathCount }
      : `Synced ${result.accessPathCount} access path(s).`,
    context
  );
}
