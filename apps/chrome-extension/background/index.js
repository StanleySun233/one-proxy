import { appendLog } from './diagnostics.js';
import { applyProxy } from './pac.js';
import { getState, configureStateEffects, handleStateStorageChange, persistState } from './state.js';
import { getComputedState, registerMessageHandler } from './messages.js';
import { runProxyMonitor } from './monitor.js';

async function broadcastState() {
  try {
    await chrome.runtime.sendMessage({ type: 'state-updated', payload: await getComputedState() });
  } catch (_error) {
  }
}

function ensureMonitorAlarm() {
  if (chrome.alarms) {
    chrome.alarms.create('proxy-monitor', { periodInMinutes: 1 });
  }
}

configureStateEffects(async (state) => {
  await applyProxy(state);
  await broadcastState();
});

chrome.runtime.onInstalled.addListener(async () => {
  ensureMonitorAlarm();
  await persistState(await getState());
  await appendLog('info', 'extension_installed');
});

chrome.runtime.onStartup.addListener(async () => {
  ensureMonitorAlarm();
  await applyProxy(await getState());
  await appendLog('info', 'extension_startup');
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

chrome.storage.onChanged.addListener(async (changes, areaName) => {
  const nextState = handleStateStorageChange(changes, areaName);
  if (!nextState) {
    return;
  }
  await applyProxy(nextState);
  await broadcastState();
});

registerMessageHandler();
