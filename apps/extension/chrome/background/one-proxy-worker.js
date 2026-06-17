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
const SESSION_STORAGE_KEY = 'oneProxySession';
const PERSISTENT_SESSION_STORAGE_KEY = 'oneProxyPersistentSession';

const DEFAULT_ROUTE_EVALUATION = {
  defaultClientMode: 'direct',
  defaultNodeMode: 'deny',
  ruleOrder: 'priority_asc_then_id_asc',
  noMatchNodeDenyReason: 'route_not_found',
  supportedMatchTypes: ['domain', 'domain_suffix', 'ip', 'ip_cidr', 'protocol', 'default'],
  supportedActions: ['chain', 'direct', 'deny']
};

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
    nodes: [],
    accessPaths: [],
    routes: [],
    routeEvaluation: DEFAULT_ROUTE_EVALUATION
  },
  selection: {
    activeAccessPathId: ''
  },
  localOverrides: {
    directHosts: [],
    proxyHosts: []
  },
  localHelper: {
    enabled: false,
    scheme: 'SOCKS5',
    host: '127.0.0.1',
    port: 1080
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

function normalizeNode(node) {
  return {
    id: String(node.id || ''),
    name: String(node.name || ''),
    mode: String(node.mode || ''),
    scopeKey: String(node.scopeKey || ''),
    parentNodeId: String(node.parentNodeId || ''),
    enabled: Boolean(node.enabled),
    status: String(node.status || ''),
    publicHost: String(node.publicHost || ''),
    publicPort: Number(node.publicPort || 0)
  };
}

function normalizeAccessPath(path) {
  return {
    id: String(path.id || ''),
    name: String(path.name || ''),
    chainId: String(path.chainId || ''),
    mode: String(path.mode || ''),
    protocol: String(path.protocol || ''),
    serviceType: String(path.serviceType || ''),
    targetNodeId: String(path.targetNodeId || ''),
    entryNodeId: String(path.entryNodeId || ''),
    relayNodeIds: uniqueStrings(path.relayNodeIds),
    listenHost: String(path.listenHost || ''),
    listenPort: Number(path.listenPort || 0),
    targetProtocol: String(path.targetProtocol || ''),
    targetHost: String(path.targetHost || ''),
    targetPort: Number(path.targetPort || 0),
    targetSni: String(path.targetSni || ''),
    tlsMode: String(path.tlsMode || ''),
    authMode: String(path.authMode || ''),
    enabled: Boolean(path.enabled),
    options: path.options && typeof path.options === 'object' ? { ...path.options } : {},
    topology: Array.isArray(path.topology) ? path.topology.map(normalizeTopologyHop) : [],
    health: normalizeAccessPathHealth(path.health)
  };
}

function normalizeAccessPathHealth(health) {
  return {
    status: String((health && health.status) || 'unknown'),
    reason: String((health && health.reason) || ''),
    checkedAt: String((health && health.checkedAt) || '')
  };
}

function normalizeRoute(route) {
  return {
    id: String(route.id || ''),
    priority: Number(route.priority || 0),
    matchType: String(route.matchType || ''),
    matchValue: String(route.matchValue || ''),
    actionType: String(route.actionType || ''),
    chainId: String(route.chainId || ''),
    accessPathId: String(route.accessPathId || ''),
    destinationScope: String(route.destinationScope || ''),
    enabled: Boolean(route.enabled),
    topology: Array.isArray(route.topology) ? route.topology.map(normalizeTopologyHop) : []
  };
}

function normalizeTopologyHop(hop) {
  const nodeId = String(hop.nodeId || '');
  const nodeName = String(hop.nodeName || '');
  return {
    nodeId,
    nodeName,
    mode: String(hop.mode || ''),
    scopeKey: String(hop.scopeKey || ''),
    publicHost: String(hop.publicHost || ''),
    publicPort: Number(hop.publicPort || 0),
    transport: String(hop.transport || ''),
    id: nodeId,
    name: nodeName
  };
}

function normalizeRouteEvaluation(contract) {
  const source = contract && typeof contract === 'object' ? contract : {};
  return {
    defaultClientMode: source.defaultClientMode === 'direct' ? 'direct' : DEFAULT_ROUTE_EVALUATION.defaultClientMode,
    defaultNodeMode: source.defaultNodeMode === 'deny' ? 'deny' : DEFAULT_ROUTE_EVALUATION.defaultNodeMode,
    ruleOrder: source.ruleOrder === 'priority_asc_then_id_asc' ? source.ruleOrder : DEFAULT_ROUTE_EVALUATION.ruleOrder,
    noMatchNodeDenyReason: source.noMatchNodeDenyReason === 'route_not_found' ? source.noMatchNodeDenyReason : DEFAULT_ROUTE_EVALUATION.noMatchNodeDenyReason,
    supportedMatchTypes: uniqueStrings(source.supportedMatchTypes || DEFAULT_ROUTE_EVALUATION.supportedMatchTypes),
    supportedActions: uniqueStrings(source.supportedActions || DEFAULT_ROUTE_EVALUATION.supportedActions)
  };
}

function normalizeTenantMembership(membership) {
  return {
    tenantId: String(membership.tenantId || ''),
    tenantName: String(membership.tenantName || ''),
    role: String(membership.role || ''),
    joinedAt: String(membership.joinedAt || '')
  };
}

function publicSession(rawSession) {
  return {
    account: String(rawSession.account || ''),
    accessToken: String(rawSession.accessToken || ''),
    refreshToken: String(rawSession.refreshToken || ''),
    expiresAt: String(rawSession.expiresAt || ''),
    proxyToken: String(rawSession.proxyToken || ''),
    proxyTokenExpiresAt: String(rawSession.proxyTokenExpiresAt || ''),
    mustRotatePassword: Boolean(rawSession.mustRotatePassword),
    tenantMemberships: Array.isArray(rawSession.tenantMemberships) ? rawSession.tenantMemberships.map(normalizeTenantMembership) : [],
    activeTenantId: String(rawSession.activeTenantId || '')
  };
}

function mergeState(raw) {
  const rest = raw || {};
  const rawSession = publicSession(rest.session || {});
  const remote = rest.remote || {};
  const selection = rest.selection || {};
  const state = {
    ...DEFAULT_STATE,
    enabled: Boolean(rest.enabled),
    themeMode: rest.themeMode === 'dark' ? 'dark' : 'vivid',
    controlPlaneUrl: String(rest.controlPlaneUrl || ''),
    session: rawSession,
    remote: {
      policyRevision: String(remote.policyRevision || ''),
      fetchedAt: String(remote.fetchedAt || ''),
      nodes: Array.isArray(remote.nodes) ? remote.nodes.map(normalizeNode) : [],
      accessPaths: Array.isArray(remote.accessPaths) ? remote.accessPaths.map(normalizeAccessPath) : [],
      routes: Array.isArray(remote.routes) ? remote.routes.map(normalizeRoute) : [],
      routeEvaluation: normalizeRouteEvaluation(remote.routeEvaluation)
    },
    selection: {
      activeAccessPathId: String(selection.activeAccessPathId || '')
    },
    localOverrides: {
      directHosts: uniqueStrings(rest.localOverrides && rest.localOverrides.directHosts),
      proxyHosts: uniqueStrings(rest.localOverrides && rest.localOverrides.proxyHosts)
    },
    localHelper: {
      enabled: Boolean(rest.localHelper && rest.localHelper.enabled),
      scheme: rest.localHelper && rest.localHelper.scheme === 'PROXY' ? 'PROXY' : 'SOCKS5',
      host: String((rest.localHelper && rest.localHelper.host) || '127.0.0.1').trim(),
      port: Number((rest.localHelper && rest.localHelper.port) || 1080)
    },
    monitor: {
      targetUrl: String((rest.monitor && rest.monitor.targetUrl) || ''),
      lastRunAt: String((rest.monitor && rest.monitor.lastRunAt) || ''),
      results: Array.isArray(rest.monitor && rest.monitor.results) ? rest.monitor.results : []
    }
  };
  if (!state.session.tenantMemberships.find((membership) => membership.tenantId === state.session.activeTenantId)) {
    state.session.activeTenantId = state.session.tenantMemberships.length === 1 ? state.session.tenantMemberships[0].tenantId : '';
  }
  const selectedPath = state.remote.accessPaths.find((path) => path.id === state.selection.activeAccessPathId);
  if (!selectedPath) {
    const firstEnabled = state.remote.accessPaths.find((path) => path.enabled);
    state.selection.activeAccessPathId = (firstEnabled || state.remote.accessPaths[0] || {}).id || '';
  }
  return state;
}

function sessionSecretsFrom(state) {
  return {
    accessToken: state.session.accessToken,
    refreshToken: state.session.refreshToken,
    proxyToken: state.session.proxyToken
  };
}

function stateWithoutSessionSecrets(state) {
  return {
    ...state,
    session: {
      ...state.session,
      accessToken: '',
      refreshToken: '',
      proxyToken: ''
    }
  };
}

function mergeSessionSecrets(durableState, secrets) {
  return {
    ...durableState,
    session: {
      ...(durableState.session || {}),
      ...(secrets || {})
    }
  };
}

function activePathView(path, state) {
  if (!path) {
    return null;
  }
  const entryNode = state.remote.nodes.find((node) => node.id === path.entryNodeId);
  return {
    ...path,
    proxyScheme: path.protocol === 'https' ? 'HTTPS' : 'PROXY',
    proxyHost: path.listenHost,
    proxyPort: path.listenPort,
    entryNodeName: (entryNode && entryNode.name) || path.entryNodeId,
    entryNodeId: path.entryNodeId
  };
}

function getState() {
  if (stateCache) {
    return Promise.resolve(structuredClone(stateCache));
  }
  return Promise.all([
    chrome.storage.local.get(STORAGE_KEY),
    chrome.storage.local.get(PERSISTENT_SESSION_STORAGE_KEY),
    chrome.storage.session.get(SESSION_STORAGE_KEY)
  ]).then(([stored, persistentSessionStored, sessionStored]) => {
    const durableState = stateWithoutSessionSecrets(mergeState(stored[STORAGE_KEY] || {}));
    stateCache = mergeState(mergeSessionSecrets(durableState, {
      ...(persistentSessionStored[PERSISTENT_SESSION_STORAGE_KEY] || {}),
      ...(sessionStored[SESSION_STORAGE_KEY] || {})
    }));
    return structuredClone(stateCache);
  });
}

function activeAccessPathFrom(state) {
  const path = state.remote.accessPaths.find((item) => item.id === state.selection.activeAccessPathId) || state.remote.accessPaths[0] || null;
  return activePathView(path, state);
}

function accessPathById(state, accessPathId) {
  const path = state.remote.accessPaths.find((item) => item.id === accessPathId) || null;
  return activePathView(path, state);
}

function persistState(nextState) {
  stateCache = mergeState(nextState);
  return Promise.all([
    chrome.storage.local.set({ [STORAGE_KEY]: stateWithoutSessionSecrets(stateCache) }),
    chrome.storage.local.set({ [PERSISTENT_SESSION_STORAGE_KEY]: sessionSecretsFrom(stateCache) }),
    chrome.storage.session.set({ [SESSION_STORAGE_KEY]: sessionSecretsFrom(stateCache) })
  ])
    .then(() => persistEffects(stateCache))
    .then(() => structuredClone(stateCache));
}

function setPartialState(mutator) {
  return getState()
    .then((current) => mutator(structuredClone(current)))
    .then((next) => persistState(next));
}

function handleStateStorageChange(changes, areaName) {
  if (areaName === 'local' && changes[STORAGE_KEY]) {
    const secrets = stateCache ? sessionSecretsFrom(stateCache) : {};
    stateCache = mergeState(mergeSessionSecrets(stateWithoutSessionSecrets(mergeState(changes[STORAGE_KEY].newValue || {})), secrets));
    return stateCache;
  }
  if (areaName === 'local' && changes[PERSISTENT_SESSION_STORAGE_KEY]) {
    stateCache = mergeState(mergeSessionSecrets(stateCache || {}, changes[PERSISTENT_SESSION_STORAGE_KEY].newValue || {}));
    return stateCache;
  }
  if (areaName === 'session' && changes[SESSION_STORAGE_KEY]) {
    stateCache = mergeState(mergeSessionSecrets(stateCache || {}, changes[SESSION_STORAGE_KEY].newValue || {}));
    return stateCache;
  }
  return null;
}

function wildcardToRegExp(pattern) {
  return new RegExp(`^${String(pattern || '').replace(/[.+?^${}()|[\]\\]/g, '\\$&').replaceAll('*', '.*')}$`, 'i');
}

function sanitizeHost(value) {
  return String(value || '').trim().toLowerCase();
}

function hostMatches(patterns, host) {
  const cleanHost = sanitizeHost(host);
  return uniqueStrings(patterns).some((pattern) => hostMatchesPattern(pattern, cleanHost));
}

function hostMatchesPattern(pattern, host) {
  const cleanPattern = sanitizeHost(pattern);
  if (cleanPattern.startsWith('*.') || cleanPattern.startsWith('.')) {
    return domainSuffixMatches(cleanPattern, host);
  }
  return wildcardToRegExp(cleanPattern).test(host);
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
  if (!cleanHost) {
    return { mode: 'unknown', source: 'no_site', host: '', topology: [] };
  }
  if (!state.enabled) {
    return { ...emptyResult(parsed), mode: 'direct', source: 'proxy_off', host: cleanHost };
  }
  return evaluateClientRoute(state, parsed);
}

function evaluateClientRoute(state, input) {
  const parsed = normalizeRouteInput(input);
  const cleanHost = sanitizeHost(parsed.host);
  if (isLocalSafetyDirect(state, parsed)) {
    return {
      ...emptyResult(parsed),
      mode: 'direct',
      source: 'local_safety_direct',
      host: cleanHost
    };
  }
  const route = firstMatchingRoute(state, parsed);
  if (!route) {
    return {
      ...emptyResult(parsed),
      mode: 'direct',
      source: 'default_direct',
      host: cleanHost
    };
  }
  return applyRouteAction(state, route, parsed);
}

function emptyResult(parsed) {
  return {
    routeId: '',
    chainId: '',
    accessPathId: '',
    targetHost: sanitizeHost(parsed.host),
    targetPort: parsed.port,
    protocol: parsed.protocol,
    topology: [],
    denyReason: '',
    host: sanitizeHost(parsed.host),
    port: parsed.port,
    rule: null
  };
}

function applyRouteAction(state, route, parsed) {
  const base = {
    ...emptyResult(parsed),
    source: 'policy',
    routeId: route.id,
    chainId: route.chainId,
    accessPathId: route.accessPathId,
    rule: route
  };
  if (route.actionType === 'direct') {
    return { ...base, mode: 'direct' };
  }
  if (route.actionType === 'deny') {
    return { ...base, mode: 'deny', denyReason: 'route_denied' };
  }
  if (route.actionType === 'chain') {
    const accessPath = accessPathById(state, route.accessPathId);
    if (!isUsableAccessPath(accessPath)) {
      return { ...base, mode: 'deny', denyReason: 'access_path_unavailable' };
    }
    return {
      ...base,
      mode: 'proxy',
      topology: route.topology.length > 0 ? route.topology : accessPath.topology
    };
  }
  return { ...base, mode: 'deny', denyReason: 'route_denied' };
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
    return normalizeRouteInput({ url: raw, host: raw.split('/')[0], protocol: 'http', port: 80 });
  }
}

