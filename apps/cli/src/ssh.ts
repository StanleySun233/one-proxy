import { spawn } from 'node:child_process';
import { ensureDaemon, readConfig, readState } from './daemon/lifecycle.js';
import { resolveRoute } from './daemon/router.js';
import type { RouteResult } from './daemon/router.js';

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
  constructor(public code: string, message: string) {
    super(message);
  }
}

export async function runSsh(argv: string[]) {
  const plan = await buildSshCommandPlan(argv);
  return await spawnSsh(plan.executable, plan.args);
}

export async function buildSshCommandPlan(argv: string[]): Promise<SshCommandPlan> {
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

function proxyCommand(proxyHost: string, proxyPort: number) {
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

function shellQuote(value: string) {
  return `'${value.replace(/'/g, "'\\''")}'`;
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
