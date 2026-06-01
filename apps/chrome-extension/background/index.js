const STORAGE_KEY = 'oneProxyState';
const LOG_KEY = 'oneProxyDiagnostics';
const MAX_LOG_ENTRIES = 120;

const DEFAULT_STATE = {
  enabled: false,
  themeMode: 'vivid',
  controlPlaneUrl: '',
  session: {
    account: '',
    accessToken: '',
    refreshToken: '',
    expiresAt: '',
    mustRotatePassword: false
  },
  remote: {
    policyRevision: '',
    fetchedAt: '',
    groups: []
  },
  selection: {
    activeGroupId: ''
  },
  localOverrides: {
    directHosts: [],
    proxyHosts: []
  },
  monitor: {
    targetUrl: '',
    lastRunAt: '',
    results: []
  }
};

let stateCache = null;

async function appendLog(level, event, details = {}) {
  const entry = {
    at: new Date().toISOString(),
    level,
    event,
    details
  };
  const stored = await chrome.storage.local.get(LOG_KEY);
  const logs = Array.isArray(stored[LOG_KEY]) ? stored[LOG_KEY] : [];
  const nextLogs = [...logs, entry].slice(-MAX_LOG_ENTRIES);
  await chrome.storage.local.set({ [LOG_KEY]: nextLogs });
  return entry;
}

async function diagnosticLogs() {
  const stored = await chrome.storage.local.get(LOG_KEY);
  return Array.isArray(stored[LOG_KEY]) ? stored[LOG_KEY] : [];
}

async function clearDiagnosticLogs() {
  await chrome.storage.local.set({ [LOG_KEY]: [] });
  await appendLog('info', 'diagnostics_cleared');
  return diagnosticLogs();
}

function uniqueStrings(items) {
  return [...new Set((items || []).map((item) => String(item || '').trim()).filter(Boolean))];
}

function normalizeGroup(group) {
  const routes = Array.isArray(group.routes) ? group.routes : [];
  const topology = Array.isArray(group.topology) ? group.topology : [];
  return {
    id: '',
    name: '',
    entryNodeId: '',
    entryNodeName: '',
    proxyScheme: 'PROXY',
    proxyHost: '',
    proxyPort: 0,
    proxyDefault: false,
    proxyHosts: [],
    proxyCidrs: [],
    directHosts: [],
    directCidrs: [],
    routes: [],
    topology: [],
    ...group,
    proxyHosts: uniqueStrings(group.proxyHosts),
    proxyCidrs: uniqueStrings(group.proxyCidrs),
    directHosts: uniqueStrings(group.directHosts),
    directCidrs: uniqueStrings(group.directCidrs),
    routes: routes.map(normalizeRoute),
    topology: topology.map(normalizeTopologyNode)
  };
}

function normalizeRoute(route) {
  return {
    id: '',
    priority: 0,
    matchType: '',
    matchValue: '',
    actionType: '',
    chainId: '',
    destinationScope: '',
    topology: [],
    ...route,
    topology: Array.isArray(route.topology) ? route.topology.map(normalizeTopologyNode) : []
  };
}

function normalizeTopologyNode(node) {
  return {
    id: '',
    name: '',
    mode: '',
    scopeKey: '',
    publicHost: '',
    publicPort: 0,
    ...node
  };
}

function mergeState(raw) {
  const state = {
    ...DEFAULT_STATE,
    ...raw,
    session: {
      ...DEFAULT_STATE.session,
      ...(raw.session || {})
    },
    remote: {
      ...DEFAULT_STATE.remote,
      ...(raw.remote || {})
    },
    selection: {
      ...DEFAULT_STATE.selection,
      ...(raw.selection || {})
    },
    localOverrides: {
      ...DEFAULT_STATE.localOverrides,
      ...(raw.localOverrides || {})
    },
    monitor: {
      ...DEFAULT_STATE.monitor,
      ...(raw.monitor || {})
    }
  };
  state.remote.groups = Array.isArray(state.remote.groups) ? state.remote.groups.map(normalizeGroup) : [];
  state.localOverrides.directHosts = uniqueStrings(state.localOverrides.directHosts);
  state.localOverrides.proxyHosts = uniqueStrings(state.localOverrides.proxyHosts);
  if (!state.remote.groups.find((group) => group.id === state.selection.activeGroupId)) {
    state.selection.activeGroupId = (state.remote.groups[0] && state.remote.groups[0].id) || '';
  }
  return state;
}

