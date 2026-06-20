import { getExtensionPageStatus, syncRemoteConfig } from './api.js';
import { accessPathById, getState } from './state.js';
import { routePreviewForUrl, urlHostname } from './routing.js';
import { tabMetricsSnapshot } from './page-metrics.js';

const STATUS_BUBBLE_LABELS = [
  'account',
  'activeAccessPath',
  'policyRevision',
  'syncedAt',
  'tenant',
  'statusBubbleTitle',
  'statusBubbleUpload',
  'statusBubbleDownload',
  'statusBubbleLatency',
  'statusBubbleRequests',
  'statusBubbleRoute',
  'statusBubbleOpenedAt',
  'statusBubbleRequestMixShort',
  'statusBubbleCorrelation',
  'statusBubbleCorrelated',
  'statusBubbleNotCorrelated',
  'statusBubbleCache',
  'statusBubbleCacheStoredAt',
  'statusBubbleCacheResponses',
  'statusBubbleStatusCode',
  'statusBubbleHttpErrors',
  'statusBubbleErrorCodes',
  'statusBubbleLastError',
  'statusBubbleTopology',
  'statusBubbleUserMachine',
  'statusBubbleWebsite',
  'statusBubbleRoundTrip',
  'statusBubbleTransport',
  'statusBubbleDirectQUIC',
  'statusBubbleRelay',
  'statusBubbleFallback',
  'statusBubbleRefresh',
  'statusBubbleCopy',
  'statusBubbleCopied',
  'statusBubbleUnknown'
];
const PATH_HEALTH_TTL_MS = 60000;
const pathHealthCache = new Map();
const STATUS_BUBBLE_FALLBACK_LABELS = {
  account: 'Account',
  activeAccessPath: 'Active access path',
  policyRevision: 'Policy',
  syncedAt: 'Synced',
  tenant: 'Tenant',
  statusBubbleTitle: 'Proxy status',
  statusBubbleUpload: 'Upload',
  statusBubbleDownload: 'Download',
  statusBubbleLatency: 'Latency',
  statusBubbleRequests: 'Requests',
  statusBubbleRoute: 'Route',
  statusBubbleOpenedAt: 'Opened',
  statusBubbleRequestMixShort: 'P / D / F',
  statusBubbleCorrelation: 'IO source',
  statusBubbleCorrelated: 'Node correlated',
  statusBubbleNotCorrelated: 'Not correlated',
  statusBubbleCache: 'Cache',
  statusBubbleCacheStoredAt: 'Cache stored',
  statusBubbleCacheResponses: 'Cache responses',
  statusBubbleStatusCode: 'HTTP status',
  statusBubbleHttpErrors: 'HTTP errors',
  statusBubbleErrorCodes: 'Error codes',
  statusBubbleLastError: 'Last error',
  statusBubbleTopology: 'Path',
  statusBubbleUserMachine: 'User machine',
  statusBubbleWebsite: 'Website',
  statusBubbleRoundTrip: 'RTT',
  statusBubbleTransport: 'Transport',
  statusBubbleDirectQUIC: 'Direct QUIC',
  statusBubbleRelay: 'Relay',
  statusBubbleFallback: 'Fallback',
  statusBubbleRefresh: 'Refresh',
  statusBubbleCopy: 'Copy diagnostics',
  statusBubbleCopied: 'Copied',
  statusBubbleUnknown: 'Unknown'
};

function tenantFrom(state) {
  const membership = state.session.tenantMemberships.find((item) => item.tenantId === state.session.activeTenantId);
  return {
    id: state.session.activeTenantId || '',
    name: (membership && membership.tenantName) || state.session.activeTenantId || ''
  };
}

function labels() {
  return Object.fromEntries(STATUS_BUBBLE_LABELS.map((key) => {
    const message = chrome.i18n.getMessage(key);
    return [key, message || STATUS_BUBBLE_FALLBACK_LABELS[key] || key];
  }));
}

function accessPathForRoute(state, route) {
  return route && route.accessPathId ? accessPathById(state, route.accessPathId) : null;
}

function accessPathProbeBaseUrl(accessPath) {
  if (!accessPath || !accessPath.listenHost || !accessPath.listenPort) {
    throw new Error('status_bubble_access_path_proxy_required');
  }
  return `http://${accessPath.listenHost}:${accessPath.listenPort}`;
}

