import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const extensionPath = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const manifest = JSON.parse(readFileSync(path.join(extensionPath, 'manifest.json'), 'utf8'));
const serviceWorkerPath = `/${manifest.background.service_worker}`;
const userDataDir = mkdtempSync(path.join(os.tmpdir(), 'oneproxy-extension-'));

let context;
let optionsPage;
function cleanup() {
  const close = context ? context.close() : Promise.resolve();
  return close.finally(() => {
    rmSync(userDataDir, { recursive: true, force: true });
  });
}

chromium.launchPersistentContext(userDataDir, {
    headless: false,
    args: [
      `--disable-extensions-except=${extensionPath}`,
      `--load-extension=${extensionPath}`
    ]
  })
  .then((createdContext) => {
    context = createdContext;

    const serviceWorkerReady = context.serviceWorkers()[0]
      ? Promise.resolve(context.serviceWorkers()[0])
      : context.waitForEvent('serviceworker', { timeout: 15000 });

    return serviceWorkerReady.then((serviceWorker) => {
      if (!serviceWorker.url().endsWith(serviceWorkerPath)) {
        throw new Error(`unexpected_service_worker:${serviceWorker.url()}`);
      }

      const extensionId = new URL(serviceWorker.url()).host;
      return context.newPage()
        .then((page) => {
          optionsPage = page;
          return page.goto(`chrome-extension://${extensionId}/options/index.html`).then(() => page);
        })
        .then((page) => page.evaluate(() => new Promise((resolve, reject) => {
          const manifest = chrome.runtime.getManifest();
          chrome.storage.local.set({ oneProxyServiceWorkerSmoke: 'ok' }, () => {
            if (chrome.runtime.lastError) {
              reject(new Error(chrome.runtime.lastError.message));
              return;
            }
            chrome.storage.local.get('oneProxyServiceWorkerSmoke', (stored) => {
              if (chrome.runtime.lastError) {
                reject(new Error(chrome.runtime.lastError.message));
                return;
              }
              resolve({
                id: chrome.runtime.id,
                name: manifest.name,
                version: manifest.version,
                stored: stored.oneProxyServiceWorkerSmoke
              });
            });
          });
        })))
        .then((result) => {
          if (result.id !== extensionId || result.name !== 'One Proxy' || result.stored !== 'ok') {
            throw new Error('service_worker_runtime_check_failed');
          }

          return serviceWorker.evaluate(() => {
            const originalFetch = globalThis.fetch;
            const requests = [];
            globalThis.fetch = (url, options = {}) => {
              const headers = new Headers(options.headers || {});
              requests.push({
                url: String(url),
                accessTokenPresent: Boolean(headers.get('X-One-Proxy-Access-Token')),
                tenantIdPresent: Boolean(headers.get('X-One-Proxy-Tenant-ID')),
                refreshTokenPresent: Boolean(headers.get('X-One-Proxy-Refresh-Token')),
                body: options.body || ''
              });
              if (String(url).endsWith('/api/auth/login')) {
                return Promise.resolve(new Response(JSON.stringify({
                  code: 0,
                  message: 'ok',
                  data: {
                    account: { account: 'admin', mustRotatePassword: false },
                    accessToken: 'access-token',
                    refreshToken: 'refresh-token',
                    expiresAt: '2026-07-04T00:00:00Z',
                    tenantMemberships: [{ tenantId: 'tenant-1', tenantName: 'Default' }],
                    activeTenantId: 'tenant-1'
                  }
                }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
              }
              if (String(url).endsWith('/api/proxy/extension/bootstrap')) {
                return Promise.resolve(new Response(JSON.stringify({
                  code: 0,
                  message: 'ok',
                  data: {
                    schemaVersion: 'v2.1.0',
                    account: { account: 'admin', mustRotatePassword: false },
                    tenant: { tenantId: 'tenant-1', tenantName: 'Default', role: 'owner', joinedAt: '2026-06-04T00:00:00Z' },
                    policyRevision: 'rev-1',
                    fetchedAt: '2026-06-04T00:00:00Z',
                    proxyToken: 'proxy-token',
                    proxyTokenExpiresAt: '2026-07-04T00:00:00Z',
                    nodes: [{
                      id: 'node-1',
                      name: 'Node 1',
                      mode: 'edge',
                      scopeKey: 'tenant-1',
                      parentNodeId: '',
                      enabled: true,
                      status: 'online',
                      publicHost: '127.0.0.1',
                      publicPort: 2988
                    }],
                    accessPaths: [{
                      id: 'path-1',
                      name: 'Test access path',
                      chainId: 'chain-1',
                      mode: 'forward',
                      protocol: 'http',
                      serviceType: 'http_forward_proxy',
                      targetNodeId: 'node-1',
                      entryNodeId: 'node-1',
                      relayNodeIds: [],
                      listenHost: '127.0.0.1',
                      listenPort: 2988,
                      targetProtocol: 'http',
                      targetHost: '',
                      targetPort: 0,
                      targetSni: '',
                      tlsMode: '',
                      authMode: 'proxy_token',
                      enabled: true,
                      options: {},
                      topology: [{
                        nodeId: 'node-1',
                        nodeName: 'Node 1',
                        mode: 'edge',
                        scopeKey: 'tenant-1',
                        publicHost: '127.0.0.1',
                        publicPort: 2988,
                        transport: 'public_http'
                      }],
                      health: { status: 'available', reason: '', checkedAt: '2026-06-04T00:00:00Z' }
                    }],
                    routes: [{
                      id: 'route-1',
                      priority: 1,
                      matchType: 'ip_cidr',
                      matchValue: '172.20.116.0/24',
                      actionType: 'chain',
                      chainId: 'chain-1',
                      accessPathId: 'path-1',
                      destinationScope: '',
                      enabled: true,
                      topology: [{
                        nodeId: 'node-1',
                        nodeName: 'Node 1',
                        mode: 'edge',
                        scopeKey: 'tenant-1',
                        publicHost: '127.0.0.1',
                        publicPort: 2988,
                        transport: 'public_http'
                      }]
                    }],
                    routeEvaluation: {
                      defaultClientMode: 'direct',
                      defaultNodeMode: 'deny',
                      ruleOrder: 'priority_asc_then_id_asc',
                      noMatchNodeDenyReason: 'route_not_found',
                      supportedMatchTypes: ['domain', 'domain_suffix', 'ip', 'ip_cidr', 'protocol', 'default'],
                      supportedActions: ['chain', 'direct', 'deny']
                    }
                  }
                }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
              }
              if (String(url).includes('/api/proxy/extension/page/status')) {
                const parsed = new URL(String(url));
                const strict = parsed.searchParams.get('routeId') || parsed.searchParams.get('chainId');
                const host = parsed.searchParams.get('host') || '';
                let data = { status: 'ok', latencyMs: 42, uploadBytes: 1234, downloadBytes: 5678, requestCount: 2, failureCount: 0, correlated: true };
                if (strict || host === '172.20.116.6' || host === '172.20.116.7') {
                  data = {
                    status: 'unknown',
                    latencyMs: 0,
                    uploadBytes: 0,
                    downloadBytes: 0,
                    requestCount: 0,
                    failureCount: 0,
                    correlated: false,
                    linkTimings: [{ fromNodeId: 'node-1', toNodeId: 'target', roundTripMs: 23, sampleTsMs: Date.now(), count: 1 }]
                  };
                }
                return Promise.resolve(new Response(JSON.stringify({ code: 0, message: 'ok', data }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
              }
              if (String(url).endsWith('/healthz')) {
                return Promise.resolve(new Response(JSON.stringify({ status: 'ok' }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
              }
              if (String(url).includes('/api/control/relay/probe')) {
                return Promise.resolve(new Response(JSON.stringify({
                  pathTimings: [{ fromNodeId: 'node-1', toNodeId: 'target', roundTripMs: 23, sampleTsMs: Date.now(), count: 1 }]
                }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
              }
              if (String(url).endsWith('/api/auth/refresh')) {
                return Promise.resolve(new Response(JSON.stringify({
                  code: 0,
                  message: 'ok',
                  data: {
                    account: { account: 'admin', mustRotatePassword: false },
                    accessToken: 'access-token-2',
                    refreshToken: 'refresh-token-2',
                    expiresAt: '2026-07-04T00:00:00Z',
                    tenantMemberships: [{ tenantId: 'tenant-1', tenantName: 'Default' }],
                    activeTenantId: 'tenant-1'
                  }
                }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
              }
              return originalFetch(url, options);
            };
            globalThis.__oneProxySmokeRequests = requests;
          }).then(() => optionsPage.evaluate(() => chrome.runtime.sendMessage({
            type: 'login',
            controlPlaneUrl: 'https://panel.oneproxy.test',
            account: 'admin',
            password: 'secret'
          }))).then((messageResult) => {
            if (messageResult.error) {
              throw new Error(messageResult.error);
            }
            if (!messageResult.session.authenticated || !messageResult.session.proxyTokenAvailable) {
              throw new Error('service_worker_login_state_missing_token');
            }
            return serviceWorker.evaluate(() => globalThis.__oneProxySmokeRequests);
          }).then((requests) => {
            const bootstrap = requests.find((request) => request.url.endsWith('/api/proxy/extension/bootstrap'));
            if (!bootstrap || !bootstrap.accessTokenPresent || !bootstrap.tenantIdPresent) {
              throw new Error('service_worker_bootstrap_missing_one_proxy_headers');
            }
            return optionsPage.evaluate(() => chrome.runtime.sendMessage({ type: 'sync-remote-config' }));
          }).then((syncResult) => {
            if (syncResult.error) {
              throw new Error(syncResult.error);
            }
            if (!syncResult.remote || syncResult.remote.accessPaths.length !== 1 || syncResult.remote.routes.length !== 1) {
              throw new Error('service_worker_sync_missing_access_paths_or_routes');
            }
            if (!syncResult.activeAccessPath || syncResult.activeAccessPath.id !== 'path-1') {
              throw new Error('service_worker_sync_missing_active_access_path');
            }
            return optionsPage.evaluate(() => chrome.runtime.sendMessage({ type: 'get-diagnostic-logs' }));
          }).then((logs) => {
            const synced = [...logs].reverse().find((entry) => entry.event === 'remote_config_synced');
            if (!synced || synced.details.accessPaths !== 1 || synced.details.routes !== 1 || Object.hasOwn(synced.details, 'groups')) {
              throw new Error('service_worker_sync_log_contract_invalid');
            }
            return serviceWorker.evaluate(() => globalThis.__oneProxySmokeRequests);
          }).then((requests) => {
            const bootstraps = requests.filter((request) => request.url.endsWith('/api/proxy/extension/bootstrap'));
            const latestBootstrap = bootstraps[bootstraps.length - 1];
            if (!latestBootstrap || !latestBootstrap.accessTokenPresent || !latestBootstrap.tenantIdPresent) {
              throw new Error('service_worker_sync_missing_one_proxy_headers');
            }
            return optionsPage.evaluate(() => chrome.runtime.sendMessage({ type: 'set-enabled', enabled: true }));
          }).then((enableResult) => {
            if (enableResult.error || !enableResult.enabled) {
              throw new Error(enableResult.error || 'service_worker_enable_failed');
            }
            return optionsPage.evaluate(() => chrome.runtime.sendMessage({ type: 'get-diagnostic-logs' }));
          }).then((logs) => {
            const applied = [...logs].reverse().find((entry) => entry.event === 'proxy_applied' && entry.details.enabled === true);
            if (!applied || applied.details.activeAccessPathId !== 'path-1' || applied.details.accessPaths !== 1 || applied.details.routes !== 1 || applied.details.enabledRoutes !== 1 || applied.details.chainRoutes !== 1 || applied.details.proxyTarget !== 'PROXY 127.0.0.1:2988' || Object.hasOwn(applied.details, 'groups')) {
              throw new Error('service_worker_proxy_applied_contract_invalid');
            }
            return optionsPage.evaluate(() => chrome.runtime.sendMessage({
              type: 'set-local-helper',
              enabled: true,
              scheme: 'SOCKS5',
              host: '127.0.0.1',
              port: 1080
            }));
          }).then((helperResult) => {
            if (helperResult.error || !helperResult.localHelper.enabled) {
              throw new Error(helperResult.error || 'service_worker_local_helper_failed');
            }
            return optionsPage.evaluate(() => chrome.runtime.sendMessage({
              type: 'status-bubble-page-status',
              url: 'http://172.20.116.5/'
            }));
          }).then((blockedStatusResult) => {
            if (!blockedStatusResult || blockedStatusResult.error !== 'message_not_allowed') {
              throw new Error('service_worker_status_bubble_extension_page_not_blocked');
            }
            return context.route('http://172.20.116.5/**', (route) => route.fulfill({
              status: 200,
              contentType: 'text/html',
              body: '<!doctype html><title>One Proxy smoke</title><main>ok</main>'
            }));
          }).then(() => context.newPage()).then((contentPage) => contentPage.goto('http://172.20.116.5/')
            .then(() => contentPage.waitForSelector('#one-proxy-status-root.opsb-green', { timeout: 10000 }))
            .then(() => contentPage.click('#one-proxy-status-root .opsb-icon'))
            .then(() => contentPage.waitForSelector('#one-proxy-status-root .opsb-hop-latency', { timeout: 10000 }))
            .then(() => contentPage.evaluate(() => {
              const root = document.getElementById('one-proxy-status-root');
              return {
                className: root ? root.className : '',
                title: root && root.querySelector('.opsb-icon') ? root.querySelector('.opsb-icon').title : '',
                hopLatencies: root ? [...root.querySelectorAll('.opsb-hop-latency')].map((node) => node.textContent || '') : []
              };
            }))
            .then((bubbleResult) => ({ bubbleResult, result })));
          }).then(({ bubbleResult, result }) => {
            if (!String(bubbleResult.className || '').includes('opsb-green') || !bubbleResult.title) {
              throw new Error('service_worker_status_bubble_content_script_failed');
            }
            if (!bubbleResult.hopLatencies[0] || bubbleResult.hopLatencies[0].includes('-')) {
              throw new Error('service_worker_status_bubble_missing_entry_latency');
            }
            return serviceWorker.evaluate(() => globalThis.__oneProxySmokeRequests)
              .then((requests) => ({ requests, result }));
          }).then(({ requests, result }) => {
            const pageStatusRequests = requests.filter((request) => request.url.includes('/api/proxy/extension/page/status'));
            if (pageStatusRequests.length < 2 || !pageStatusRequests[0].url.includes('routeId=route-1') || pageStatusRequests[1].url.includes('routeId=')) {
              throw new Error('service_worker_status_bubble_host_fallback_missing');
            }
            console.log(`chrome_extension_service_worker_ok id=${result.id} version=${result.version}`);
          });
        });
    })
  .then(cleanup, (error) => cleanup().then(() => {
    throw error;
  }));
