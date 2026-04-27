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
  }
};

let stateCache = null;

function uniqueStrings(items) {
  return [...new Set((items || []).map((item) => String(item || '').trim()).filter(Boolean))];
}

function normalizeGroup(group) {
  return {
    id: '',
    name: '',
    entryNodeId: '',
    entryNodeName: '',
    proxyScheme: 'PROXY',
    proxyHost: '',
    proxyPort: 0,
    proxyHosts: [],
    proxyCidrs: [],
    directHosts: [],
    directCidrs: [],
    ...group,
    proxyHosts: uniqueStrings(group.proxyHosts),
    proxyCidrs: uniqueStrings(group.proxyCidrs),
    directHosts: uniqueStrings(group.directHosts),
    directCidrs: uniqueStrings(group.directCidrs)
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
  return 'DIRECT';
}
`;
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
  return {
    state,
    session: state.session,
    remote: state.remote,
    activeGroup: activeGroupFrom(state),
    currentTab: await getCurrentTabInfo()
  };
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
    throw new Error((payload && payload.message) || 'request_failed');
  }
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
  return syncRemoteConfig(nextState);
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
  return getComputedState();
}

async function setPartialState(mutator) {
  const current = await getState();
  const next = await mutator(structuredClone(current));
  await persistState(next);
  return getComputedState();
}

chrome.runtime.onInstalled.addListener(async () => {
  await persistState(await getState());
});

chrome.runtime.onStartup.addListener(async () => {
  await applyProxy(await getState());
});

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
      case 'logout':
        sendResponse(await logout());
        return;
      case 'sync-remote-config':
        sendResponse(await syncRemoteConfig());
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
      default:
        sendResponse(null);
    }
  })().catch((error) => {
    sendResponse({ error: error.message || 'unexpected_error' });
  });
  return true;
});
