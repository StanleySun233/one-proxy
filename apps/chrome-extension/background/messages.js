import { clearDiagnosticLogs, diagnosticLogs, appendLog } from './diagnostics.js';
import { login, logout, selectTenant, syncRemoteConfig, testConnection } from './api.js';
import { activeGroupFrom, getState, persistState, setPartialState, uniqueStrings } from './state.js';
import { pacSummary } from './pac.js';
import { routePreviewForUrl, sanitizeHost } from './routing.js';
import { testUrlRoute } from './monitor.js';

export async function getCurrentTabInfo() {
  const queries = [{ active: true, currentWindow: true }, { active: true, lastFocusedWindow: true }];
  for (const query of queries) {
    const tabs = await chrome.tabs.query(query);
    const tab = tabs[0];
    if (!tab || !tab.url) {
      continue;
    }
    try {
      const parsed = new URL(tab.url);
      return {
        url: tab.url,
        host: parsed.hostname
      };
    } catch (_error) {
    }
  }
  return null;
}

export async function getComputedState() {
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

async function computedAfter(operation) {
  const result = await operation();
  return result || getComputedState();
}

export function registerMessageHandler() {
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
          sendResponse(await computedAfter(() => setPartialState((state) => ({ ...state, enabled: Boolean(message.enabled) }))));
          return;
        case 'set-theme-mode':
          sendResponse(await computedAfter(() => setPartialState((state) => ({ ...state, themeMode: message.themeMode === 'dark' ? 'dark' : 'vivid' }))));
          return;
        case 'set-control-plane-url':
          sendResponse(await computedAfter(() => setPartialState((state) => ({ ...state, controlPlaneUrl: String(message.controlPlaneUrl || '').trim() }))));
          return;
        case 'login':
          sendResponse(await computedAfter(() => login(message.controlPlaneUrl, message.account, message.password)));
          return;
        case 'test-connection':
          sendResponse(await testConnection(message.controlPlaneUrl));
          return;
        case 'logout':
          sendResponse(await computedAfter(() => logout()));
          return;
        case 'sync-remote-config':
          sendResponse(await computedAfter(() => syncRemoteConfig()));
          return;
        case 'select-tenant':
          sendResponse(await computedAfter(() => selectTenant(message.tenantId)));
          return;
        case 'test-url-route':
          sendResponse(await testUrlRoute(message.url, { saveMonitorTarget: Boolean(message.saveMonitorTarget) }));
          return;
        case 'select-group':
          sendResponse(await computedAfter(() => setPartialState((state) => ({
            ...state,
            selection: {
              ...state.selection,
              activeGroupId: message.groupId || ''
            }
          }))));
          return;
        case 'set-local-overrides':
          sendResponse(await computedAfter(() => setPartialState((state) => ({
            ...state,
            localOverrides: {
              directHosts: uniqueStrings(message.directHosts),
              proxyHosts: uniqueStrings(message.proxyHosts)
            }
          }))));
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
}
