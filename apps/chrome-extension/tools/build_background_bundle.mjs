import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const entry = path.join(root, 'background/index.js');
const output = path.join(root, 'background/service-worker.js');

function importsOf(file) {
  const dir = path.dirname(file);
  const source = readFileSync(file, 'utf8');
  return [...source.matchAll(/^\s*import\s+(?:[^'"]+\s+from\s+)?['"](.+?)['"];?\s*$/gm)]
    .map((match) => match[1])
    .filter((specifier) => specifier.startsWith('.'))
    .map((specifier) => path.resolve(dir, specifier.endsWith('.js') ? specifier : `${specifier}.js`));
}

function stripModuleSyntax(file) {
  return readFileSync(file, 'utf8')
    .replace(/^\s*import\s+(?:[^'"]+\s+from\s+)?['"].+?['"];?\s*$/gm, '')
    .replace(/^export\s+const\s+/gm, 'const ')
    .replace(/^export\s+function\s+/gm, 'function ');
}

const seen = new Set();
const ordered = [];

function visit(file) {
  if (seen.has(file)) {
    return;
  }
  seen.add(file);
  for (const imported of importsOf(file)) {
    visit(imported);
  }
  ordered.push(file);
}

visit(entry);

const bundle = ordered
  .map((file) => stripModuleSyntax(file).trim())
  .filter(Boolean)
  .join('\n\n');

mkdirSync(path.dirname(output), { recursive: true });
writeFileSync(output, `${bundle}\n`);
