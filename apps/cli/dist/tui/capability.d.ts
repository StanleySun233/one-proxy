export { footerRowsForTerminal as footerRowsForTerminalHeight } from './footer.ts';
export declare const tuiUnavailableWarning = "onep tui: unavailable, using standard terminal mode";
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
    stdin: {
        isTTY?: boolean;
    };
    stdout: {
        isTTY?: boolean;
        rows?: number;
    };
    stderr: {
        isTTY?: boolean;
    };
    env?: NodeJS.ProcessEnv;
    platform: NodeJS.Platform;
    ptyAvailable: boolean;
};
export type TuiCapabilityProbeResult = {
    available: boolean;
    reason: TuiCapabilityProbeReason | null;
    color: boolean;
};
export type TuiUnavailableReason = 'not_requested' | 'not_interactive' | 'json' | 'not_tty' | 'dumb_terminal' | 'unsupported_platform' | 'terminal_too_small' | 'pty_unavailable';
export type TuiCapabilityProbeReason = Exclude<TuiUnavailableReason, 'json'> | 'json_output';
export declare const minimumTuiRows = 10;
export declare function detectTuiCapability(input: TuiCapabilityInput): TuiCapabilityResult;
export declare function detectTuiCapability(input: TuiCapabilityProbeInput): TuiCapabilityProbeResult;
export declare function currentTuiCapabilityInput(requested: boolean, interactive: boolean, json: boolean, ptyAvailable: boolean): TuiCapabilityInput;
