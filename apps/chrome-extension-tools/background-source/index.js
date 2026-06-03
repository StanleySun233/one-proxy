import { appendLog } from './diagnostics.js';
import { applyProxy } from './pac.js';
import { getState, configureStateEffects, handleStateStorageChange, persistState } from './state.js';
import { getComputedState, registerMessageHandler } from './messages.js';
import { runProxyMonitor } from './monitor.js';
import { registerProxyAuthHandler, updateProxyAuthCache } from './proxy-auth.js';

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
