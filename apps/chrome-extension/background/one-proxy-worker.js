const LOG_KEY = 'oneProxyDiagnostics';
const MAX_LOG_ENTRIES = 120;

function appendLog(level, event, details = {}) {
  const entry = {
    at: new Date().toISOString(),
    level,
    event,
    details
  };
  return chrome.storage.local.get(LOG_KEY)
    .then((stored) => {
      const logs = Array.isArray(stored[LOG_KEY]) ? stored[LOG_KEY] : [];
      const nextLogs = [...logs, entry].slice(-MAX_LOG_ENTRIES);
      return chrome.storage.local.set({ [LOG_KEY]: nextLogs });
    })
    .then(() => entry);
}

function diagnosticLogs() {
  return chrome.storage.local.get(LOG_KEY)
    .then((stored) => Array.isArray(stored[LOG_KEY]) ? stored[LOG_KEY] : []);
}

function clearDiagnosticLogs() {
  return chrome.storage.local.set({ [LOG_KEY]: [] })
    .then(() => appendLog('info', 'diagnostics_cleared'))
    .then(() => diagnosticLogs());
}

const STORAGE_KEY = 'oneProxyState';

const DEFAULT_STATE = {
  enabled: false,
  themeMode: 'vivid',
  controlPlaneUrl: '',
  session: {
    account: '',
    accessToken: '',
    refreshToken: '',
    expiresAt: '',
    proxyToken: '',
    proxyTokenExpiresAt: '',
    mustRotatePassword: false,
    tenantMemberships: [],
    activeTenantId: ''
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
let persistEffects = () => Promise.resolve();

function configureStateEffects(effects) {
  persistEffects = typeof effects === 'function' ? effects : () => Promise.resolve();
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

function normalizeTenantMembership(membership) {
  return {
    tenantId: '',
    tenantName: '',
    role: '',
    joinedAt: '',
    ...membership
  };
}

function mergeState(raw) {
  const { proxyAuth: _proxyAuth, ...rest } = raw || {};
  const state = {
    ...DEFAULT_STATE,
    ...rest,
    session: {
      ...DEFAULT_STATE.session,
      ...(rest.session || {})
    },
    remote: {
      ...DEFAULT_STATE.remote,
      ...(rest.remote || {})
    },
    selection: {
      ...DEFAULT_STATE.selection,
      ...(rest.selection || {})
    },
    localOverrides: {
      ...DEFAULT_STATE.localOverrides,
      ...(rest.localOverrides || {})
    },
    monitor: {
      ...DEFAULT_STATE.monitor,
      ...(rest.monitor || {})
    }
  };
  state.remote.groups = Array.isArray(state.remote.groups) ? state.remote.groups.map(normalizeGroup) : [];
  state.session.tenantMemberships = Array.isArray(state.session.tenantMemberships) ? state.session.tenantMemberships.map(normalizeTenantMembership) : [];
  if (!state.session.tenantMemberships.find((membership) => membership.tenantId === state.session.activeTenantId)) {
    state.session.activeTenantId = state.session.tenantMemberships.length === 1 ? state.session.tenantMemberships[0].tenantId : '';
  }
  state.localOverrides.directHosts = uniqueStrings(state.localOverrides.directHosts);
  state.localOverrides.proxyHosts = uniqueStrings(state.localOverrides.proxyHosts);
  if (!state.remote.groups.find((group) => group.id === state.selection.activeGroupId)) {
    state.selection.activeGroupId = (state.remote.groups[0] && state.remote.groups[0].id) || '';
  }
  return state;
}

function getState() {
  if (stateCache) {
    return Promise.resolve(structuredClone(stateCache));
  }
  return chrome.storage.local.get(STORAGE_KEY)
    .then((stored) => {
      stateCache = mergeState(stored[STORAGE_KEY] || {});
      return structuredClone(stateCache);
    });
}

function activeGroupFrom(state) {
  return state.remote.groups.find((group) => group.id === state.selection.activeGroupId) || state.remote.groups[0] || null;
}

function persistState(nextState) {
  stateCache = mergeState(nextState);
  return chrome.storage.local.set({ [STORAGE_KEY]: stateCache })
    .then(() => persistEffects(stateCache))
    .then(() => structuredClone(stateCache));
}

function setPartialState(mutator) {
  return getState()
    .then((current) => mutator(structuredClone(current)))
    .then((next) => persistState(next));
}

function handleStateStorageChange(changes, areaName) {
  if (areaName !== 'local' || !changes[STORAGE_KEY]) {
    return null;
  }
  stateCache = mergeState(changes[STORAGE_KEY].newValue || {});
  return stateCache;
}

function wildcardToRegExp(pattern) {
  return new RegExp(`^${String(pattern || '').replace(/[.+?^${}()|[\]\\]/g, '\\$&').replaceAll('*', '.*')}$`, 'i');
}

function sanitizeHost(value) {
  return String(value || '').trim().toLowerCase();
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
  const matchedRoute = matchGroupRoute(group, parsed);
  if (matchedRoute) {
    const mode = matchedRoute.actionType === 'direct' ? 'direct' : 'proxy';
    return {
      mode,
      source: routeSource(matchedRoute),
      host: cleanHost,
      protocol: parsed.protocol,
      port: parsed.port,
      rule: matchedRoute,
      topology: mode === 'proxy' ? (matchedRoute.topology && matchedRoute.topology.length ? matchedRoute.topology : group.topology) : []
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

function matchGroupRoute(group, parsed) {
  const routes = [...(group.routes || [])].sort((left, right) => Number(left.priority || 0) - Number(right.priority || 0));
  for (const route of routes) {
    if (!routeMatches(route, parsed)) {
      continue;
    }
    return route;
  }
  return null;
}

function routeMatches(route, target) {
  const parsed = typeof target === 'object' && target ? target : parseTargetUrl(target);
  const value = String(route.matchValue || '').toLowerCase();
  const cleanHost = sanitizeHost(parsed.host);
  switch (route.matchType) {
    case 'domain':
      return cleanHost === value;
    case 'domain_suffix':
      return domainSuffixMatches(value, cleanHost);
    case 'ip_cidr':
      return cidrMatches([route.matchValue], cleanHost);
    case 'ip_range':
      return ipRangeMatches(value, cleanHost);
    case 'port':
      return Number(parsed.port) === Number(value);
    case 'url_regex':
      return urlRegexMatches(route.matchValue, parsed.url);
    case 'default':
      return true;
    default:
      return false;
  }
}

function domainSuffixMatches(value, host) {
  const suffix = value.startsWith('*.') ? value.slice(1) : value;
  return suffix.startsWith('.') && host.endsWith(suffix);
}

function ipRangeMatches(value, host) {
  const ip = ipv4ToNumber(host);
  if (ip === null) {
    return false;
  }
  const [start, end] = value.split('-', 2).map((item) => ipv4ToNumber(item.trim()));
  return start !== null && end !== null && ip >= Math.min(start, end) && ip <= Math.max(start, end);
}

function urlRegexMatches(pattern, url) {
  try {
    return new RegExp(pattern).test(url || '');
  } catch (_error) {
    return false;
  }
}

function routeSource(route) {
  if (route.actionType === 'direct') {
    return 'remote_direct';
  }
  if (route.matchType === 'default') {
    return 'proxy_default';
  }
  if (route.actionType === 'chain') {
    return 'remote_proxy';
  }
  return 'remote_proxy';
}

function urlHostname(value) {
  try {
    return new URL(value).hostname;
  } catch (_error) {
    return '';
  }
}

function escapePacString(value) {
  return String(value || '').replaceAll('\\', '\\\\').replaceAll("'", "\\'");
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

function applyProxy(state) {
  return chrome.proxy.settings.set({
    value: {
      mode: 'pac_script',
      pacScript: {
        data: buildPacScript(state)
      }
    },
    scope: 'regular'
  }).then(() => appendLog('info', 'proxy_applied', pacSummary(state)));
}

function authHeaders(token) {
  return token ? { Authorization: `Bearer ${token}` } : {};
}

function tenantHeaders(state) {
  return state.session && state.session.activeTenantId ? { 'X-Tenant-ID': state.session.activeTenantId } : {};
}

function readJSON(response) {
  return response.json().catch(() => null);
}

function apiRequest(state, path, options = {}) {
  const controlPlaneUrl = String(state.controlPlaneUrl || '').trim().replace(/\/$/, '');
  if (!controlPlaneUrl) {
    return Promise.reject(new Error('missing_control_plane_url'));
  }
  const headers = {
    ...(options.body ? { 'Content-Type': 'application/json' } : {}),
    ...(options.headers || {})
  };
  if (options.auth !== false && state.session.accessToken) {
    Object.assign(headers, authHeaders(state.session.accessToken));
  }
  if (options.tenant !== false) {
    Object.assign(headers, tenantHeaders(state));
  }
  return fetch(`${controlPlaneUrl}${path}`, {
    method: options.method || 'GET',
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined
  }).then((response) => {
    if (response.status === 401 && options.allowRefresh !== false && state.session.refreshToken) {
      return refreshSession(state).then((refreshed) => apiRequest(refreshed, path, { ...options, allowRefresh: false }));
    }
    return readJSON(response).then((payload) => {
      if (!response.ok) {
        const message = (payload && payload.message) || 'request_failed';
        return appendLog('error', 'api_request_failed', { path, status: response.status, message })
          .then(() => { throw new Error(message); });
      }
      return appendLog('info', 'api_request_ok', { path, status: response.status })
        .then(() => payload ? payload.data : null);
    });
  });
}

function login(controlPlaneUrl, account, password) {
  return fetch(`${String(controlPlaneUrl || '').trim().replace(/\/$/, '')}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ account, password })
  }).then((response) => readJSON(response).then((payload) => {
    if (!response.ok) {
      const message = (payload && payload.message) || 'login_failed';
      return appendLog('error', 'login_failed', { status: response.status, message })
        .then(() => { throw new Error(message); });
    }
    const memberships = Array.isArray(payload.data.tenantMemberships) ? payload.data.tenantMemberships : [];
    const activeTenantId = payload.data.activeTenantId || (memberships.length === 1 ? memberships[0].tenantId : '');
    return getState()
      .then((state) => {
        const nextState = mergeState({
          ...state,
          controlPlaneUrl: String(controlPlaneUrl || '').trim(),
          session: {
            account: payload.data.account.account,
            accessToken: payload.data.accessToken,
            refreshToken: payload.data.refreshToken,
            expiresAt: payload.data.expiresAt,
            proxyToken: '',
            proxyTokenExpiresAt: '',
            mustRotatePassword: Boolean(payload.data.mustRotatePassword),
            tenantMemberships: memberships,
            activeTenantId
          }
        });
        return persistState(nextState).then(() => nextState);
      })
      .then((nextState) => appendLog('info', 'login_ok', { account: nextState.session.account, controlPlaneUrl: nextState.controlPlaneUrl })
        .then(() => nextState.session.activeTenantId ? syncRemoteConfig(nextState) : null));
  }));
}

function testConnection(controlPlaneUrl) {
  const baseUrl = String(controlPlaneUrl || '').trim().replace(/\/$/, '');
  if (!baseUrl) {
    return Promise.reject(new Error('missing_control_plane_url'));
  }
  return fetch(`${baseUrl}/healthz`)
    .catch((error) => appendLog('error', 'test_connection_failed', { controlPlaneUrl: baseUrl, message: error.message || 'connection_failed' })
      .then(() => { throw new Error('connection_failed'); }))
    .then((response) => readJSON(response).then((payload) => {
      if (!response.ok) {
        const message = (payload && payload.message) || 'connection_failed';
        return appendLog('error', 'test_connection_failed', { controlPlaneUrl: baseUrl, status: response.status, message })
          .then(() => { throw new Error(message); });
      }
      return appendLog('info', 'test_connection_ok', { controlPlaneUrl: baseUrl })
        .then(() => payload ? payload.data : null);
    }));
}

function refreshSession(sourceState) {
  return (sourceState ? Promise.resolve(sourceState) : getState()).then((source) => {
    const state = mergeState(source);
    if (!state.controlPlaneUrl || !state.session.refreshToken) {
      throw new Error('missing_refresh_token');
    }
    return fetch(`${state.controlPlaneUrl.replace(/\/$/, '')}/api/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: state.session.refreshToken })
    }).then((response) => readJSON(response).then((payload) => {
      if (!response.ok) {
        const message = (payload && payload.message) || 'refresh_failed';
        return appendLog('error', 'refresh_failed', { status: response.status, message })
          .then(() => { throw new Error(message); });
      }
      const memberships = Array.isArray(payload.data.tenantMemberships) ? payload.data.tenantMemberships : state.session.tenantMemberships;
      const activeTenantId = state.session.activeTenantId || payload.data.activeTenantId || (memberships.length === 1 ? memberships[0].tenantId : '');
      const nextState = mergeState({
        ...state,
        session: {
          account: payload.data.account.account,
          accessToken: payload.data.accessToken,
          refreshToken: payload.data.refreshToken,
          expiresAt: payload.data.expiresAt,
          proxyToken: state.session.proxyToken,
          proxyTokenExpiresAt: state.session.proxyTokenExpiresAt,
          mustRotatePassword: Boolean(payload.data.mustRotatePassword),
          tenantMemberships: memberships,
          activeTenantId
        }
      });
      return persistState(nextState)
        .then(() => appendLog('info', 'refresh_ok', { account: nextState.session.account }))
        .then(() => nextState);
    }));
  });
}

function syncRemoteConfig(sourceState) {
  return (sourceState ? Promise.resolve(sourceState) : getState()).then((source) => {
    const state = mergeState(source);
    if (!state.session.activeTenantId) {
      throw new Error('tenant_required');
    }
    return apiRequest(state, '/api/v1/extension/bootstrap')
      .then((data) => {
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
            mustRotatePassword: Boolean(data.account && data.account.mustRotatePassword),
            proxyToken: data.proxyToken || '',
            proxyTokenExpiresAt: data.proxyTokenExpiresAt || ''
          }
        });
        return persistState(nextState)
          .then(() => appendLog('info', 'remote_config_synced', {
            policyRevision: nextState.remote.policyRevision,
            groups: nextState.remote.groups.length
          }));
      })
      .then(() => null);
  });
}