function normalizeRouteInput(input) {
  const parsed = typeof input === 'object' && input ? input : parseTargetUrl(input);
  const protocol = String(parsed.protocol || 'http').replace(':', '').toLowerCase();
  return {
    url: String(parsed.url || ''),
    host: sanitizeHost(parsed.host),
    port: Number(parsed.port || defaultPort(protocol)),
    protocol,
    accessPathId: String(parsed.accessPathId || '')
  };
}

function defaultPort(protocol) {
  if (protocol === 'http' || protocol === 'ws') {
    return 80;
  }
  if (protocol === 'https' || protocol === 'wss' || protocol === 'connect') {
    return 443;
  }
  if (protocol === 'ssh') {
    return 22;
  }
  return 0;
}

function sortedEnabledRoutes(state) {
  return [...(state.remote.routes || [])]
    .filter((route) => route.enabled)
    .sort((left, right) => {
      const priority = Number(left.priority || 0) - Number(right.priority || 0);
      return priority || String(left.id || '').localeCompare(String(right.id || ''));
    });
}

function firstMatchingRoute(state, parsed) {
  return sortedEnabledRoutes(state).find((route) => routeMatches(route, parsed)) || null;
}

function routeMatches(route, target) {
  const parsed = normalizeRouteInput(target);
  const value = String(route.matchValue || '').toLowerCase();
  const cleanHost = sanitizeHost(parsed.host);
  switch (route.matchType) {
    case 'domain':
      return cleanHost === value;
    case 'domain_suffix':
      return domainSuffixMatches(value, cleanHost);
    case 'ip':
      return cleanHost === value;
    case 'ip_cidr':
      return cidrMatches([route.matchValue], cleanHost);
    case 'protocol':
      return parsed.protocol === value;
    case 'default':
      return true;
    default:
      return false;
  }
}

