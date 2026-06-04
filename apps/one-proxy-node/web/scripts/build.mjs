import { copyFile, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const dist = join(root, "dist");
const assets = join(dist, "assets");
const packageJson = JSON.parse(await readFile(join(root, "package.json"), "utf8"));
const builtAt = new Date().toISOString();

await rm(dist, { recursive: true, force: true });
await mkdir(assets, { recursive: true });
await copyFile(join(root, "index.html"), join(dist, "index.html"));
await copyFile(join(root, "src", "main.js"), join(assets, "main.js"));
await copyFile(join(root, "src", "styles.css"), join(assets, "styles.css"));
await writeFile(join(dist, "build-meta.json"), JSON.stringify({
  name: packageJson.name,
  version: packageJson.version,
  builtAt,
  staticOutput: packageJson.oneProxyNodeConsole.staticOutput,
  routes: packageJson.oneProxyNodeConsole.routes
}, null, 2));
