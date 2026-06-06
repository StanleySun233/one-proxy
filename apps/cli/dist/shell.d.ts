import type { CliContext } from './main.ts';
export declare function startActivatedShell(): Promise<number>;
export declare function shellCommand(_args: string[], _context: CliContext): Promise<number>;
