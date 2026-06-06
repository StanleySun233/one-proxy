import { spawn } from 'node:child_process';
import type { CliContext } from './main.ts';
import { proxyEnv } from './session-env.ts';
import { startDaemonSession } from './daemon/lifecycle.ts';

function defaultShell(): string {
  if (process.env.ONEPROXY_SHELL) {
    return process.env.ONEPROXY_SHELL;
  }
  if (process.platform === 'win32') {
    return process.env.ComSpec || 'cmd.exe';
  }
  return process.env.SHELL || '/bin/sh';
}

function shellArgs(shell: string): string[] {
  const normalized = shell.toLowerCase();
  if (process.platform === 'win32' || normalized.includes('cmd.exe') || normalized.includes('powershell') || normalized.includes('pwsh')) {
    return [];
  }
  return ['-i'];
}

export async function startActivatedShell(): Promise<number> {
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

export async function shellCommand(_args: string[], _context: CliContext): Promise<number> {
  return startActivatedShell();
}