function entryProbeUrl(accessPath) {
  return `${accessPathProbeBaseUrl(accessPath)}/healthz`;
}

function measureEntryLatency(accessPath) {
  const startedAt = Date.now();
  return fetch(entryProbeUrl(accessPath), { cache: 'no-store' })
    .then((response) => {
      if (!response.ok) {
        throw new Error(`status_bubble_entry_probe_failed:${response.status}`);
      }
      return Date.now() - startedAt;
    });
}

function routeTopology(accessPath, route) {
  const topology = Array.isArray(route.topology) && route.topology.length > 0 ? route.topology : (accessPath && Array.isArray(accessPath.topology) ? accessPath.topology : []);
  if (topology.length > 0) {
    return topology;
  }
  if (accessPath && (accessPath.entryNodeId || accessPath.entryNodeName)) {
    return [{ id: accessPath.entryNodeId || accessPath.entryNodeName, name: accessPath.entryNodeName || accessPath.entryNodeId, mode: 'edge' }];
  }
  return [];
}

function pathHealthKey(accessPath, route) {
  const topologyIds = routeTopology(accessPath, route).map((node) => node.id).filter(Boolean).join('>');
  return [
    accessPath.id || '',
    accessPath.listenHost || '',
    accessPath.listenPort || '',
    topologyIds,
    route.protocol || '',
    route.host || '',
    route.port || ''
  ].join('|');
}

function entryNodeId(accessPath, route) {
  const first = routeTopology(accessPath, route)[0];
  if (!first || !first.id) {
    throw new Error('status_bubble_entry_node_required');
  }
  return first.id;
}

function oneProxyTokenHeaders(state) {
  const token = state.session && state.session.proxyToken ? String(state.session.proxyToken) : '';
  return token ? { 'X-One-Proxy-Token': token } : {};
}

function requestNodePathHealth(state, accessPath, route) {
  const endpoint = `${accessPathProbeBaseUrl(accessPath)}/api/control/relay/probe`;
  const remainingHopNodeIds = routeTopology(accessPath, route).map((node) => node.id).filter(Boolean).slice(1);
  return fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...oneProxyTokenHeaders(state) },
    body: JSON.stringify({
      protocol: route.protocol,
      remainingHopNodeIds,
      targetHost: route.host,
      targetPort: route.port
    })
  }).then((response) => response.json()
    .then((body) => {
      if (!response.ok) {
        throw new Error((body && body.message) || `path_health_failed:${response.status}`);
      }
      if (!body || !Array.isArray(body.pathTimings)) {
        throw new Error('path_health_timings_required');
      }
      return body.pathTimings;
    }));
}

function measurePathHealth(state, accessPath, route) {
  if (state.localHelper && state.localHelper.enabled) {
    return Promise.resolve({
      sampleTsMs: Date.now(),
      linkTimings: []
    });
  }
  const key = pathHealthKey(accessPath, route);
  const cached = pathHealthCache.get(key);
  const now = Date.now();
  if (cached && now - cached.sampleTsMs < PATH_HEALTH_TTL_MS) {
    return Promise.resolve(cached);
  }
  return Promise.all([
    measureEntryLatency(accessPath),
    requestNodePathHealth(state, accessPath, route)
  ]).then(([entryLatencyMs, nodeTimings]) => {
    const result = {
      sampleTsMs: Date.now(),
      linkTimings: [
        {
          fromNodeId: 'user',
          toNodeId: entryNodeId(accessPath, route),
          roundTripMs: entryLatencyMs,
          sampleTsMs: Date.now(),
          count: 1
        },
        ...nodeTimings
      ]
    };
    pathHealthCache.set(key, result);
    return result;
  });
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
  const requestCount = Number((remoteStatus && remoteStatus.requestCount) || (metrics && metrics.requestCount) || 0);
  const rawProxiedRequestCount = Number((metrics && metrics.proxiedRequestCount) || 0);
  const rawDirectRequestCount = Number((metrics && metrics.directRequestCount) || 0);
  const proxiedRequestCount = route.mode === 'proxy'
    ? Math.max(rawProxiedRequestCount, requestCount - rawDirectRequestCount)
    : rawProxiedRequestCount;
  const directRequestCount = route.mode === 'proxy'
    ? rawDirectRequestCount
    : Math.max(rawDirectRequestCount, requestCount - rawProxiedRequestCount);
  return {
    host: route.host || '',
    openedAt: (metrics && metrics.openedAt) || new Date().toISOString(),
    requestCount,
    proxiedRequestCount,
    directRequestCount,
    failureCount: Number((remoteStatus && remoteStatus.failureCount) || (metrics && metrics.failureCount) || 0),
    statusCode: Number((remoteStatus && remoteStatus.statusCode) || (metrics && metrics.statusCode) || 0),
    httpErrorCount: Number((metrics && metrics.httpErrorCount) || 0),
    errorCodeCount: (metrics && metrics.errorCodeCount) || {},
    cacheStatus: String((remoteStatus && remoteStatus.cacheStatus) || (metrics && metrics.cacheStatus) || ''),
    cacheStoredAt: String((remoteStatus && remoteStatus.cacheStoredAt) || (metrics && metrics.cacheStoredAt) || ''),
    cacheAgeSeconds: Number((metrics && metrics.cacheAgeSeconds) || 0),
    cacheResponseCount: Number((metrics && metrics.cacheResponseCount) || 0)
  };
}

