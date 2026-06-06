import type { CliContext } from './main.ts';
type MonitorProxy = {
    port: number;
    close: () => Promise<void>;
};
declare function monitorLogName(executable: string, now?: Date): string;
declare function startMonitorProxy(logPath: string, onActivity?: () => void): Promise<MonitorProxy>;
export declare function monitorCommand(args: string[], _context: CliContext): Promise<number>;
export declare const monitorInternals: {
    monitorLogName: typeof monitorLogName;
    startMonitorProxy: typeof startMonitorProxy;
};
export {};
