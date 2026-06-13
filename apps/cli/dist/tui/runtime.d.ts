import { type TuiStatusSnapshot } from './footer.ts';
import { type PtyAdapter } from './pty.ts';
export type TuiCommand = {
    executable: string;
    args: string[];
    cwd?: string;
    env?: NodeJS.ProcessEnv;
};
export type TuiRuntimeOptions = {
    command: TuiCommand;
    snapshot: TuiStatusSnapshot | (() => TuiStatusSnapshot | Promise<TuiStatusSnapshot>);
    requested: boolean;
    interactive?: boolean;
    json?: boolean;
    ptyAdapter?: PtyAdapter;
};
export type TuiSessionOptions = {
    command: string;
    args: string[];
    cwd?: string;
    env?: NodeJS.ProcessEnv;
    status: TuiStatusSnapshot | (() => TuiStatusSnapshot | Promise<TuiStatusSnapshot>);
    stdin?: RuntimeInput;
    stdout?: RuntimeOutput;
    stderr?: RuntimeOutput;
    ptyAdapter: PtyAdapter;
};
export type TuiRuntimeResult = {
    ran: boolean;
    exitCode?: number;
};
export declare function runTuiRuntime(options: TuiRuntimeOptions): Promise<TuiRuntimeResult>;
export declare function runTuiCommand(options: {
    executable: string;
    args: string[];
    cwd?: string;
    env?: NodeJS.ProcessEnv;
    status?: unknown;
}): Promise<{
    available: boolean;
    exitCode?: number;
}>;
export declare function runTuiSession(options: TuiSessionOptions): Promise<number>;
type RuntimeInput = NodeJS.ReadStream & {
    isTTY?: boolean;
    setRawMode?: (mode: boolean) => void;
};
type RuntimeOutput = NodeJS.WriteStream & {
    isTTY?: boolean;
    columns?: number;
    rows?: number;
};
export {};
