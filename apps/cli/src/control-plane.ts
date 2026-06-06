import * as readline from 'node:readline/promises';
import { stdin as input, stdout as output } from 'node:process';
import type { CliContext } from './main.js';
import {
  clearTokens,
  readConfig,
  readState,
  readTokens,
  writeConfig,
  writeState,
  writeTokens,
  type Account,
  type OneProxyConfig,
  type OneProxyTokens,
  type RouteGroup,
  type RouteRule
} from './storage.js';

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

type Group = {
  id: string;
  name: string;
  enabled?: boolean;
};

type LoginResult = {
  account: Account;
  accessToken?: string;
  refreshToken?: string;
  expiresAt?: string;
  tokens?: {
    accessToken?: string;
    refreshToken?: string;
    expiresAt?: string;
  };
  tenantMemberships?: Tenant[];
  activeTenantId?: string | null;
};

type ExtensionBootstrap = {
  policyRevision?: string;
  fetchedAt?: string;
  proxyToken?: string;
  proxyTokenExpiresAt?: string;
  groups?: Array<{
    id: string;
    name: string;
    entryNodeId?: string;
    proxyHost?: string;
    proxyPort?: number;
    proxyScheme?: string;
    proxyDefault?: boolean;
    proxyHosts?: string[];
    directHosts?: string[];
    routes?: Array<{
      id: string;
      matchType: string;
      matchValue: string;
      actionType: string;
      topology?: Array<{ id: string; publicHost?: string; publicPort?: number; mode?: string }>;
    }>;
  }>;
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
  let response: Response;
  try {
    response = await fetch(endpoint(config.controlPlaneUrl, path), {
      method: options.method ?? 'GET',
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body)
    });
  } catch {
    throw Object.assign(new Error('Control plane is unavailable.'), { code: 'CONTROL_PLANE_UNAVAILABLE' });
  }
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
  const rl = readline.createInterface({ input, output });
  try {
    return (await rl.question(`${label}: `)).trim();
  } finally {
    rl.close();
  }
}

