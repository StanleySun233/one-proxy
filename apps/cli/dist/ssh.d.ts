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
export declare function runSsh(argv: string[]): Promise<number>;
export declare function buildSshCommandPlan(argv: string[]): Promise<SshCommandPlan>;
export declare function parseSshTarget(argv: string[]): SshTarget;