function domainSuffixMatches(value, host) {
  const suffix = value.replace(/^\*\./, '').replace(/^\./, '');
  return Boolean(suffix) && (host === suffix || host.endsWith(`.${suffix}`));
}

function isLocalSafetyDirect(state, target) {
  const parsed = normalizeRouteInput(target);
  const controlPlaneHost = urlHostname(state.controlPlaneUrl);
  if (parsed.host && parsed.host === controlPlaneHost) {
    return true;
  }
  if (isLoopbackHost(parsed.host)) {
    return true;
  }
  const helper = state.localHelper || {};
  return Boolean(helper.enabled && parsed.host === sanitizeHost(helper.host) && Number(parsed.port || 0) === Number(helper.port || 0));
}

function isLoopbackHost(host) {
  const cleanHost = sanitizeHost(host).replace(/^\[/, '').replace(/\]$/, '');
  return cleanHost === 'localhost' ||
    cleanHost.endsWith('.localhost') ||
    cleanHost === '::1' ||
    cleanHost.startsWith('127.');
}

function isUsableAccessPath(accessPath) {
  return Boolean(accessPath &&
    accessPath.enabled &&
    accessPath.authMode === 'proxy_token' &&
    accessPath.serviceType === 'http_forward_proxy' &&
    accessPath.listenHost &&
    accessPath.listenPort > 0 &&
    (!accessPath.health || accessPath.health.status !== 'unavailable'));
}

function accessPathProxyTarget(accessPath) {
  if (!isUsableAccessPath(accessPath)) {
    return '';
  }
  const scheme = accessPath.protocol === 'https' ? 'HTTPS' : 'PROXY';
  return `${scheme} ${accessPath.listenHost}:${accessPath.listenPort}`;
}

function urlHostname(value) {
  try {
    return new URL(value).hostname.toLowerCase();
  } catch (_error) {
    return '';
  }
}

function denyProxyTarget() {
  return 'PROXY 127.0.0.1:9';
}

function localHelperTarget(state) {
  const helper = state.localHelper || {};
  return helper.enabled && helper.host && helper.port ? `${helper.scheme || 'SOCKS5'} ${helper.host}:${helper.port}` : '';
}

function compiledRules(state) {
  const helperTarget = localHelperTarget(state);
  return sortedEnabledRoutes(state).map((route) => {
    const accessPath = accessPathById(state, route.accessPathId);
    const proxyTarget = route.actionType === 'chain' && isUsableAccessPath(accessPath)
      ? helperTarget || accessPathProxyTarget(accessPath)
      : '';
    return {
      id: route.id,
      matchType: route.matchType,
      matchValue: route.matchValue,
      actionType: route.actionType,
      chainId: route.chainId,
      accessPathId: route.accessPathId,
      proxyTarget
    };
  });
}

function buildPacScript(state) {
  const helper = state.localHelper || {};
  return `
const enabled = ${state.enabled ? 'true' : 'false'};
const rules = ${JSON.stringify(compiledRules(state))};
const controlPlaneHost = ${JSON.stringify(urlHostname(state.controlPlaneUrl))};
const helperHost = ${JSON.stringify(helper.enabled ? String(helper.host || '').toLowerCase() : '')};
const helperPort = ${Number(helper.enabled ? helper.port || 0 : 0)};
const denyTarget = ${JSON.stringify(denyProxyTarget())};

function protocolFromUrl(url) {
  const index = String(url || '').indexOf(':');
  return index > 0 ? String(url).slice(0, index).toLowerCase() : 'http';
}

function portFromUrl(url, protocol) {
  const match = String(url || '').match(/^[a-z][a-z0-9+.-]*:\\/\\/(?:[^@/]*@)?(?:\\[[^\\]]+\\]|[^/:?#]+)(?::(\\d+))?/i);
  if (match && match[1]) {
    return Number(match[1]);
  }
  if (protocol === 'http' || protocol === 'ws') {
    return 80;
  }
  if (protocol === 'https' || protocol === 'wss' || protocol === 'connect') {
    return 443;
  }
  if (protocol === 'ssh') {
    return 22;
  }
  return 0;
}

function sanitizeHost(host) {
  return String(host || '').toLowerCase();
}

function domainSuffixMatches(value, host) {
  const suffix = String(value || '').replace(/^\\*\\./, '').replace(/^\\./, '');
  return Boolean(suffix) && (host === suffix || dnsDomainIs(host, '.' + suffix));
}

function ipv4ToNumber(value) {
  const parts = String(value || '').split('.');
  if (parts.length !== 4) {
    return null;
  }
  let result = 0;
  for (let index = 0; index < parts.length; index += 1) {
    const octet = Number(parts[index]);
    if (!Number.isInteger(octet) || octet < 0 || octet > 255) {
      return null;
    }
    result = (result * 256) + octet;
  }
  return result;
}

function cidrMatches(pattern, host) {
  const ip = ipv4ToNumber(host);
  if (ip === null) {
    return false;
  }
  const parts = String(pattern || '').split('/');
  const networkIp = ipv4ToNumber(parts[0]);
  const prefix = Number(parts[1]);
  if (networkIp === null || !Number.isInteger(prefix) || prefix < 0 || prefix > 32) {
    return false;
  }
  const mask = prefix === 0 ? 0 : (0xffffffff << (32 - prefix)) >>> 0;
  return (ip & mask) === (networkIp & mask);
}

function isLoopbackHost(host) {
  return host === 'localhost' ||
    dnsDomainIs(host, '.localhost') ||
    host === '::1' ||
    host.indexOf('127.') === 0;
}

function isLocalSafetyDirect(host, port) {
  if (host && host === controlPlaneHost) {
    return true;
  }
  if (isLoopbackHost(host)) {
    return true;
  }
  return Boolean(helperHost && host === helperHost && Number(port || 0) === helperPort);
}

function routeMatches(rule, target) {
  switch (rule.matchType) {
    case 'domain':
      return target.host === String(rule.matchValue || '').toLowerCase();
    case 'domain_suffix':
      return domainSuffixMatches(rule.matchValue, target.host);
    case 'ip':
      return target.host === String(rule.matchValue || '').toLowerCase();
    case 'ip_cidr':
      return cidrMatches(rule.matchValue, target.host);
    case 'protocol':
      return target.protocol === String(rule.matchValue || '').toLowerCase();
    case 'default':
      return true;
    default:
      return false;
  }
}

function FindProxyForURL(url, host) {
  if (!enabled) {
    return 'DIRECT';
  }
  const protocol = protocolFromUrl(url);
  const port = portFromUrl(url, protocol);
  const target = {
    host: sanitizeHost(host),
    protocol,
    port
  };
  if (isLocalSafetyDirect(target.host, target.port)) {
    return 'DIRECT';
  }
  for (let index = 0; index < rules.length; index += 1) {
    const rule = rules[index];
    if (!routeMatches(rule, target)) {
      continue;
    }
    if (rule.actionType === 'direct') {
      return 'DIRECT';
    }
    if (rule.actionType === 'chain') {
      return rule.proxyTarget || denyTarget;
    }
    return denyTarget;
  }
  return 'DIRECT';
}
`;
}

