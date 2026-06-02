import { clearDiagnosticLogs, diagnosticLogs, appendLog } from './diagnostics.js';
import { login, logout, selectTenant, syncRemoteConfig, testConnection } from './api.js';
import { activeGroupFrom, getState, persistState, setPartialState, uniqueStrings } from './state.js';
import { pacSummary } from './pac.js';
import { routePreviewForUrl, sanitizeHost } from './routing.js';
import { testUrlRoute } from './monitor.js';

export function getCurrentTabInfo() {
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

export function getComputedState() {
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

export function registerMessageHandler() {
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
