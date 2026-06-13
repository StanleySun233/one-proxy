import assert from 'node:assert/strict';
import test from 'node:test';

import { buildPacScript } from './background-source/pac.js';
import { routeMatches, routePreviewForHost } from './background-source/routing.js';

function stateWithProxyHost(host) {
  return {
    enabled: true,
    controlPlaneUrl: 'https://panel.example.com',
    localOverrides: { directHosts: [], proxyHosts: [] },
    localHelper: {},
    selection: { activeGroupId: 'group-1' },
    remote: {
      groups: [
        {
          id: 'group-1',
          proxyHost: '127.0.0.1',
          proxyPort: 18080,
          proxyScheme: 'PROXY',
          proxyHosts: [host],
          directHosts: [],
          proxyCidrs: [],
          directCidrs: [],
          routes: [],
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
  for (const host of ['.openai.com', '*.openai.com']) {
    assert.equal(routePreviewForHost(stateWithProxyHost(host), 'openai.com').mode, 'proxy');
    assert.equal(routePreviewForHost(stateWithProxyHost(host), 'api.openai.com').mode, 'proxy');
  }
});

test('pac host entries include root domain for suffix patterns', () => {
  const dotScript = buildPacScript(stateWithProxyHost('.openai.com'));
  assert.match(dotScript, /const proxyHosts = \["openai\.com","\*\.openai\.com"\]/);

  const wildcardScript = buildPacScript(stateWithProxyHost('*.openai.com'));
  assert.match(wildcardScript, /const proxyHosts = \["openai\.com","\*\.openai\.com"\]/);
});
