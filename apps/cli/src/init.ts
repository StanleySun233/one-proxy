import * as readline from 'node:readline/promises';
import { emitKeypressEvents } from 'node:readline';
import { stdin as input, stdout as output } from 'node:process';
import type { CliContext } from './main.ts';
import { promptPassword, promptText } from './prompt.ts';
import { startActivatedShell } from './shell.ts';
import {
  addProfile,
  writeConfig,
  writeState,
  writeTokens,
  type Account,
  type OneProxyTokens,
  type RouteGroup
} from './storage.ts';
import { routeRulesFromBootstrap } from './control-plane.ts';

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
  accessToken?: string;
  refreshToken?: string;
  expiresAt?: string;
  tokens?: {
    accessToken?: string;
    refreshToken?: string;
    expiresAt?: string;
  };
};

function endpoint(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/\/+$/, '')}/api${path}`;
}

function normalizePanelUrl(input: string): string {
  const raw = input.trim();
  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(raw) ? raw : `https://${raw}`;
  const url = new URL(withScheme);
  url.pathname = url.pathname.replace(/\/+$/, '');
  url.search = '';
  url.hash = '';
  return url.toString().replace(/\/+$/, '');
}

async function request<T>(controlPlaneUrl: string, path: string, options: { method?: string; accessToken?: string; tenantId?: string; body?: unknown } = {}): Promise<T> {
  const headers = new Headers();
  if (options.accessToken) {
    headers.set('X-One-Proxy-Access-Token', options.accessToken);
  }
  if (options.tenantId) {
    headers.set('X-One-Proxy-Tenant-ID', options.tenantId);
  }
  if (options.body !== undefined) {
    headers.set('Content-Type', 'application/json');
  }
  const response = await fetch(endpoint(controlPlaneUrl, path), {
    method: options.method ?? 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  });
  const raw = await response.text();
  const envelope = raw ? (JSON.parse(raw) as Envelope<T>) : null;
  if (!response.ok || !envelope || envelope.code !== 0) {
    throw Object.assign(new Error(envelope?.message || `Control plane request failed with HTTP ${response.status}.`), {
      code: response.status === 401 ? 'AUTH_REQUIRED' : 'CONTROL_PLANE_UNAVAILABLE'
    });
  }
  return envelope.data;
}

async function healthCheck(controlPlaneUrl: string): Promise<void> {
  const response = await fetch(`${controlPlaneUrl.replace(/\/+$/, '')}/healthz`);
  if (!response.ok) {
    throw Object.assign(new Error(`Panel health check failed with HTTP ${response.status}.`), { code: 'CONTROL_PLANE_UNAVAILABLE' });
  }
}

async function prompt(label: string): Promise<string> {
  return promptText(label);
}

function tenantIdOf(tenant: Tenant): string {
  return tenant.id || tenant.tenantId || '';
}

function tenantNameOf(tenant: Tenant): string {
  return tenant.name || tenant.tenantName || tenantIdOf(tenant);
}

function tokenFromLogin(result: LoginResult): OneProxyTokens {
  return {
    schemaVersion: 1,
    account: result.account,
    accessToken: result.tokens?.accessToken || result.accessToken,
    refreshToken: result.tokens?.refreshToken || result.refreshToken,
    proxyToken: undefined,
    accessTokenExpiresAt: result.tokens?.expiresAt || result.expiresAt,
    refreshTokenExpiresAt: undefined,
    proxyTokenExpiresAt: undefined
  };
}

function profileNameFromUrl(panelUrl: string): string {
  const url = new URL(panelUrl);
  return `${url.hostname}${url.port ? `-${url.port}` : ''}`.toLowerCase().replace(/[^a-z0-9._-]+/g, '-');
}