function pacSummary(state) {
  const helperTarget = localHelperTarget(state);
  const activePath = accessPathById(state, state.selection.activeAccessPathId);
  const activeTarget = helperTarget || accessPathProxyTarget(activePath) || 'DIRECT';
  const rules = compiledRules(state);
  return {
    enabled: Boolean(state.enabled),
    activeAccessPathId: activePath ? activePath.id : '',
    activeAccessPathName: activePath ? activePath.name : '',
    proxyTarget: activeTarget,
    localHelper: helperTarget,
    accessPaths: state.remote.accessPaths.length,
    routes: state.remote.routes.length,
    enabledRoutes: rules.length,
    chainRoutes: rules.filter((rule) => rule.actionType === 'chain').length,
    directRoutes: rules.filter((rule) => rule.actionType === 'direct').length,
    denyRoutes: rules.filter((rule) => rule.actionType === 'deny').length,
    proxyTargets: uniqueStrings(rules.map((rule) => rule.proxyTarget)).length
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
  return token ? { 'X-One-Proxy-Access-Token': token } : {};
}

function tenantHeaders(state) {
  return state.session && state.session.activeTenantId ? { 'X-One-Proxy-Tenant-ID': state.session.activeTenantId } : {};
}

function readJSON(response) {
  return response.json().catch(() => null);
}

function normalizeControlPlaneUrl(value) {
  const clean = String(value || '').trim().replace(/\/+$/, '');
  if (!clean) {
    return '';
  }
  return /^[a-z][a-z0-9+.-]*:\/\//i.test(clean) ? clean : `https://${clean}`;
}

function apiRequest(state, path, options = {}) {
  const controlPlaneUrl = normalizeControlPlaneUrl(state.controlPlaneUrl);
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
  const normalizedControlPlaneUrl = normalizeControlPlaneUrl(controlPlaneUrl);
  if (!normalizedControlPlaneUrl) {
    return Promise.reject(new Error('missing_control_plane_url'));
  }
  return fetch(`${normalizedControlPlaneUrl}/api/auth/login`, {
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
          controlPlaneUrl: normalizedControlPlaneUrl,
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
  const baseUrl = normalizeControlPlaneUrl(controlPlaneUrl);
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
    return fetch(`${normalizeControlPlaneUrl(state.controlPlaneUrl)}/api/auth/refresh`, {
      method: 'POST',
      headers: { 'X-One-Proxy-Refresh-Token': state.session.refreshToken }
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
    return apiRequest(state, '/api/proxy/extension/bootstrap')
      .then((data) => {
        if (!data || data.schemaVersion !== 'v2.1.0') {
          throw new Error('invalid_bootstrap');
        }
        const nextState = mergeState({
          ...state,
          remote: {
            policyRevision: data.policyRevision || '',
            fetchedAt: data.fetchedAt || '',
            nodes: Array.isArray(data.nodes) ? data.nodes : [],
            accessPaths: Array.isArray(data.accessPaths) ? data.accessPaths : [],
            routes: Array.isArray(data.routes) ? data.routes : [],
            routeEvaluation: data.routeEvaluation || DEFAULT_STATE.remote.routeEvaluation
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
            accessPaths: nextState.remote.accessPaths.length,
            routes: nextState.remote.routes.length
          }));
      })
      .then(() => null);
  });
}

function syncRemoteConfigIfReady(sourceState) {
  return (sourceState ? Promise.resolve(sourceState) : getState()).then((source) => {
    const state = mergeState(source);
    if (!state.controlPlaneUrl || !state.session.accessToken || !state.session.activeTenantId) {
      return null;
    }
    return syncRemoteConfig(state).catch((error) => appendLog('error', 'auto_remote_sync_failed', {
      message: error.message || 'sync_failed'
    }).then(() => null));
  });
}

function getExtensionPageStatus(state, params) {
  const query = new URLSearchParams();
  if (params.host) {
    query.set('host', params.host);
  }
  if (params.routeId) {
    query.set('routeId', params.routeId);
  }
  if (params.chainId) {
    query.set('chainId', params.chainId);
  }
  return apiRequest(state, `/api/proxy/extension/page/status?${query.toString()}`);
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
      ? fetch(`${normalizeControlPlaneUrl(state.controlPlaneUrl)}/api/auth/logout`, {
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
  const accessPath = route && route.accessPathId ? accessPathById(state, route.accessPathId) : null;
  const parsed = parseTargetUrl(targetUrl);
  if (!state.enabled || !accessPath || !accessPath.listenHost || !accessPath.listenPort || !route || route.mode !== 'proxy') {
    return probeProtocols().map((protocol) => ({ protocol, status: 'skipped', latencyMs: 0, message: 'proxy_not_applied' }));
  }
  const remainingHopNodeIds = (route.topology || []).map((node) => node.id).filter(Boolean).slice(1);
  const endpoint = `http://${accessPath.listenHost}:${accessPath.listenPort}/api/control/relay/probe`;
  return Promise.all(probeProtocols().map((protocol) => runNodeProbe(state, endpoint, {
    protocol,
    remainingHopNodeIds,
    targetHost: parsed.host,
    targetPort: parsed.port
  })));
}

function probeProtocols() {
  return ['http', 'https', 'ws', 'tcp', 'udp'];
}

function oneProxyTokenHeaders(state) {
  const token = state.session && state.session.proxyToken ? String(state.session.proxyToken) : '';
  return token ? { 'X-One-Proxy-Token': token } : {};
}

function runNodeProbe(state, endpoint, payload) {
  const startedAt = Date.now();
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 8000);
  return fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...oneProxyTokenHeaders(state) },
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

const pagesByTab = new Map();
const requestsById = new Map();

function nowIso() {
  return new Date().toISOString();
}

function pageFor(tabId, url, options = {}) {
  const host = hostOf(url);
  const current = pagesByTab.get(tabId);
  if (!options.reset && current && (!url || current.url === url || current.host === host || !host)) {
    return current;
  }
  const page = {
    url,
    host,
    openedAt: nowIso(),
    requestCount: 0,
    responseCount: 0,
    proxiedRequestCount: 0,
    directRequestCount: 0,
    failureCount: 0,
    uploadBytes: 0,
    downloadBytes: 0,
    latencyMs: 0,
    latencyTotalMs: 0,
    latencyCount: 0,
    statusCode: 0,
    httpErrorCount: 0,
    errorCodeCount: {},
    cacheStatus: '',
    cacheStoredAt: '',
    cacheAgeSeconds: 0,
    cacheResponseCount: 0,
    lastErrorCode: '',
    lastErrorMessage: ''
  };
  pagesByTab.set(tabId, page);
  return page;
}

function hostOf(url) {
  try {
    return new URL(url).hostname;
  } catch (_error) {
    return '';
  }
}

function estimateUploadBytes(requestBody) {
  if (!requestBody) {
    return 0;
  }
  if (Array.isArray(requestBody.raw)) {
    return requestBody.raw.reduce((total, item) => total + (item.bytes ? item.bytes.byteLength : 0), 0);
  }
  if (!requestBody.formData) {
    return 0;
  }
  return Object.entries(requestBody.formData).reduce((total, [key, values]) => {
    const valueBytes = (values || []).reduce((sum, value) => sum + String(value || '').length, 0);
    return total + key.length + valueBytes;
  }, 0);
}

function contentLength(headers) {
  const header = headerValue(headers, 'content-length');
  const value = header ? Number(header) : 0;
  return Number.isFinite(value) && value > 0 ? value : 0;
}

function headerValue(headers, name) {
  const header = (headers || []).find((item) => String(item.name || '').toLowerCase() === name);
  return header ? String(header.value || '') : '';
}

function trackCacheHeaders(page, headers) {
  const cacheStatus = headerValue(headers, 'x-one-proxy-cache');
  if (!cacheStatus) {
    return;
  }
  page.cacheStatus = cacheStatus;
  page.cacheStoredAt = headerValue(headers, 'x-one-proxy-cache-stored-at');
  page.cacheAgeSeconds = Number(headerValue(headers, 'x-one-proxy-cache-age-seconds') || 0) || 0;
  page.cacheResponseCount += 1;
}

function trackStatusHeaders(page, details) {
  const statusCode = Number(details.statusCode || 0);
  if (statusCode <= 0) {
    return;
  }
  page.statusCode = statusCode;
  if (statusCode < 400) {
    return;
  }
  const errorCode = headerValue(details.responseHeaders, 'x-one-proxy-error') || `http_${statusCode}`;
  page.failureCount += 1;
  page.httpErrorCount += 1;
  page.lastErrorCode = errorCode;
  page.lastErrorMessage = errorCode;
  page.errorCodeCount[errorCode] = Number(page.errorCodeCount[errorCode] || 0) + 1;
}

function recordLatency(page, tracked) {
  if (!page || !tracked || tracked.responseSeen) {
    return;
  }
  const latencyMs = Date.now() - tracked.startedAt;
  if (latencyMs <= 0) {
    return;
  }
  tracked.responseSeen = true;
  page.responseCount += 1;
  page.latencyTotalMs += latencyMs;
  page.latencyCount += 1;
  page.latencyMs = Math.round(page.latencyTotalMs / page.latencyCount);
}

function trackStarted(details) {
  if (details.tabId < 0 || !details.url) {
    return;
  }
  let page;
  if (details.type === 'main_frame') {
    page = pageFor(details.tabId, details.url, { reset: true });
  } else {
    page = pageFor(details.tabId, details.documentUrl || details.initiator || details.url);
  }
  page.requestCount += 1;
  const uploadBytes = estimateUploadBytes(details.requestBody);
  page.uploadBytes += uploadBytes;
  requestsById.set(details.requestId, { tabId: details.tabId, uploadBytes, startedAt: Date.now(), responseSeen: false });
  getState()
    .then((state) => {
      const route = routePreviewForUrl(state, details.url);
      if (route.mode === 'proxy') {
        page.proxiedRequestCount += 1;
      } else {
        page.directRequestCount += 1;
      }
    })
    .catch(() => {
      page.directRequestCount += 1;
    });
}

function trackHeaders(details) {
  const tracked = requestsById.get(details.requestId);
  if (!tracked) {
    return;
  }
  const page = pagesByTab.get(tracked.tabId);
  if (!page) {
    return;
  }
  recordLatency(page, tracked);
  trackStatusHeaders(page, details);
  trackCacheHeaders(page, details.responseHeaders);
  page.downloadBytes += contentLength(details.responseHeaders);
}

function trackFinished(details) {
  const tracked = requestsById.get(details.requestId);
  if (tracked) {
    const page = pagesByTab.get(tracked.tabId);
    recordLatency(page, tracked);
  }
  requestsById.delete(details.requestId);
}

function trackFailed(details) {
  const tracked = requestsById.get(details.requestId);
  if (tracked) {
    const page = pagesByTab.get(tracked.tabId);
    if (page) {
      page.failureCount += 1;
      page.lastErrorCode = details.error || 'request_failed';
      page.lastErrorMessage = details.error || 'request_failed';
      page.errorCodeCount[page.lastErrorCode] = Number(page.errorCodeCount[page.lastErrorCode] || 0) + 1;
    }
  }
  requestsById.delete(details.requestId);
}

function tabMetricsSnapshot(sender, url) {
  const tabId = sender && sender.tab ? sender.tab.id : -1;
  if (tabId < 0) {
    return null;
  }
  const page = pageFor(tabId, url || (sender.tab && sender.tab.url) || '');
  return structuredClone(page);
}

function registerPageMetrics() {
  if (!chrome.webRequest) {
    return;
  }
  chrome.webRequest.onBeforeRequest.addListener(trackStarted, { urls: ['<all_urls>'] }, ['requestBody']);
  chrome.webRequest.onHeadersReceived.addListener(trackHeaders, { urls: ['<all_urls>'] }, ['responseHeaders']);
  chrome.webRequest.onCompleted.addListener(trackFinished, { urls: ['<all_urls>'] });
  chrome.webRequest.onErrorOccurred.addListener(trackFailed, { urls: ['<all_urls>'] });
  if (chrome.tabs && chrome.tabs.onRemoved) {
    chrome.tabs.onRemoved.addListener((tabId) => pagesByTab.delete(tabId));
  }
}

const STATUS_BUBBLE_LABELS = [
  'account',
  'activeAccessPath',
  'policyRevision',
  'syncedAt',
  'tenant',
  'statusBubbleTitle',
  'statusBubbleUpload',
  'statusBubbleDownload',
  'statusBubbleLatency',
  'statusBubbleRequests',
  'statusBubbleRoute',
  'statusBubbleOpenedAt',
  'statusBubbleRequestMixShort',
  'statusBubbleCorrelation',
  'statusBubbleCorrelated',
  'statusBubbleNotCorrelated',
  'statusBubbleCache',
  'statusBubbleCacheStoredAt',
  'statusBubbleCacheResponses',
  'statusBubbleStatusCode',
  'statusBubbleHttpErrors',
  'statusBubbleErrorCodes',
  'statusBubbleLastError',
  'statusBubbleTopology',
  'statusBubbleUserMachine',
  'statusBubbleWebsite',
  'statusBubbleRoundTrip',
  'statusBubbleTransport',
  'statusBubbleDirectQUIC',
  'statusBubbleRelay',
  'statusBubbleFallback',
  'statusBubbleRefresh',
  'statusBubbleCopy',
  'statusBubbleCopied',
  'statusBubbleUnknown'
];
const PATH_HEALTH_TTL_MS = 60000;
const pathHealthCache = new Map();
const STATUS_BUBBLE_FALLBACK_LABELS = {
  account: 'Account',
  activeAccessPath: 'Active access path',
  policyRevision: 'Policy',
  syncedAt: 'Synced',
  tenant: 'Tenant',
  statusBubbleTitle: 'Proxy status',
  statusBubbleUpload: 'Upload',
  statusBubbleDownload: 'Download',
  statusBubbleLatency: 'Latency',
  statusBubbleRequests: 'Requests',
  statusBubbleRoute: 'Route',
  statusBubbleOpenedAt: 'Opened',
  statusBubbleRequestMixShort: 'P / D / F',
  statusBubbleCorrelation: 'IO source',
  statusBubbleCorrelated: 'Node correlated',
  statusBubbleNotCorrelated: 'Not correlated',
  statusBubbleCache: 'Cache',
  statusBubbleCacheStoredAt: 'Cache stored',
  statusBubbleCacheResponses: 'Cache responses',
  statusBubbleStatusCode: 'HTTP status',
  statusBubbleHttpErrors: 'HTTP errors',
  statusBubbleErrorCodes: 'Error codes',
  statusBubbleLastError: 'Last error',
  statusBubbleTopology: 'Path',
  statusBubbleUserMachine: 'User machine',
  statusBubbleWebsite: 'Website',
  statusBubbleRoundTrip: 'RTT',
  statusBubbleTransport: 'Transport',
  statusBubbleDirectQUIC: 'Direct QUIC',
  statusBubbleRelay: 'Relay',
  statusBubbleFallback: 'Fallback',
  statusBubbleRefresh: 'Refresh',
  statusBubbleCopy: 'Copy diagnostics',
  statusBubbleCopied: 'Copied',
  statusBubbleUnknown: 'Unknown'
};

function tenantFrom(state) {
  const membership = state.session.tenantMemberships.find((item) => item.tenantId === state.session.activeTenantId);
  return {
    id: state.session.activeTenantId || '',
    name: (membership && membership.tenantName) || state.session.activeTenantId || ''
  };
}

function labels() {
  return Object.fromEntries(STATUS_BUBBLE_LABELS.map((key) => {
    const message = chrome.i18n.getMessage(key);
    return [key, message || STATUS_BUBBLE_FALLBACK_LABELS[key] || key];
  }));
}

function accessPathForRoute(state, route) {
  return route && route.accessPathId ? accessPathById(state, route.accessPathId) : null;
}

function accessPathProbeBaseUrl(accessPath) {
  if (!accessPath || !accessPath.listenHost || !accessPath.listenPort) {
    throw new Error('status_bubble_access_path_proxy_required');
  }
  return `http://${accessPath.listenHost}:${accessPath.listenPort}`;
}

function entryProbeUrl(accessPath) {
  return `${accessPathProbeBaseUrl(accessPath)}/healthz`;
}

function measureEntryLatency(accessPath) {
  const startedAt = Date.now();
  return fetch(entryProbeUrl(accessPath), { cache: 'no-store' })
    .then((response) => {
      if (!response.ok) {
        throw new Error(`status_bubble_entry_probe_failed:${response.status}`);
      }
      return Date.now() - startedAt;
    });
}

function routeTopology(accessPath, route) {
  const topology = Array.isArray(route.topology) && route.topology.length > 0 ? route.topology : (accessPath && Array.isArray(accessPath.topology) ? accessPath.topology : []);
  if (topology.length > 0) {
    return topology;
  }
  if (accessPath && (accessPath.entryNodeId || accessPath.entryNodeName)) {
    return [{ id: accessPath.entryNodeId || accessPath.entryNodeName, name: accessPath.entryNodeName || accessPath.entryNodeId, mode: 'edge' }];
  }
  return [];
}

function pathHealthKey(accessPath, route) {
  const topologyIds = routeTopology(accessPath, route).map((node) => node.id).filter(Boolean).join('>');
  return [
    accessPath.id || '',
    accessPath.listenHost || '',
    accessPath.listenPort || '',
    topologyIds,
    route.protocol || '',
    route.host || '',
    route.port || ''
  ].join('|');
}

function entryNodeId(accessPath, route) {
  const first = routeTopology(accessPath, route)[0];
  if (!first || !first.id) {
    throw new Error('status_bubble_entry_node_required');
  }
  return first.id;
}

function oneProxyTokenHeaders(state) {
  const token = state.session && state.session.proxyToken ? String(state.session.proxyToken) : '';
  return token ? { 'X-One-Proxy-Token': token } : {};
}

function requestNodePathHealth(state, accessPath, route) {
  const endpoint = `${accessPathProbeBaseUrl(accessPath)}/api/control/relay/probe`;
  const remainingHopNodeIds = routeTopology(accessPath, route).map((node) => node.id).filter(Boolean).slice(1);
  return fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...oneProxyTokenHeaders(state) },
    body: JSON.stringify({
      protocol: route.protocol,
      remainingHopNodeIds,
      targetHost: route.host,
      targetPort: route.port
    })
  }).then((response) => response.json()
    .then((body) => {
      if (!response.ok) {
        throw new Error((body && body.message) || `path_health_failed:${response.status}`);
      }
      if (!body || !Array.isArray(body.pathTimings)) {
        throw new Error('path_health_timings_required');
      }
      return body.pathTimings;
    }));
}

function measurePathHealth(state, accessPath, route) {
  if (state.localHelper && state.localHelper.enabled) {
    return Promise.resolve({
      sampleTsMs: Date.now(),
      linkTimings: []
    });
  }
  const key = pathHealthKey(accessPath, route);
  const cached = pathHealthCache.get(key);
  const now = Date.now();
  if (cached && now - cached.sampleTsMs < PATH_HEALTH_TTL_MS) {
    return Promise.resolve(cached);
  }
  return Promise.all([
    measureEntryLatency(accessPath),
    requestNodePathHealth(state, accessPath, route)
  ]).then(([entryLatencyMs, nodeTimings]) => {
    const result = {
      sampleTsMs: Date.now(),
      linkTimings: [
        {
          fromNodeId: 'user',
          toNodeId: entryNodeId(accessPath, route),
          roundTripMs: entryLatencyMs,
          sampleTsMs: Date.now(),
          count: 1
        },
        ...nodeTimings
      ]
    };
    pathHealthCache.set(key, result);
    return result;
  });
}

function colorFor(status, latencyMs, routeMode) {
  if (routeMode !== 'proxy') {
    return 'gray';
  }
  if (status === 'error') {
    return 'red';
  }
  if (status === 'slow' || Number(latencyMs || 0) > 1000) {
    return 'yellow';
  }
  if (status === 'ok') {
    return 'green';
  }
  return 'gray';
}

function routePayload(route) {
  const rule = route.rule || null;
  if (!rule) {
    return {
      id: '',
      source: route.source || '',
      matchType: '',
      matchValue: '',
      chainId: ''
    };
  }
  return {
    id: rule.id || '',
    source: route.source || '',
    matchType: rule.matchType || '',
    matchValue: rule.matchValue || '',
    chainId: rule.chainId || ''
  };
}

function mergePageSnapshot(route, metrics, remoteStatus) {
  const requestCount = Number((remoteStatus && remoteStatus.requestCount) || (metrics && metrics.requestCount) || 0);
  const rawProxiedRequestCount = Number((metrics && metrics.proxiedRequestCount) || 0);
  const rawDirectRequestCount = Number((metrics && metrics.directRequestCount) || 0);
  const proxiedRequestCount = route.mode === 'proxy'
    ? Math.max(rawProxiedRequestCount, requestCount - rawDirectRequestCount)
    : rawProxiedRequestCount;
  const directRequestCount = route.mode === 'proxy'
    ? rawDirectRequestCount
    : Math.max(rawDirectRequestCount, requestCount - rawProxiedRequestCount);
  return {
    host: route.host || '',
    openedAt: (metrics && metrics.openedAt) || new Date().toISOString(),
    requestCount,
    proxiedRequestCount,
    directRequestCount,
    failureCount: Number((remoteStatus && remoteStatus.failureCount) || (metrics && metrics.failureCount) || 0),
    statusCode: Number((remoteStatus && remoteStatus.statusCode) || (metrics && metrics.statusCode) || 0),
    httpErrorCount: Number((metrics && metrics.httpErrorCount) || 0),
    errorCodeCount: (metrics && metrics.errorCodeCount) || {},
    cacheStatus: String((remoteStatus && remoteStatus.cacheStatus) || (metrics && metrics.cacheStatus) || ''),
    cacheStoredAt: String((remoteStatus && remoteStatus.cacheStoredAt) || (metrics && metrics.cacheStoredAt) || ''),
    cacheAgeSeconds: Number((metrics && metrics.cacheAgeSeconds) || 0),
    cacheResponseCount: Number((metrics && metrics.cacheResponseCount) || 0)
  };
}

function pathPayload(state, accessPath, route, status, pageHost) {
  const topology = routeTopology(accessPath, route);
  const transport = state.localHelper && state.localHelper.enabled ? 'direct_quic' : (status && status.path && status.path.transport ? String(status.path.transport) : 'relay');
  const nodes = [
    { id: 'user', name: 'User machine', kind: 'user', transport: 'client' },
    ...topology.map((node, index) => ({
      id: node.id,
      name: node.name || node.id,
      kind: 'node',
      mode: node.mode || '',
      transport: index === 0 ? transport : 'relay_ws_parent'
    })),
    { id: pageHost || 'website', name: pageHost || 'Website', kind: 'web', transport: 'target' }
  ];
  return {
    mode: transport,
    transport,
    fallbackReason: '',
    nodes
  };
}

function requestRemoteStatus(state, route, routeInfo) {
  if (route.mode !== 'proxy') {
    return Promise.resolve(null);
  }
  if (!state.session.accessToken || !state.session.activeTenantId) {
    throw new Error('status_bubble_session_required');
  }
  return getExtensionPageStatus(state, {
    host: route.host,
    routeId: routeInfo.id,
    chainId: routeInfo.chainId
  }).then((status) => {
    if (isUncorrelatedStatus(status) && (routeInfo.id || routeInfo.chainId)) {
      return getExtensionPageStatus(state, { host: route.host });
    }
    return status;
  });
}

function isUncorrelatedStatus(status) {
  return !status || (!status.correlated && String(status.status || '') === 'unknown');
}

function emptyPathHealth() {
  return {
    sampleTsMs: Date.now(),
    linkTimings: []
  };
}

function normalizePageMetrics(metrics) {
  if (!metrics || typeof metrics !== 'object') {
    return null;
  }
  return {
    openedAt: String(metrics.openedAt || ''),
    requestCount: Number(metrics.requestCount || 0),
    responseCount: Number(metrics.responseCount || 0),
    proxiedRequestCount: Number(metrics.proxiedRequestCount || 0),
    directRequestCount: Number(metrics.directRequestCount || 0),
    failureCount: Number(metrics.failureCount || 0),
    uploadBytes: Number(metrics.uploadBytes || 0),
    downloadBytes: Number(metrics.downloadBytes || 0),
    latencyMs: Number(metrics.latencyMs || 0),
    statusCode: Number(metrics.statusCode || 0),
    httpErrorCount: Number(metrics.httpErrorCount || 0),
    errorCodeCount: metrics.errorCodeCount && typeof metrics.errorCodeCount === 'object' ? metrics.errorCodeCount : {},
    cacheStatus: String(metrics.cacheStatus || ''),
    cacheStoredAt: String(metrics.cacheStoredAt || ''),
    cacheAgeSeconds: Number(metrics.cacheAgeSeconds || 0),
    cacheResponseCount: Number(metrics.cacheResponseCount || 0),
    lastErrorCode: String(metrics.lastErrorCode || ''),
    lastErrorMessage: String(metrics.lastErrorMessage || '')
  };
}

function mergeLocalMetrics(primary, fallback) {
  if (!primary) {
    return fallback;
  }
  if (!fallback) {
    return primary;
  }
  return {
    ...primary,
    openedAt: primary.openedAt || fallback.openedAt,
    requestCount: maxNumber(primary.requestCount, fallback.requestCount),
    responseCount: maxNumber(primary.responseCount, fallback.responseCount),
    proxiedRequestCount: maxNumber(primary.proxiedRequestCount, fallback.proxiedRequestCount),
    directRequestCount: maxNumber(primary.directRequestCount, fallback.directRequestCount),
    failureCount: maxNumber(primary.failureCount, fallback.failureCount),
    uploadBytes: maxNumber(primary.uploadBytes, fallback.uploadBytes),
    downloadBytes: maxNumber(primary.downloadBytes, fallback.downloadBytes),
    latencyMs: firstPositive(primary.latencyMs, fallback.latencyMs),
    statusCode: firstPositive(primary.statusCode, fallback.statusCode),
    httpErrorCount: maxNumber(primary.httpErrorCount, fallback.httpErrorCount),
    errorCodeCount: mergeErrorCodeCount(primary.errorCodeCount, fallback.errorCodeCount),
    cacheStatus: primary.cacheStatus || fallback.cacheStatus,
    cacheStoredAt: primary.cacheStoredAt || fallback.cacheStoredAt,
    cacheAgeSeconds: maxNumber(primary.cacheAgeSeconds, fallback.cacheAgeSeconds),
    cacheResponseCount: maxNumber(primary.cacheResponseCount, fallback.cacheResponseCount),
    lastErrorCode: primary.lastErrorCode || fallback.lastErrorCode,
    lastErrorMessage: primary.lastErrorMessage || fallback.lastErrorMessage
  };
}

function maxNumber(primary, fallback) {
  return Math.max(Number(primary || 0), Number(fallback || 0));
}

function firstPositive(primary, fallback) {
  const first = Number(primary || 0);
  return first > 0 ? first : Number(fallback || 0);
}

function mergeErrorCodeCount(primary, fallback) {
  const result = { ...(fallback || {}) };
  Object.entries(primary || {}).forEach(([key, value]) => {
    if (key) {
      result[key] = Math.max(Number(result[key] || 0), Number(value || 0));
    }
  });
  return result;
}

function pathLatencyMs(pathHealth) {
  const first = pathHealth && Array.isArray(pathHealth.linkTimings) ? pathHealth.linkTimings[0] : null;
  return first ? Number(first.roundTripMs || first.rttMs || 0) : 0;
}

function statusFrom(remoteStatus, metrics, routeMode, pathHealth) {
  const remote = String((remoteStatus && remoteStatus.status) || '').trim();
  if (remote && remote !== 'unknown') {
    return remote;
  }
  if (metrics && Number(metrics.failureCount || 0) > 0) {
    return 'error';
  }
  if (metrics && Number(metrics.responseCount || 0) > 0 && (Number(metrics.proxiedRequestCount || 0) > 0 || routeMode === 'proxy')) {
    return 'ok';
  }
  if (routeMode === 'proxy' && pathLatencyMs(pathHealth) > 0) {
    return 'ok';
  }
  return remote || 'unknown';
}

function statusWithLatency(status, latencyMs, routeMode) {
  if (status !== 'unknown' || routeMode !== 'proxy') {
    return status;
  }
  return Number(latencyMs || 0) > 0 ? 'ok' : status;
}

function effectiveLatencyMs(remoteStatus, metrics, pathHealth) {
  const remoteLatencyMs = Number((remoteStatus && remoteStatus.latencyMs) || 0);
  if (remoteLatencyMs > 0) {
    return remoteLatencyMs;
  }
  const localLatencyMs = Number((metrics && metrics.latencyMs) || 0);
  if (localLatencyMs > 0) {
    return localLatencyMs;
  }
  return pathLatencyMs(pathHealth);
}

function normalizeNodeTimings(items) {
  return (Array.isArray(items) ? items : []).map((item) => ({
    nodeId: item.nodeId || '',
    processAvgMs: Number(item.processAvgMs || 0),
    responseProcessAvgMs: Number(item.responseProcessAvgMs || 0),
    sampleTsMs: Number(item.sampleTsMs || 0),
    count: Number(item.count || 1)
  })).filter((item) => item.nodeId);
}

function normalizeLinkTimings(items) {
  return (Array.isArray(items) ? items : []).map((item) => ({
    fromNodeId: item.fromNodeId || '',
    toNodeId: item.toNodeId || '',
    roundTripMs: Number(item.roundTripMs || item.rttMs || 0),
    sampleTsMs: Number(item.sampleTsMs || 0),
    count: Number(item.count || 1)
  })).filter((item) => item.fromNodeId && item.toNodeId);
}

function shouldDisplay(state, route) {
  const controlPlaneHost = urlHostname(state.controlPlaneUrl);
  if (!route.host || route.host === controlPlaneHost) {
    return false;
  }
  return route.mode === 'proxy';
}

function getStatusBubblePageStatus(message, sender) {
  const url = message.url || (sender && sender.tab && sender.tab.url) || '';
  return (message.refresh ? syncRemoteConfig() : Promise.resolve(null))
    .then(() => getState())
    .then((state) => {
      const route = routePreviewForUrl(state, url);
      const accessPath = accessPathForRoute(state, route);
      const routeInfo = routePayload(route);
      const metrics = mergeLocalMetrics(tabMetricsSnapshot(sender, url), normalizePageMetrics(message.pageMetrics));
      if (!shouldDisplay(state, route)) {
        return {
          display: false
        };
      }
      return Promise.all([
        Promise.resolve().then(() => requestRemoteStatus(state, route, routeInfo)).catch((error) => ({
          status: 'unknown',
          lastErrorCode: 'page_status_failed',
          lastErrorMessage: error.message || 'page_status_failed'
        })),
        Promise.resolve().then(() => measurePathHealth(state, accessPath, route)).catch(() => emptyPathHealth())
      ]).then(([remoteStatus, pathHealth]) => {
        const status = remoteStatus || { status: 'unknown' };
        const page = mergePageSnapshot(route, metrics, status);
        const uploadBytes = Number(status.uploadBytes || (metrics && metrics.uploadBytes) || 0);
        const downloadBytes = Number(status.downloadBytes || (metrics && metrics.downloadBytes) || 0);
        const latencyMs = effectiveLatencyMs(status, metrics, pathHealth);
        const actualNodeTimings = normalizeNodeTimings(status.nodeTimings);
        const actualLinkTimings = normalizeLinkTimings(status.linkTimings);
        const linkTimings = actualLinkTimings.length > 0 ? actualLinkTimings : normalizeLinkTimings(pathHealth.linkTimings);
        const lastErrorCode = status.lastErrorCode || (metrics && metrics.lastErrorCode) || '';
        const lastErrorMessage = status.lastErrorMessage || (metrics && metrics.lastErrorMessage) || '';
        const displayStatus = statusWithLatency(statusFrom(status, metrics, route.mode, pathHealth), latencyMs, route.mode);
        const cache = {
          status: page.cacheStatus || '',
          storedAt: page.cacheStoredAt || '',
          ageSeconds: page.cacheAgeSeconds || 0,
          responseCount: page.cacheResponseCount || 0
        };
        return {
          status: displayStatus,
          color: cache.status ? 'yellow' : colorFor(displayStatus, latencyMs, route.mode),
          display: shouldDisplay(state, route),
          account: state.session.account || '',
          tenant: tenantFrom(state),
          accessPath: accessPath ? { id: accessPath.id || '', name: accessPath.name || accessPath.entryNodeName || accessPath.id || '' } : { id: '', name: '' },
          route: routeInfo,
          page,
          io: {
            uploadBytes,
            downloadBytes,
            correlated: Boolean(status.correlated || (metrics && metrics.responseCount > 0))
          },
          cache,
          latencyMs,
          path: pathPayload(state, accessPath, route, status, page.host),
          labels: labels(),
          nodeTimings: actualNodeTimings,
          linkTimings,
          policyRevision: status.policyRevision || state.remote.policyRevision || '',
          configFetchedAt: state.remote.fetchedAt || '',
          lastError: {
            code: lastErrorCode,
            message: lastErrorMessage
          }
        };
      });
    });
}

const INTERNAL_MESSAGE_TYPES = new Set([
  'get-state',
  'get-diagnostic-logs',
  'clear-diagnostic-logs',
  'record-diagnostic-event',
  'set-enabled',
  'set-theme-mode',
  'set-control-plane-url',
  'login',
  'test-connection',
  'logout',
  'sync-remote-config',
  'select-tenant',
  'test-url-route',
  'select-access-path',
  'set-local-overrides',
  'set-local-helper',
  'add-current-host-to-direct',
  'add-current-host-to-proxy',
  'remove-current-host-override'
]);
const CONTENT_MESSAGE_TYPES = new Set(['status-bubble-page-status']);

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
    return computedStateView(state, currentTab);
  });
}

