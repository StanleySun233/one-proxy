import { activeGroupFrom, getState, persistState } from './state.js';
import { parseTargetUrl, routePreviewForUrl } from './routing.js';

export async function testUrlRoute(targetUrl, options = {}) {
  const state = await getState();
  const route = routePreviewForUrl(state, targetUrl);
  const results = await runProxyProbes(state, targetUrl, route);
  if (options.saveMonitorTarget) {
    await persistState({
      ...state,
      monitor: {
        targetUrl: parseTargetUrl(targetUrl).url || String(targetUrl || '').trim(),
        lastRunAt: new Date().toISOString(),
        results
      }
    });
  }
  return { route, results };
}

export async function runProxyMonitor() {
  const state = await getState();
  if (!state.enabled || !state.monitor.targetUrl) {
    return;
  }
  const route = routePreviewForUrl(state, state.monitor.targetUrl);
  const results = await runProxyProbes(state, state.monitor.targetUrl, route);
  await persistState({
    ...state,
    monitor: {
      ...state.monitor,
      lastRunAt: new Date().toISOString(),
      results
    }
  });
}

async function runProxyProbes(state, targetUrl, route) {
  const group = activeGroupFrom(state);
  const parsed = parseTargetUrl(targetUrl);
  if (!state.enabled || !group || !group.proxyHost || !group.proxyPort || !route || route.mode !== 'proxy') {
    return probeProtocols().map((protocol) => ({ protocol, status: 'skipped', latencyMs: 0, message: 'proxy_not_applied' }));
  }
  const remainingHopNodeIds = (route.topology || []).map((node) => node.id).filter(Boolean).slice(1);
  const endpoint = `http://${group.proxyHost}:${group.proxyPort}/api/v1/control-relay/probe`;
  return Promise.all(probeProtocols().map((protocol) => runNodeProbe(endpoint, {
    protocol,
    remainingHopNodeIds,
    targetHost: parsed.host,
    targetPort: parsed.port
  })));
}

function probeProtocols() {
  return ['http', 'https', 'ws', 'tcp', 'udp'];
}

async function runNodeProbe(endpoint, payload) {
  const startedAt = Date.now();
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 8000);
  try {
    const response = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
      signal: controller.signal
    });
    let body = null;
    try {
      body = await response.json();
    } catch (_error) {
    }
    return {
      protocol: payload.protocol,
      status: response.ok && body && body.status ? body.status : 'failed',
      latencyMs: Date.now() - startedAt,
      message: body && body.message ? body.message : `http_${response.status}`
    };
  } catch (error) {
    return {
      protocol: payload.protocol,
      status: 'failed',
      latencyMs: Date.now() - startedAt,
      message: error.name === 'AbortError' ? 'probe_timeout' : error.message || 'probe_failed'
    };
  } finally {
    clearTimeout(timeout);
  }
}
