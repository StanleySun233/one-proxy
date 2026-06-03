import { copyFileSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const toolsRoot = path.dirname(fileURLToPath(import.meta.url));
const extensionRoot = path.resolve(toolsRoot, '../chrome-extension');
const backgroundEntry = path.join(toolsRoot, 'background-source/index.js');
const backgroundOutput = path.join(extensionRoot, 'background/one-proxy-worker.js');

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

visit(backgroundEntry);

const backgroundBundle = ordered
  .map((file) => stripModuleSyntax(file).trim())
  .filter(Boolean)
  .join('\n\n');

mkdirSync(path.dirname(backgroundOutput), { recursive: true });
writeFileSync(backgroundOutput, `${backgroundBundle}\n`);

function writePageBundle(sourceDir, output) {
  const files = [
    path.join(toolsRoot, 'page-source/shared/locale.js'),
    path.join(toolsRoot, 'page-source/shared/theme.js'),
    path.join(toolsRoot, `page-source/${sourceDir}/index.js`)
  ];
  const bundle = files
    .map((file) => stripModuleSyntax(file).trim())
    .filter(Boolean)
    .join('\n\n');
  mkdirSync(path.dirname(output), { recursive: true });
  writeFileSync(output, `${bundle}\n`);
}

writePageBundle('popup', path.join(extensionRoot, 'popup/runtime.js'));
writePageBundle('options', path.join(extensionRoot, 'options/runtime.js'));

function copyContentAsset(name) {
  const source = path.join(toolsRoot, 'content-source', name);
  const output = path.join(extensionRoot, 'content', name);
  mkdirSync(path.dirname(output), { recursive: true });
  copyFileSync(source, output);
}

copyContentAsset('status-bubble.js');
copyContentAsset('status-bubble.css');
