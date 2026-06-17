import assert from 'node:assert/strict';
import test from 'node:test';

import { buildPacScript } from '../tools/background-source/pac.js';
import { routeMatches, routePreviewForHost } from '../tools/background-source/routing.js';

function stateWithDomainSuffixRoute(matchValue) {
  return {
    enabled: true,
    controlPlaneUrl: 'https://panel.example.com',
    localOverrides: { directHosts: [], proxyHosts: [] },
    localHelper: {},
    selection: { activeAccessPathId: 'path-1' },
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
