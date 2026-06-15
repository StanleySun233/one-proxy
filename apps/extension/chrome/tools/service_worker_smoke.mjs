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
                accessToken: headers.get('X-One-Proxy-Access-Token') || '',
                tenantId: headers.get('X-One-Proxy-Tenant-ID') || '',
                refreshToken: headers.get('X-One-Proxy-Refresh-Token') || '',
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
                    account: { account: 'admin', mustRotatePassword: false },
                    policyRevision: 'rev-1',
                    fetchedAt: '2026-06-04T00:00:00Z',
                    proxyToken: 'proxy-token',
                    proxyTokenExpiresAt: '2026-07-04T00:00:00Z',
                    groups: [{
                      id: 'group-1',
                      name: 'Test group',
                      entryNodeId: 'node-1',
                      entryNodeName: 'Node 1',
                      proxyHost: '127.0.0.1',
                      proxyPort: 2988,
                      routes: [{
                        id: 'route-1',
                        priority: 1,
                        matchType: 'ip_cidr',
                        matchValue: '172.20.116.0/24',
                        actionType: 'chain',
                        chainId: 'chain-1',
                        topology: [{ id: 'node-1', name: 'Node 1', mode: 'edge' }]
                      }],
                      topology: [{ id: 'node-1', name: 'Node 1', mode: 'edge' }]
                    }]
                  }
                }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
              }
              if (String(url).includes('/api/proxy/extension/page/status')) {
                const parsed = new URL(String(url));
                const strict = parsed.searchParams.get('routeId') || parsed.searchParams.get('chainId');
                const data = strict
                  ? { status: 'unknown', latencyMs: 0, uploadBytes: 0, downloadBytes: 0, requestCount: 0, failureCount: 0, correlated: false }
                  : { status: 'ok', latencyMs: 42, uploadBytes: 1234, downloadBytes: 5678, requestCount: 2, failureCount: 0, correlated: true };
                return Promise.resolve(new Response(JSON.stringify({ code: 0, message: 'ok', data }), { status: 200, headers: { 'Content-Type': 'application/json' } }));
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
            if (messageResult.session.accessToken !== 'access-token' || messageResult.session.proxyToken !== 'proxy-token') {
              throw new Error('service_worker_login_state_missing_token');
            }
            return serviceWorker.evaluate(() => globalThis.__oneProxySmokeRequests);
          }).then((requests) => {
            const bootstrap = requests.find((request) => request.url.endsWith('/api/proxy/extension/bootstrap'));
            if (!bootstrap || bootstrap.accessToken !== 'access-token' || bootstrap.tenantId !== 'tenant-1') {
              throw new Error('service_worker_bootstrap_missing_one_proxy_headers');
            }
            return optionsPage.evaluate(() => chrome.runtime.sendMessage({ type: 'sync-remote-config' }));
          }).then((syncResult) => {
            if (syncResult.error) {
              throw new Error(syncResult.error);
            }
            return serviceWorker.evaluate(() => globalThis.__oneProxySmokeRequests);
          }).then((requests) => {
            const bootstraps = requests.filter((request) => request.url.endsWith('/api/proxy/extension/bootstrap'));
            const latestBootstrap = bootstraps[bootstraps.length - 1];
            if (!latestBootstrap || latestBootstrap.accessToken !== 'access-token' || latestBootstrap.tenantId !== 'tenant-1') {
              throw new Error('service_worker_sync_missing_one_proxy_headers');
            }
            return optionsPage.evaluate(() => chrome.runtime.sendMessage({ type: 'set-enabled', enabled: true }));
          }).then((enableResult) => {
            if (enableResult.error || !enableResult.enabled) {
              throw new Error(enableResult.error || 'service_worker_enable_failed');
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
          }).then((statusResult) => {
            if (statusResult.error) {
              throw new Error(statusResult.error);
            }
            if (statusResult.status !== 'ok' || statusResult.io.uploadBytes !== 1234 || statusResult.io.downloadBytes !== 5678 || statusResult.page.requestCount !== 2) {
              throw new Error('service_worker_status_bubble_fallback_failed');
            }
            return serviceWorker.evaluate(() => globalThis.__oneProxySmokeRequests);
          }).then((requests) => {
            const pageStatusRequests = requests.filter((request) => request.url.includes('/api/proxy/extension/page/status'));
            if (pageStatusRequests.length < 2 || !pageStatusRequests[0].url.includes('routeId=route-1') || pageStatusRequests[1].url.includes('routeId=')) {
              throw new Error('service_worker_status_bubble_host_fallback_missing');
            }
            console.log(`chrome_extension_service_worker_ok id=${result.id} version=${result.version}`);
          });
        });
    });
  })
  .then(cleanup, (error) => cleanup().then(() => {
    throw error;
  }));
