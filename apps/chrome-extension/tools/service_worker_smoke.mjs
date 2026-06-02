import { mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const extensionPath = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const userDataDir = mkdtempSync(path.join(os.tmpdir(), 'oneproxy-extension-'));

let context;
try {
  context = await chromium.launchPersistentContext(userDataDir, {
    headless: false,
    args: [
      `--disable-extensions-except=${extensionPath}`,
      `--load-extension=${extensionPath}`
    ]
  });

  let serviceWorker = context.serviceWorkers()[0];
  if (!serviceWorker) {
    serviceWorker = await context.waitForEvent('serviceworker', { timeout: 15000 });
  }

  if (!serviceWorker.url().endsWith('/background/index.js')) {
    throw new Error(`unexpected_service_worker:${serviceWorker.url()}`);
  }

  const extensionId = new URL(serviceWorker.url()).host;
  const page = await context.newPage();
  await page.goto(`chrome-extension://${extensionId}/options/index.html`);

  const result = await page.evaluate(async () => {
    const manifest = chrome.runtime.getManifest();
    await chrome.storage.local.set({ oneProxyServiceWorkerSmoke: 'ok' });
    const stored = await chrome.storage.local.get('oneProxyServiceWorkerSmoke');
    return {
      id: chrome.runtime.id,
      name: manifest.name,
      version: manifest.version,
      stored: stored.oneProxyServiceWorkerSmoke
    };
  });

  if (result.id !== extensionId || result.name !== 'One Proxy' || result.stored !== 'ok') {
    throw new Error('service_worker_runtime_check_failed');
  }

  console.log(`chrome_extension_service_worker_ok id=${result.id} version=${result.version}`);
} finally {
  if (context) {
    await context.close();
  }
  rmSync(userDataDir, { recursive: true, force: true });
}
