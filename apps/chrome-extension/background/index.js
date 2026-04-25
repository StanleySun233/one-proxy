const STORAGE_KEY = 'oneProxyState';

const DEFAULT_STATE = {
  enabled: false,
  internalProxyEnabled: false,
  themeMode: 'vivid',
  activeGroupId: 'group-default',
  groups: [
    {
      id: 'group-default',
      name: 'Primary',
      routingMode: 'smart',
      profileIds: ['profile-default'],
      activeProfileId: 'profile-default',
      bypassHosts: ['localhost', '*.local', '*.lan'],
      proxyHosts: [],
      internalDomains: ['*.corp', '*.internal'],
      internalCidrs: ['10.0.0.0/8', '172.16.0.0/12', '192.168.0.0/16']
    }
  ],
  profiles: [
    {
      id: 'profile-default',
      name: 'Local Proxy',
      scheme: 'PROXY',
      host: '127.0.0.1',
      port: 2388
    }
  ],
  quickRules: {
    bypassHosts: [],
    proxyHosts: [],
    internalDomains: []
  }
};

let stateCache = null;

async function getState() {
  if (stateCache) {
    return structuredClone(stateCache);
  }
  const stored = await chrome.storage.local.get(STORAGE_KEY);
  stateCache = mergeState(stored[STORAGE_KEY] || {});
  return structuredClone(stateCache);
}

function mergeState(raw) {
  const state = {
    ...DEFAULT_STATE,
    ...raw,
    quickRules: {
      ...DEFAULT_STATE.quickRules,
      ...(raw.quickRules || {})
    }
  };
  state.groups = Array.isArray(raw.groups) && raw.groups.length ? raw.groups.map(normalizeGroup) : DEFAULT_STATE.groups.map(normalizeGroup);
  state.profiles = Array.isArray(raw.profiles) && raw.profiles.length ? raw.profiles : DEFAULT_STATE.profiles;
  if (!state.groups.find((group) => group.id === state.activeGroupId)) {
    state.activeGroupId = (state.groups[0] && state.groups[0].id) || DEFAULT_STATE.activeGroupId;
  }
  return state;
}

function normalizeGroup(group) {
  return {
    bypassHosts: [],
    proxyHosts: [],
    internalDomains: [],
    internalCidrs: [],
    ...group
  };
}

async function saveState(nextState) {
  stateCache = mergeState(nextState);
  await chrome.storage.local.set({ [STORAGE_KEY]: stateCache });
  await applyProxy(stateCache);
  await broadcastState();
  return structuredClone(stateCache);
}

function getActiveGroup(state) {
  return state.groups.find((group) => group.id === state.activeGroupId) || state.groups[0];
}

function getActiveProfile(state) {
  const group = getActiveGroup(state);
  if (!group) {
    return null;
  }
  return state.profiles.find((profile) => profile.id === group.activeProfileId) || state.profiles.find((profile) => group.profileIds.includes(profile.id)) || null;
}

