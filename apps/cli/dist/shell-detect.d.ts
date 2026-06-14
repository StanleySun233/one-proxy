export type ShellFamily = 'posix' | 'fish' | 'powershell' | 'cmd';
export type ShellDetectionInput = {
    env?: Record<string, string | undefined>;
    parentShell?: string;
    platform?: NodeJS.Platform;
};
export declare function detectShellPath(input?: ShellDetectionInput): string;
export declare function detectShellFamily(input?: ShellDetectionInput): ShellFamily;
