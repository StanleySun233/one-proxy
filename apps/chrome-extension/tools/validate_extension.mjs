import { execFileSync } from 'node:child_process';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const manifestPath = path.join(root, 'manifest.json');
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));

if (manifest.manifest_version !== 3) {
  throw new Error('manifest_version_must_be_3');
}

if (!manifest.background || manifest.background.type !== 'module') {
  throw new Error('background_service_worker_must_be_module');
}

if (!manifest.background.service_worker) {
  throw new Error('missing_background_service_worker');
}

const serviceWorkerPath = path.join(root, manifest.background.service_worker);
if (!existsSync(serviceWorkerPath)) {
  throw new Error(`missing_service_worker_file:${manifest.background.service_worker}`);
}

function files(dir) {
  const result = [];
  for (const name of readdirSync(dir)) {
    const item = path.join(dir, name);
    if (statSync(item).isDirectory()) {
      result.push(...files(item));
    } else {
      result.push(item);
    }
  }
  return result;
}

for (const file of files(root)) {
  if (file.endsWith('.json')) {
    JSON.parse(readFileSync(file, 'utf8'));
  }
  if (file.endsWith('.js')) {
    execFileSync(process.execPath, ['--check', file], { stdio: 'inherit' });
  }
}

console.log('chrome_extension_static_ok');
