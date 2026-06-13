var __rewriteRelativeImportExtension = (this && this.__rewriteRelativeImportExtension) || function (path, preserveJsx) {
    if (typeof path === "string" && /^\.\.?\//.test(path)) {
        return path.replace(/\.(tsx)$|((?:\.d)?)((?:\.[^./]+?)?)\.([cm]?)ts$/i, function (m, tsx, d, ext, cm) {
            return tsx ? preserveJsx ? ".jsx" : ".js" : d && (!ext || !cm) ? m : (d + ext + "." + cm.toLowerCase() + "js");
        });
    }
    return path;
};
import { spawn } from 'node:child_process';
import { ensureDaemon, readConfig, readState } from "./daemon/lifecycle.js";
import { resolveRoute } from "./daemon/router.js";
export class SshCommandError extends Error {
    code;
    constructor(code, message) {
        super(message);
        this.code = code;
    }
}
export async function runSsh(argv, context = { json: false }) {
    const parsed = parseSshCommandArgs(argv);
    const plan = await buildSshCommandPlan(parsed.args);
    if (parsed.tui && !context.json) {
        const tuiExitCode = await tryRunSshTui(plan);
        if (tuiExitCode !== null) {
            return tuiExitCode;
        }
        process.stderr.write('onep tui: unavailable, using standard terminal mode\n');
    }
    if (parsed.tui && context.json) {
        process.stderr.write('onep tui: unavailable, using standard terminal mode\n');
    }
    return await spawnSsh(plan.executable, plan.args);
}
export function parseSshCommandArgs(argv) {
    return stripTuiFlag(argv);
}
export async function buildSshCommandPlan(argv) {
    const target = parseSshTarget(argv);
    const { metadata } = await ensureDaemon();
    const [config, state] = await Promise.all([readConfig(), readState()]);
    const route = resolveRoute({ config, state, target: `ssh://${target.host}:${target.port}`, protocol: 'ssh' });
    const args = ['-p', String(target.port)];
    if (route.mode === 'proxy') {
        args.push('-o', `ProxyCommand=${proxyCommand(metadata.bindings.host, metadata.bindings.httpPort)}`);
    }
    args.push(target.original);
    return {
        executable: 'ssh',
        args,
        route
    };
}
export function parseSshTarget(argv) {
    let port = 22;
    let target = '';
    for (let index = 0; index < argv.length; index += 1) {
        const value = argv[index];
        if (value === '-p') {
            port = Number(argv[index + 1]);
            index += 1;
        }
        else if (!target) {
            target = value;
        }
        else {
            throw new SshCommandError('INVALID_TARGET', 'onep ssh accepts one SSH target');
        }
    }
    if (!target || !Number.isInteger(port) || port < 1 || port > 65535) {
        throw new SshCommandError('INVALID_TARGET', 'Invalid SSH target or port');
    }
    const at = target.lastIndexOf('@');
    const user = at > 0 ? target.slice(0, at) : undefined;
    const host = (at > 0 ? target.slice(at + 1) : target).toLowerCase();
    if (!host) {
        throw new SshCommandError('INVALID_TARGET', 'Invalid SSH host');
    }
    return { user, host, port, original: target };
}
function proxyCommand(proxyHost, proxyPort) {
    const helper = [
        "const net=require('node:net')",
        'const [host,port,proxyHost,proxyPort]=process.argv.slice(1)',
        'const socket=net.connect(Number(proxyPort),proxyHost,()=>socket.write(`CONNECT ${host}:${port} HTTP/1.1\\r\\nHost: ${host}:${port}\\r\\n\\r\\n`))',
        "let buffer=''",
        "socket.on('data',(chunk)=>{if(buffer!==null){buffer+=chunk.toString('latin1');const index=buffer.indexOf('\\r\\n\\r\\n');if(index>=0){if(!/^HTTP\\/1\\.[01] 2\\d\\d/.test(buffer))process.exit(1);const rest=Buffer.from(buffer.slice(index+4),'latin1');if(rest.length)process.stdout.write(rest);buffer=null;socket.pipe(process.stdout);process.stdin.pipe(socket)}}else process.stdout.write(chunk)})",
        "socket.on('error',()=>process.exit(1))"
    ].join(';');
    return `${shellQuote(process.execPath)} -e ${JSON.stringify(helper)} %h %p ${proxyHost} ${proxyPort}`;
}
function shellQuote(value) {
    return `'${value.replace(/'/g, "'\\''")}'`;
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
async function tryRunSshTui(plan) {
    try {
        const runtimePath = './tui/runtime.ts';
        const statusPath = './tui/status.ts';
        const [runtime, status] = await Promise.all([
            import(__rewriteRelativeImportExtension(runtimePath)),
            import(__rewriteRelativeImportExtension(statusPath))
        ]);
        if (!runtime.runTuiCommand || !status.buildTuiStatusSnapshot) {
            return null;
        }
        const result = await runtime.runTuiCommand({
            executable: plan.executable,
            args: plan.args,
            env: process.env,
            status: await status.buildTuiStatusSnapshot({ route: plan.route })
        });
        if (typeof result === 'number') {
            return result;
        }
        if (result.available === false || typeof result.exitCode !== 'number') {
            return null;
        }
        return result.exitCode;
    }
    catch {
        return null;
    }
}
async function spawnSsh(executable, args) {
    const child = spawn(executable, args, { stdio: 'inherit' });
    return await new Promise((resolve, reject) => {
        child.once('error', (error) => {
            reject(new SshCommandError(error.code === 'ENOENT' ? 'COMMAND_NOT_FOUND' : 'SSH_FAILED', error.message));
        });
        child.once('exit', (code, signal) => {
            resolve(signal ? 1 : code ?? 1);
        });
    });
}
//# sourceMappingURL=ssh.js.map