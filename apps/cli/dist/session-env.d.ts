import type { CliContext } from './main.ts';
import { type DaemonBindings } from './storage.ts';
export declare function ensureSessionProxyBindings(): Promise<DaemonBindings>;
export declare function proxyEnv(bindings: DaemonBindings): Record<string, string>;
export declare function sessionProxyEnv(): Promise<Record<string, string>>;
export declare function envOn(): Promise<void>;
export declare function envOff(): Promise<void>;
export declare function runCommand(args: string[], _context: CliContext): Promise<number>;
