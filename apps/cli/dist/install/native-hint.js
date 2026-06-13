import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
const nativeTools = ['make', 'g++', 'python3'];
export function missingNativeBuildTools() {
    return nativeTools.filter((tool) => !commandExists(tool));
}
export function nativeBuildToolsHint(missing) {
    const list = missing.join(', ');
    return [
        `OneProxy TUI native dependency is not ready. Missing build tools: ${list}.`,
        '',
        'Install the complete native build toolchain, then reinstall OneProxy CLI:',
        '',
        '  Ubuntu/Debian:',
        '    sudo apt update',
        '    sudo apt install -y build-essential python3',
        '',
        '  RHEL/CentOS/Fedora:',
        '    sudo dnf groupinstall -y "Development Tools"',
        '    sudo dnf install -y python3',
        '',
        '  Alpine:',
        '    sudo apk add --no-cache build-base python3',
        '',
        'Then run:',
        '    npm install -g @stanleysun233/oneproxy-cli'
    ].join('\n');
}
export async function nodePtyReady() {
    try {
        const dynamicImport = new Function('specifier', 'return import(specifier)');
        await dynamicImport('node-pty');
        return true;
    }
    catch {
        return false;
    }
}
export async function printNativeDependencyHint() {
    if (await nodePtyReady()) {
        return;
    }
    const missing = missingNativeBuildTools();
    if (missing.length === 0) {
        return;
    }
    process.stderr.write(`\n${nativeBuildToolsHint(missing)}\n\n`);
}
function commandExists(command) {
    const result = spawnSync('sh', ['-c', `command -v ${command}`], { stdio: 'ignore' });
    return result.status === 0;
}
if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
    void printNativeDependencyHint();
}
//# sourceMappingURL=native-hint.js.map