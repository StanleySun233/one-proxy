import { activeGroupFrom, getState, persistState } from './state.js';
import { parseTargetUrl, routePreviewForUrl } from './routing.js';

export function testUrlRoute(targetUrl, options = {}) {
  return getState().then((state) => {
    const route = routePreviewForUrl(state, targetUrl);
    return runProxyProbes(state, targetUrl, route)
      .then((results) => {
        if (!options.saveMonitorTarget) {
          return { route, results };
        }
        return persistState({
          ...state,
          monitor: {
            targetUrl: parseTargetUrl(targetUrl).url || String(targetUrl || '').trim(),
            lastRunAt: new Date().toISOString(),
            results
          }
        }).then(() => ({ route, results }));
      });
  });
}

export function runProxyMonitor() {
  return getState().then((state) => {
    if (!state.enabled || !state.monitor.targetUrl) {
      return;
    }
    const route = routePreviewForUrl(state, state.monitor.targetUrl);
    return runProxyProbes(state, state.monitor.targetUrl, route)
      .then((results) => persistState({
        ...state,
        monitor: {
          ...state.monitor,
          lastRunAt: new Date().toISOString(),
          results
        }
      }));
  });
}

function runProxyProbes(state, targetUrl, route) {
  const group = activeGroupFrom(state);
  const parsed = parseTargetUrl(targetUrl);
  if (!state.enabled || !group || !group.proxyHost || !group.proxyPort || !route || route.mode !== 'proxy') {
    return probeProtocols().map((protocol) => ({ protocol, status: 'skipped', latencyMs: 0, message: 'proxy_not_applied' }));
  }
  const remainingHopNodeIds = (route.topology || []).map((node) => node.id).filter(Boolean).slice(1);
  const endpoint = `http://${group.proxyHost}:${group.proxyPort}/api/control/relay/probe`;
  return Promise.all(probeProtocols().map((protocol) => runNodeProbe(state, endpoint, {
    protocol,
    remainingHopNodeIds,
    targetHost: parsed.host,
    targetPort: parsed.port
  })));
}

function probeProtocols() {
  return ['http', 'https', 'ws', 'tcp', 'udp'];
}

function oneProxyTokenHeaders(state) {
  const token = state.session && state.session.proxyToken ? String(state.session.proxyToken) : '';
  return token ? { 'X-One-Proxy-Token': token } : {};
}

function runNodeProbe(state, endpoint, payload) {
  const startedAt = Date.now();
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 8000);
  return fetch(endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', ...oneProxyTokenHeaders(state) },
    body: JSON.stringify(payload),
    signal: controller.signal
  })
    .then((response) => response.json()
      .catch(() => null)
      .then((body) => ({
        protocol: payload.protocol,
        status: response.ok && body && body.status ? body.status : 'failed',
        latencyMs: Date.now() - startedAt,
        message: body && body.message ? body.message : `http_${response.status}`
      })))
    .catch((error) => ({
      protocol: payload.protocol,
      status: 'failed',
      latencyMs: Date.now() - startedAt,
      message: error.name === 'AbortError' ? 'probe_timeout' : error.message || 'probe_failed'
    }))
    .finally(() => clearTimeout(timeout));
}
