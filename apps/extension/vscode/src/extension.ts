import * as fs from 'node:fs/promises';
import * as os from 'node:os';
import * as path from 'node:path';
import * as vscode from 'vscode';

type ApiEnvelope<T> = {
  data?: T;
  message?: string;
};

type LoginResponse = {
  account: { account: string };
  accessToken: string;
  refreshToken: string;
  tenantMemberships: TenantMembership[];
  activeTenantId: string | null;
};

type TenantMembership = {
  tenantId: string;
  tenantName: string;
};

type BootstrapResponse = {
  proxyToken: string;
  proxyTokenExpiresAt: string;
  accessPaths: AccessPathSnapshot[];
};

type Session = {
  panelUrl: string;
  account: string;
  accessToken: string;
  refreshToken: string;
  tenantId: string;
  proxyToken: string;
  proxyTokenExpiresAt: string;
};

type AccessPathSnapshot = {
  id: string;
  name: string;
  protocol: string;
  serviceType: string;
  targetProtocol: string;
  targetHost: string;
  targetPort: number;
  listenHost: string;
  listenPort: number;
  enabled: boolean;
};

type SshConnection = {
  accessPathId: string;
  accessPathName: string;
  hostAlias: string;
  user: string;
  targetHost: string;
  targetPort: number;
  endpointHost: string;
  endpointPort: number;
  proxyEnabled: boolean;
};

const sessionKey = 'oneproxy.session';
const sshConnectionKey = 'oneproxy.sshConnection';

export function activate(context: vscode.ExtensionContext) {
  syncExtensionSessionIfReady(context).catch(() => {});
  context.subscriptions.push(
    vscode.commands.registerCommand('oneproxy.login', () => login(context)),
    vscode.commands.registerCommand('oneproxy.selectSshTarget', () => runWithSyncedSession(context, () => selectSshTarget(context))),
    vscode.commands.registerCommand('oneproxy.writeSshConfig', () => runWithSyncedSession(context, () => writeSshConfig(context))),
    vscode.commands.registerCommand('oneproxy.connectRemoteSsh', () => runWithSyncedSession(context, () => connectRemoteSsh(context))),
    vscode.commands.registerCommand('oneproxy.setSshProxyMode', () => runWithSyncedSession(context, () => setSshProxyMode(context)))
  );
}

export function deactivate() {}

