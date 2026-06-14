import type { CliContext } from './main.ts';
import { type DaemonBindings } from './storage.ts';
export declare function ensureSessionProxyBindings(): Promise<DaemonBindings>;
export declare function proxyEnv(bindings: DaemonBindings, extraNoProxyHosts?: string[]): Record<string, string>;
export declare function proxyOnlyEnv(bindings: DaemonBindings): Record<string, string>;
export declare function sessionProxyEnv(bindings?: DaemonBindings): Promise<Record<string, string>>;
export declare function parseEnvCommandArgs(argv: string[]): {};
export declare function envOn(args?: string[]): Promise<void>;
export declare function envOff(args?: string[]): Promise<void>;
export declare function runCommand(args: string[], context: CliContext): Promise<number>;
export declare function parseRunCommandArgs(argv: string[]): {
    args: string[];
    tui: boolean;
};
