import { clearDiagnosticLogs, diagnosticLogs, appendLog } from './diagnostics.js';
import { login, logout, normalizeControlPlaneUrl, selectTenant, syncRemoteConfig, testConnection } from './api.js';
import { accessPathsView, getState, persistState, setPartialState, uniqueStrings } from './state.js';
import { pacSummary } from './pac.js';
import { routePreviewForUrl, sanitizeHost } from './routing.js';
import { testUrlRoute } from './monitor.js';
import { getStatusBubblePageStatus } from './status-bubble.js';

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
  'set-access-path-enabled',
  'set-local-overrides',
  'set-local-helper',
  'add-current-host-to-direct',
  'add-current-host-to-proxy',
  'remove-current-host-override'
]);
const CONTENT_MESSAGE_TYPES = new Set(['status-bubble-page-status']);

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
    accessPaths: accessPathsView(state),
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
    case 'set-access-path-enabled':
      return computedAfter(() => setPartialState((state) => {
        const accessPathId = String(message.accessPathId || '');
        const disabled = uniqueStrings(state.accessPathSwitches && state.accessPathSwitches.disabledAccessPathIds)
          .filter((id) => id !== accessPathId);
        if (accessPathId && !message.enabled) {
          disabled.push(accessPathId);
        }
        return {
          ...state,
          accessPathSwitches: {
            ...(state.accessPathSwitches || {}),
            disabledAccessPathIds: uniqueStrings(disabled)
          }
        };
      }));
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

export function registerMessageHandler() {
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
