import { spawn } from 'node:child_process';
import { emitKeypressEvents } from 'node:readline';
import { readConfig, writeConfig } from "./storage.js";
const packageName = '@stanleysun233/oneproxy-cli';
const latestUrl = `https://registry.npmjs.org/${encodeURIComponent(packageName)}/latest`;
const timeoutMs = 1500;
export async function maybeHandleCliUpdate(command, context, currentVersion) {
    if (!shouldOfferUpdate(command, context)) {
        return false;
    }
    const latestVersion = await latestCliVersion().catch(() => '');
    if (!latestVersion || compareVersions(latestVersion, currentVersion) <= 0) {
        return false;
    }
    const config = await readConfig();
    if (config.ignoredCliVersion === latestVersion) {
        return false;
    }
    const choice = await promptUpdateChoice(currentVersion, latestVersion);
    if (choice === 'ignore') {
        await writeConfig({ ...config, ignoredCliVersion: latestVersion });
        return false;
    }
    if (choice !== 'update') {
        return false;
    }
    const code = await installVersion(latestVersion);
    if (code !== 0) {
        process.stderr.write(`Update failed with exit code ${code}.\n`);
        return false;
    }
    process.stderr.write(`Updated OneProxy CLI to ${latestVersion}. Re-run your command.\n`);
    return true;
}
function shouldOfferUpdate(command, context) {
    if (context.json || process.env.ONEPROXY_DAEMON_CHILD === '1' || process.env.ONEPROXY_SKIP_UPDATE_CHECK === '1') {
        return false;
    }
    if (!process.stdin.isTTY || !process.stderr.isTTY) {
        return false;
    }
    return !new Set(['daemon', 'env', 'version', 'help', '--help', '-h']).has(command);
}
async function latestCliVersion() {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    try {
        const response = await fetch(latestUrl, { signal: controller.signal });
        if (!response.ok) {
            return '';
        }
        const metadata = await response.json();
        return metadata.version || '';
    }
    finally {
        clearTimeout(timer);
    }
}
function compareVersions(left, right) {
    const leftParts = versionParts(left);
    const rightParts = versionParts(right);
    for (let index = 0; index < Math.max(leftParts.length, rightParts.length); index += 1) {
        const delta = (leftParts[index] ?? 0) - (rightParts[index] ?? 0);
        if (delta !== 0) {
            return delta;
        }
    }
    return 0;
}
function versionParts(value) {
    return value.split('-', 1)[0].split('.').map((part) => Number(part)).map((part) => Number.isFinite(part) ? part : 0);
}
async function promptUpdateChoice(currentVersion, latestVersion) {
    const input = process.stdin;
    const output = process.stderr;
    const choices = [
        { label: 'Update Now', value: 'update' },
        { label: 'Later', value: 'later' },
        { label: 'This version ignore', value: 'ignore' }
    ];
    let index = 0;
    let handler;
    const wasRaw = input.isRaw;
    emitKeypressEvents(input);
    input.setRawMode(true);
    input.resume();
    output.write(`\nOneProxy CLI ${latestVersion} is available. Current version: ${currentVersion}.\n`);
    for (const choice of choices) {
        output.write(`  ${choice.label}\n`);
    }
    const render = () => {
        output.write('\x1b[?25l');
        output.write(`\x1b[${choices.length}F`);
        for (let i = 0; i < choices.length; i += 1) {
            output.write(`${i === index ? '>' : ' '} ${i + 1}. ${choices[i].label}\x1b[K\n`);
        }
    };
    render();
    const selected = await new Promise((resolve) => {
        handler = (_chunk, key) => {
            if (key.ctrl && key.name === 'c') {
                resolve('later');
                return;
            }
            if (key.name === 'up') {
                index = index === 0 ? choices.length - 1 : index - 1;
                render();
            }
            if (key.name === 'down') {
                index = index === choices.length - 1 ? 0 : index + 1;
                render();
            }
            if (key.name === 'return') {
                resolve(choices[index].value);
            }
            const numeric = Number(key.name);
            if (Number.isInteger(numeric) && numeric >= 1 && numeric <= choices.length) {
                resolve(choices[numeric - 1].value);
            }
        };
        input.on('keypress', handler);
    });
    if (handler) {
        input.off('keypress', handler);
    }
    input.setRawMode(wasRaw);
    if (!wasRaw) {
        input.pause();
    }
    output.write('\x1b[?25h\n');
    return selected;
}
function installVersion(version) {
    const executable = process.platform === 'win32' ? 'npm.cmd' : 'npm';
    const child = spawn(executable, ['install', '-g', `${packageName}@${version}`], { stdio: 'inherit' });
    return new Promise((resolve, reject) => {
        child.once('error', reject);
        child.once('exit', (code, signal) => resolve(signal ? 1 : code ?? 0));
    });
}
//# sourceMappingURL=update-check.js.map