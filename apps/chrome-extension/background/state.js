const STORAGE_KEY = 'oneProxyState';

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
let persistEffects = async () => {};

export function configureStateEffects(effects) {
  persistEffects = typeof effects === 'function' ? effects : async () => {};
}

export function uniqueStrings(items) {
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

export function mergeState(raw) {
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
  state.localOverrides.directHosts = uniqueStrings(state.localOverrides.directHosts);
  state.localOverrides.proxyHosts = uniqueStrings(state.localOverrides.proxyHosts);
  if (!state.remote.groups.find((group) => group.id === state.selection.activeGroupId)) {
    state.selection.activeGroupId = (state.remote.groups[0] && state.remote.groups[0].id) || '';
  }
  return state;
}

export async function getState() {
  if (stateCache) {
    return structuredClone(stateCache);
  }
  const stored = await chrome.storage.local.get(STORAGE_KEY);
  stateCache = mergeState(stored[STORAGE_KEY] || {});
  return structuredClone(stateCache);
}

export function activeGroupFrom(state) {
  return state.remote.groups.find((group) => group.id === state.selection.activeGroupId) || state.remote.groups[0] || null;
}

export async function persistState(nextState) {
  stateCache = mergeState(nextState);
  await chrome.storage.local.set({ [STORAGE_KEY]: stateCache });
  await persistEffects(stateCache);
  return structuredClone(stateCache);
}

export async function setPartialState(mutator) {
  const current = await getState();
  const next = await mutator(structuredClone(current));
  await persistState(next);
}

export function handleStateStorageChange(changes, areaName) {
  if (areaName !== 'local' || !changes[STORAGE_KEY]) {
    return null;
  }
  stateCache = mergeState(changes[STORAGE_KEY].newValue || {});
  return stateCache;
}
