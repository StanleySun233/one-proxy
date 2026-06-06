import * as fs from 'node:fs/promises';
import * as fsSync from 'node:fs';
import * as net from 'node:net';
import * as os from 'node:os';
import * as path from 'node:path';
export const loopbackHost = '127.0.0.1';
const portRangeStart = 10000;
const portRangeEnd = 60999;
const commonPorts = new Set([
    20, 21, 22, 25, 53, 67, 68, 80, 110, 123, 143, 161, 389, 443, 445, 465, 587, 631, 993, 995,
    1433, 1521, 2049, 2375, 2376, 3000, 3306, 3389, 5000, 5432, 5601, 5672, 5900, 6379, 8000,
    8080, 8443, 9000, 9200, 9300, 11211, 27017
]);
export function oneProxyHome() {
    return process.env.ONEPROXY_HOME || path.join(os.homedir(), '.oneproxy');
}
export function profilesFile() {
    return path.join(oneProxyHome(), 'profiles.json');
}
export function activeProfileName() {
    const envProfile = process.env.ONEPROXY_PROFILE || process.env.ONEPROXY_ACTIVE_PROFILE;
    if (envProfile) {
        return profileKey(envProfile);
    }
    if (fsSync.existsSync(profilesFile())) {
        const index = JSON.parse(fsSync.readFileSync(profilesFile(), 'utf8'));
        if (index.activeProfile) {
            return profileKey(index.activeProfile);
        }
    }
    return 'default';
}
export function profileRoot(name = activeProfileName()) {
    return path.join(oneProxyHome(), 'profiles', profileKey(name));
}
export function storageFile(name) {
    const names = {
        config: 'config.json',
        state: 'state.json',
        tokens: 'tokens.json',
        daemon: 'daemon.json',
        log: 'onep.log'
    };
    return path.join(profileRoot(), names[name]);
}
export async function ensureStorageRoot() {
    await fs.mkdir(oneProxyHome(), { recursive: true, mode: 0o700 });
}
export async function ensureProfileRoot() {
    await fs.mkdir(profileRoot(), { recursive: true, mode: 0o700 });
}
async function readJson(file) {
    try {
        return JSON.parse(await fs.readFile(file, 'utf8'));
    }
    catch (error) {
        if (error.code === 'ENOENT') {
            return null;
        }
        throw error;
    }
}
async function writeJson(file, value, mode = 0o600) {
    await fs.mkdir(path.dirname(file), { recursive: true, mode: 0o700 });
    await fs.writeFile(file, `${JSON.stringify(value, null, 2)}\n`, { mode });
    if (process.platform !== 'win32') {
        await fs.chmod(file, mode);
    }
}
function uniqueHosts(items) {
    return [...new Set((items ?? []).map((item) => item.trim().toLowerCase()).filter(Boolean))].sort();
}
export function defaultConfig() {
    return {
        schemaVersion: 1,
        profileName: activeProfileName(),
        overrides: { direct: [], proxy: [] }
    };
}
export async function readConfig() {
    const config = await readJson(storageFile('config'));
    return {
        ...defaultConfig(),
        ...config,
        overrides: {
            direct: uniqueHosts(config?.overrides?.direct),
            proxy: uniqueHosts(config?.overrides?.proxy)
        }
    };
}
export async function writeConfig(config) {
    await writeJson(storageFile('config'), {
        ...config,
        schemaVersion: 1,
        profileName: config.profileName || activeProfileName(),
        overrides: {
            direct: uniqueHosts(config.overrides.direct),
            proxy: uniqueHosts(config.overrides.proxy)
        }
    });
}
export async function readProfilesIndex() {
    const index = await readJson(profilesFile());
    return {
        schemaVersion: 1,
        activeProfile: index?.activeProfile,
        profiles: index?.profiles ?? {}
    };
}
export async function writeProfilesIndex(index) {
    await writeJson(profilesFile(), {
        schemaVersion: 1,
        activeProfile: index.activeProfile,
        profiles: index.profiles
    });
}
export function profileKey(name) {
    const value = name.trim().toLowerCase();
    if (!/^[a-z0-9][a-z0-9._-]*$/.test(value)) {
        throw Object.assign(new Error(`Invalid profile name: ${name}`), { code: 'SYNTAX_ERROR', exitCode: 2 });
    }
    return value;
}
export async function addProfile(name, controlPlaneUrl) {
    const key = profileKey(name);
    const index = await readProfilesIndex();
    const profile = { name: key, controlPlaneUrl };
    index.profiles[key] = profile;
    index.activeProfile = key;
    await writeProfilesIndex(index);
    await writeConfig({ ...(await readConfig()), profileName: key, controlPlaneUrl });
    return profile;
}
export async function useProfile(name) {
    const key = profileKey(name);
    const index = await readProfilesIndex();
    const profile = index.profiles[key];
    if (!profile) {
        throw Object.assign(new Error(`Profile not found: ${name}`), { code: 'PROFILE_REQUIRED' });
    }
    index.activeProfile = key;
    await writeProfilesIndex(index);
    process.env.ONEPROXY_PROFILE = key;
    await writeConfig({ ...(await readConfig()), profileName: key, controlPlaneUrl: profile.controlPlaneUrl });
    return profile;
}
export async function readTokens() {
    return readJson(storageFile('tokens'));
}
export async function writeTokens(tokens) {
    await writeJson(storageFile('tokens'), { ...tokens, schemaVersion: 1 });
}
export async function clearTokens() {
    try {
        await fs.rm(storageFile('tokens'));
    }
    catch (error) {
        if (error.code !== 'ENOENT') {
            throw error;
        }
    }
}
export async function readState() {
    const state = await readJson(storageFile('state'));
    return {
        schemaVersion: 1,
        ...state,
        routeGroups: state?.routeGroups ?? []
    };
}
export async function writeState(state) {
    await writeJson(storageFile('state'), { ...state, schemaVersion: 1 });
}
export async function readDaemonMetadata() {
    return readJson(storageFile('daemon'));
}
export async function writeDaemonMetadata(metadata) {
    await writeJson(storageFile('daemon'), { ...metadata, schemaVersion: 1 });
}
export async function appendLog(message) {
    await ensureStorageRoot();
    await fs.appendFile(storageFile('log'), `${new Date().toISOString()} ${message}\n`, { mode: 0o600 });
}
function isExcludedPort(port) {
    return commonPorts.has(port) || port < portRangeStart || port > portRangeEnd;
}
export async function isLoopbackPortAvailable(port) {
    if (isExcludedPort(port)) {
        return false;
    }
    const server = net.createServer();
    return new Promise((resolve) => {
        server.once('error', () => resolve(false));
        server.listen(port, loopbackHost, () => {
            server.close(() => resolve(true));
        });
    });
}
export async function scanAvailableProxyPortPairs() {
    const pairs = [];
    for (let port = portRangeStart; port < portRangeEnd; port += 1) {
        if (isExcludedPort(port) || isExcludedPort(port + 1)) {
            continue;
        }
        const [httpAvailable, httpsAvailable] = await Promise.all([
            isLoopbackPortAvailable(port),
            isLoopbackPortAvailable(port + 1)
        ]);
        if (httpAvailable && httpsAvailable) {
            pairs.push([port, port + 1]);
        }
    }
    return pairs;
}
export function processIsRunning(pid) {
    try {
        process.kill(pid, 0);
        return true;
    }
    catch {
        return false;
    }
}
//# sourceMappingURL=storage.js.map