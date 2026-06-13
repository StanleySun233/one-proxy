import { spawn } from 'node:child_process';
import { proxyEnv } from "./session-env.js";
import { startDaemonSession } from "./daemon/lifecycle.js";
import { runTuiCommand } from "./tui/runtime.js";
import { buildTuiStatusSnapshot } from "./tui/status.js";
function defaultShell() {
    if (process.env.ONEPROXY_SHELL) {
        return process.env.ONEPROXY_SHELL;
    }
    if (process.platform === 'win32') {
        return process.env.ComSpec || 'cmd.exe';
    }
    return process.env.SHELL || '/bin/sh';
}
function shellArgs(shell) {
    const normalized = shell.toLowerCase();
    if (process.platform === 'win32' || normalized.includes('cmd.exe') || normalized.includes('powershell') || normalized.includes('pwsh')) {
        return [];
    }
    return ['-i'];
}
export async function startActivatedShell() {
    const shell = defaultShell();
    const session = await startDaemonSession();
    process.stdout.write(`OneProxy shell active: ${shell}\n`);
    process.stdout.write('Exit this shell to turn it off.\n');
    const child = spawn(shell, shellArgs(shell), {
        stdio: 'inherit',
        env: {
            ...process.env,
            ...proxyEnv(session.metadata.bindings)
        }
    });
    return new Promise((resolve, reject) => {
        child.once('error', (error) => {
            session.end().then(() => reject(error), reject);
        });
        child.once('exit', (code, signal) => {
            session.end().then(() => {
                if (signal) {
                    resolve(1);
                    return;
                }
                resolve(code ?? 0);
            }, reject);
        });
    });
}
export async function shellCommand(_args, _context) {
    const parsed = parseShellCommandArgs(_args);
    if (!_context.json) {
        const tuiExitCode = await tryStartActivatedShellTui();
        if (tuiExitCode !== null) {
            return tuiExitCode;
        }
    }
    return startActivatedShell();
}
export function parseShellCommandArgs(argv) {
    return stripTuiFlag(argv);
}
function stripTuiFlag(argv) {
    const args = [];
    let tui = false;
    for (const value of argv) {
        if (value === '--tui') {
            tui = true;
        }
        else {
            args.push(value);
        }
    }
    return { args, tui };
}
async function tryStartActivatedShellTui() {
    const shell = defaultShell();
    const session = await startDaemonSession();
    try {
        const result = await runTuiCommand({
            executable: shell,
            args: shellArgs(shell),
            env: {
                ...process.env,
                ...proxyEnv(session.metadata.bindings)
            },
            status: await buildTuiStatusSnapshot()
        });
        if (!result.available) {
            return null;
        }
        if (typeof result.exitCode !== 'number') {
            throw Object.assign(new Error('TUI exited without a child exit code'), { code: 'TUI_RUNTIME_ERROR' });
        }
        return result.exitCode;
    }
    finally {
        await session.end();
    }
}
//# sourceMappingURL=shell.js.map