function selectTenant(tenantId) {
  return getState().then((state) => {
    const activeTenantId = String(tenantId || '').trim();
    const nextState = mergeState({
      ...state,
      enabled: false,
      session: {
        ...state.session,
        activeTenantId,
        proxyToken: '',
        proxyTokenExpiresAt: ''
      },
      remote: DEFAULT_STATE.remote,
      selection: DEFAULT_STATE.selection
    });
    return persistState(nextState)
      .then(() => appendLog('info', 'tenant_selected', { tenantId: activeTenantId }))
      .then(() => syncRemoteConfig(nextState));
  });
}

function logout() {
  return getState().then((state) => {
    const logoutRequest = state.controlPlaneUrl && state.session.accessToken
      ? fetch(`${state.controlPlaneUrl.replace(/\/$/, '')}/api/v1/auth/logout`, {
        method: 'POST',
        headers: authHeaders(state.session.accessToken)
      }).catch(() => {})
      : Promise.resolve();
    const nextState = mergeState({
      ...state,
      enabled: false,
      session: DEFAULT_STATE.session,
      remote: DEFAULT_STATE.remote,
      selection: DEFAULT_STATE.selection
    });
    return logoutRequest
      .then(() => persistState(nextState))
      .then(() => appendLog('info', 'logout_ok'));
  });
}