function pathPayload(state, accessPath, route, status, pageHost) {
  const topology = routeTopology(accessPath, route);
  const transport = state.localHelper && state.localHelper.enabled ? 'direct_quic' : (status && status.path && status.path.transport ? String(status.path.transport) : 'relay');
  const nodes = [
    { id: 'user', name: 'User machine', kind: 'user', transport: 'client' },
    ...topology.map((node, index) => ({
      id: node.id,
      name: node.name || node.id,
      kind: 'node',
      mode: node.mode || '',
      transport: index === 0 ? transport : 'relay_ws_parent'
    })),
    { id: pageHost || 'website', name: pageHost || 'Website', kind: 'web', transport: 'target' }
  ];
  return {
    mode: transport,
    transport,
    fallbackReason: '',
    nodes
  };
}

function requestRemoteStatus(state, route, routeInfo) {
  if (route.mode !== 'proxy') {
    return Promise.resolve(null);
  }
  if (!state.session.accessToken || !state.session.activeTenantId) {
    throw new Error('status_bubble_session_required');
  }
  return getExtensionPageStatus(state, {
    host: route.host,
    routeId: routeInfo.id,
    chainId: routeInfo.chainId
  }).then((status) => {
    if (isUncorrelatedStatus(status) && (routeInfo.id || routeInfo.chainId)) {
      return getExtensionPageStatus(state, { host: route.host });
    }
    return status;
  });
}

function isUncorrelatedStatus(status) {
  return !status || (!status.correlated && String(status.status || '') === 'unknown');
}

function emptyPathHealth() {
  return {
    sampleTsMs: Date.now(),
    linkTimings: []
  };
}

function normalizePageMetrics(metrics) {
  if (!metrics || typeof metrics !== 'object') {
    return null;
  }
  return {
    openedAt: String(metrics.openedAt || ''),
    requestCount: Number(metrics.requestCount || 0),
    responseCount: Number(metrics.responseCount || 0),
    proxiedRequestCount: Number(metrics.proxiedRequestCount || 0),
    directRequestCount: Number(metrics.directRequestCount || 0),
    failureCount: Number(metrics.failureCount || 0),
    uploadBytes: Number(metrics.uploadBytes || 0),
    downloadBytes: Number(metrics.downloadBytes || 0),
    latencyMs: Number(metrics.latencyMs || 0),
    statusCode: Number(metrics.statusCode || 0),
    httpErrorCount: Number(metrics.httpErrorCount || 0),
    errorCodeCount: metrics.errorCodeCount && typeof metrics.errorCodeCount === 'object' ? metrics.errorCodeCount : {},
    cacheStatus: String(metrics.cacheStatus || ''),
    cacheStoredAt: String(metrics.cacheStoredAt || ''),
    cacheAgeSeconds: Number(metrics.cacheAgeSeconds || 0),
    cacheResponseCount: Number(metrics.cacheResponseCount || 0),
    lastErrorCode: String(metrics.lastErrorCode || ''),
    lastErrorMessage: String(metrics.lastErrorMessage || '')
  };
}

