import { getExtensionPageStatus, syncRemoteConfig } from './api.js';
import { appendLog } from './diagnostics.js';
import { activeGroupFrom, getState } from './state.js';
import { routePreviewForUrl, urlHostname } from './routing.js';
import { tabMetricsSnapshot } from './page-metrics.js';

function tenantFrom(state) {
  const membership = state.session.tenantMemberships.find((item) => item.tenantId === state.session.activeTenantId);
  return {
    id: state.session.activeTenantId || '',
    name: (membership && membership.tenantName) || state.session.activeTenantId || ''
  };
}

function colorFor(status, latencyMs, routeMode) {
  if (routeMode !== 'proxy') {
    return 'gray';
  }
  if (status === 'error') {
    return 'red';
  }
  if (status === 'slow' || Number(latencyMs || 0) > 1000) {
    return 'yellow';
  }
  if (status === 'ok') {
    return 'green';
  }
  return 'gray';
}

function routePayload(route) {
  const rule = route.rule || null;
  if (!rule) {
    return {
      id: '',
      source: route.source || '',
      matchType: '',
      matchValue: '',
      chainId: ''
    };
  }
  return {
    id: rule.id || '',
    source: route.source || '',
    matchType: rule.matchType || '',
    matchValue: rule.matchValue || '',
    chainId: rule.chainId || ''
  };
}

function mergePageSnapshot(route, metrics, remoteStatus) {
  return {
    host: route.host || '',
    openedAt: (metrics && metrics.openedAt) || new Date().toISOString(),
    requestCount: Number((remoteStatus && remoteStatus.requestCount) || (metrics && metrics.requestCount) || 0),
    proxiedRequestCount: Number((metrics && metrics.proxiedRequestCount) || 0),
    directRequestCount: Number((metrics && metrics.directRequestCount) || 0),
    failureCount: Number((remoteStatus && remoteStatus.failureCount) || (metrics && metrics.failureCount) || 0)
  };
}

function fallbackStatus(metrics) {
  return {
    status: (metrics && metrics.failureCount) ? 'error' : 'unknown',
    latencyMs: 0,
    uploadBytes: metrics ? metrics.uploadBytes : 0,
    downloadBytes: metrics ? metrics.downloadBytes : 0,
    requestCount: metrics ? metrics.requestCount : 0,
    failureCount: metrics ? metrics.failureCount : 0,
    lastErrorCode: metrics ? metrics.lastErrorCode : '',
    lastErrorMessage: metrics ? metrics.lastErrorMessage : '',
    policyRevision: '',
    correlated: false
  };
}

function requestRemoteStatus(state, route, routeInfo) {
  if (route.mode !== 'proxy' || !state.session.accessToken || !state.session.activeTenantId) {
    return Promise.resolve(null);
  }
  return getExtensionPageStatus(state, {
    host: route.host,
    routeId: routeInfo.id,
    chainId: routeInfo.chainId
  }).catch((error) => appendLog('error', 'status_bubble_page_status_failed', {
    message: error.message || 'page_status_failed',
    host: route.host
  }).then(() => null));
}

function shouldDisplay(state, route) {
  const controlPlaneHost = urlHostname(state.controlPlaneUrl);
  if (!route.host || route.host === controlPlaneHost) {
    return false;
  }
  return route.mode === 'proxy';
}

export function getStatusBubblePageStatus(message, sender) {
  const url = message.url || (sender && sender.tab && sender.tab.url) || '';
  return (message.refresh ? syncRemoteConfig().catch(() => null) : Promise.resolve(null))
    .then(() => getState())
    .then((state) => {
      const group = activeGroupFrom(state);
      const route = routePreviewForUrl(state, url);
      const routeInfo = routePayload(route);
      const metrics = tabMetricsSnapshot(sender, url);
      return requestRemoteStatus(state, route, routeInfo).then((remoteStatus) => {
        const status = remoteStatus || fallbackStatus(metrics);
        const page = mergePageSnapshot(route, metrics, status);
        const uploadBytes = Number(status.uploadBytes || (metrics && metrics.uploadBytes) || 0);
        const downloadBytes = Number(status.downloadBytes || (metrics && metrics.downloadBytes) || 0);
        const latencyMs = Number(status.latencyMs || 0);
        const lastErrorCode = status.lastErrorCode || (metrics && metrics.lastErrorCode) || '';
        const lastErrorMessage = status.lastErrorMessage || (metrics && metrics.lastErrorMessage) || '';
        return {
          status: status.status || 'unknown',
          color: colorFor(status.status, latencyMs, route.mode),
          display: shouldDisplay(state, route),
          account: state.session.account || '',
          tenant: tenantFrom(state),
          group: group ? { id: group.id || '', name: group.name || group.entryNodeName || group.id || '' } : { id: '', name: '' },
          route: routeInfo,
          page,
          io: {
            uploadBytes,
            downloadBytes,
            correlated: Boolean(status.correlated)
          },
          latencyMs,
          topology: route.topology || [],
          policyRevision: status.policyRevision || state.remote.policyRevision || '',
          configFetchedAt: state.remote.fetchedAt || '',
          lastError: {
            code: lastErrorCode,
            message: lastErrorMessage
          }
        };
      });
    });
}
