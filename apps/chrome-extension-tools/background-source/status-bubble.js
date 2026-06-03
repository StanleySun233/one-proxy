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
  'statusBubbleLastError',
  'statusBubbleTopology',
  'statusBubbleUserMachine',
  'statusBubbleWebsite',
  'statusBubbleRoundTrip',
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

function pathHealthKey(group, route) {
  const topologyIds = (route.topology || []).map((node) => node.id).filter(Boolean).join('>');
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

function entryNodeId(route) {
  const first = route.topology && route.topology[0];
  if (!first || !first.id) {
    throw new Error('status_bubble_entry_node_required');
  }
  return first.id;
}

function requestNodePathHealth(group, route) {
  const endpoint = `http://${group.proxyHost}:${group.proxyPort}/api/v1/control-relay/probe`;
  const remainingHopNodeIds = (route.topology || []).map((node) => node.id).filter(Boolean).slice(1);
  return fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
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

function measurePathHealth(group, route) {
  const key = pathHealthKey(group, route);
  const cached = pathHealthCache.get(key);
  const now = Date.now();
  if (cached && now - cached.sampleTsMs < PATH_HEALTH_TTL_MS) {
    return Promise.resolve(cached);
  }
  return Promise.all([
    measureEntryLatency(group),
    requestNodePathHealth(group, route)
  ]).then(([entryLatencyMs, nodeTimings]) => {
    const result = {
      sampleTsMs: Date.now(),
      linkTimings: [
        {
          fromNodeId: 'user',
          toNodeId: entryNodeId(route),
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
  return {
    host: route.host || '',
    openedAt: (metrics && metrics.openedAt) || new Date().toISOString(),
    requestCount: Number((remoteStatus && remoteStatus.requestCount) || (metrics && metrics.requestCount) || 0),
    proxiedRequestCount: Number((metrics && metrics.proxiedRequestCount) || 0),
    directRequestCount: Number((metrics && metrics.directRequestCount) || 0),
    failureCount: Number((remoteStatus && remoteStatus.failureCount) || (metrics && metrics.failureCount) || 0)
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
  });
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
      const metrics = tabMetricsSnapshot(sender, url);
      if (!shouldDisplay(state, route)) {
        return {
          display: false
        };
      }
      return Promise.all([
        requestRemoteStatus(state, route, routeInfo),
        measurePathHealth(group, route)
      ]).then(([remoteStatus, pathHealth]) => {
        if (!remoteStatus) {
          throw new Error('status_bubble_page_status_required');
        }
        const status = remoteStatus;
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
          labels: labels(),
          nodeTimings: [],
          linkTimings: pathHealth.linkTimings,
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
