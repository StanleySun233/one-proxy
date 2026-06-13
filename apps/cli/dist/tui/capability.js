export { footerRowsForTerminal as footerRowsForTerminalHeight } from "./footer.js";
export const tuiUnavailableWarning = '! TUI failed to start; falling back to standard terminal mode.';
export const minimumTuiRows = 10;
const supportedPlatforms = new Set(['darwin', 'linux']);
export function detectTuiCapability(input) {
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
function isProbeInput(input) {
    return 'stdin' in input;
}
function normalizeProbeInput(input) {
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
function probeReason(reason) {
    return reason === 'json' ? 'json_output' : reason;
}
function unavailableReason(input) {
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
export function currentTuiCapabilityInput(requested, interactive, json, ptyAvailable) {
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
//# sourceMappingURL=capability.js.map