import { getState } from './state.js';
import { routePreviewForUrl } from './routing.js';

const pagesByTab = new Map();
const requestsById = new Map();

function nowIso() {
  return new Date().toISOString();
}

function pageFor(tabId, url) {
  const current = pagesByTab.get(tabId);
  if (current && current.url === url) {
    return current;
  }
  const page = {
    url,
    host: hostOf(url),
    openedAt: nowIso(),
    requestCount: 0,
    proxiedRequestCount: 0,
    directRequestCount: 0,
    failureCount: 0,
    uploadBytes: 0,
    downloadBytes: 0,
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
  const header = (headers || []).find((item) => String(item.name || '').toLowerCase() === 'content-length');
  const value = header ? Number(header.value) : 0;
  return Number.isFinite(value) && value > 0 ? value : 0;
}

function trackStarted(details) {
  if (details.tabId < 0 || !details.url) {
    return;
  }
  if (details.type === 'main_frame') {
    pagesByTab.delete(details.tabId);
  }
  const page = pageFor(details.tabId, details.documentUrl || details.initiator || details.url);
  page.requestCount += 1;
  const uploadBytes = estimateUploadBytes(details.requestBody);
  page.uploadBytes += uploadBytes;
  requestsById.set(details.requestId, { tabId: details.tabId, uploadBytes });
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
  page.downloadBytes += contentLength(details.responseHeaders);
}

function trackFinished(details) {
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
