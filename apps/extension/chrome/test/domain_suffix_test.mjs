import assert from 'node:assert/strict';
import test from 'node:test';

import { buildPacScript, pacSummary } from '../tools/background-source/pac.js';
import { runProxyProbes } from '../tools/background-source/monitor.js';
import { routeMatches, routePreviewForHost } from '../tools/background-source/routing.js';

function stateWithDomainSuffixRoute(matchValue) {
  return {
    enabled: true,
    controlPlaneUrl: 'https://panel.example.com',
    localOverrides: { directHosts: [], proxyHosts: [] },
    localHelper: {},
    accessPathSwitches: { disabledAccessPathIds: [] },
    remote: {
      nodes: [
        {
          id: 'node-1',
          name: 'Node 1',
          mode: 'edge',
          scopeKey: 'tenant-1',
          parentNodeId: '',
          enabled: true,
          status: 'online'
        }
      ],
      accessPaths: [
        {
          id: 'path-1',
          name: 'Path 1',
          chainId: 'chain-1',
          mode: 'forward',
          protocol: 'http',
          serviceType: 'http_forward_proxy',
          targetNodeId: 'node-1',
          entryNodeId: 'node-1',
          relayNodeIds: [],
          listenHost: '127.0.0.1',
          listenPort: 18080,
          targetProtocol: 'http',
          targetHost: '',
          targetPort: 0,
          targetSni: '',
          tlsMode: '',
          authMode: 'proxy_token',
          enabled: true,
          options: {},
          topology: [{ id: 'node-1', name: 'Node 1', mode: 'edge' }],
          health: { status: 'available', reason: '', checkedAt: '' }
        }
      ],
      routes: [
        {
          id: 'route-1',
          priority: 1,
          matchType: 'domain_suffix',
          matchValue,
          actionType: 'chain',
          chainId: 'chain-1',
          accessPathId: 'path-1',
          destinationScope: '',
          enabled: true,
          topology: []
        }
      ]
    }
  };
}

function stateWithParallelAccessPaths(disabledAccessPathIds = []) {
  const state = stateWithDomainSuffixRoute('.one.example');
  state.accessPathSwitches = { disabledAccessPathIds };
  state.remote.accessPaths.push({
    ...state.remote.accessPaths[0],
    id: 'path-2',
    name: 'Path 2',
    chainId: 'chain-2',
    listenPort: 18081
  });
  state.remote.routes.push({
    id: 'route-2',
    priority: 2,
    matchType: 'domain_suffix',
    matchValue: '.two.example',
    actionType: 'chain',
    chainId: 'chain-2',
    accessPathId: 'path-2',
    destinationScope: '',
    enabled: true,
    topology: []
  });
  return state;
}

test('domain suffix routes match root and subdomains', () => {
  for (const matchValue of ['.openai.com', '*.openai.com']) {
    assert.equal(routeMatches({ matchType: 'domain_suffix', matchValue }, 'https://openai.com'), true);
    assert.equal(routeMatches({ matchType: 'domain_suffix', matchValue }, 'https://api.openai.com'), true);
    assert.equal(routeMatches({ matchType: 'domain_suffix', matchValue }, 'https://notopenai.com'), false);
  }
});

test('route preview treats wildcard host entries as root plus subdomains', () => {
  for (const matchValue of ['.openai.com', '*.openai.com']) {
    assert.equal(routePreviewForHost(stateWithDomainSuffixRoute(matchValue), 'openai.com').mode, 'proxy');
    assert.equal(routePreviewForHost(stateWithDomainSuffixRoute(matchValue), 'api.openai.com').mode, 'proxy');
  }
});

test('pac rules use latest suffix routes and access path proxy targets', () => {
  const dotScript = buildPacScript(stateWithDomainSuffixRoute('.openai.com'));
  assert.match(dotScript, /"matchType":"domain_suffix"/);
  assert.match(dotScript, /"matchValue":"\.openai\.com"/);
  assert.match(dotScript, /"proxyTarget":"PROXY 127\.0\.0\.1:18080"/);

  const wildcardScript = buildPacScript(stateWithDomainSuffixRoute('*.openai.com'));
  assert.match(wildcardScript, /"matchValue":"\*\.openai\.com"/);
  assert.match(wildcardScript, /"proxyTarget":"PROXY 127\.0\.0\.1:18080"/);
});

test('parallel access paths route independently by route accessPathId', () => {
  const state = stateWithParallelAccessPaths();

  const oneRoute = routePreviewForHost(state, 'api.one.example');
  assert.equal(oneRoute.mode, 'proxy');
  assert.equal(oneRoute.accessPathId, 'path-1');

  const twoRoute = routePreviewForHost(state, 'api.two.example');
  assert.equal(twoRoute.mode, 'proxy');
  assert.equal(twoRoute.accessPathId, 'path-2');

  const script = buildPacScript(state);
  assert.match(script, /"accessPathId":"path-1","proxyTarget":"PROXY 127\.0\.0\.1:18080"/);
  assert.match(script, /"accessPathId":"path-2","proxyTarget":"PROXY 127\.0\.0\.1:18081"/);
});