async function login(context: vscode.ExtensionContext) {
  const panelUrl = trimUrl(await prompt('Panel URL', 'https://panel.example.com'));
  const account = await prompt('Account');
  const password = await prompt('Password', undefined, true);
  const payload = await requestEnvelope<LoginResponse>(`${panelUrl}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ account, password })
  }, 'login_failed');
  const tenantId = await selectTenant(payload);
  const bootstrap = await extensionBootstrap(panelUrl, payload.accessToken, tenantId);
  const session: Session = {
    panelUrl,
    account: payload.account.account,
    accessToken: payload.accessToken,
    refreshToken: payload.refreshToken,
    tenantId,
    proxyToken: bootstrap.proxyToken,
    proxyTokenExpiresAt: bootstrap.proxyTokenExpiresAt
  };
  await context.secrets.store(sessionKey, JSON.stringify(session));
  vscode.window.showInformationMessage(`Logged in as ${session.account} tenant ${session.tenantId}`);
}

async function selectSshTarget(context: vscode.ExtensionContext): Promise<SshConnection> {
  const current = await readSshConnection(context);
  const items = await listSshConnections(context, current);
  const selected = await vscode.window.showQuickPick(items, { title: 'SSH target', ignoreFocusOut: true });
  if (!selected) {
    throw new Error('ssh_target_required');
  }
  const user = await prompt('SSH user', selected.connection.user || os.userInfo().username);
  const connection = { ...selected.connection, user };
  await storeSshConnection(context, connection);
  await writeConnectionSshConfig(context, connection);
  vscode.window.showInformationMessage(`SSH config updated: ${connection.hostAlias}`);
  return connection;
}

async function writeSshConfig(context: vscode.ExtensionContext) {
  const connection = await ensureSshConnection(context);
  await writeConnectionSshConfig(context, connection);
  vscode.window.showInformationMessage(`SSH config updated: ${connection.hostAlias}`);
}

async function connectRemoteSsh(context: vscode.ExtensionContext) {
  await ensureRemoteSshInstalled();
  const connection = await ensureSshConnection(context);
  await writeConnectionSshConfig(context, connection);
  const uri = vscode.Uri.parse(`vscode-remote://ssh-remote+${encodeURIComponent(connection.hostAlias)}/`);
  await vscode.commands.executeCommand('vscode.openFolder', uri, true);
}

async function setSshProxyMode(context: vscode.ExtensionContext) {
  const connection = await ensureSshConnection(context);
  const selected = await vscode.window.showQuickPick([
    { label: 'Use OneProxy', description: 'Route Remote-SSH through the selected OneProxy access path.', proxyEnabled: true },
    { label: 'Direct', description: 'Connect to the target host without ProxyCommand.', proxyEnabled: false }
  ], { title: 'SSH proxy mode', ignoreFocusOut: true });
  if (!selected) {
    throw new Error('proxy_mode_required');
  }
  const next = { ...connection, proxyEnabled: selected.proxyEnabled };
  await storeSshConnection(context, next);
  await writeConnectionSshConfig(context, next);
  vscode.window.showInformationMessage(`SSH proxy mode: ${selected.label}`);
}

async function ensureSshConnection(context: vscode.ExtensionContext): Promise<SshConnection> {
  const connection = await readSshConnection(context);
  return connection || selectSshTarget(context);
}

async function listSshConnections(context: vscode.ExtensionContext, current: SshConnection | null) {
  const { bootstrap } = await syncBootstrap(context);
  const items = bootstrap.accessPaths
    .filter((item) => item.enabled && item.targetProtocol === 'ssh' && item.listenHost && item.listenPort > 0)
    .map((item) => {
      const connection: SshConnection = {
        accessPathId: item.id,
        accessPathName: item.name,
        hostAlias: current && current.accessPathId === item.id ? current.hostAlias : sshAlias(item),
        user: current && current.accessPathId === item.id ? current.user : os.userInfo().username,
        targetHost: item.targetHost,
        targetPort: item.targetPort,
        endpointHost: item.listenHost,
        endpointPort: item.listenPort,
        proxyEnabled: current && current.accessPathId === item.id ? current.proxyEnabled : true
      };
      return {
        label: item.name || item.id,
        description: `${item.targetHost}:${item.targetPort}`,
        detail: `Access path ${item.listenHost}:${item.listenPort}`,
        connection
      };
    });
  if (items.length === 0) {
    throw new Error('no_ssh_access_path');
  }
  return items;
}

async function writeConnectionSshConfig(context: vscode.ExtensionContext, connection: SshConnection) {
  let tokenFile = '';
  if (connection.proxyEnabled) {
    const session = await syncProxyToken(context);
    tokenFile = await writeTokenFile(session.proxyToken);
  }
  const cliPath = config().get<string>('cliPath') || 'oneproxy';
  const block = sshConfigBlock({ ...connection, cliPath, tokenFile });
  await upsertSshConfigBlock(connection.hostAlias, block);
}

async function syncProxyToken(context: vscode.ExtensionContext): Promise<Session> {
  return syncExtensionSession(context, await readSession(context));
}

async function syncExtensionSessionIfReady(context: vscode.ExtensionContext): Promise<void> {
  const raw = await context.secrets.get(sessionKey);
  if (!raw) {
    return;
  }
  await syncExtensionSession(context, JSON.parse(raw) as Session);
}

async function syncExtensionSession(context: vscode.ExtensionContext, session: Session): Promise<Session> {
  return (await syncBootstrap(context, session)).session;
}

async function syncBootstrap(context: vscode.ExtensionContext, session?: Session): Promise<{ session: Session; bootstrap: BootstrapResponse }> {
  const current = session ?? await readSession(context);
  const result = await extensionBootstrapWithRefresh(context, current);
  const next = {
    ...result.session,
    proxyToken: result.bootstrap.proxyToken,
    proxyTokenExpiresAt: result.bootstrap.proxyTokenExpiresAt
  };
  await context.secrets.store(sessionKey, JSON.stringify(next));
  return { session: next, bootstrap: result.bootstrap };
}

async function runWithSyncedSession<T>(context: vscode.ExtensionContext, operation: () => Promise<T>): Promise<T> {
  await syncExtensionSessionIfReady(context).catch(() => {});
  return operation();
}

async function extensionBootstrapWithRefresh(context: vscode.ExtensionContext, session: Session): Promise<{ session: Session; bootstrap: BootstrapResponse }> {
  const response = await fetch(`${session.panelUrl}/api/proxy/extension/bootstrap`, {
    headers: authHeaders(session)
  });
  if (response.status !== 401) {
    return { session, bootstrap: await readEnvelope<BootstrapResponse>(response, 'bootstrap_failed') };
  }
  const refreshed = await refreshSession(context, session);
  const retry = await fetch(`${refreshed.panelUrl}/api/proxy/extension/bootstrap`, {
    headers: authHeaders(refreshed)
  });
  return { session: refreshed, bootstrap: await readEnvelope<BootstrapResponse>(retry, 'bootstrap_failed') };
}

async function refreshSession(context: vscode.ExtensionContext, session: Session): Promise<Session> {
  const payload = await requestEnvelope<LoginResponse>(`${session.panelUrl}/api/auth/refresh`, {
    method: 'POST',
    headers: { 'X-One-Proxy-Refresh-Token': session.refreshToken }
  }, 'refresh_failed');
  const tenantId = payload.tenantMemberships.find((item) => item.tenantId === session.tenantId)
    ? session.tenantId
    : await selectTenant(payload);
  const next = {
    ...session,
    account: payload.account.account,
    accessToken: payload.accessToken,
    refreshToken: payload.refreshToken,
    tenantId
  };
  await context.secrets.store(sessionKey, JSON.stringify(next));
  return next;
}

async function readSession(context: vscode.ExtensionContext): Promise<Session> {
  const raw = await context.secrets.get(sessionKey);
  if (!raw) {
    throw new Error('not_logged_in');
  }
  return JSON.parse(raw) as Session;
}

async function readSshConnection(context: vscode.ExtensionContext): Promise<SshConnection | null> {
  const raw = await context.secrets.get(sshConnectionKey);
  return raw ? JSON.parse(raw) as SshConnection : null;
}

async function storeSshConnection(context: vscode.ExtensionContext, connection: SshConnection) {
  await context.secrets.store(sshConnectionKey, JSON.stringify(connection));
}

async function selectTenant(session: LoginResponse): Promise<string> {
  if (session.activeTenantId) {
    return session.activeTenantId;
  }
  if (session.tenantMemberships.length === 1) {
    return session.tenantMemberships[0].tenantId;
  }
  const selected = await vscode.window.showQuickPick(
    session.tenantMemberships.map((membership) => ({
      label: membership.tenantName || membership.tenantId,
      description: membership.tenantId,
      tenantId: membership.tenantId
    })),
    { title: 'Tenant', ignoreFocusOut: true }
  );
  if (!selected) {
    throw new Error('tenant_required');
  }
  return selected.tenantId;
}

async function extensionBootstrap(panelUrl: string, accessToken: string, tenantId: string): Promise<BootstrapResponse> {
  const response = await fetch(`${panelUrl}/api/proxy/extension/bootstrap`, {
    headers: {
      'X-One-Proxy-Access-Token': accessToken,
      'X-One-Proxy-Tenant-ID': tenantId
    }
  });
  return readEnvelope<BootstrapResponse>(response, 'bootstrap_failed');
}

async function requestEnvelope<T>(url: string, init: RequestInit, fallback: string): Promise<T> {
  const response = await fetch(url, init);
  return readEnvelope<T>(response, fallback);
}

async function readEnvelope<T>(response: Response, fallback: string): Promise<T> {
  const payload = await response.json() as ApiEnvelope<T>;
  if (!response.ok || !payload.data) {
    throw new Error(payload.message || fallback);
  }
  return payload.data;
}

async function ensureRemoteSshInstalled() {
  const extension = vscode.extensions.getExtension('ms-vscode-remote.remote-ssh');
  if (!extension) {
    throw new Error('Remote-SSH extension is not installed');
  }
}

async function prompt(title: string, value?: string, password = false): Promise<string> {
  const result = await vscode.window.showInputBox({ title, value, password, ignoreFocusOut: true });
  if (!result) {
    throw new Error('input_cancelled');
  }
  return result;
}

function trimUrl(value: string): string {
  return value.trim().replace(/\/+$/, '');
}

function authHeaders(session: Session): Record<string, string> {
  return {
    'X-One-Proxy-Access-Token': session.accessToken,
    'X-One-Proxy-Tenant-ID': session.tenantId
  };
}

function config() {
  return vscode.workspace.getConfiguration('oneproxy');
}

async function writeTokenFile(token: string): Promise<string> {
  const dir = path.join(os.homedir(), '.config', 'oneproxy');
  await fs.mkdir(dir, { recursive: true, mode: 0o700 });
  const tokenFile = path.join(dir, 'proxy-token');
  await fs.writeFile(tokenFile, `${token}\n`, { mode: 0o600 });
  await fs.chmod(tokenFile, 0o600);
  return tokenFile;
}

function sshConfigBlock(input: SshConnection & { cliPath: string; tokenFile: string }): string {
  const lines = [
    `Host ${input.hostAlias}`,
    `  HostName ${input.targetHost}`,
    `  Port ${input.targetPort}`,
    `  User ${input.user}`
  ];
  if (input.proxyEnabled) {
    lines.push(`  ProxyCommand ${shellQuote(input.cliPath)} proxy-command --entry-host ${shellQuote(input.endpointHost)} --entry-port ${shellQuote(String(input.endpointPort))} --target-host %h --target-port %p --token-file ${shellQuote(input.tokenFile)}`);
  }
  return `${lines.join('\n')}\n`;
}

async function upsertSshConfigBlock(hostAlias: string, block: string) {
  const sshDir = path.join(os.homedir(), '.ssh');
  const configPath = path.join(sshDir, 'config');
  await fs.mkdir(sshDir, { recursive: true, mode: 0o700 });
  let current = '';
  try {
    current = await fs.readFile(configPath, 'utf8');
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== 'ENOENT') {
      throw error;
    }
  }
  const pattern = new RegExp(`(^|\\n)Host ${escapeRegExp(hostAlias)}\\n(?:[ \\t].*\\n?)*`, 'm');
  const next = current.match(pattern) ? current.replace(pattern, `\n${block}`) : `${current.trimEnd()}\n\n${block}`;
  await fs.writeFile(configPath, next.trimStart(), { mode: 0o600 });
}

function sshAlias(item: AccessPathSnapshot): string {
  const prefix = config().get<string>('sshConfigHostPrefix') || 'oneproxy';
  return `${prefix}-${slug(item.name || item.id)}-${slug(item.id)}`;
}

function slug(value: string): string {
  const clean = value.toLowerCase().replace(/[^a-z0-9_-]+/g, '-').replace(/^-+|-+$/g, '');
  return clean || 'ssh';
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}