function testUrlRoute(targetUrl, options = {}) {
  return getState().then((state) => {
    const route = routePreviewForUrl(state, targetUrl);
    return runProxyProbes(state, targetUrl, route)
      .then((results) => {
        if (!options.saveMonitorTarget) {
          return { route, results };
        }
        return persistState({
          ...state,
          monitor: {
            targetUrl: parseTargetUrl(targetUrl).url || String(targetUrl || '').trim(),
            lastRunAt: new Date().toISOString(),
            results
          }
        }).then(() => ({ route, results }));
      });
  });
}

function runProxyMonitor() {
  return getState().then((state) => {
    if (!state.enabled || !state.monitor.targetUrl) {
      return;
    }
    const route = routePreviewForUrl(state, state.monitor.targetUrl);
    return runProxyProbes(state, state.monitor.targetUrl, route)
      .then((results) => persistState({
        ...state,
        monitor: {
          ...state.monitor,
          lastRunAt: new Date().toISOString(),
          results
        }
      }));
  });
}

function runProxyProbes(state, targetUrl, route) {
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

function runNodeProbe(endpoint, payload) {
  const startedAt = Date.now();
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 8000);
  return fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
    signal: controller.signal
  })
    .then((response) => response.json()
      .catch(() => null)
      .then((body) => ({
        protocol: payload.protocol,
        status: response.ok && body && body.status ? body.status : 'failed',
        latencyMs: Date.now() - startedAt,
        message: body && body.message ? body.message : `http_${response.status}`
      })))
    .catch((error) => ({
      protocol: payload.protocol,
      status: 'failed',
      latencyMs: Date.now() - startedAt,
      message: error.name === 'AbortError' ? 'probe_timeout' : error.message || 'probe_failed'
    }))
    .finally(() => clearTimeout(timeout));
}

