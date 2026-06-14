import * as fsSync from 'node:fs';

export type ShellFamily = 'posix' | 'fish' | 'powershell' | 'cmd';

export type ShellDetectionInput = {
  shellOverride?: string;
  env?: Record<string, string | undefined>;
  parentShell?: string;
  platform?: NodeJS.Platform;
};

export function detectShellPath(input: ShellDetectionInput = {}): string {
  const env = input.env ?? process.env;
  const platform = input.platform ?? process.platform;
  const explicit = firstNonEmpty(input.shellOverride, env.ONEPROXY_SHELL);
  if (explicit) {
    return explicit;
  }
  const parentShell = input.parentShell ?? readParentShellName(platform);
  if (parentShell && shellFamilyFrom(parentShell)) {
    return parentShell;
  }
  if (platform === 'win32') {
    return firstNonEmpty(env.ComSpec, env.SHELL) ?? 'cmd.exe';
  }
  return firstNonEmpty(env.SHELL, env.ComSpec) ?? '/bin/sh';
}

export function detectShellFamily(input: ShellDetectionInput = {}): ShellFamily {
  return shellFamilyFrom(detectShellPath(input)) ?? 'posix';
}

function readParentShellName(platform: NodeJS.Platform): string {
  if (platform !== 'linux' || !process.ppid) {
    return '';
  }
  return firstRecognizedProcValue(`/proc/${process.ppid}/comm`, `/proc/${process.ppid}/cmdline`);
}

function firstRecognizedProcValue(...files: string[]): string {
  for (const file of files) {
    const value = readProcValue(file);
    if (value && shellFamilyFrom(value)) {
      return value;
    }
  }
  return '';
}

function readProcValue(file: string): string {
  try {
    const body = fsSync.readFileSync(file, 'utf8').trim();
    return body.split('\0')[0] ?? '';
  } catch {
    return '';
  }
}

function shellFamilyFrom(value: string): ShellFamily | null {
  const shell = shellName(value);
  if (shell === 'fish') {
    return 'fish';
  }
  if (shell === 'powershell' || shell === 'pwsh' || shell === 'powershell_ise') {
    return 'powershell';
  }
  if (shell === 'cmd') {
    return 'cmd';
  }
  if (['sh', 'bash', 'zsh', 'dash', 'ksh', 'ash'].includes(shell)) {
    return 'posix';
  }
  return null;
}

function shellName(value: string): string {
  const normalized = value.trim().replace(/\\/g, '/').split(/\s+/)[0] ?? '';
  const base = normalized.split('/').filter(Boolean).pop() ?? normalized;
  return base.toLowerCase().replace(/\.exe$/, '');
}

function firstNonEmpty(...values: Array<string | undefined>): string | undefined {
  return values.map((value) => value?.trim()).find(Boolean);
}
