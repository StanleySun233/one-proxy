import type { CliContext } from './main.ts';
import type { RouteResult } from './daemon/router.ts';
export type SshTarget = {
    user?: string;
    host: string;
    port: number;
    original: string;
};
export type SshCommandPlan = {
    executable: string;
    args: string[];
    route: RouteResult;
};
export declare class SshCommandError extends Error {
    code: string;
    constructor(code: string, message: string);
}
export declare function runSsh(argv: string[], context?: CliContext): Promise<number>;
export declare function parseSshCommandArgs(argv: string[]): {
    args: string[];
    tui: boolean;
};
export declare function buildSshCommandPlan(argv: string[]): Promise<SshCommandPlan>;
export declare function parseSshTarget(argv: string[]): SshTarget;
export declare function buildSshProxyCommand(proxyHost: string, proxyPort: number): string;