function mergeLocalMetrics(primary, fallback) {
  if (!primary) {
    return fallback;
  }
  if (!fallback) {
    return primary;
  }
  return {
    ...primary,
    openedAt: primary.openedAt || fallback.openedAt,
    requestCount: maxNumber(primary.requestCount, fallback.requestCount),
    responseCount: maxNumber(primary.responseCount, fallback.responseCount),
    proxiedRequestCount: maxNumber(primary.proxiedRequestCount, fallback.proxiedRequestCount),
    directRequestCount: maxNumber(primary.directRequestCount, fallback.directRequestCount),
    failureCount: maxNumber(primary.failureCount, fallback.failureCount),
    uploadBytes: maxNumber(primary.uploadBytes, fallback.uploadBytes),
    downloadBytes: maxNumber(primary.downloadBytes, fallback.downloadBytes),
    latencyMs: firstPositive(primary.latencyMs, fallback.latencyMs),
    statusCode: firstPositive(primary.statusCode, fallback.statusCode),
    httpErrorCount: maxNumber(primary.httpErrorCount, fallback.httpErrorCount),
    errorCodeCount: mergeErrorCodeCount(primary.errorCodeCount, fallback.errorCodeCount),
    cacheStatus: primary.cacheStatus || fallback.cacheStatus,
    cacheStoredAt: primary.cacheStoredAt || fallback.cacheStoredAt,
    cacheAgeSeconds: maxNumber(primary.cacheAgeSeconds, fallback.cacheAgeSeconds),
    cacheResponseCount: maxNumber(primary.cacheResponseCount, fallback.cacheResponseCount),
    lastErrorCode: primary.lastErrorCode || fallback.lastErrorCode,
    lastErrorMessage: primary.lastErrorMessage || fallback.lastErrorMessage
  };
}

function maxNumber(primary, fallback) {
  return Math.max(Number(primary || 0), Number(fallback || 0));
}

function firstPositive(primary, fallback) {
  const first = Number(primary || 0);
  return first > 0 ? first : Number(fallback || 0);
}

function mergeErrorCodeCount(primary, fallback) {
  const result = { ...(fallback || {}) };
  Object.entries(primary || {}).forEach(([key, value]) => {
    if (key) {
      result[key] = Math.max(Number(result[key] || 0), Number(value || 0));
    }
  });
  return result;
}

function pathLatencyMs(pathHealth) {
  const first = pathHealth && Array.isArray(pathHealth.linkTimings) ? pathHealth.linkTimings[0] : null;
  return first ? Number(first.roundTripMs || first.rttMs || 0) : 0;
}

function statusFrom(remoteStatus, metrics, routeMode, pathHealth) {
  const remote = String((remoteStatus && remoteStatus.status) || '').trim();
  if (remote && remote !== 'unknown') {
    return remote;
  }
  if (metrics && Number(metrics.failureCount || 0) > 0) {
    return 'error';
  }
  if (metrics && Number(metrics.responseCount || 0) > 0 && (Number(metrics.proxiedRequestCount || 0) > 0 || routeMode === 'proxy')) {
    return 'ok';
  }
  if (routeMode === 'proxy' && pathLatencyMs(pathHealth) > 0) {
    return 'ok';
  }
  return remote || 'unknown';
}

function statusWithLatency(status, latencyMs, routeMode) {
  if (status !== 'unknown' || routeMode !== 'proxy') {
    return status;
  }
  return Number(latencyMs || 0) > 0 ? 'ok' : status;
}

function effectiveLatencyMs(remoteStatus, metrics, pathHealth) {
  const remoteLatencyMs = Number((remoteStatus && remoteStatus.latencyMs) || 0);
  if (remoteLatencyMs > 0) {
    return remoteLatencyMs;
  }
  const localLatencyMs = Number((metrics && metrics.latencyMs) || 0);
  if (localLatencyMs > 0) {
    return localLatencyMs;
  }
  return pathLatencyMs(pathHealth);
}