function escapePacString(value) {
  return String(value).replaceAll('\\', '\\\\').replaceAll("'", "\\'");
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

function buildPacScript(state) {
  const group = getActiveGroup(state);
  const profile = getActiveProfile(state);
  const proxyTarget = profile ? `${profile.scheme} ${profile.host}:${profile.port}` : 'DIRECT';
  const bypassHosts = [...new Set([...(group ? group.bypassHosts || [] : []), ...(state.quickRules.bypassHosts || [])])];
  const proxyHosts = [...new Set([...(group ? group.proxyHosts || [] : []), ...(state.quickRules.proxyHosts || [])])];
  const internalDomains = [...new Set([...(group ? group.internalDomains || [] : []), ...(state.quickRules.internalDomains || [])])];
  const internalCidrs = [...new Set(group ? group.internalCidrs || [] : [])]
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => {
      const [network, prefix] = item.split('/');
      const mask = cidrToMask(prefix);
      return mask ? { network, mask } : null;
    })
    .filter(Boolean);
  const mode = (group && group.routingMode) || 'smart';

  return `
const enabled = ${state.enabled ? 'true' : 'false'};
const internalProxyEnabled = ${state.internalProxyEnabled ? 'true' : 'false'};
const proxyTarget = '${escapePacString(proxyTarget)}';
const mode = '${escapePacString(mode)}';
const bypassHosts = ${JSON.stringify(bypassHosts)};
const proxyHosts = ${JSON.stringify(proxyHosts)};
const internalDomains = ${JSON.stringify(internalDomains)};
const internalCidrs = ${JSON.stringify(internalCidrs)};

function hostMatches(patterns, host) {
  for (const pattern of patterns) {
    if (shExpMatch(host, pattern)) {
      return true;
    }
  }
  return false;
}

function isReservedIp(ip) {
  return isInNet(ip, '127.0.0.0', '255.0.0.0') ||
    isInNet(ip, '10.0.0.0', '255.0.0.0') ||
    isInNet(ip, '172.16.0.0', '255.240.0.0') ||
    isInNet(ip, '192.168.0.0', '255.255.0.0') ||
    isInNet(ip, '169.254.0.0', '255.255.0.0') ||
    isInNet(ip, '100.64.0.0', '255.192.0.0');
}

function isInternalHost(host) {
  if (isPlainHostName(host) || dnsDomainIs(host, '.local')) {
    return true;
  }
  if (hostMatches(internalDomains, host)) {
    return true;
  }
  const resolved = dnsResolve(host);
  if (!resolved) {
    return false;
  }
  if (isReservedIp(resolved)) {
    return true;
  }
  for (const item of internalCidrs) {
    if (isInNet(resolved, item.network, item.mask)) {
      return true;
    }
  }
  return false;
}

function FindProxyForURL(url, host) {
  if (!enabled) {
    return 'DIRECT';
  }
  if (hostMatches(bypassHosts, host)) {
    return 'DIRECT';
  }
  if (!hostMatches(proxyHosts, host)) {
    return 'DIRECT';
  }
  const internal = isInternalHost(host);
  if (mode === 'all') {
    return internal && !internalProxyEnabled ? 'DIRECT' : proxyTarget;
  }
  if (mode === 'internal') {
    return internal && internalProxyEnabled ? proxyTarget : 'DIRECT';
  }
  if (mode === 'external') {
    return internal ? 'DIRECT' : proxyTarget;
  }
  if (internal) {
    return internalProxyEnabled ? proxyTarget : 'DIRECT';
  }
  return proxyTarget;
}
`;
}

async function applyProxy(state) {
  const pacScript = buildPacScript(state);
  await chrome.proxy.settings.set({
    value: {
      mode: 'pac_script',
      pacScript: {
        data: pacScript
      }
    },
    scope: 'regular'
  });
}

async function broadcastState() {
  const payload = await getComputedState();
  try {
    await chrome.runtime.sendMessage({ type: 'state-updated', payload });
  } catch (_error) {
  }
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
  const activeGroup = getActiveGroup(state);
  const activeProfile = getActiveProfile(state);
  const currentTab = await getCurrentTabInfo();
  return {
    state,
    activeGroup,
    activeProfile,
    currentTab
  };
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
  const bucket = Array.isArray(state.quickRules[kind]) ? state.quickRules[kind] : [];
  if (!bucket.includes(clean)) {
    bucket.push(clean);
  }
  state.quickRules[kind] = bucket;
  await saveState(state);
  return getComputedState();
}

chrome.runtime.onInstalled.addListener(async () => {
  await saveState(await getState());
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
    if (message && message.type === 'get-state') {
      sendResponse(await getComputedState());
      return;
    }
    if (message && message.type === 'set-state') {
      sendResponse(await saveState(message.payload));
      return;
    }
    if (message && message.type === 'add-current-host-to-bypass') {
      const info = await getCurrentTabInfo();
      sendResponse(await addHostToRule('bypassHosts', (info && info.host) || ''));
      return;
    }
    if (message && message.type === 'add-current-host-to-proxy') {
      const info = await getCurrentTabInfo();
      sendResponse(await addHostToRule('proxyHosts', (info && info.host) || ''));
      return;
    }
    if (message && message.type === 'add-current-host-to-internal') {
      const info = await getCurrentTabInfo();
      sendResponse(await addHostToRule('internalDomains', (info && info.host) || ''));
      return;
    }
    sendResponse(null);
  })();
  return true;
});
