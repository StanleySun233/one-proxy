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

export type TuiUnavailableReason =
  | 'not_requested'
  | 'not_interactive'
  | 'json'
  | 'not_tty'
  | 'dumb_terminal'
  | 'unsupported_platform'
  | 'terminal_too_small'
  | 'pty_unavailable';

export const minimumTuiRows = 10;
const supportedPlatforms = new Set<NodeJS.Platform>(['darwin', 'linux']);

export function detectTuiCapability(input: TuiCapabilityInput): TuiCapabilityResult {
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
