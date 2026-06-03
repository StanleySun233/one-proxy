import { execFileSync } from 'node:child_process';
import { existsSync, readdirSync, readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../chrome-extension');
const manifestPath = path.join(root, 'manifest.json');
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));

if (manifest.manifest_version !== 3) {
  throw new Error('manifest_version_must_be_3');
}

if (!manifest.background) {
  throw new Error('missing_background');
}

if (manifest.background.type) {
  throw new Error('background_service_worker_must_be_classic');
}

if (!manifest.background.service_worker) {
  throw new Error('missing_background_service_worker');
}

const serviceWorkerPath = path.join(root, manifest.background.service_worker);
if (!existsSync(serviceWorkerPath)) {
  throw new Error(`missing_service_worker_file:${manifest.background.service_worker}`);
}

for (const script of manifest.content_scripts || []) {
  for (const js of script.js || []) {
    if (!existsSync(path.join(root, js))) {
      throw new Error(`missing_content_script_file:${js}`);
    }
  }
  for (const css of script.css || []) {
    if (!existsSync(path.join(root, css))) {
      throw new Error(`missing_content_style_file:${css}`);
    }
  }
}

const serviceWorkerSource = readFileSync(serviceWorkerPath, 'utf8');
if (/\bawait\b/.test(serviceWorkerSource)) {
  throw new Error(`service_worker_entry_contains_await:${manifest.background.service_worker}`);
}
if (/^\s*import\s/m.test(serviceWorkerSource) || /^\s*export\s/m.test(serviceWorkerSource)) {
  throw new Error(`service_worker_entry_contains_module_syntax:${manifest.background.service_worker}`);
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
  if (file.endsWith('.html') && /type=["']module["']/.test(readFileSync(file, 'utf8'))) {
    throw new Error(`runtime_html_contains_module_script:${path.relative(root, file)}`);
  }
  if (file.endsWith('.js')) {
    const relative = path.relative(root, file);
    if (/^background\//.test(relative) && relative !== manifest.background.service_worker) {
      throw new Error(`unexpected_background_runtime_file:${relative}`);
    }
    if (/^(background|shared|options|popup|content)\//.test(relative) && /\bawait\b/.test(readFileSync(file, 'utf8'))) {
      throw new Error(`runtime_file_contains_await:${relative}`);
    }
    if (/^(background|shared|options|popup|content)\//.test(relative) && (/^\s*import\s/m.test(readFileSync(file, 'utf8')) || /^\s*export\s/m.test(readFileSync(file, 'utf8')))) {
      throw new Error(`runtime_file_contains_module_syntax:${relative}`);
    }
    execFileSync(process.execPath, ['--check', file], { stdio: 'inherit' });
  }
}

console.log('chrome_extension_static_ok');