function getCurrentTabInfo() {
  const queries = [{ active: true, currentWindow: true }, { active: true, lastFocusedWindow: true }];
  function next(index) {
    if (index >= queries.length) {
      return Promise.resolve(null);
    }
    return chrome.tabs.query(queries[index]).then((tabs) => {
      const tab = tabs[0];
    if (!tab || !tab.url) {
        return next(index + 1);
    }
    try {
      const parsed = new URL(tab.url);
      return {
        url: tab.url,
        host: parsed.hostname
      };
    } catch (_error) {
        return next(index + 1);
    }
    });
  }
  return next(0);
}

function getComputedState() {
  return Promise.all([getState(), getCurrentTabInfo()]).then(([state, currentTab]) => {
  return {
    state,
    session: state.session,
    remote: state.remote,
    activeGroup: activeGroupFrom(state),
    currentTab,
    currentRoute: routePreviewForUrl(state, currentTab && currentTab.url),
    monitorRoute: routePreviewForUrl(state, state.monitor.targetUrl)
  };
  });
}

function addHostToRule(kind, host) {
  const clean = sanitizeHost(host);
  if (!clean) {
    return getComputedState();
  }
  return getState().then((state) => {
  const overrides = {
    ...state.localOverrides,
    [kind]: uniqueStrings([...(state.localOverrides[kind] || []), clean])
  };
  return persistState({ ...state, localOverrides: overrides })
    .then(() => appendLog('info', 'local_override_added', { kind, host: clean }))
    .then(() => getComputedState());
  });
}

