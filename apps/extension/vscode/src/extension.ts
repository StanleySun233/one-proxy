import * as fs from 'node:fs/promises';
import * as os from 'node:os';
import * as path from 'node:path';
import * as vscode from 'vscode';

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

const sessionKey = 'oneproxy.session';

export function activate(context: vscode.ExtensionContext) {
  context.subscriptions.push(
    vscode.commands.registerCommand('oneproxy.login', () => login(context)),
    vscode.commands.registerCommand('oneproxy.writeSshConfig', () => writeSshConfig(context)),
    vscode.commands.registerCommand('oneproxy.connectRemoteSsh', () => connectRemoteSsh(context))
  );
}

export function deactivate() {}

async function login(context: vscode.ExtensionContext) {
  const panelUrl = await prompt('Panel URL', 'https://panel.example.com');
  const account = await prompt('Account');
  const password = await prompt('Password', undefined, true);
  const response = await fetch(`${trimUrl(panelUrl)}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ account, password })
  });
  const payload = await response.json() as { data?: LoginResponse; message?: string };
  if (!response.ok || !payload.data) {
    throw new Error(payload.message || 'login_failed');
  }
  const tenantId = await selectTenant(payload.data);
  const bootstrap = await extensionBootstrap(trimUrl(panelUrl), payload.data.accessToken, tenantId);
  const session: Session = {
    panelUrl: trimUrl(panelUrl),
    account: payload.data.account.account,
    accessToken: payload.data.accessToken,
    refreshToken: payload.data.refreshToken,
    tenantId,
    proxyToken: bootstrap.proxyToken,
    proxyTokenExpiresAt: bootstrap.proxyTokenExpiresAt
  };
  await context.secrets.store(sessionKey, JSON.stringify(session));
  vscode.window.showInformationMessage(`Logged in as ${session.account} tenant ${session.tenantId}`);
}

async function writeSshConfig(context: vscode.ExtensionContext) {
  const session = await readSession(context);
  const entryHost = await prompt('OneProxy node host', '172.20.116.58');
  const entryPort = await prompt('OneProxy node port', '2333');
  const targetHost = await prompt('Target host', '172.20.116.91');
  const targetPort = await prompt('Target SSH port', '22');
  const user = await prompt('SSH user', os.userInfo().username);
  const hostAlias = config().get<string>('sshConfigHost') || 'oneproxy-remote';
  const cliPath = config().get<string>('cliPath') || 'oneproxy';
  const tokenFile = await writeTokenFile(session.proxyToken);
  const block = sshConfigBlock({ hostAlias, user, targetHost, targetPort, entryHost, entryPort, cliPath, tokenFile });
  await upsertSshConfigBlock(hostAlias, block);
  vscode.window.showInformationMessage(`SSH config updated: ${hostAlias}`);
}

async function connectRemoteSsh(context: vscode.ExtensionContext) {
  await ensureRemoteSshInstalled();
  const hostAlias = config().get<string>('sshConfigHost') || 'oneproxy-remote';
  await writeSshConfig(context);
  const uri = vscode.Uri.parse(`vscode-remote://ssh-remote+${encodeURIComponent(hostAlias)}/`);
  await vscode.commands.executeCommand('vscode.openFolder', uri, true);
}

async function readSession(context: vscode.ExtensionContext): Promise<Session> {
  const raw = await context.secrets.get(sessionKey);
  if (!raw) {
    throw new Error('not_logged_in');
  }
  return JSON.parse(raw) as Session;
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
  const payload = await response.json() as { data?: BootstrapResponse; message?: string };
  if (!response.ok || !payload.data) {
    throw new Error(payload.message || 'bootstrap_failed');
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
  return value.trim().replace(/\/$/, '');
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

function sshConfigBlock(input: {
  hostAlias: string;
  user: string;
  targetHost: string;
  targetPort: string;
  entryHost: string;
  entryPort: string;
  cliPath: string;
  tokenFile: string;
}): string {
  return [
    `Host ${input.hostAlias}`,
    `  HostName ${input.targetHost}`,
    `  Port ${input.targetPort}`,
    `  User ${input.user}`,
    `  ProxyCommand ${shellQuote(input.cliPath)} proxy-command --entry-host ${shellQuote(input.entryHost)} --entry-port ${shellQuote(input.entryPort)} --target-host %h --target-port %p --token-file ${shellQuote(input.tokenFile)}`,
    ''
  ].join('\n');
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

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}