function normalizeNodeTimings(items) {
  return (Array.isArray(items) ? items : []).map((item) => ({
    nodeId: item.nodeId || '',
    processAvgMs: Number(item.processAvgMs || 0),
    responseProcessAvgMs: Number(item.responseProcessAvgMs || 0),
    sampleTsMs: Number(item.sampleTsMs || 0),
    count: Number(item.count || 1)
  })).filter((item) => item.nodeId);
}

function normalizeLinkTimings(items) {
  return (Array.isArray(items) ? items : []).map((item) => ({
    fromNodeId: item.fromNodeId || '',
    toNodeId: item.toNodeId || '',
    roundTripMs: Number(item.roundTripMs || item.rttMs || 0),
    sampleTsMs: Number(item.sampleTsMs || 0),
    count: Number(item.count || 1)
  })).filter((item) => item.fromNodeId && item.toNodeId);
}

function mergeLinkTimings(remoteItems, measuredItems) {
  const result = [];
  const indexes = new Map();
  [...measuredItems, ...remoteItems].forEach((item) => {
    const key = `${item.fromNodeId}\u0000${item.toNodeId}`;
    if (indexes.has(key)) {
      result[indexes.get(key)] = item;
      return;
    }
    indexes.set(key, result.length);
    result.push(item);
  });
  return result;
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
  return (message.refresh ? syncRemoteConfig() : Promise.resolve(null))
    .then(() => getState())
    .then((state) => {
      const route = routePreviewForUrl(state, url);
      const accessPath = accessPathForRoute(state, route);
      const routeInfo = routePayload(route);
      const metrics = mergeLocalMetrics(tabMetricsSnapshot(sender, url), normalizePageMetrics(message.pageMetrics));
      if (!shouldDisplay(state, route)) {
        return {
          display: false
        };
      }
      return Promise.all([
        Promise.resolve().then(() => requestRemoteStatus(state, route, routeInfo)).catch((error) => ({
          status: 'unknown',
          lastErrorCode: 'page_status_failed',
          lastErrorMessage: error.message || 'page_status_failed'
        })),
        Promise.resolve().then(() => measurePathHealth(state, accessPath, route)).catch(() => emptyPathHealth())
      ]).then(([remoteStatus, pathHealth]) => {
        const status = remoteStatus || { status: 'unknown' };
        const page = mergePageSnapshot(route, metrics, status);
        const uploadBytes = Number(status.uploadBytes || (metrics && metrics.uploadBytes) || 0);
        const downloadBytes = Number(status.downloadBytes || (metrics && metrics.downloadBytes) || 0);
        const latencyMs = effectiveLatencyMs(status, metrics, pathHealth);
        const actualNodeTimings = normalizeNodeTimings(status.nodeTimings);
        const actualLinkTimings = normalizeLinkTimings(status.linkTimings);
        const measuredLinkTimings = normalizeLinkTimings(pathHealth.linkTimings);
        const linkTimings = mergeLinkTimings(actualLinkTimings, measuredLinkTimings);
        const lastErrorCode = status.lastErrorCode || (metrics && metrics.lastErrorCode) || '';
        const lastErrorMessage = status.lastErrorMessage || (metrics && metrics.lastErrorMessage) || '';
        const displayStatus = statusWithLatency(statusFrom(status, metrics, route.mode, pathHealth), latencyMs, route.mode);
        const cache = {
          status: page.cacheStatus || '',
          storedAt: page.cacheStoredAt || '',
          ageSeconds: page.cacheAgeSeconds || 0,
          responseCount: page.cacheResponseCount || 0
        };
        return {
          status: displayStatus,
          color: cache.status ? 'yellow' : colorFor(displayStatus, latencyMs, route.mode),
          display: shouldDisplay(state, route),
          account: state.session.account || '',
          tenant: tenantFrom(state),
          accessPath: accessPath ? { id: accessPath.id || '', name: accessPath.name || accessPath.entryNodeName || accessPath.id || '' } : { id: '', name: '' },
          route: routeInfo,
          page,
          io: {
            uploadBytes,
            downloadBytes,
            correlated: Boolean(status.correlated || (metrics && metrics.responseCount > 0))
          },
          cache,
          latencyMs,
          path: pathPayload(state, accessPath, route, status, page.host),
          labels: labels(),
          nodeTimings: actualNodeTimings,
          linkTimings,
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