function removeHostFromRule(host) {
  const clean = sanitizeHost(host);
  if (!clean) {
    return getComputedState();
  }
  return getState().then((state) => {
  const overrides = {
    ...state.localOverrides,
    directHosts: uniqueStrings(state.localOverrides.directHosts).filter((item) => item !== clean),
    proxyHosts: uniqueStrings(state.localOverrides.proxyHosts).filter((item) => item !== clean)
  };
  return persistState({ ...state, localOverrides: overrides })
    .then(() => appendLog('info', 'local_override_removed', { host: clean }))
    .then(() => getComputedState());
  });
}

function computedAfter(operation) {
  return Promise.resolve()
    .then(() => operation())
    .then((result) => result || getComputedState());
}

function handleMessage(message) {
  if (!message || !message.type) {
    return Promise.resolve(null);
  }
  switch (message.type) {
    case 'get-state':
      return getComputedState();
    case 'get-diagnostic-logs':
      return diagnosticLogs();
    case 'clear-diagnostic-logs':
      return clearDiagnosticLogs();
    case 'record-diagnostic-event':
      return getState()
        .then((state) => appendLog('info', message.event || 'diagnostic_event', pacSummary(state)))
        .then(() => diagnosticLogs());
    case 'set-enabled':
      return computedAfter(() => setPartialState((state) => ({ ...state, enabled: Boolean(message.enabled) })));
    case 'set-theme-mode':
      return computedAfter(() => setPartialState((state) => ({ ...state, themeMode: message.themeMode === 'dark' ? 'dark' : 'vivid' })));
    case 'set-control-plane-url':
      return computedAfter(() => setPartialState((state) => ({ ...state, controlPlaneUrl: String(message.controlPlaneUrl || '').trim() })));
    case 'login':
      return computedAfter(() => login(message.controlPlaneUrl, message.account, message.password));
    case 'test-connection':
      return testConnection(message.controlPlaneUrl);
    case 'logout':
      return computedAfter(() => logout());
    case 'sync-remote-config':
      return computedAfter(() => syncRemoteConfig());
    case 'select-tenant':
      return computedAfter(() => selectTenant(message.tenantId));
    case 'test-url-route':
      return testUrlRoute(message.url, { saveMonitorTarget: Boolean(message.saveMonitorTarget) });
    case 'select-group':
      return computedAfter(() => setPartialState((state) => ({
        ...state,
        selection: {
          ...state.selection,
          activeGroupId: message.groupId || ''
        }
      })));
    case 'set-local-overrides':
      return computedAfter(() => setPartialState((state) => ({
        ...state,
        localOverrides: {
          directHosts: uniqueStrings(message.directHosts),
          proxyHosts: uniqueStrings(message.proxyHosts)
        }
      })));
    case 'add-current-host-to-direct':
      return getCurrentTabInfo().then((info) => addHostToRule('directHosts', (info && info.host) || ''));
    case 'add-current-host-to-proxy':
      return getCurrentTabInfo().then((info) => addHostToRule('proxyHosts', (info && info.host) || ''));
    case 'remove-current-host-override':
      return getCurrentTabInfo().then((info) => removeHostFromRule((info && info.host) || ''));
    default:
      return Promise.resolve(null);
  }
}