test('disabling one access path does not disable other parallel paths', () => {
  const state = stateWithParallelAccessPaths(['path-2']);

  const oneRoute = routePreviewForHost(state, 'api.one.example');
  assert.equal(oneRoute.mode, 'proxy');
  assert.equal(oneRoute.accessPathId, 'path-1');

  const twoRoute = routePreviewForHost(state, 'api.two.example');
  assert.equal(twoRoute.mode, 'deny');
  assert.equal(twoRoute.denyReason, 'access_path_unavailable');

  const script = buildPacScript(state);
  assert.match(script, /"accessPathId":"path-1","proxyTarget":"PROXY 127\.0\.0\.1:18080"/);
  assert.match(script, /"accessPathId":"path-2","proxyTarget":""/);
});

test('disabling all access paths denies only chain routes', () => {
  const state = stateWithParallelAccessPaths(['path-1', 'path-2']);

  assert.equal(routePreviewForHost(state, 'api.one.example').mode, 'deny');
  assert.equal(routePreviewForHost(state, 'api.two.example').mode, 'deny');

  const summary = pacSummary(state);
  assert.equal(summary.enabledAccessPaths, 0);
  assert.equal(summary.proxyTarget, '');
  assert.equal(summary.proxyTargets, 0);
});

test('control-plane disabled access path remains unavailable', () => {
  const state = stateWithParallelAccessPaths();
  state.remote.accessPaths[0].enabled = false;

  const oneRoute = routePreviewForHost(state, 'api.one.example');
  assert.equal(oneRoute.mode, 'deny');
  assert.equal(oneRoute.denyReason, 'access_path_unavailable');

  const twoRoute = routePreviewForHost(state, 'api.two.example');
  assert.equal(twoRoute.mode, 'proxy');
  assert.equal(twoRoute.accessPathId, 'path-2');
});

test('unavailable access path health denies only its own route', () => {
  const state = stateWithParallelAccessPaths();
  state.remote.accessPaths[1].health = { status: 'unavailable', reason: 'probe_failed', checkedAt: '' };

  assert.equal(routePreviewForHost(state, 'api.one.example').mode, 'proxy');
  const twoRoute = routePreviewForHost(state, 'api.two.example');
  assert.equal(twoRoute.mode, 'deny');
  assert.equal(twoRoute.denyReason, 'access_path_unavailable');
});

test('local helper applies to every usable chain route without selecting one access path', () => {
  const state = stateWithParallelAccessPaths();
  state.localHelper = { enabled: true, scheme: 'SOCKS5', host: '127.0.0.1', port: 1080 };

  const script = buildPacScript(state);
  assert.match(script, /"accessPathId":"path-1","proxyTarget":"SOCKS5 127\.0\.0\.1:1080"/);
  assert.match(script, /"accessPathId":"path-2","proxyTarget":"SOCKS5 127\.0\.0\.1:1080"/);
});

test('direct and deny routes do not depend on access path switches', () => {
  const state = stateWithParallelAccessPaths(['path-1', 'path-2']);
  state.remote.routes.unshift({
    id: 'route-direct',
    priority: 0,
    matchType: 'domain_suffix',
    matchValue: '.direct.example',
    actionType: 'direct',
    chainId: '',
    accessPathId: '',
    destinationScope: '',
    enabled: true,
    topology: []
  }, {
    id: 'route-deny',
    priority: 0,
    matchType: 'domain_suffix',
    matchValue: '.deny.example',
    actionType: 'deny',
    chainId: '',
    accessPathId: '',
    destinationScope: '',
    enabled: true,
    topology: []
  });

  assert.equal(routePreviewForHost(state, 'api.direct.example').mode, 'direct');
  assert.equal(routePreviewForHost(state, 'api.deny.example').mode, 'deny');
});

test('multiple enabled proxy targets are preserved without a global active target', () => {
  const state = stateWithParallelAccessPaths();
  const summary = pacSummary(state);

  assert.equal(summary.enabledAccessPaths, 2);
  assert.equal(summary.proxyTarget, '');
  assert.equal(summary.proxyTargets, 2);
});

test('removing a disabled access path switch restores that path route', () => {
  const disabled = stateWithParallelAccessPaths(['path-2']);
  assert.equal(routePreviewForHost(disabled, 'api.two.example').mode, 'deny');

  const restored = stateWithParallelAccessPaths([]);
  const route = routePreviewForHost(restored, 'api.two.example');
  assert.equal(route.mode, 'proxy');
  assert.equal(route.accessPathId, 'path-2');
});

test('runProxyProbes always returns a Promise for skipped proxy states', async () => {
  const state = stateWithParallelAccessPaths();
  state.enabled = false;
  const result = runProxyProbes(state, 'https://api.one.example', routePreviewForHost(state, 'api.one.example'));

  assert.equal(typeof result.then, 'function');
  const probes = await result;
  assert.deepEqual(probes.map((item) => item.status), ['skipped', 'skipped', 'skipped', 'skipped', 'skipped']);
});
