const messages = new Map();

function text(key, substitutions) {
  if (!chrome.i18n || typeof chrome.i18n.getMessage !== 'function') {
    throw new Error('chrome_i18n_unavailable');
  }
  if (substitutions) {
    const message = chrome.i18n.getMessage(key, substitutions);
    if (!message) {
      throw new Error(`missing_i18n_message:${key}`);
    }
    return message;
  }
  if (!messages.has(key)) {
    const message = chrome.i18n.getMessage(key);
    if (!message) {
      throw new Error(`missing_i18n_message:${key}`);
    }
    messages.set(key, message);
  }
  return messages.get(key);
}

function applyLanguage(root = document) {
  document.documentElement.lang = chrome.i18n.getUILanguage().startsWith('zh') ? 'zh-CN' : 'en';
  root.querySelectorAll('[data-lang]').forEach((element) => {
    element.textContent = text(element.dataset.lang);
  });
  root.querySelectorAll('[data-lang-title]').forEach((element) => {
    element.title = text(element.dataset.langTitle);
  });
}

function normalizeThemeMode(mode) {
  return mode === 'dark' ? 'dark' : 'vivid';
}

function applyThemeMode(mode, root = document) {
  const normalized = normalizeThemeMode(mode);
  const documentElement = root.documentElement || document.documentElement;
  documentElement.dataset.theme = normalized;
  root.querySelectorAll('[data-theme-mode]').forEach((button) => {
    button.classList.toggle('is-active', button.dataset.themeMode === normalized);
  });
  return normalized;
}

function bindThemeMode(container, onChange) {
  if (!container) {
    return;
  }
  container.querySelectorAll('[data-theme-mode]').forEach((button) => {
    button.addEventListener('click', () => {
      const mode = applyThemeMode(button.dataset.themeMode);
      Promise.resolve(onChange(mode)).catch(() => {});
    });
  });
}

const masterToggle = document.getElementById('masterToggle');
const masterValue = document.getElementById('masterValue');
const accountName = document.getElementById('accountName');
const syncMeta = document.getElementById('syncMeta');
const statusMessage = document.getElementById('statusMessage');
const groupSelect = document.getElementById('groupSelect');
const currentSite = document.getElementById('currentSite');
const entryNodeName = document.getElementById('entryNodeName');
const entryNodeAddress = document.getElementById('entryNodeAddress');
const policyRevision = document.getElementById('policyRevision');
const ruleCounts = document.getElementById('ruleCounts');
const routeCard = document.getElementById('routeCard');
const currentRouteValue = document.getElementById('currentRouteValue');
const currentRouteSource = document.getElementById('currentRouteSource');
const testUrlInput = document.getElementById('testUrlInput');
const testUrlButton = document.getElementById('testUrlButton');
const monitorMeta = document.getElementById('monitorMeta');
const topologyView = document.getElementById('topologyView');
const probeGrid = document.getElementById('probeGrid');
const addDirect = document.getElementById('addDirect');
const addProxy = document.getElementById('addProxy');
const removeOverride = document.getElementById('removeOverride');
const openOptions = document.getElementById('openOptions');
const syncButton = document.getElementById('syncButton');
const themeMode = document.getElementById('themeMode');

let viewState = null;

function sendMessage(message) {
  return chrome.runtime.sendMessage(message);
}

function setToggleState(element, enabled) {
  element.classList.toggle('is-on', enabled);
  element.classList.toggle('is-off', !enabled);
}

function setStatus(kind, message) {
  statusMessage.className = `status-strip is-${kind}`;
  statusMessage.textContent = message;
}

function countLabel(remote) {
  const routes = remote && Array.isArray(remote.routes) ? remote.routes : [];
  if (routes.length === 0) {
    return text('noRules');
  }
  return text('ruleSummary', [
    String(routes.filter((route) => route.actionType === 'chain').length),
    String(routes.filter((route) => route.actionType === 'direct').length)
  ]);
}

