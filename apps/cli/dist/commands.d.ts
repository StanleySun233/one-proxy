import type { CliContext } from './main.ts';
type ErrorResult = {
    error: {
        code: string;
        message: string;
        details?: Record<string, unknown>;
    };
};
export declare function writeError(error: ErrorResult['error'], context: CliContext): void;
export declare function statusCommand(_args: string[], context: CliContext): Promise<void>;
export declare function overrideCommand(args: string[], context: CliContext): Promise<void>;
export declare function routeCommand(args: string[], context: CliContext): Promise<void>;
export declare function testCommand(args: string[], context: CliContext): Promise<void>;
export declare function doctor(_args: string[], context: CliContext): Promise<number>;
export {};