function sessionView(session) {
  return {
    account: session.account || '',
    expiresAt: session.expiresAt || '',
    mustRotatePassword: Boolean(session.mustRotatePassword),
    tenantMemberships: session.tenantMemberships || [],
    activeTenantId: session.activeTenantId || '',
    authenticated: Boolean(session.accessToken),
    proxyTokenAvailable: Boolean(session.proxyToken),
    proxyTokenExpiresAt: session.proxyTokenExpiresAt || ''
  };
}

function stateView(state) {
  return {
    ...state,
    session: sessionView(state.session)
  };
}

function computedStateView(state, currentTab) {
  return {
    state: stateView(state),
    session: sessionView(state.session),
    remote: state.remote,
    activeAccessPath: activeAccessPathFrom(state),
    currentTab,
    currentRoute: routePreviewForUrl(state, currentTab && currentTab.url),
    monitorRoute: routePreviewForUrl(state, state.monitor.targetUrl)
  };
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

function handleMessage(message, sender) {
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
      return computedAfter(() => setPartialState((state) => ({ ...state, controlPlaneUrl: normalizeControlPlaneUrl(message.controlPlaneUrl) })));
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
    case 'status-bubble-page-status':
      return getStatusBubblePageStatus(message, sender);
    case 'select-access-path':
      return computedAfter(() => setPartialState((state) => ({
        ...state,
        selection: {
          ...state.selection,
          activeAccessPathId: message.accessPathId || ''
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
    case 'set-local-helper':
      return computedAfter(() => setPartialState((state) => ({
        ...state,
        localHelper: {
          enabled: Boolean(message.enabled),
          scheme: message.scheme === 'PROXY' ? 'PROXY' : 'SOCKS5',
          host: String(message.host || '127.0.0.1').trim(),
          port: Number(message.port || 1080)
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

function extensionPageSender(sender) {
  const senderUrl = String((sender && sender.url) || '');
  return senderUrl.startsWith(chrome.runtime.getURL(''));
}

function contentSender(sender) {
  return Boolean(sender && sender.tab && !extensionPageSender(sender));
}

function allowedMessage(message, sender) {
  if (sender && sender.id && sender.id !== chrome.runtime.id) {
    return false;
  }
  if (contentSender(sender)) {
    return CONTENT_MESSAGE_TYPES.has(message.type);
  }
  return INTERNAL_MESSAGE_TYPES.has(message.type);
}

function registerMessageHandler() {
  chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (!message || !message.type) {
      sendResponse(null);
      return false;
    }
    if (!allowedMessage(message, sender)) {
      sendResponse({ error: 'message_not_allowed' });
      return false;
    }
    handleMessage(message, sender).then(sendResponse).catch((error) => {
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
  targets: new Set(),
  token: ''
};

function proxyTargetKey(host, port) {
  return `${String(host || '').toLowerCase()}:${Number(port || 0)}`;
}

function proxyAuthTargetsFrom(state) {
  return new Set((state.remote.accessPaths || [])
    .filter(isUsableAccessPath)
    .map((path) => proxyTargetKey(path.listenHost, path.listenPort)));
}

function updateProxyAuthCache(state) {
  proxyAuthCache = {
    targets: proxyAuthTargetsFrom(state),
    token: state.session && state.session.proxyToken ? String(state.session.proxyToken) : ''
  };
}

function matchesProxyChallenge(details) {
  if (!details || !details.isProxy || !proxyAuthCache.token) {
    return false;
  }
  const challenger = details.challenger || {};
  return proxyAuthCache.targets.has(proxyTargetKey(challenger.host, challenger.port));
}

function registerProxyAuthHandler() {
  if (!chrome.webRequest || !chrome.webRequest.onAuthRequired) {
    return;
  }
  chrome.webRequest.onAuthRequired.addListener(
    (details, callback) => {
      if (!matchesProxyChallenge(details)) {
        callback({});
        return;
      }
      const challenger = details.challenger || {};
      appendLog('info', 'proxy_auth_supplied', {
        host: String(challenger.host || ''),
        port: Number(challenger.port || 0)
      }).catch(() => {});
      callback({
        authCredentials: {
          username: 'token',
          password: proxyAuthCache.token
        }
      });
    },
    { urls: ['<all_urls>'] },
    ['asyncBlocking']
  );
}

let startupSyncPromise = null;

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

function syncRemoteOnce() {
  if (!startupSyncPromise) {
    startupSyncPromise = getState()
      .then((state) => syncRemoteConfigIfReady(state))
      .then(() => getState());
  }
  return startupSyncPromise;
}

configureStateEffects((state) => {
  updateProxyAuthCache(state);
  return applyProxy(state).then(() => broadcastState());
});

chrome.runtime.onInstalled.addListener(() => {
  ensureMonitorAlarm();
  syncRemoteOnce()
    .then((state) => persistState(state))
    .then(() => appendLog('info', 'extension_installed'))
    .catch((error) => appendLog('error', 'extension_installed_failed', { message: error.message || 'install_failed' }));
});

chrome.runtime.onStartup.addListener(() => {
  ensureMonitorAlarm();
  syncRemoteOnce()
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
  registerPageMetrics();
  registerProxyAuthHandler();
  registerMessageHandler();
  return syncRemoteOnce().then((state) => updateProxyAuthCache(state));
}

bootstrap().catch((error) => {
  appendLog('error', 'service_worker_bootstrap_failed', { message: error.message || 'bootstrap_failed' }).catch(() => {});
});
