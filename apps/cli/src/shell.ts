import { spawn } from 'node:child_process';
import type { CliContext } from './main.ts';
import { sessionProxyEnv } from './session-env.ts';
import { startDaemonSession } from './daemon/lifecycle.ts';
import { runTuiCommand } from './tui/runtime.ts';
import { buildTuiStatusSnapshot } from './tui/status.ts';
import { detectShellPath } from './shell-detect.ts';

function shellArgs(shell: string): string[] {
  const normalized = shell.toLowerCase();
  if (process.platform === 'win32' || normalized.includes('cmd.exe') || normalized.includes('powershell') || normalized.includes('pwsh')) {
    return [];
  }
  return ['-i'];
}

export async function startActivatedShell(): Promise<number> {
  const shell = detectShellPath();
  const session = await startDaemonSession();
  process.stdout.write(`OneProxy shell active: ${shell}\n`);
  process.stdout.write('Exit this shell to turn it off.\n');
  const child = spawn(shell, shellArgs(shell), {
    stdio: 'inherit',
    env: {
      ...process.env,
      ...(await sessionProxyEnv(session.metadata.bindings))
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
  parseShellCommandArgs(_args);
  if (!_context.json) {
    const tuiExitCode = await tryStartActivatedShellTui();
    if (tuiExitCode !== null) {
      return tuiExitCode;
    }
  }
  return startActivatedShell();
}

export function parseShellCommandArgs(argv: string[]): { args: string[]; tui: boolean } {
  const args: string[] = [];
  let tui = false;
  for (const value of argv) {
    if (value === '--tui') {
      tui = true;
      continue;
    }
    throw Object.assign(new Error(`Unknown shell option: ${value}`), { code: 'SYNTAX_ERROR', exitCode: 2 });
  }
  return { args, tui };
}

async function tryStartActivatedShellTui(): Promise<number | null> {
  const shell = detectShellPath();
  const session = await startDaemonSession();
  try {
    const result = await runTuiCommand({
      executable: shell,
      args: shellArgs(shell),
      env: {
        ...process.env,
        ...(await sessionProxyEnv(session.metadata.bindings))
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
  } finally {
    await session.end();
  }
}
