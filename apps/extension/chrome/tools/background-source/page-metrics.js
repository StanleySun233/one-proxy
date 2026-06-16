import { getState } from './state.js';
import { routePreviewForUrl } from './routing.js';

const pagesByTab = new Map();
const requestsById = new Map();

function nowIso() {
  return new Date().toISOString();
}

function pageFor(tabId, url, options = {}) {
  const host = hostOf(url);
  const current = pagesByTab.get(tabId);
  if (!options.reset && current && (!url || current.url === url || current.host === host || !host)) {
    return current;
  }
  const page = {
    url,
    host,
    openedAt: nowIso(),
    requestCount: 0,
    responseCount: 0,
    proxiedRequestCount: 0,
    directRequestCount: 0,
    failureCount: 0,
    uploadBytes: 0,
    downloadBytes: 0,
    latencyMs: 0,
    latencyTotalMs: 0,
    latencyCount: 0,
    cacheStatus: '',
    cacheStoredAt: '',
    cacheAgeSeconds: 0,
    cacheResponseCount: 0,
    lastErrorCode: '',
    lastErrorMessage: ''
  };
  pagesByTab.set(tabId, page);
  return page;
}

function hostOf(url) {
  try {
    return new URL(url).hostname;
  } catch (_error) {
    return '';
  }
}

function estimateUploadBytes(requestBody) {
  if (!requestBody) {
    return 0;
  }
  if (Array.isArray(requestBody.raw)) {
    return requestBody.raw.reduce((total, item) => total + (item.bytes ? item.bytes.byteLength : 0), 0);
  }
  if (!requestBody.formData) {
    return 0;
  }
  return Object.entries(requestBody.formData).reduce((total, [key, values]) => {
    const valueBytes = (values || []).reduce((sum, value) => sum + String(value || '').length, 0);
    return total + key.length + valueBytes;
  }, 0);
}

function contentLength(headers) {
  const header = headerValue(headers, 'content-length');
  const value = header ? Number(header) : 0;
  return Number.isFinite(value) && value > 0 ? value : 0;
}

function headerValue(headers, name) {
  const header = (headers || []).find((item) => String(item.name || '').toLowerCase() === name);
  return header ? String(header.value || '') : '';
}

function trackCacheHeaders(page, headers) {
  const cacheStatus = headerValue(headers, 'x-one-proxy-cache');
  if (!cacheStatus) {
    return;
  }
  page.cacheStatus = cacheStatus;
  page.cacheStoredAt = headerValue(headers, 'x-one-proxy-cache-stored-at');
  page.cacheAgeSeconds = Number(headerValue(headers, 'x-one-proxy-cache-age-seconds') || 0) || 0;
  page.cacheResponseCount += 1;
}

function recordLatency(page, tracked) {
  if (!page || !tracked || tracked.responseSeen) {
    return;
  }
  const latencyMs = Date.now() - tracked.startedAt;
  if (latencyMs <= 0) {
    return;
  }
  tracked.responseSeen = true;
  page.responseCount += 1;
  page.latencyTotalMs += latencyMs;
  page.latencyCount += 1;
  page.latencyMs = Math.round(page.latencyTotalMs / page.latencyCount);
}

function trackStarted(details) {
  if (details.tabId < 0 || !details.url) {
    return;
  }
  let page;
  if (details.type === 'main_frame') {
    page = pageFor(details.tabId, details.url, { reset: true });
  } else {
    page = pageFor(details.tabId, details.documentUrl || details.initiator || details.url);
  }
  page.requestCount += 1;
  const uploadBytes = estimateUploadBytes(details.requestBody);
  page.uploadBytes += uploadBytes;
  requestsById.set(details.requestId, { tabId: details.tabId, uploadBytes, startedAt: Date.now(), responseSeen: false });
  getState()
    .then((state) => {
      const route = routePreviewForUrl(state, details.url);
      if (route.mode === 'proxy') {
        page.proxiedRequestCount += 1;
      } else {
        page.directRequestCount += 1;
      }
    })
    .catch(() => {
      page.directRequestCount += 1;
    });
}

function trackHeaders(details) {
  const tracked = requestsById.get(details.requestId);
  if (!tracked) {
    return;
  }
  const page = pagesByTab.get(tracked.tabId);
  if (!page) {
    return;
  }
  recordLatency(page, tracked);
  trackCacheHeaders(page, details.responseHeaders);
  page.downloadBytes += contentLength(details.responseHeaders);
}

function trackFinished(details) {
  const tracked = requestsById.get(details.requestId);
  if (tracked) {
    const page = pagesByTab.get(tracked.tabId);
    recordLatency(page, tracked);
  }
  requestsById.delete(details.requestId);
}

function trackFailed(details) {
  const tracked = requestsById.get(details.requestId);
  if (tracked) {
    const page = pagesByTab.get(tracked.tabId);
    if (page) {
      page.failureCount += 1;
      page.lastErrorCode = details.error || 'request_failed';
      page.lastErrorMessage = details.error || 'request_failed';
    }
  }
  requestsById.delete(details.requestId);
}

export function tabMetricsSnapshot(sender, url) {
  const tabId = sender && sender.tab ? sender.tab.id : -1;
  if (tabId < 0) {
    return null;
  }
  const page = pageFor(tabId, url || (sender.tab && sender.tab.url) || '');
  return structuredClone(page);
}

export function registerPageMetrics() {
  if (!chrome.webRequest) {
    return;
  }
  chrome.webRequest.onBeforeRequest.addListener(trackStarted, { urls: ['<all_urls>'] }, ['requestBody']);
  chrome.webRequest.onHeadersReceived.addListener(trackHeaders, { urls: ['<all_urls>'] }, ['responseHeaders']);
  chrome.webRequest.onCompleted.addListener(trackFinished, { urls: ['<all_urls>'] });
  chrome.webRequest.onErrorOccurred.addListener(trackFailed, { urls: ['<all_urls>'] });
  if (chrome.tabs && chrome.tabs.onRemoved) {
    chrome.tabs.onRemoved.addListener((tabId) => pagesByTab.delete(tabId));
  }
}
