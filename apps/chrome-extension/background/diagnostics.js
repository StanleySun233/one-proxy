const LOG_KEY = 'oneProxyDiagnostics';
const MAX_LOG_ENTRIES = 120;

export async function appendLog(level, event, details = {}) {
  const entry = {
    at: new Date().toISOString(),
    level,
    event,
    details
  };
  const stored = await chrome.storage.local.get(LOG_KEY);
  const logs = Array.isArray(stored[LOG_KEY]) ? stored[LOG_KEY] : [];
  const nextLogs = [...logs, entry].slice(-MAX_LOG_ENTRIES);
  await chrome.storage.local.set({ [LOG_KEY]: nextLogs });
  return entry;
}

export async function diagnosticLogs() {
  const stored = await chrome.storage.local.get(LOG_KEY);
  return Array.isArray(stored[LOG_KEY]) ? stored[LOG_KEY] : [];
}

export async function clearDiagnosticLogs() {
  await chrome.storage.local.set({ [LOG_KEY]: [] });
  await appendLog('info', 'diagnostics_cleared');
  return diagnosticLogs();
}
