export type IsolatedRunInput = {
    executable: string;
    args: string[];
    env: NodeJS.ProcessEnv;
    proxyPort: number;
};
type FirewallRule = {
    command: 'iptables' | 'ip6tables';
    add: string[];
    remove: string[];
};
export declare function runProxyOnlyIsolatedCommand(input: IsolatedRunInput): Promise<number>;
export declare function runProxyOnlyBestEffortCommand(input: IsolatedRunInput): Promise<number>;
export declare function isProxyIsolationUnavailable(error: unknown): boolean;
export declare function proxyIsolationHelp(error: unknown): string[];
declare function firewallRules(proxyPort: number, cgroupPath: string): FirewallRule[];
export declare const runIsolationInternals: {
    isProxyIsolationUnavailable: typeof isProxyIsolationUnavailable;
    proxyIsolationHelp: typeof proxyIsolationHelp;
    firewallRules: typeof firewallRules;
};
export {};
