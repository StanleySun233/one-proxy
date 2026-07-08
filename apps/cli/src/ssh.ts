import { spawn } from 'node:child_process';
import type { CliContext } from './main.ts';
import { ensureDaemon, readConfig, readState } from './daemon/lifecycle.ts';
import { resolveRoute } from './daemon/router.ts';
import type { RouteResult } from './daemon/router.ts';
import { runTuiCommand } from './tui/runtime.ts';
import { buildTuiStatusSnapshot } from './tui/status.ts';

export type SshTarget = {
  user?: string;
  host: string;
  port: number;
  original: string;
};

export type SshCommandPlan = {
  executable: string;
  args: string[];
  route: RouteResult;
};

export class SshCommandError extends Error {
  code: string;

  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }
}

export async function runSsh(argv: string[], context: CliContext = { json: false }) {
  const parsed = parseSshCommandArgs(argv);
  const plan = await buildSshCommandPlan(parsed.args);
  if (!context.json) {
    const tuiExitCode = await tryRunSshTui(plan);
    if (tuiExitCode !== null) {
      return tuiExitCode;
    }
  }
  return await spawnSsh(plan.executable, plan.args);
}

export function parseSshCommandArgs(argv: string[]) {
  return stripTuiFlag(argv);
}

export async function buildSshCommandPlan(argv: string[]): Promise<SshCommandPlan> {
  const target = parseSshTarget(argv);
  const { metadata } = await ensureDaemon();
  const [config, state] = await Promise.all([readConfig(), readState()]);
  const route = resolveRoute({ config, state, target: `ssh://${target.host}:${target.port}`, protocol: 'ssh' });
  const args = ['-p', String(target.port)];
  if (route.mode === 'proxy') {
    args.push('-o', `ProxyCommand=${buildSshProxyCommand(metadata.bindings.host, metadata.bindings.httpPort)}`);
  }
  args.push(target.original);
  return {
    executable: 'ssh',
    args,
    route
  };
}

export function parseSshTarget(argv: string[]): SshTarget {
  let port = 22;
  let target = '';
  for (let index = 0; index < argv.length; index += 1) {
    const value = argv[index];
    if (value === '-p') {
      port = Number(argv[index + 1]);
      index += 1;
    } else if (!target) {
      target = value;
    } else {
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

export function buildSshProxyCommand(proxyHost: string, proxyPort: number) {
  const helper = [
    "const net=require('node:net')",
    'const [host,port,proxyHost,proxyPort]=process.argv.slice(1)',
    "const target=host+':'+port",
    "const request='CONNECT '+target+' HTTP/1.1\\r\\nHost: '+target+'\\r\\n\\r\\n'",
    'const socket=net.connect(Number(proxyPort),proxyHost,()=>socket.write(request))',
    'let buffer=Buffer.alloc(0)',
    "const onData=(chunk)=>{buffer=Buffer.concat([buffer,chunk]);const index=buffer.indexOf('\\r\\n\\r\\n');if(index<0){if(buffer.length>65536)process.exit(1);return}socket.off('data',onData);const header=buffer.subarray(0,index).toString('ascii');if(!/^HTTP\\/1\\.[01] 2\\d\\d/.test(header))process.exit(1);const rest=buffer.subarray(index+4);if(rest.length)socket.unshift(rest);socket.pipe(process.stdout);process.stdin.pipe(socket)}",
    "socket.on('data',onData)",
    "socket.on('error',()=>process.exit(1))"
  ].join(';');
  return `${shellQuote(process.execPath)} -e ${JSON.stringify(helper)} %h %p ${proxyHost} ${proxyPort}`;
}

function shellQuote(value: string) {
  return `'${value.replace(/'/g, "'\\''")}'`;
}

function stripTuiFlag(argv: string[]) {
  const args: string[] = [];
  let tui = false;
  for (const value of argv) {
    if (value === '--tui') {
      tui = true;
    } else {
      args.push(value);
    }
  }
  return { args, tui };
}

async function tryRunSshTui(plan: SshCommandPlan): Promise<number | null> {
  const result = await runTuiCommand({
    executable: plan.executable,
    args: plan.args,
    env: process.env,
    status: await buildTuiStatusSnapshot({ route: plan.route })
  });
  if (!result.available) {
    return null;
  }
  if (typeof result.exitCode !== 'number') {
    throw Object.assign(new Error('TUI exited without a child exit code'), { code: 'TUI_RUNTIME_ERROR' });
  }
  return result.exitCode;
}

async function spawnSsh(executable: string, args: string[]) {
  const child = spawn(executable, args, { stdio: 'inherit' });
  return await new Promise<number>((resolve, reject) => {
    child.once('error', (error: NodeJS.ErrnoException) => {
      reject(new SshCommandError(error.code === 'ENOENT' ? 'COMMAND_NOT_FOUND' : 'SSH_FAILED', error.message));
    });
    child.once('exit', (code, signal) => {
      resolve(signal ? 1 : code ?? 1);
    });
  });
}
