import { getExtensionPageStatus, syncRemoteConfig } from './api.js';
import { activeGroupFrom, getState } from './state.js';
import { routePreviewForUrl, urlHostname } from './routing.js';
import { tabMetricsSnapshot } from './page-metrics.js';

const STATUS_BUBBLE_LABELS = [
  'account',
  'activeGroup',
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
    if (!message) {
      throw new Error(`missing_i18n_message:${key}`);
    }
    return [key, message];
  }));
}

function entryProbeUrl(group) {
  if (!group || !group.proxyHost || !group.proxyPort) {
    throw new Error('status_bubble_entry_proxy_required');
  }
  return `http://${group.proxyHost}:${group.proxyPort}/healthz`;
}

function measureEntryLatency(group) {
  const startedAt = Date.now();
  return fetch(entryProbeUrl(group), { cache: 'no-store' })
    .then((response) => {
      if (!response.ok) {
        throw new Error(`status_bubble_entry_probe_failed:${response.status}`);
      }
      return Date.now() - startedAt;
    });
}

function routeTopology(group, route) {
  const topology = Array.isArray(route.topology) && route.topology.length > 0 ? route.topology : (group && Array.isArray(group.topology) ? group.topology : []);
  if (topology.length > 0) {
    return topology;
  }
  if (group && (group.entryNodeId || group.entryNodeName)) {
    return [{ id: group.entryNodeId || group.entryNodeName, name: group.entryNodeName || group.entryNodeId, mode: 'edge' }];
  }
  return [];
}

function pathHealthKey(group, route) {
  const topologyIds = routeTopology(group, route).map((node) => node.id).filter(Boolean).join('>');
  return [
    group.id || '',
    group.proxyHost || '',
    group.proxyPort || '',
    topologyIds,
    route.protocol || '',
    route.host || '',
    route.port || ''
  ].join('|');
}

function entryNodeId(group, route) {
  const first = routeTopology(group, route)[0];
  if (!first || !first.id) {
    throw new Error('status_bubble_entry_node_required');
  }
  return first.id;
}

function oneProxyTokenHeaders(state) {
  const token = state.session && state.session.proxyToken ? String(state.session.proxyToken) : '';
  return token ? { 'X-One-Proxy-Token': token } : {};
}

function requestNodePathHealth(state, group, route) {
  const endpoint = `http://${group.proxyHost}:${group.proxyPort}/api/control/relay/probe`;
  const remainingHopNodeIds = routeTopology(group, route).map((node) => node.id).filter(Boolean).slice(1);
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

function measurePathHealth(state, group, route) {
  if (state.localHelper && state.localHelper.enabled) {
    return Promise.resolve({
      sampleTsMs: Date.now(),
      linkTimings: []
    });
  }
  const key = pathHealthKey(group, route);
  const cached = pathHealthCache.get(key);
  const now = Date.now();
  if (cached && now - cached.sampleTsMs < PATH_HEALTH_TTL_MS) {
    return Promise.resolve(cached);
  }
  return Promise.all([
    measureEntryLatency(group),
    requestNodePathHealth(state, group, route)
  ]).then(([entryLatencyMs, nodeTimings]) => {
    const result = {
      sampleTsMs: Date.now(),
      linkTimings: [
        {
          fromNodeId: 'user',
          toNodeId: entryNodeId(group, route),
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
  const proxiedRequestCount = Number((metrics && metrics.proxiedRequestCount) || 0) || (route.mode === 'proxy' ? requestCount : 0);
  const directRequestCount = Number((metrics && metrics.directRequestCount) || 0) || (route.mode === 'proxy' ? 0 : requestCount);
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

function pathPayload(state, group, route, status, pageHost) {
  const topology = routeTopology(group, route);
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
  if (primary && Number(primary.requestCount || 0) > 0) {
    return primary;
  }
  if (!primary) {
    return fallback;
  }
  if (!fallback) {
    return primary;
  }
  return {
    ...primary,
    ...fallback,
    openedAt: primary.openedAt || fallback.openedAt
  };
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
      const group = activeGroupFrom(state);
      const route = routePreviewForUrl(state, url);
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
        Promise.resolve().then(() => measurePathHealth(state, group, route)).catch(() => emptyPathHealth())
      ]).then(([remoteStatus, pathHealth]) => {
        const status = remoteStatus || { status: 'unknown' };
        const page = mergePageSnapshot(route, metrics, status);
        const uploadBytes = Number(status.uploadBytes || (metrics && metrics.uploadBytes) || 0);
        const downloadBytes = Number(status.downloadBytes || (metrics && metrics.downloadBytes) || 0);
        const latencyMs = effectiveLatencyMs(status, metrics, pathHealth);
        const actualNodeTimings = normalizeNodeTimings(status.nodeTimings);
        const actualLinkTimings = normalizeLinkTimings(status.linkTimings);
        const linkTimings = actualLinkTimings.length > 0 ? actualLinkTimings : normalizeLinkTimings(pathHealth.linkTimings);
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
          group: group ? { id: group.id || '', name: group.name || group.entryNodeName || group.id || '' } : { id: '', name: '' },
          route: routeInfo,
          page,
          io: {
            uploadBytes,
            downloadBytes,
            correlated: Boolean(status.correlated || (metrics && metrics.responseCount > 0))
          },
          cache,
          latencyMs,
          path: pathPayload(state, group, route, status, page.host),
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
