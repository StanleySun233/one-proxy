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
        .then((page) => page.goto(`chrome-extension://${extensionId}/options/index.html`).then(() => page))
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

          console.log(`chrome_extension_service_worker_ok id=${result.id} version=${result.version}`);
        });
    });
  })
  .then(cleanup, (error) => cleanup().then(() => {
    throw error;
  }));