function tokenFromLogin(result: LoginResult): OneProxyTokens {
  const accessTokenExpiresAt = result.tokens?.expiresAt || result.expiresAt;
  return {
    schemaVersion: 1,
    account: result.account,
    accessToken: result.tokens?.accessToken || result.accessToken,
    refreshToken: result.tokens?.refreshToken || result.refreshToken,
    proxyToken: undefined,
    accessTokenExpiresAt,
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
  const config = await readConfig();
  const controlPlaneUrl = optionValue(args, '--control-plane') || process.env.ONEPROXY_CONTROL_PLANE_URL || config.controlPlaneUrl;
  const account = await promptMissing(optionValue(args, '--account') || process.env.ONEPROXY_ACCOUNT, 'Account');
  const password = await promptMissing(process.env.ONEPROXY_PASSWORD, 'Password');
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
  await writeConfig({ ...nextConfig, activeTenantId: defaultTenantId || config.activeTenantId });
  await writeTokens(tokens);
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
    await request(config, '/auth/logout', { method: 'POST', accessToken: tokens.accessToken }).catch(() => undefined);
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
    refreshToken: tokens.refreshToken,
    body: { refreshToken: tokens.refreshToken }
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
  await writeConfig({ ...config, activeTenantId: tenantIdOf(tenant), activeGroupId: undefined });
  print(context.json ? { activeTenantId: tenantIdOf(tenant) } : `Active tenant: ${tenantNameOf(tenant)}`, context);
}

export async function groupList(_args: string[], context: CliContext): Promise<void> {
  const config = await readConfig();
  if (!config.activeTenantId) {
    throw Object.assign(new Error('Active tenant is required. Run onep tenant use <name-or-id>.'), { code: 'TENANT_REQUIRED' });
  }
  const groups = await authenticatedRequest<Group[]>('/groups', { tenantId: config.activeTenantId });
  if (context.json) {
    print({ groups }, context);
    return;
  }
  print(groups.map((group) => `${group.id}\t${group.name}${group.enabled === false ? '\tdisabled' : ''}`).join('\n'), context);
}

export async function groupUse(args: string[], context: CliContext): Promise<void> {
  const config = await readConfig();
  if (!config.activeTenantId) {
    throw Object.assign(new Error('Active tenant is required. Run onep tenant use <name-or-id>.'), { code: 'TENANT_REQUIRED' });
  }
  const target = args[0].toLowerCase();
  const groups = await authenticatedRequest<Group[]>('/groups', { tenantId: config.activeTenantId });
  const group = groups.find((item) => item.id.toLowerCase() === target || item.name.toLowerCase() === target);
  if (!group) {
    throw Object.assign(new Error(`Group not found: ${args[0]}`), { code: 'GROUP_REQUIRED' });
  }
  await writeConfig({ ...config, activeGroupId: group.id });
  print(context.json ? { activeGroupId: group.id } : `Active group: ${group.name}`, context);
}

function routeRulesFromBootstrap(group: NonNullable<ExtensionBootstrap['groups']>[number]): RouteRule[] {
  const directHosts = (group.directHosts ?? []).map((host) => ({
    id: `direct:${host}`,
    type: 'domain' as const,
    pattern: host,
    mode: 'direct' as const
  }));
  const proxyHosts = (group.proxyHosts ?? []).map((host) => ({
    id: `proxy:${host}`,
    type: 'domain' as const,
    pattern: host,
    mode: 'proxy' as const
  }));
  const routes = (group.routes ?? []).map((route) => ({
    id: route.id,
    type: route.matchType === 'suffix' ? ('suffix' as const) : ('domain' as const),
    pattern: route.matchValue,
    mode: route.actionType === 'direct' ? ('direct' as const) : ('proxy' as const)
  }));
  return [...routes, ...directHosts, ...proxyHosts];
}

export async function sync(_args: string[], context: CliContext): Promise<void> {
  const config = await readConfig();
  if (!config.activeTenantId) {
    throw Object.assign(new Error('Active tenant is required. Run onep tenant use <name-or-id>.'), { code: 'TENANT_REQUIRED' });
  }
  const tokens = await requireTokens();
  const bootstrap = await request<ExtensionBootstrap>(config, '/proxy/extension/bootstrap', {
    accessToken: tokens.accessToken,
    tenantId: config.activeTenantId
  });
  const routeGroups: RouteGroup[] = (bootstrap.groups ?? []).map((group) => ({
    id: group.id,
    tenantId: config.activeTenantId || '',
    name: group.name,
    rules: routeRulesFromBootstrap(group)
  }));
  const activeGroup = routeGroups.find((group) => group.id === config.activeGroupId) || routeGroups.find((group) => group.id);
  const entryGroup = (bootstrap.groups ?? []).find((group) => group.id === activeGroup?.id);
  const state = {
    ...(await readState()),
    schemaVersion: 1,
    bootstrap: {
      tenantId: config.activeTenantId,
      groupId: activeGroup?.id,
      entryNodes: entryGroup?.proxyHost
        ? [{ id: entryGroup.entryNodeId || entryGroup.id, host: entryGroup.proxyHost, port: entryGroup.proxyPort || 443, protocol: entryGroup.proxyScheme || 'connect' }]
        : []
    },
    policyRevision: bootstrap.policyRevision,
    fetchedAt: bootstrap.fetchedAt || new Date().toISOString(),
    routeGroups
  };
  await writeState(state);
  await writeTokens({
    ...tokens,
    proxyToken: bootstrap.proxyToken || tokens.proxyToken,
    proxyTokenExpiresAt: bootstrap.proxyTokenExpiresAt || tokens.proxyTokenExpiresAt
  });
  if (activeGroup?.id && activeGroup.id !== config.activeGroupId) {
    await writeConfig({ ...config, activeGroupId: activeGroup.id });
  }
  print(context.json ? { synced: true, policyRevision: state.policyRevision, groupCount: routeGroups.length } : `Synced ${routeGroups.length} group(s).`, context);
}