async function getState() {
  if (stateCache) {
    return structuredClone(stateCache);
  }
  const stored = await chrome.storage.local.get(STORAGE_KEY);
  stateCache = mergeState(stored[STORAGE_KEY] || {});
  return structuredClone(stateCache);
}

function activeGroupFrom(state) {
  return state.remote.groups.find((group) => group.id === state.selection.activeGroupId) || state.remote.groups[0] || null;
}

function wildcardToRegExp(pattern) {
  return new RegExp(`^${String(pattern || '').replace(/[.+?^${}()|[\]\\]/g, '\\$&').replaceAll('*', '.*')}$`, 'i');
}

function hostMatches(patterns, host) {
  const cleanHost = sanitizeHost(host);
  return uniqueStrings(patterns).some((pattern) => wildcardToRegExp(pattern).test(cleanHost));
}

function ipv4ToNumber(value) {
  const parts = String(value || '').split('.');
  if (parts.length !== 4) {
    return null;
  }
  let result = 0;
  for (const part of parts) {
    const octet = Number(part);
    if (!Number.isInteger(octet) || octet < 0 || octet > 255) {
      return null;
    }
    result = (result * 256) + octet;
  }
  return result;
}

function cidrMatches(patterns, host) {
  const ip = ipv4ToNumber(host);
  if (ip === null) {
    return false;
  }
  return uniqueStrings(patterns).some((pattern) => {
    const [network, prefixValue] = pattern.split('/');
    const networkIp = ipv4ToNumber(network);
    const prefix = Number(prefixValue);
    if (networkIp === null || !Number.isInteger(prefix) || prefix < 0 || prefix > 32) {
      return false;
    }
    const mask = prefix === 0 ? 0 : (0xffffffff << (32 - prefix)) >>> 0;
    return (ip & mask) === (networkIp & mask);
  });
}

function routePreviewForHost(state, host) {
  return routePreviewForUrl(state, host ? `http://${host}` : '');
}

