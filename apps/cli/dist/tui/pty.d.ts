export type PtyProcess = {
    pid: number;
    write(data: string | Buffer): void;
    resize(columns: number, rows: number): void;
    kill(signal?: string): void;
    onData(callback: (data: string) => void): void;
    onExit(callback: (event: {
        exitCode: number;
        signal?: number;
    }) => void): void;
};
export type PtySpawnOptions = {
    columns: number;
    cols?: number;
    rows: number;
    cwd?: string;
    env?: NodeJS.ProcessEnv;
};
export type PtyAdapter = {
    spawn(executable: string, args: string[], options: PtySpawnOptions): PtyProcess;
};
export declare function isNodePtyAvailable(): Promise<boolean>;
export declare function loadPtyAdapter(): Promise<PtyAdapter | null>;