async function selectTenant(tenants: Tenant[]): Promise<Tenant> {
  if (!input.isTTY) {
    throw Object.assign(new Error('onep init requires an interactive terminal for tenant selection.'), { code: 'SYNTAX_ERROR', exitCode: 2 });
  }
  let index = 0;
  emitKeypressEvents(input);
  input.setRawMode(true);
  input.resume();
  output.write('\nSelect tenant:\n');
  const render = () => {
    output.write('\x1b[?25l');
    output.write(`\x1b[${tenants.length}F`);
    for (let i = 0; i < tenants.length; i += 1) {
      const tenant = tenants[i];
      output.write(`${i === index ? '>' : ' '} ${tenantNameOf(tenant)} (${tenantIdOf(tenant)})\x1b[K\n`);
    }
  };
  for (const tenant of tenants) {
    output.write(`  ${tenantNameOf(tenant)} (${tenantIdOf(tenant)})\n`);
  }
  render();
  await new Promise((resolve) => setTimeout(resolve, 100));
  let handler: ((chunk: string, key: any) => void) | undefined;
  const selected = await new Promise<Tenant>((resolve) => {
    handler = (_chunk, key) => {
      if (key.name === 'up') {
        index = index === 0 ? tenants.length - 1 : index - 1;
        render();
      }
      if (key.name === 'down') {
        index = index === tenants.length - 1 ? 0 : index + 1;
        render();
      }
      if (key.name === 'return') {
        resolve(tenants[index]);
      }
    };
    input.on('keypress', handler);
  });
  if (handler) {
    input.off('keypress', handler);
  }
  input.setRawMode(false);
  output.write('\x1b[?25h\n');
  return selected;
}

export async function initCommand(_args: string[], _context: CliContext): Promise<number | void> {
  const panelUrl = normalizePanelUrl(await prompt('Panel URL: '));
  process.stdout.write('Testing panel reachability...\n');
  await healthCheck(panelUrl);
  process.stdout.write('Panel reachable.\n');

  const account = await prompt('Account: ');
  const password = await promptPassword('Password: ');
  const profileName = profileNameFromUrl(panelUrl);
  process.env.ONEPROXY_PROFILE = profileName;
  await addProfile(profileName, panelUrl);

  const login = await request<LoginResult>(panelUrl, '/auth/login', {
    method: 'POST',
    body: { account, password }
  });
  const tokens = tokenFromLogin(login);
  await writeTokens(tokens);

  const tenants = await request<{ tenants: Tenant[] }>(panelUrl, '/tenants', { accessToken: tokens.accessToken }).then((result) => result.tenants);
  if (tenants.length === 0) {
    throw Object.assign(new Error('No tenants are available for this account.'), { code: 'TENANT_REQUIRED' });
  }
  const tenant = await selectTenant(tenants);
  const activeTenantId = tenantIdOf(tenant);
  await writeConfig({
    schemaVersion: 1,
    profileName,
    controlPlaneUrl: panelUrl,
    activeTenantId,
    overrides: { direct: [], proxy: [] }
  });

  const bootstrap = await request<any>(panelUrl, '/proxy/extension/bootstrap', {
    accessToken: tokens.accessToken,
    tenantId: activeTenantId
  });
  const routeGroups: RouteGroup[] = (bootstrap.groups ?? []).map((group: any) => ({
    id: group.id,
    tenantId: activeTenantId,
    name: group.name,
    rules: routeRulesFromBootstrap(group)
  }));
  const activeGroup = routeGroups.find((group) => group.id);
  const entryGroup = (bootstrap.groups ?? []).find((group: any) => group.id === activeGroup?.id);
  await writeState({
    schemaVersion: 1,
    bootstrap: {
      tenantId: activeTenantId,
      groupId: activeGroup?.id,
      entryNodes: entryGroup?.proxyHost
        ? [{ id: entryGroup.entryNodeId || entryGroup.id, host: entryGroup.proxyHost, port: entryGroup.proxyPort || 443, protocol: entryGroup.proxyScheme || 'connect' }]
        : []
    },
    policyRevision: bootstrap.policyRevision,
    fetchedAt: bootstrap.fetchedAt || new Date().toISOString(),
    routeGroups
  });
  await writeTokens({
    ...tokens,
    proxyToken: bootstrap.proxyToken || tokens.proxyToken,
    proxyTokenExpiresAt: bootstrap.proxyTokenExpiresAt || tokens.proxyTokenExpiresAt
  });
  if (activeGroup?.id) {
    await writeConfig({
      schemaVersion: 1,
      profileName,
      controlPlaneUrl: panelUrl,
      activeTenantId,
      activeGroupId: activeGroup.id,
      overrides: { direct: [], proxy: [] }
    });
  }

  process.stdout.write(`Initialized profile ${profileName} with tenant ${tenantNameOf(tenant)}.\n`);
  const activate = (await prompt('Enter an activated OneProxy shell now? [y/N]: ')).toLowerCase();
  if (activate === 'y' || activate === 'yes') {
    return startActivatedShell();
  }
}
