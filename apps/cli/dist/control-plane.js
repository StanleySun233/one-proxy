import { promptPassword, promptText } from "./prompt.js";
import { clearTokens, activeProfileName, addProfile, readConfig, readState, readTokens, writeConfig, writeState, writeTokens, useProfile } from "./storage.js";
function print(value, context) {
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
function endpoint(baseUrl, path) {
    return `${baseUrl.replace(/\/+$/, '')}/api${path}`;
}
async function request(config, path, options = {}) {
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
    const envelope = raw ? JSON.parse(raw) : null;
    if (!response.ok || !envelope || envelope.code !== 0) {
        const code = response.status === 401 ? 'AUTH_REQUIRED' : 'CONTROL_PLANE_UNAVAILABLE';
        throw Object.assign(new Error(envelope?.message || `Control plane request failed with HTTP ${response.status}.`), { code });
    }
    return envelope.data;
}
function optionValue(args, name) {
    const index = args.indexOf(name);
    return index >= 0 ? args[index + 1] : undefined;
}
async function promptMissing(value, label) {
    if (value) {
        return value;
    }
    return promptText(`${label}: `);
}
async function promptMissingPassword(value) {
    if (value) {
        return value;
    }
    return promptPassword('Password: ');
}
function tokenFromLogin(result) {
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
function tenantIdOf(tenant) {
    return tenant.id || tenant.tenantId || '';
}
function tenantNameOf(tenant) {
    return tenant.name || tenant.tenantName || tenantIdOf(tenant);
}
async function requireTokens() {
    const tokens = await readTokens();
    if (!tokens?.accessToken) {
        throw Object.assign(new Error('Authentication is required. Run onep login.'), { code: 'AUTH_REQUIRED' });
    }
    return tokens;
}
export async function login(args, context) {
    const requestedProfile = optionValue(args, '--profile') || process.env.ONEPROXY_PROFILE;
    const requestedControlPlaneUrl = optionValue(args, '--control-plane') || process.env.ONEPROXY_CONTROL_PLANE_URL;
    if (requestedProfile && requestedControlPlaneUrl) {
        process.env.ONEPROXY_PROFILE = requestedProfile;
        await addProfile(requestedProfile, requestedControlPlaneUrl);
    }
    else if (requestedProfile) {
        await useProfile(requestedProfile);
    }
    else if (requestedControlPlaneUrl) {
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
    const result = await request(nextConfig, '/auth/login', {
        method: 'POST',
        body: { account, password }
    });
    const tokens = tokenFromLogin(result);
    const memberships = result.tenantMemberships ?? [];
    const defaultTenantId = result.activeTenantId || (memberships.length === 1 ? tenantIdOf(memberships[0]) : undefined);
    await writeConfig({ ...nextConfig, activeTenantId: defaultTenantId || config.activeTenantId });
    await writeTokens(tokens);
    print(context.json
        ? { account: tokens.account, activeTenantId: defaultTenantId ?? null, tenantSelectionRequired: !defaultTenantId }
        : `Logged in as ${tokens.account?.email || tokens.account?.account || tokens.account?.id}.` +
            (defaultTenantId ? ` Active tenant: ${defaultTenantId}` : ' Run onep tenant list, then onep tenant use <name-or-id>.'), context);
}
export async function logout(_args, context) {
    const config = await readConfig();
    const tokens = await readTokens();
    if (tokens?.accessToken) {
        await request(config, '/auth/logout', { method: 'POST', accessToken: tokens.accessToken });
    }
    await clearTokens();
    print(context.json ? { loggedOut: true } : 'Logged out.', context);
}
export async function refreshSession() {
    const config = await readConfig();
    const tokens = await readTokens();
    if (!tokens?.refreshToken) {
        throw Object.assign(new Error('Refresh token is missing. Run onep login.'), { code: 'AUTH_REQUIRED' });
    }
    const result = await request(config, '/auth/refresh', {
        method: 'POST',
        refreshToken: tokens.refreshToken,
        body: { refreshToken: tokens.refreshToken }
    });
    const nextTokens = { ...tokens, ...tokenFromLogin(result) };
    await writeTokens(nextTokens);
    return nextTokens;
}
async function authenticatedRequest(path, options = {}) {
    const config = await readConfig();
    const tokens = await requireTokens();
    return request(config, path, { ...options, accessToken: tokens.accessToken });
}
export async function tenantList(_args, context) {
    const tenants = await authenticatedRequest('/tenants').then((result) => result.tenants);
    if (context.json) {
        print({ tenants }, context);
        return;
    }
    print(tenants.map((tenant) => `${tenantIdOf(tenant)}\t${tenantNameOf(tenant)}`).join('\n'), context);
}
export async function tenantUse(args, context) {
    const target = args[0].toLowerCase();
    const tenants = await authenticatedRequest('/tenants').then((result) => result.tenants);
    const tenant = tenants.find((item) => tenantIdOf(item).toLowerCase() === target || tenantNameOf(item).toLowerCase() === target);
    if (!tenant) {
        throw Object.assign(new Error(`Tenant not found: ${args[0]}`), { code: 'TENANT_REQUIRED' });
    }
    const config = await readConfig();
    const nextTenantId = tenantIdOf(tenant);
    const state = await readState();
    const currentGroup = state.routeGroups.find((group) => group.id === config.activeGroupId);
    await writeConfig({
        ...config,
        activeTenantId: nextTenantId,
        activeGroupId: currentGroup?.tenantId === nextTenantId ? config.activeGroupId : undefined
    });
    print(context.json ? { activeTenantId: tenantIdOf(tenant) } : `Active tenant: ${tenantNameOf(tenant)}`, context);
}
export async function groupList(_args, context) {
    const config = await readConfig();
    if (!config.activeTenantId) {
        throw Object.assign(new Error('Active tenant is required. Run onep tenant use <name-or-id>.'), { code: 'TENANT_REQUIRED' });
    }
    const state = await readState();
    const groups = state.routeGroups.filter((group) => group.tenantId === config.activeTenantId);
    if (context.json) {
        print({ groups }, context);
        return;
    }
    print(groups.map((group) => `${group.id}\t${group.name || group.id}`).join('\n'), context);
}
export async function groupUse(args, context) {
    const config = await readConfig();
    if (!config.activeTenantId) {
        throw Object.assign(new Error('Active tenant is required. Run onep tenant use <name-or-id>.'), { code: 'TENANT_REQUIRED' });
    }
    const target = args[0].toLowerCase();
    const state = await readState();
    const groups = state.routeGroups.filter((group) => group.tenantId === config.activeTenantId);
    const group = groups.find((item) => item.id.toLowerCase() === target || (item.name || '').toLowerCase() === target);
    if (!group) {
        throw Object.assign(new Error(`Group not found in synced state: ${args[0]}. Run onep sync first.`), { code: 'GROUP_REQUIRED' });
    }
    await writeConfig({ ...config, activeGroupId: group.id });
    print(context.json ? { activeGroupId: group.id } : `Active group: ${group.name || group.id}`, context);
}
export function routeRulesFromBootstrap(group) {
    const directHosts = (group.directHosts ?? []).map((host) => ({
        id: `direct:${host}`,
        type: routeTypeFromHostPattern(host),
        pattern: host,
        mode: 'direct'
    }));
    const proxyHosts = (group.proxyHosts ?? []).map((host) => ({
        id: `proxy:${host}`,
        type: routeTypeFromHostPattern(host),
        pattern: host,
        mode: 'proxy'
    }));
    const routes = (group.routes ?? []).map((route) => ({
        id: route.id,
        type: routeTypeFromMatchType(route.matchType),
        pattern: route.matchValue,
        mode: route.actionType === 'direct' ? 'direct' : 'proxy'
    }));
    return [...routes, ...directHosts, ...proxyHosts];
}
function routeTypeFromMatchType(matchType) {
    const normalized = matchType.toLowerCase();
    if (normalized === 'suffix' || normalized === 'cidr' || normalized === 'wildcard') {
        return normalized;
    }
    if (normalized === 'domain_suffix') {
        return 'suffix';
    }
    if (normalized === 'ip_cidr') {
        return 'cidr';
    }
    if (normalized === 'default') {
        return 'wildcard';
    }
    return 'domain';
}
function routeTypeFromHostPattern(host) {
    const normalized = host.trim().toLowerCase();
    if (normalized.startsWith('*.') || normalized.startsWith('.')) {
        return 'suffix';
    }
    if (normalized.includes('*')) {
        return 'wildcard';
    }
    return 'domain';
}
export async function sync(_args, context) {
    const config = await readConfig();
    if (!config.activeTenantId) {
        throw Object.assign(new Error('Active tenant is required. Run onep tenant use <name-or-id>.'), { code: 'TENANT_REQUIRED' });
    }
    const tokens = await requireTokens();
    const bootstrap = await request(config, '/proxy/extension/bootstrap', {
        accessToken: tokens.accessToken,
        tenantId: config.activeTenantId
    });
    const routeGroups = (bootstrap.groups ?? []).map((group) => ({
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
//# sourceMappingURL=control-plane.js.map