function sourceLabel(route) {
  const source = route && route.source;
  if (source === 'policy' && route.mode === 'proxy') {
    return text('routeSource_remote_proxy');
  }
  if (source === 'policy' && route.mode === 'direct') {
    return text('routeSource_remote_direct');
  }
  if (source === 'policy' && route.mode === 'deny') {
    return route.denyReason || text('routeSource_unknown');
  }
  if (source === 'local_safety_direct') {
    return text('routeSource_local_direct');
  }
  return text(`routeSource_${source || 'unknown'}`);
}

function routeLabel(route) {
  if (route && route.source === 'no_site') {
    return text('noSite');
  }
  if (route && route.mode === 'proxy') {
    return text('routeProxy');
  }
  if (route && route.mode === 'direct') {
    return text('routeDirect');
  }
  if (route && route.mode === 'deny') {
    return route.denyReason || text('routeUnknown');
  }
  return text('routeUnknown');
}

function renderTopology(route) {
  topologyView.innerHTML = '';
  const nodes = route && Array.isArray(route.topology) ? route.topology : [];
  if (nodes.length === 0) {
    const empty = document.createElement('span');
    empty.className = 'topology-empty';
    empty.textContent = route && route.mode === 'direct' ? text('routeDirect') : text('noTopology');
    topologyView.appendChild(empty);
    return;
  }
  nodes.forEach((node, index) => {
    if (index > 0) {
      const link = document.createElement('span');
      link.className = 'topology-link';
      link.textContent = '>';
      topologyView.appendChild(link);
    }
    const item = document.createElement('span');
    item.className = `topology-node is-${node.mode || 'node'}`;
    item.textContent = node.name || node.id || '-';
    topologyView.appendChild(item);
  });
}

function renderProbes(results) {
  probeGrid.innerHTML = '';
  const items = Array.isArray(results) ? results : [];
  if (items.length === 0) {
    const empty = document.createElement('span');
    empty.className = 'probe-empty';
    empty.textContent = text('noProbeData');
    probeGrid.appendChild(empty);
    return;
  }
  for (const item of items) {
    const cell = document.createElement('div');
    cell.className = `probe-cell is-${item.status || 'unknown'}`;
    const protocol = document.createElement('strong');
    protocol.textContent = String(item.protocol || '-').toUpperCase();
    const meta = document.createElement('span');
    meta.textContent = item.status === 'skipped' ? text('probeSkipped') : `${item.latencyMs || 0}ms`;
    cell.append(protocol, meta);
    probeGrid.appendChild(cell);
  }
}

function render(bundle) {
  viewState = bundle;
  const { state, session, remote, activeAccessPath, currentTab, currentRoute, monitorRoute } = bundle;
  applyThemeMode(state.themeMode);
  accountName.textContent = session.account || text('notSignedIn');
  syncMeta.textContent = remote.fetchedAt ? `${text('syncedAt')} ${new Date(remote.fetchedAt).toLocaleTimeString()}` : text('notSynced');
  currentSite.textContent = (currentTab && currentTab.host) || text('noSite');
  masterValue.textContent = state.enabled ? text('on') : text('off');
  setToggleState(masterToggle, state.enabled);
  policyRevision.textContent = remote.policyRevision || '-';
  ruleCounts.textContent = countLabel(remote);
  entryNodeName.textContent = activeAccessPath ? (activeAccessPath.name || activeAccessPath.entryNodeName) : text('noGroup');
  entryNodeAddress.textContent = activeAccessPath ? `${activeAccessPath.proxyScheme} ${activeAccessPath.proxyHost}:${activeAccessPath.proxyPort}` : '-';
  currentRouteValue.textContent = routeLabel(currentRoute);
  currentRouteSource.textContent = sourceLabel(currentRoute);
  routeCard.classList.toggle('is-proxy', Boolean(currentRoute && currentRoute.mode === 'proxy'));
  if (!testUrlInput.value) {
    testUrlInput.value = (state.monitor && state.monitor.targetUrl) || (currentTab && currentTab.url) || '';
  }
  monitorMeta.textContent = state.monitor && state.monitor.lastRunAt ? new Date(state.monitor.lastRunAt).toLocaleTimeString() : text('notTested');
  renderTopology(state.monitor && state.monitor.targetUrl ? monitorRoute : currentRoute);
  renderProbes(state.monitor && state.monitor.results);

  groupSelect.innerHTML = '';
  for (const accessPath of remote.accessPaths || []) {
    const option = document.createElement('option');
    option.value = accessPath.id;
    option.textContent = accessPath.name;
    option.selected = activeAccessPath && accessPath.id === activeAccessPath.id;
    groupSelect.appendChild(option);
  }
  groupSelect.disabled = !(remote.accessPaths || []).length;
  masterToggle.disabled = !activeAccessPath;
  addDirect.disabled = !(currentTab && currentTab.host);
  addProxy.disabled = !(currentTab && currentTab.host);
  removeOverride.disabled = !(currentTab && currentTab.host);
  testUrlButton.disabled = !(remote.routes || []).length;

  if (!session.authenticated) {
    setStatus('warning', text('statusLoginRequired'));
  } else if (!activeAccessPath) {
    setStatus('warning', text('statusNoGroups'));
  } else if (!state.enabled) {
    setStatus('idle', text('statusReadyOff'));
  } else {
    setStatus('ok', text('statusReadyOn'));
  }
}

