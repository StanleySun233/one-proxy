import { spawn } from 'node:child_process';
import type { CliContext } from './main.ts';
import { sessionProxyEnv } from './session-env.ts';

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
  process.stdout.write(`OneProxy shell active: ${shell}\n`);
  process.stdout.write('Exit this shell to turn it off.\n');
  const child = spawn(shell, shellArgs(shell), {
    stdio: 'inherit',
    env: {
      ...process.env,
      ...(await sessionProxyEnv())
    }
  });
  return new Promise((resolve, reject) => {
    child.once('error', reject);
    child.once('exit', (code, signal) => {
      if (signal) {
        resolve(1);
        return;
      }
      resolve(code ?? 0);
    });
  });
}

export async function shellCommand(_args: string[], _context: CliContext): Promise<number> {
  return startActivatedShell();
}
