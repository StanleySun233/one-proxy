export type PtyProcess = {
  pid: number;
  write(data: string | Buffer): void;
  resize(columns: number, rows: number): void;
  kill(signal?: string): void;
  onData(callback: (data: string) => void): void;
  onExit(callback: (event: { exitCode: number; signal?: number }) => void): void;
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

type NodePtyModule = {
  spawn(executable: string, args: string[], options: {
    cols: number;
    rows: number;
    cwd?: string;
    env?: NodeJS.ProcessEnv;
  }): {
    pid: number;
    write(data: string): void;
    resize(columns: number, rows: number): void;
    kill(signal?: string): void;
    onData(callback: (data: string) => void): void;
    onExit(callback: (event: { exitCode: number; signal?: number }) => void): void;
  };
};

export async function isNodePtyAvailable(): Promise<boolean> {
  return (await loadNodePty()) !== null;
}

export async function loadPtyAdapter(): Promise<PtyAdapter | null> {
  const nodePty = await loadNodePty();
  if (!nodePty) {
    return null;
  }
  return {
    spawn(executable, args, options) {
      const child = nodePty.spawn(executable, args, {
        cols: options.columns,
        rows: options.rows,
        cwd: options.cwd,
        env: options.env
      });
      return {
        pid: child.pid,
        write(data) {
          child.write(typeof data === 'string' ? data : data.toString('utf8'));
        },
        resize(columns, rows) {
          child.resize(columns, rows);
        },
        kill(signal) {
          child.kill(signal);
        },
        onData(callback) {
          child.onData(callback);
        },
        onExit(callback) {
          child.onExit(callback);
        }
      };
    }
  };
}

async function loadNodePty(): Promise<NodePtyModule | null> {
  try {
    const dynamicImport = new Function('specifier', 'return import(specifier)') as (specifier: string) => Promise<unknown>;
    return await dynamicImport('node-pty') as NodePtyModule;
  } catch {
    return null;
  }
}
