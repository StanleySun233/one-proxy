import { spawn } from 'node:child_process';
import type { CliContext } from './main.ts';
import { sessionProxyEnv } from './session-env.ts';
import { startDaemonSession } from './daemon/lifecycle.ts';
import { runTuiCommand } from './tui/runtime.ts';
import { buildTuiStatusSnapshot } from './tui/status.ts';
import { detectShellPath } from './shell-detect.ts';

function defaultShell(shellOverride?: string): string {
  return detectShellPath({ shellOverride });
}

function shellArgs(shell: string): string[] {
  const normalized = shell.toLowerCase();
  if (process.platform === 'win32' || normalized.includes('cmd.exe') || normalized.includes('powershell') || normalized.includes('pwsh')) {
    return [];
  }
  return ['-i'];
}

export async function startActivatedShell(shellOverride?: string): Promise<number> {
  const shell = defaultShell(shellOverride);
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
  const parsed = parseShellCommandArgs(_args);
  if (!_context.json) {
    const tuiExitCode = await tryStartActivatedShellTui(parsed.shell);
    if (tuiExitCode !== null) {
      return tuiExitCode;
    }
  }
  return startActivatedShell(parsed.shell);
}

export function parseShellCommandArgs(argv: string[]): { args: string[]; tui: boolean; shell?: string } {
  const args: string[] = [];
  let tui = false;
  let shell: string | undefined;
  for (let index = 0; index < argv.length; index += 1) {
    const value = argv[index];
    if (value === '--tui') {
      tui = true;
      continue;
    }
    if (value === '--shell') {
      shell = argv[index + 1];
      index += 1;
      if (!shell) {
        throw Object.assign(new Error('shell --shell requires a shell name.'), { code: 'SYNTAX_ERROR', exitCode: 2 });
      }
      continue;
    }
    args.push(value);
  }
  if (shell) {
    return { args, tui, shell };
  }
  return { args, tui };
}

async function tryStartActivatedShellTui(shellOverride?: string): Promise<number | null> {
  const shell = defaultShell(shellOverride);
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
