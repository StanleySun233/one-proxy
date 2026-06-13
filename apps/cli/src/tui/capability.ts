export { footerRowsForTerminal as footerRowsForTerminalHeight } from './footer.ts';

export const tuiUnavailableWarning = 'onep tui: unavailable, using standard terminal mode';

export type TuiCapabilityInput = {
  requested: boolean;
  interactive: boolean;
  json: boolean;
  stdinIsTty: boolean;
  stdoutIsTty: boolean;
  stderrIsTty: boolean;
  term?: string;
  platform: NodeJS.Platform;
  rows?: number;
  ptyAvailable: boolean;
};

export type TuiCapabilityResult = {
  enabled: boolean;
  warn: boolean;
  reason?: TuiUnavailableReason;
};

export type TuiCapabilityProbeInput = {
  requested: boolean;
  interactive?: boolean;
  json: boolean;
  stdin: { isTTY?: boolean };
  stdout: { isTTY?: boolean; rows?: number };
  stderr: { isTTY?: boolean };
  env?: NodeJS.ProcessEnv;
  platform: NodeJS.Platform;
  ptyAvailable: boolean;
};

export type TuiCapabilityProbeResult = {
  available: boolean;
  reason: TuiCapabilityProbeReason | null;
  color: boolean;
};

export type TuiUnavailableReason =
  | 'not_requested'
  | 'not_interactive'
  | 'json'
  | 'not_tty'
  | 'dumb_terminal'
  | 'unsupported_platform'
  | 'terminal_too_small'
  | 'pty_unavailable';

export type TuiCapabilityProbeReason = Exclude<TuiUnavailableReason, 'json'> | 'json_output';

export const minimumTuiRows = 10;
const supportedPlatforms = new Set<NodeJS.Platform>(['darwin', 'linux']);

export function detectTuiCapability(input: TuiCapabilityInput): TuiCapabilityResult;
export function detectTuiCapability(input: TuiCapabilityProbeInput): TuiCapabilityProbeResult;
export function detectTuiCapability(input: TuiCapabilityInput | TuiCapabilityProbeInput): TuiCapabilityResult | TuiCapabilityProbeResult {
  if (isProbeInput(input)) {
    const normalized = normalizeProbeInput(input);
    const reason = unavailableReason(normalized);
    return {
      available: reason === null,
      reason: probeReason(reason),
      color: Boolean(input.stdout.isTTY && input.env?.TERM !== 'dumb' && !input.env?.NO_COLOR)
    };
  }

  const reason = unavailableReason(input);
  if (!reason) {
    return { enabled: true, warn: false };
  }
  return {
    enabled: false,
    warn: input.requested && reason !== 'not_requested',
    reason
  };
}

function isProbeInput(input: TuiCapabilityInput | TuiCapabilityProbeInput): input is TuiCapabilityProbeInput {
  return 'stdin' in input;
}

function normalizeProbeInput(input: TuiCapabilityProbeInput): TuiCapabilityInput {
  return {
    requested: input.requested,
    interactive: input.interactive ?? true,
    json: input.json,
    stdinIsTty: Boolean(input.stdin.isTTY),
    stdoutIsTty: Boolean(input.stdout.isTTY),
    stderrIsTty: Boolean(input.stderr.isTTY),
    term: input.env?.TERM,
    platform: input.platform,
    rows: input.stdout.rows,
    ptyAvailable: input.ptyAvailable
  };
}

function probeReason(reason: TuiUnavailableReason | null): TuiCapabilityProbeReason | null {
  return reason === 'json' ? 'json_output' : reason;
}

function unavailableReason(input: TuiCapabilityInput): TuiUnavailableReason | null {
  if (!input.requested) {
    return 'not_requested';
  }
  if (!input.interactive) {
    return 'not_interactive';
  }
  if (input.json) {
    return 'json';
  }
  if (!input.stdinIsTty || !input.stdoutIsTty || !input.stderrIsTty) {
    return 'not_tty';
  }
  if (input.term === 'dumb') {
    return 'dumb_terminal';
  }
  if (!supportedPlatforms.has(input.platform)) {
    return 'unsupported_platform';
  }
  if ((input.rows ?? 0) < minimumTuiRows) {
    return 'terminal_too_small';
  }
  if (!input.ptyAvailable) {
    return 'pty_unavailable';
  }
  return null;
}

export function currentTuiCapabilityInput(requested: boolean, interactive: boolean, json: boolean, ptyAvailable: boolean): TuiCapabilityInput {
  return {
    requested,
    interactive,
    json,
    stdinIsTty: Boolean(process.stdin.isTTY),
    stdoutIsTty: Boolean(process.stdout.isTTY),
    stderrIsTty: Boolean(process.stderr.isTTY),
    term: process.env.TERM,
    platform: process.platform,
    rows: process.stdout.rows,
    ptyAvailable
  };
}