masterToggle.addEventListener('click', () => {
  if (!viewState || !viewState.activeAccessPath) {
    return;
  }
  sendMessage({ type: 'set-enabled', enabled: !viewState.state.enabled }).then((result) => {
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
  });
});

groupSelect.addEventListener('change', (event) => {
  sendMessage({ type: 'select-access-path', accessPathId: event.target.value }).then((result) => {
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
  });
});

addDirect.addEventListener('click', () => {
  sendMessage({ type: 'add-current-host-to-direct' }).then((result) => {
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
  });
});

addProxy.addEventListener('click', () => {
  sendMessage({ type: 'add-current-host-to-proxy' }).then((result) => {
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
  });
});

removeOverride.addEventListener('click', () => {
  sendMessage({ type: 'remove-current-host-override' }).then((result) => {
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
  });
});

syncButton.addEventListener('click', () => {
  sendMessage({ type: 'sync-remote-config' }).then((result) => {
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
  setStatus('ok', text('statusSynced'));
  });
});

testUrlButton.addEventListener('click', () => {
  const targetUrl = testUrlInput.value.trim();
  if (!targetUrl) {
    setStatus('error', text('statusInvalidTarget'));
    return;
  }
  testUrlButton.disabled = true;
  setStatus('idle', text('statusTesting'));
  sendMessage({ type: 'test-url-route', url: targetUrl, saveMonitorTarget: true })
    .then((result) => {
    if (result && result.error) {
      setStatus('error', result.error);
      return;
    }
    renderTopology(result.route);
    renderProbes(result.results);
      return sendMessage({ type: 'get-state' })
        .then((refreshed) => {
          render(refreshed);
          setStatus('ok', text('statusTested'));
        });
    })
    .catch((error) => setStatus('error', error.message || 'test_failed'))
    .finally(() => {
      testUrlButton.disabled = !viewState || !(viewState.remote.routes || []).length;
    });
});

bindThemeMode(themeMode, (mode) => sendMessage({ type: 'set-theme-mode', themeMode: mode }).then((result) => {
  if (result && result.error) {
    setStatus('error', result.error);
    if (viewState && viewState.state) {
      applyThemeMode(viewState.state.themeMode);
    }
    return;
  }
  render(result);
}));

openOptions.addEventListener('click', () => chrome.runtime.openOptionsPage());

chrome.runtime.onMessage.addListener((message) => {
  if (message && message.type === 'state-updated') {
    render(message.payload);
  }
});

function init() {
  applyLanguage();
  sendMessage({ type: 'get-state' }).then(render);
}

init();
