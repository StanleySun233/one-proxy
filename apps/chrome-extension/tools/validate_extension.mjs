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

if (/\bawait\b/.test(readFileSync(serviceWorkerPath, 'utf8'))) {
  throw new Error(`service_worker_entry_contains_await:${manifest.background.service_worker}`);
}

function stripLineForDepth(line) {
  return line
    .replace(/'([^'\\]|\\.)*'/g, "''")
    .replace(/"([^"\\]|\\.)*"/g, '""')
    .replace(/`([^`\\]|\\.)*`/g, '``')
    .replace(/\/\/.*$/, '');
}

function assertNoTopLevelAwait(file) {
  const lines = readFileSync(file, 'utf8').split('\n');
  let depth = 0;
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (depth === 0 && /\bawait\b/.test(stripLineForDepth(line))) {
      throw new Error(`top_level_await:${path.relative(root, file)}:${index + 1}`);
    }
    const stripped = stripLineForDepth(line);
    for (const char of stripped) {
      if (char === '{') {
        depth += 1;
      } else if (char === '}') {
        depth = Math.max(0, depth - 1);
      }
    }
  }
}

function importedModules(file) {
  const dir = path.dirname(file);
  const source = readFileSync(file, 'utf8');
  const imports = [...source.matchAll(/^\s*import\s+(?:[^'"]+\s+from\s+)?['"](.+?)['"]/gm)]
    .map((match) => match[1])
    .filter((specifier) => specifier.startsWith('.'))
    .map((specifier) => path.resolve(dir, specifier.endsWith('.js') ? specifier : `${specifier}.js`));
  return imports.filter((item) => existsSync(item));
}

function serviceWorkerGraph(entry) {
  const seen = new Set();
  const queue = [entry];
  while (queue.length) {
    const file = queue.shift();
    if (seen.has(file)) {
      continue;
    }
    seen.add(file);
    queue.push(...importedModules(file));
  }
  return [...seen];
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

for (const file of serviceWorkerGraph(serviceWorkerPath)) {
  assertNoTopLevelAwait(file);
}

for (const file of files(root)) {
  if (file.endsWith('.json')) {
    JSON.parse(readFileSync(file, 'utf8'));
  }
  if (file.endsWith('.js')) {
    const relative = path.relative(root, file);
    if (/^(background|shared|options|popup)\//.test(relative) && /\bawait\b/.test(readFileSync(file, 'utf8'))) {
      throw new Error(`runtime_file_contains_await:${relative}`);
    }
    execFileSync(process.execPath, ['--check', file], { stdio: 'inherit' });
  }
}

console.log('chrome_extension_static_ok');
