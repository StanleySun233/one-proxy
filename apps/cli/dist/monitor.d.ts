import type { CliContext } from './main.ts';
type ProcessInfo = {
    pid: number;
    parentPid: number;
    name: string;
};
type Endpoint = {
    address: string;
    port: number | null;
};
type MonitorLogEvent = {
    timestamp: string;
    source: 'netstat' | 'procfs' | 'lsof';
    process: string;
    pid: number;
    protocol: 'tcp' | 'udp';
    localAddress: string;
    localPort: number | null;
    remoteAddress: string;
    remotePort: number | null;
    remoteHost: string;
    domain: string | null;
    domainSource: null;
    state: string | null;
};
declare function monitorLogName(executable: string, now?: Date): string;
declare function parseEndpoint(value: string): Endpoint;
declare function parseWindowsNetstat(output: string): MonitorLogEvent[];
declare function parseWindowsProcesses(output: string): ProcessInfo[];
declare function watchedProcesses(rootPid: number, previousPids: Set<number>, processes: ProcessInfo[]): Set<number>;
declare function parseLinuxEndpoint(value: string, family: 'ipv4' | 'ipv6'): Endpoint;
declare function parseLsof(output: string): MonitorLogEvent[];
export declare function monitorCommand(args: string[], _context: CliContext): Promise<number>;
export declare const monitorInternals: {
    monitorLogName: typeof monitorLogName;
    parseEndpoint: typeof parseEndpoint;
    parseLsof: typeof parseLsof;
    parseLinuxEndpoint: typeof parseLinuxEndpoint;
    parseWindowsNetstat: typeof parseWindowsNetstat;
    parseWindowsProcesses: typeof parseWindowsProcesses;
    watchedProcesses: typeof watchedProcesses;
};
export {};