function registerMessageHandler() {
  chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
    handleMessage(message).then(sendResponse).catch((error) => {
      appendLog('error', 'runtime_message_failed', {
        type: message && message.type ? message.type : '',
        message: error.message || 'unexpected_error'
      }).catch(() => {});
      sendResponse({ error: error.message || 'unexpected_error' });
    });
    return true;
  });
}

let proxyAuthCache = {
  host: '',
  port: 0,
  token: ''
};

function updateProxyAuthCache(state) {
  const group = activeGroupFrom(state);
  proxyAuthCache = {
    host: group && group.proxyHost ? String(group.proxyHost) : '',
    port: group && group.proxyPort ? Number(group.proxyPort) : 0,
    token: state.session && state.session.proxyToken ? String(state.session.proxyToken) : ''
  };
}

function matchesProxyChallenge(details) {
  if (!details || !details.isProxy || !proxyAuthCache.token) {
    return false;
  }
  const challenger = details.challenger || {};
  return challenger.host === proxyAuthCache.host && Number(challenger.port || 0) === proxyAuthCache.port;
}

function registerProxyAuthHandler() {
  if (!chrome.webRequest || !chrome.webRequest.onAuthRequired) {
    return;
  }
  chrome.webRequest.onAuthRequired.addListener(
    (details) => {
      if (!matchesProxyChallenge(details)) {
        return {};
      }
      appendLog('info', 'proxy_auth_supplied', {
        host: proxyAuthCache.host,
        port: proxyAuthCache.port
      }).catch(() => {});
      return {
        authCredentials: {
          username: 'token',
          password: proxyAuthCache.token
        }
      };
    },
    { urls: ['<all_urls>'] },
    ['blocking']
  );
}

function broadcastState() {
  return getComputedState()
    .then((payload) => chrome.runtime.sendMessage({ type: 'state-updated', payload }))
    .catch(() => {});
}

function ensureMonitorAlarm() {
  if (chrome.alarms) {
    chrome.alarms.create('proxy-monitor', { periodInMinutes: 1 });
  }
}

configureStateEffects((state) => {
  updateProxyAuthCache(state);
  return applyProxy(state).then(() => broadcastState());
});

chrome.runtime.onInstalled.addListener(() => {
  ensureMonitorAlarm();
  getState()
    .then((state) => persistState(state))
    .then(() => appendLog('info', 'extension_installed'))
    .catch((error) => appendLog('error', 'extension_installed_failed', { message: error.message || 'install_failed' }));
});

chrome.runtime.onStartup.addListener(() => {
  ensureMonitorAlarm();
  getState()
    .then((state) => {
      updateProxyAuthCache(state);
      return applyProxy(state);
    })
    .then(() => appendLog('info', 'extension_startup'))
    .catch((error) => appendLog('error', 'extension_startup_failed', { message: error.message || 'startup_failed' }));
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

chrome.storage.onChanged.addListener((changes, areaName) => {
  const nextState = handleStateStorageChange(changes, areaName);
  if (!nextState) {
    return;
  }
  applyProxy(nextState)
    .then(() => broadcastState())
    .catch((error) => appendLog('error', 'storage_change_apply_failed', { message: error.message || 'apply_failed' }));
});

function bootstrap() {
  registerProxyAuthHandler();
  registerMessageHandler();
  return getState().then((state) => updateProxyAuthCache(state));
}

bootstrap().catch((error) => {
  appendLog('error', 'service_worker_bootstrap_failed', { message: error.message || 'bootstrap_failed' }).catch(() => {});
});
