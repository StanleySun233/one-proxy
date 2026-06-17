const STORAGE_KEY = 'oneProxyState';
const SESSION_STORAGE_KEY = 'oneProxySession';

export const DEFAULT_ROUTE_EVALUATION = {
  defaultClientMode: 'direct',
  defaultNodeMode: 'deny',
  ruleOrder: 'priority_asc_then_id_asc',
  noMatchNodeDenyReason: 'route_not_found',
  supportedMatchTypes: ['domain', 'domain_suffix', 'ip', 'ip_cidr', 'protocol', 'default'],
  supportedActions: ['chain', 'direct', 'deny']
};

export const DEFAULT_STATE = {
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

export function configureStateEffects(effects) {
  persistEffects = typeof effects === 'function' ? effects : () => Promise.resolve();
}

export function uniqueStrings(items) {
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

export function mergeState(raw) {
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

export function getState() {
  if (stateCache) {
    return Promise.resolve(structuredClone(stateCache));
  }
  return Promise.all([
    chrome.storage.local.get(STORAGE_KEY),
    chrome.storage.session.get(SESSION_STORAGE_KEY)
  ]).then(([stored, sessionStored]) => {
    const durableState = stateWithoutSessionSecrets(mergeState(stored[STORAGE_KEY] || {}));
    stateCache = mergeState(mergeSessionSecrets(durableState, sessionStored[SESSION_STORAGE_KEY] || {}));
    return structuredClone(stateCache);
  });
}

export function activeAccessPathFrom(state) {
  const path = state.remote.accessPaths.find((item) => item.id === state.selection.activeAccessPathId) || state.remote.accessPaths[0] || null;
  return activePathView(path, state);
}

export function accessPathById(state, accessPathId) {
  const path = state.remote.accessPaths.find((item) => item.id === accessPathId) || null;
  return activePathView(path, state);
}

export function persistState(nextState) {
  stateCache = mergeState(nextState);
  return Promise.all([
    chrome.storage.local.set({ [STORAGE_KEY]: stateWithoutSessionSecrets(stateCache) }),
    chrome.storage.session.set({ [SESSION_STORAGE_KEY]: sessionSecretsFrom(stateCache) })
  ])
    .then(() => persistEffects(stateCache))
    .then(() => structuredClone(stateCache));
}

export function setPartialState(mutator) {
  return getState()
    .then((current) => mutator(structuredClone(current)))
    .then((next) => persistState(next));
}

export function handleStateStorageChange(changes, areaName) {
  if (areaName === 'local' && changes[STORAGE_KEY]) {
    const secrets = stateCache ? sessionSecretsFrom(stateCache) : {};
    stateCache = mergeState(mergeSessionSecrets(stateWithoutSessionSecrets(mergeState(changes[STORAGE_KEY].newValue || {})), secrets));
    return stateCache;
  }
  if (areaName === 'session' && changes[SESSION_STORAGE_KEY]) {
    stateCache = mergeState(mergeSessionSecrets(stateCache || {}, changes[SESSION_STORAGE_KEY].newValue || {}));
    return stateCache;
  }
  return null;
}
