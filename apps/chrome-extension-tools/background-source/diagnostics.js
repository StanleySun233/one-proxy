const LOG_KEY = 'oneProxyDiagnostics';
const MAX_LOG_ENTRIES = 120;

export function appendLog(level, event, details = {}) {
  const entry = {
    at: new Date().toISOString(),
    level,
    event,
    details
  };
  return chrome.storage.local.get(LOG_KEY)
    .then((stored) => {
      const logs = Array.isArray(stored[LOG_KEY]) ? stored[LOG_KEY] : [];
      const nextLogs = [...logs, entry].slice(-MAX_LOG_ENTRIES);
      return chrome.storage.local.set({ [LOG_KEY]: nextLogs });
    })
    .then(() => entry);
}

export function diagnosticLogs() {
  return chrome.storage.local.get(LOG_KEY)
    .then((stored) => Array.isArray(stored[LOG_KEY]) ? stored[LOG_KEY] : []);
}

export function clearDiagnosticLogs() {
  return chrome.storage.local.set({ [LOG_KEY]: [] })
    .then(() => appendLog('info', 'diagnostics_cleared'))
    .then(() => diagnosticLogs());
}