function routePreviewForUrl(state, value) {
  const parsed = parseTargetUrl(value);
  const cleanHost = sanitizeHost(parsed.host);
  const group = activeGroupFrom(state);
  if (!cleanHost) {
    return { mode: 'unknown', source: 'no_site', host: '', topology: [] };
  }
  if (!state.enabled) {
    return { mode: 'direct', source: 'proxy_off', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
  }
  if (!group || !group.proxyHost || !group.proxyPort) {
    return { mode: 'direct', source: 'no_proxy_target', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
  }
  if (hostMatches(['localhost', '*.local', '*.lan', urlHostname(state.controlPlaneUrl), group.proxyHost, ...(group.directHosts || []), ...(state.localOverrides.directHosts || [])], cleanHost)) {
    const local = hostMatches(state.localOverrides.directHosts, cleanHost);
    return { mode: 'direct', source: local ? 'local_direct' : 'remote_direct', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
  }
  if (cidrMatches(group.directCidrs || [], cleanHost)) {
    return { mode: 'direct', source: 'remote_direct', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
  }
  const matchedRoute = matchGroupRoute(group, cleanHost, parsed.protocol);
  if (matchedRoute) {
    return {
      mode: 'proxy',
      source: routeSource(matchedRoute),
      host: cleanHost,
      protocol: parsed.protocol,
      port: parsed.port,
      rule: matchedRoute,
      topology: matchedRoute.topology && matchedRoute.topology.length ? matchedRoute.topology : group.topology
    };
  }
  if (hostMatches([...(group.proxyHosts || []), ...(state.localOverrides.proxyHosts || [])], cleanHost)) {
    const local = hostMatches(state.localOverrides.proxyHosts, cleanHost);
    return { mode: 'proxy', source: local ? 'local_proxy' : 'remote_proxy', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: group.topology || [] };
  }
  if (cidrMatches(group.proxyCidrs || [], cleanHost)) {
    return { mode: 'proxy', source: 'remote_proxy', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: group.topology || [] };
  }
  if (group.proxyDefault) {
    return { mode: 'proxy', source: 'proxy_default', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: group.topology || [] };
  }
  return { mode: 'direct', source: 'default_direct', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
}

function parseTargetUrl(value) {
  const raw = String(value || '').trim();
  if (!raw) {
    return { url: '', host: '', protocol: 'http', port: 80 };
  }
  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(raw) ? raw : `http://${raw}`;
  try {
    const parsed = new URL(withScheme);
    const protocol = parsed.protocol.replace(':', '').toLowerCase() || 'http';
    const port = Number(parsed.port) || defaultPort(protocol);
    return { url: parsed.href, host: parsed.hostname, protocol, port };
  } catch (_error) {
    return { url: raw, host: raw.split('/')[0], protocol: 'http', port: 80 };
  }
}

function defaultPort(protocol) {
  if (protocol === 'https' || protocol === 'wss') {
    return 443;
  }
  return 80;
}

function matchGroupRoute(group, host, protocol) {
  const routes = [...(group.routes || [])].sort((left, right) => Number(left.priority || 0) - Number(right.priority || 0));
  for (const route of routes) {
    if (!routeMatches(route, host, protocol)) {
      continue;
    }
    return route;
  }
  return null;
}

function routeMatches(route, host, protocol) {
  const value = String(route.matchValue || '').toLowerCase();
  const cleanHost = sanitizeHost(host);
  switch (route.matchType) {
    case 'domain':
      return cleanHost === value;
    case 'domain_suffix':
      return cleanHost.endsWith(value);
    case 'ip':
      return cleanHost === value;
    case 'ip_cidr':
      return cidrMatches([route.matchValue], cleanHost);
    case 'protocol':
      return String(protocol || '').toLowerCase() === value;
    case 'default':
      return true;
    default:
      return false;
  }
}

function routeSource(route) {
  if (route.matchType === 'default') {
    return 'proxy_default';
  }
  if (route.actionType === 'chain') {
    return 'remote_proxy';
  }
  return 'remote_proxy';
}

function escapePacString(value) {
  return String(value || '').replaceAll('\\', '\\\\').replaceAll("'", "\\'");
}

function urlHostname(value) {
  try {
    return new URL(value).hostname;
  } catch (_error) {
    return '';
  }
}

function cidrToMask(prefix) {
  const bits = Number(prefix);
  if (!Number.isInteger(bits) || bits < 0 || bits > 32) {
    return null;
  }
  const octets = [];
  let remaining = bits;
  for (let index = 0; index < 4; index += 1) {
    const value = remaining >= 8 ? 255 : remaining <= 0 ? 0 : 256 - 2 ** (8 - remaining);
    octets.push(value);
    remaining -= 8;
  }
  return octets.join('.');
}

function cidrEntries(items) {
  return uniqueStrings(items)
    .map((item) => {
      const [network, prefix] = item.split('/');
      const mask = cidrToMask(prefix);
      if (!network || !mask) {
        return null;
      }
      return { network, mask };
    })
    .filter(Boolean);
}

function buildPacScript(state) {
  const group = activeGroupFrom(state);
  const proxyTarget = group && group.proxyHost && group.proxyPort ? `${group.proxyScheme || 'PROXY'} ${group.proxyHost}:${group.proxyPort}` : 'DIRECT';
  const directHosts = uniqueStrings([
    'localhost',
    '*.local',
    '*.lan',
    urlHostname(state.controlPlaneUrl),
    group ? group.proxyHost : '',
    ...(group ? group.directHosts : []),
    ...(state.localOverrides.directHosts || [])
  ]);
  const proxyHosts = uniqueStrings([
    ...(group ? group.proxyHosts : []),
    ...(state.localOverrides.proxyHosts || [])
  ]);
  const directCidrs = cidrEntries(group ? group.directCidrs : []);
  const proxyCidrs = cidrEntries(group ? group.proxyCidrs : []);
  return `
const enabled = ${state.enabled ? 'true' : 'false'};
const proxyTarget = '${escapePacString(proxyTarget)}';
const proxyDefault = ${group && group.proxyDefault ? 'true' : 'false'};
const directHosts = ${JSON.stringify(directHosts)};
const proxyHosts = ${JSON.stringify(proxyHosts)};
const directCidrs = ${JSON.stringify(directCidrs)};
const proxyCidrs = ${JSON.stringify(proxyCidrs)};

function hostMatches(patterns, host) {
  for (const pattern of patterns) {
    if (shExpMatch(host, pattern)) {
      return true;
    }
  }
  return false;
}

function inCidrs(cidrs, ip) {
  if (!ip) {
    return false;
  }
  for (const item of cidrs) {
    if (isInNet(ip, item.network, item.mask)) {
      return true;
    }
  }
  return false;
}

function isLocalOnly(host, ip) {
  if (isPlainHostName(host) || dnsDomainIs(host, '.local')) {
    return true;
  }
  if (!ip) {
    return false;
  }
  return isInNet(ip, '127.0.0.0', '255.0.0.0') ||
    isInNet(ip, '169.254.0.0', '255.255.0.0');
}

function FindProxyForURL(url, host) {
  if (!enabled || proxyTarget === 'DIRECT') {
    return 'DIRECT';
  }
  const resolved = dnsResolve(host);
  if (hostMatches(directHosts, host)) {
    return 'DIRECT';
  }
  if (inCidrs(directCidrs, resolved)) {
    return 'DIRECT';
  }
  if (hostMatches(proxyHosts, host)) {
    return proxyTarget;
  }
  if (inCidrs(proxyCidrs, resolved)) {
    return proxyTarget;
  }
  if (isLocalOnly(host, resolved)) {
    return 'DIRECT';
  }
  if (proxyDefault) {
    return proxyTarget;
  }
  return 'DIRECT';
}
`;
}

function pacSummary(state) {
  const group = activeGroupFrom(state);
  return {
    enabled: Boolean(state.enabled),
    activeGroupId: group ? group.id : '',
    activeGroupName: group ? group.name : '',
    proxyTarget: group && group.proxyHost && group.proxyPort ? `${group.proxyScheme || 'PROXY'} ${group.proxyHost}:${group.proxyPort}` : 'DIRECT',
    proxyDefault: Boolean(group && group.proxyDefault),
    remoteProxyHosts: group ? uniqueStrings(group.proxyHosts).length : 0,
    remoteProxyCidrs: group ? uniqueStrings(group.proxyCidrs).length : 0,
    remoteDirectHosts: group ? uniqueStrings(group.directHosts).length : 0,
    remoteDirectCidrs: group ? uniqueStrings(group.directCidrs).length : 0,
    localProxyHosts: uniqueStrings(state.localOverrides.proxyHosts).length,
    localDirectHosts: uniqueStrings(state.localOverrides.directHosts).length
  };
}

async function applyProxy(state) {
  await chrome.proxy.settings.set({
    value: {
      mode: 'pac_script',
      pacScript: {
        data: buildPacScript(state)
      }
    },
    scope: 'regular'
  });
  await appendLog('info', 'proxy_applied', pacSummary(state));
}

async function getCurrentTabInfo() {
  const tabs = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
  const tab = tabs[0];
  if (!tab || !tab.url) {
    return null;
  }
  try {
    const parsed = new URL(tab.url);
    return {
      url: tab.url,
      host: parsed.hostname
    };
  } catch (_error) {
    return null;
  }
}

async function getComputedState() {
  const state = await getState();
  const currentTab = await getCurrentTabInfo();
  return {
    state,
    session: state.session,
    remote: state.remote,
    activeGroup: activeGroupFrom(state),
    currentTab,
    currentRoute: routePreviewForUrl(state, currentTab && currentTab.url),
    monitorRoute: routePreviewForUrl(state, state.monitor.targetUrl)
  };
}

async function testUrlRoute(targetUrl, options = {}) {
  const state = await getState();
  const route = routePreviewForUrl(state, targetUrl);
  const results = await runProxyProbes(state, targetUrl, route);
  if (options.saveMonitorTarget) {
    await persistState({
      ...state,
      monitor: {
        targetUrl: parseTargetUrl(targetUrl).url || String(targetUrl || '').trim(),
        lastRunAt: new Date().toISOString(),
        results
      }
    });
  }
  return { route, results };
}

async function runProxyMonitor() {
  const state = await getState();
  if (!state.enabled || !state.monitor.targetUrl) {
    return;
  }
  const route = routePreviewForUrl(state, state.monitor.targetUrl);
  const results = await runProxyProbes(state, state.monitor.targetUrl, route);
  await persistState({
    ...state,
    monitor: {
      ...state.monitor,
      lastRunAt: new Date().toISOString(),
      results
    }
  });
}

async function runProxyProbes(state, targetUrl, route) {
  const group = activeGroupFrom(state);
  const parsed = parseTargetUrl(targetUrl);
  if (!state.enabled || !group || !group.proxyHost || !group.proxyPort || !route || route.mode !== 'proxy') {
    return probeProtocols().map((protocol) => ({ protocol, status: 'skipped', latencyMs: 0, message: 'proxy_not_applied' }));
  }
  const remainingHopNodeIds = (route.topology || []).map((node) => node.id).filter(Boolean).slice(1);
  const endpoint = `http://${group.proxyHost}:${group.proxyPort}/api/v1/control-relay/probe`;
  return Promise.all(probeProtocols().map((protocol) => runNodeProbe(endpoint, {
    protocol,
    remainingHopNodeIds,
    targetHost: parsed.host,
    targetPort: parsed.port
  })));
}

function probeProtocols() {
  return ['http', 'https', 'ws', 'tcp', 'udp'];
}

async function runNodeProbe(endpoint, payload) {
  const startedAt = Date.now();
  try {
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    let body = null;
    try {
      body = await response.json();
    } catch (_error) {
    }
    return {
      protocol: payload.protocol,
      status: response.ok && body && body.status ? body.status : 'failed',
      latencyMs: Date.now() - startedAt,
      message: body && body.message ? body.message : `http_${response.status}`
    };
  } catch (error) {
    return {
      protocol: payload.protocol,
      status: 'failed',
      latencyMs: Date.now() - startedAt,
      message: error.message || 'probe_failed'
    };
  }
}

async function persistState(nextState) {
  stateCache = mergeState(nextState);
  await chrome.storage.local.set({ [STORAGE_KEY]: stateCache });
  await applyProxy(stateCache);
  await broadcastState();
  return structuredClone(stateCache);
}

async function broadcastState() {
  try {
    await chrome.runtime.sendMessage({ type: 'state-updated', payload: await getComputedState() });
  } catch (_error) {
  }
}

function authHeaders(token) {
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function apiRequest(state, path, options = {}) {
  const controlPlaneUrl = String(state.controlPlaneUrl || '').trim().replace(/\/$/, '');
  if (!controlPlaneUrl) {
    throw new Error('missing_control_plane_url');
  }
  const headers = {
    ...(options.body ? { 'Content-Type': 'application/json' } : {}),
    ...(options.headers || {})
  };
  if (options.auth !== false && state.session.accessToken) {
    Object.assign(headers, authHeaders(state.session.accessToken));
  }
  const response = await fetch(`${controlPlaneUrl}${path}`, {
    method: options.method || 'GET',
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined
  });
  if (response.status === 401 && options.allowRefresh !== false && state.session.refreshToken) {
    const refreshed = await refreshSession(state);
    return apiRequest(refreshed, path, { ...options, allowRefresh: false });
  }
  let payload = null;
  try {
    payload = await response.json();
  } catch (_error) {
  }
  if (!response.ok) {
    await appendLog('error', 'api_request_failed', { path, status: response.status, message: (payload && payload.message) || 'request_failed' });
    throw new Error((payload && payload.message) || 'request_failed');
  }
  await appendLog('info', 'api_request_ok', { path, status: response.status });
  return payload ? payload.data : null;
}

async function login(controlPlaneUrl, account, password) {
  const response = await fetch(`${String(controlPlaneUrl || '').trim().replace(/\/$/, '')}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ account, password })
  });
  const payload = await response.json();
  if (!response.ok) {
    await appendLog('error', 'login_failed', { status: response.status, message: (payload && payload.message) || 'login_failed' });
    throw new Error((payload && payload.message) || 'login_failed');
  }
  const nextState = mergeState({
    ...(await getState()),
    controlPlaneUrl: String(controlPlaneUrl || '').trim(),
    session: {
      account: payload.data.account.account,
      accessToken: payload.data.accessToken,
      refreshToken: payload.data.refreshToken,
      expiresAt: payload.data.expiresAt,
      mustRotatePassword: Boolean(payload.data.mustRotatePassword)
    }
  });
  await persistState(nextState);
  await appendLog('info', 'login_ok', { account: nextState.session.account, controlPlaneUrl: nextState.controlPlaneUrl });
  return syncRemoteConfig(nextState);
}

async function testConnection(controlPlaneUrl) {
  const baseUrl = String(controlPlaneUrl || '').trim().replace(/\/$/, '');
  if (!baseUrl) {
    throw new Error('missing_control_plane_url');
  }
  let response;
  try {
    response = await fetch(`${baseUrl}/healthz`);
  } catch (error) {
    await appendLog('error', 'test_connection_failed', { controlPlaneUrl: baseUrl, message: error.message || 'connection_failed' });
    throw new Error('connection_failed');
  }
  let payload = null;
  try {
    payload = await response.json();
  } catch (_error) {
  }
  if (!response.ok) {
    await appendLog('error', 'test_connection_failed', { controlPlaneUrl: baseUrl, status: response.status, message: (payload && payload.message) || 'connection_failed' });
    throw new Error((payload && payload.message) || 'connection_failed');
  }
  await appendLog('info', 'test_connection_ok', { controlPlaneUrl: baseUrl });
  return payload ? payload.data : null;
}

async function refreshSession(sourceState) {
  const state = mergeState(sourceState || (await getState()));
  if (!state.controlPlaneUrl || !state.session.refreshToken) {
    throw new Error('missing_refresh_token');
  }
  const response = await fetch(`${state.controlPlaneUrl.replace(/\/$/, '')}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refreshToken: state.session.refreshToken })
  });
  const payload = await response.json();
  if (!response.ok) {
    await appendLog('error', 'refresh_failed', { status: response.status, message: (payload && payload.message) || 'refresh_failed' });
    throw new Error((payload && payload.message) || 'refresh_failed');
  }
  const nextState = mergeState({
    ...state,
    session: {
      account: payload.data.account.account,
      accessToken: payload.data.accessToken,
      refreshToken: payload.data.refreshToken,
      expiresAt: payload.data.expiresAt,
      mustRotatePassword: Boolean(payload.data.mustRotatePassword)
    }
  });
  await persistState(nextState);
  await appendLog('info', 'refresh_ok', { account: nextState.session.account });
  return nextState;
}

async function syncRemoteConfig(sourceState) {
  const state = mergeState(sourceState || (await getState()));
  const data = await apiRequest(state, '/api/v1/extension/bootstrap');
  const nextState = mergeState({
    ...state,
    remote: {
      policyRevision: data.policyRevision || '',
      fetchedAt: data.fetchedAt || '',
      groups: Array.isArray(data.groups) ? data.groups : []
    },
    session: {
      ...state.session,
      account: data.account ? data.account.account : state.session.account,
      mustRotatePassword: Boolean(data.account && data.account.mustRotatePassword)
    }
  });
  await persistState(nextState);
  await appendLog('info', 'remote_config_synced', {
    policyRevision: nextState.remote.policyRevision,
    groups: nextState.remote.groups.length
  });
  return getComputedState();
}

async function logout() {
  const state = await getState();
  if (state.controlPlaneUrl && state.session.accessToken) {
    try {
      await fetch(`${state.controlPlaneUrl.replace(/\/$/, '')}/api/v1/auth/logout`, {
        method: 'POST',
        headers: authHeaders(state.session.accessToken)
      });
    } catch (_error) {
    }
  }
  const nextState = mergeState({
    ...state,
    enabled: false,
    session: DEFAULT_STATE.session,
    remote: DEFAULT_STATE.remote,
    selection: DEFAULT_STATE.selection
  });
  await persistState(nextState);
  await appendLog('info', 'logout_ok');
  return getComputedState();
}

function sanitizeHost(value) {
  return String(value || '').trim().toLowerCase();
}

async function addHostToRule(kind, host) {
  const clean = sanitizeHost(host);
  if (!clean) {
    return getComputedState();
  }
  const state = await getState();
  const overrides = {
    ...state.localOverrides,
    [kind]: uniqueStrings([...(state.localOverrides[kind] || []), clean])
  };
  await persistState({ ...state, localOverrides: overrides });
  await appendLog('info', 'local_override_added', { kind, host: clean });
  return getComputedState();
}

async function removeHostFromRule(host) {
  const clean = sanitizeHost(host);
  if (!clean) {
    return getComputedState();
  }
  const state = await getState();
  const overrides = {
    ...state.localOverrides,
    directHosts: uniqueStrings(state.localOverrides.directHosts).filter((item) => item !== clean),
    proxyHosts: uniqueStrings(state.localOverrides.proxyHosts).filter((item) => item !== clean)
  };
  await persistState({ ...state, localOverrides: overrides });
  await appendLog('info', 'local_override_removed', { host: clean });
  return getComputedState();
}

async function setPartialState(mutator) {
  const current = await getState();
  const next = await mutator(structuredClone(current));
  await persistState(next);
  return getComputedState();
}

function ensureMonitorAlarm() {
  if (chrome.alarms) {
    chrome.alarms.create('proxy-monitor', { periodInMinutes: 1 });
  }
}

chrome.runtime.onInstalled.addListener(async () => {
  ensureMonitorAlarm();
  await persistState(await getState());
  await appendLog('info', 'extension_installed');
});

chrome.runtime.onStartup.addListener(async () => {
  ensureMonitorAlarm();
  await applyProxy(await getState());
  await appendLog('info', 'extension_startup');
});

if (chrome.alarms) {
  chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === 'proxy-monitor') {
      runProxyMonitor().catch((error) => appendLog('error', 'proxy_monitor_failed', { message: error.message || 'probe_failed' }));
    }
  });
  ensureMonitorAlarm();
}

if (chrome.proxy && chrome.proxy.onProxyError) {
  chrome.proxy.onProxyError.addListener((details) => {
    appendLog('error', 'proxy_error', details).catch(() => {});
  });
}

chrome.storage.onChanged.addListener(async (changes, areaName) => {
  if (areaName !== 'local' || !changes[STORAGE_KEY]) {
    return;
  }
  stateCache = mergeState(changes[STORAGE_KEY].newValue || {});
  await applyProxy(stateCache);
  await broadcastState();
});

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  (async () => {
    if (!message || !message.type) {
      sendResponse(null);
      return;
    }
    switch (message.type) {
      case 'get-state':
        sendResponse(await getComputedState());
        return;
      case 'get-diagnostic-logs':
        sendResponse(await diagnosticLogs());
        return;
      case 'clear-diagnostic-logs':
        sendResponse(await clearDiagnosticLogs());
        return;
      case 'record-diagnostic-event': {
        const state = await getState();
        await appendLog('info', message.event || 'diagnostic_event', pacSummary(state));
        sendResponse(await diagnosticLogs());
        return;
      }
      case 'set-enabled':
        sendResponse(await setPartialState((state) => ({ ...state, enabled: Boolean(message.enabled) })));
        return;
      case 'set-theme-mode':
        sendResponse(await setPartialState((state) => ({ ...state, themeMode: message.themeMode || 'vivid' })));
        return;
      case 'set-control-plane-url':
        sendResponse(await setPartialState((state) => ({ ...state, controlPlaneUrl: String(message.controlPlaneUrl || '').trim() })));
        return;
      case 'login':
        sendResponse(await login(message.controlPlaneUrl, message.account, message.password));
        return;
      case 'test-connection':
        sendResponse(await testConnection(message.controlPlaneUrl));
        return;
      case 'logout':
        sendResponse(await logout());
        return;
      case 'sync-remote-config':
        sendResponse(await syncRemoteConfig());
        return;
      case 'test-url-route':
        sendResponse(await testUrlRoute(message.url, { saveMonitorTarget: Boolean(message.saveMonitorTarget) }));
        return;
      case 'select-group':
        sendResponse(await setPartialState((state) => ({
          ...state,
          selection: {
            ...state.selection,
            activeGroupId: message.groupId || ''
          }
        })));
        return;
      case 'set-local-overrides':
        sendResponse(await setPartialState((state) => ({
          ...state,
          localOverrides: {
            directHosts: uniqueStrings(message.directHosts),
            proxyHosts: uniqueStrings(message.proxyHosts)
          }
        })));
        return;
      case 'add-current-host-to-direct': {
        const info = await getCurrentTabInfo();
        sendResponse(await addHostToRule('directHosts', (info && info.host) || ''));
        return;
      }
      case 'add-current-host-to-proxy': {
        const info = await getCurrentTabInfo();
        sendResponse(await addHostToRule('proxyHosts', (info && info.host) || ''));
        return;
      }
      case 'remove-current-host-override': {
        const info = await getCurrentTabInfo();
        sendResponse(await removeHostFromRule((info && info.host) || ''));
        return;
      }
      default:
        sendResponse(null);
    }
  })().catch((error) => {
    appendLog('error', 'runtime_message_failed', {
      type: message && message.type ? message.type : '',
      message: error.message || 'unexpected_error'
    }).catch(() => {});
    sendResponse({ error: error.message || 'unexpected_error' });
  });
  return true;
});
