import { currentTuiCapabilityInput, detectTuiCapability, tuiUnavailableWarning } from './capability.ts';
import { planFooter, renderFooterLines, visibleWidth, type TuiStatusSnapshot } from './footer.ts';
import { isNodePtyAvailable, loadPtyAdapter, type PtyAdapter, type PtyProcess } from './pty.ts';

export type TuiCommand = {
  executable: string;
  args: string[];
  cwd?: string;
  env?: NodeJS.ProcessEnv;
};

export type TuiRuntimeOptions = {
  command: TuiCommand;
  snapshot: TuiStatusSnapshot | (() => TuiStatusSnapshot | Promise<TuiStatusSnapshot>);
  requested: boolean;
  interactive?: boolean;
  json?: boolean;
  ptyAdapter?: PtyAdapter;
};

export type TuiSessionOptions = {
  command: string;
  args: string[];
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  status: TuiStatusSnapshot | (() => TuiStatusSnapshot | Promise<TuiStatusSnapshot>);
  stdin?: RuntimeInput;
  stdout?: RuntimeOutput;
  stderr?: RuntimeOutput;
  ptyAdapter: PtyAdapter;
};

export type TuiRuntimeResult = {
  ran: boolean;
  exitCode?: number;
};

export async function runTuiRuntime(options: TuiRuntimeOptions): Promise<TuiRuntimeResult> {
  const ptyAvailable = Boolean(options.ptyAdapter) || await isNodePtyAvailable();
  const capability = detectTuiCapability(currentTuiCapabilityInput(options.requested, options.interactive ?? true, options.json ?? false, ptyAvailable));
  if (!capability.enabled) {
    if (capability.warn) {
      process.stderr.write(`${tuiUnavailableWarning}\n`);
    }
    return { ran: false };
  }

  const ptyAdapter = options.ptyAdapter ?? await loadPtyAdapter();
  if (!ptyAdapter) {
    process.stderr.write(`${tuiUnavailableWarning}\n`);
    return { ran: false };
  }

  const runtime = new ActiveTuiRuntime(options, ptyAdapter);
  return {
    ran: true,
    exitCode: await runtime.run()
  };
}

export async function runTuiCommand(options: {
  executable: string;
  args: string[];
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  status?: unknown;
}): Promise<{ available: boolean; exitCode?: number }> {
  const result = await runTuiRuntime({
    command: {
      executable: options.executable,
      args: options.args,
      cwd: options.cwd,
      env: options.env
    },
    snapshot: options.status as TuiStatusSnapshot,
    requested: true
  });
  return {
    available: result.ran,
    exitCode: result.exitCode
  };
}

export function runTuiSession(options: TuiSessionOptions): Promise<number> {
  const runtime = new ActiveTuiRuntime({
    command: {
      executable: options.command,
      args: options.args,
      cwd: options.cwd,
      env: options.env
    },
    snapshot: options.status,
    requested: true,
    stdin: options.stdin ?? process.stdin,
    stdout: options.stdout ?? process.stdout,
    stderr: options.stderr ?? process.stderr
  }, options.ptyAdapter);
  return runtime.run();
}

type RuntimeInput = NodeJS.ReadStream & {
  isTTY?: boolean;
  setRawMode?: (mode: boolean) => void;
};

type RuntimeOutput = NodeJS.WriteStream & {
  isTTY?: boolean;
  columns?: number;
  rows?: number;
};

type ActiveTuiRuntimeOptions = TuiRuntimeOptions & {
  stdin?: RuntimeInput;
  stdout?: RuntimeOutput;
  stderr?: RuntimeOutput;
};

class ActiveTuiRuntime {
  private child: PtyProcess | null = null;
  private footerRows = 0;
  private readonly options: ActiveTuiRuntimeOptions;
  private readonly ptyAdapter: PtyAdapter;
  private readonly stdin: RuntimeInput;
  private readonly stdout: RuntimeOutput;

  constructor(options: ActiveTuiRuntimeOptions, ptyAdapter: PtyAdapter) {
    this.options = options;
    this.ptyAdapter = ptyAdapter;
    this.stdin = options.stdin ?? process.stdin;
    this.stdout = options.stdout ?? process.stdout;
  }

  async run(): Promise<number> {
    const plan = planFooter(this.stdout.columns ?? 80, this.stdout.rows ?? 24);
    this.footerRows = plan.rows;
    this.child = this.ptyAdapter.spawn(this.options.command.executable, this.options.command.args, {
      columns: plan.childColumns,
      cols: plan.childColumns,
      rows: plan.childRows,
      cwd: this.options.command.cwd,
      env: this.options.command.env
    });
    this.wireOutput();
    this.wireInput();
    this.wireResize();
    const exit = this.waitForExit();
    await this.redrawFooter();
    return await exit;
  }

  private wireOutput(): void {
    this.child?.onData((data) => {
      this.stdout.write(data);
    });
  }

  private wireInput(): void {
    if (this.stdin.isTTY && this.stdin.setRawMode) {
      this.stdin.setRawMode(true);
    }
    this.stdin.resume();
    this.stdin.on('data', this.forwardInput);
  }

  private wireResize(): void {
    this.stdout.on('resize', this.resize);
  }

  private async waitForExit(): Promise<number> {
    const child = this.child;
    if (!child) {
      return 1;
    }
    return await new Promise((resolve) => {
      child.onExit((event) => {
        this.cleanup();
        resolve(event.signal ? 1 : event.exitCode ?? 0);
      });
    });
  }

  private readonly forwardInput = (data: Buffer): void => {
    this.child?.write(data);
  };

  private readonly resize = (): void => {
    const plan = planFooter(this.stdout.columns ?? 80, this.stdout.rows ?? 24);
    this.footerRows = plan.rows;
    this.child?.resize(plan.childColumns, plan.childRows);
    void this.redrawFooter();
  };

  private async redrawFooter(): Promise<void> {
    const rows = this.stdout.rows ?? 24;
    const columns = this.stdout.columns ?? 80;
    const lines = renderFooterLines(await this.snapshot(), {
      columns,
      rows,
      color: colorEnabled(this.stdout)
    });
    writeFooter(this.stdout, lines, columns, rows);
  }

  private async snapshot(): Promise<TuiStatusSnapshot> {
    return typeof this.options.snapshot === 'function' ? await this.options.snapshot() : this.options.snapshot;
  }

  private cleanup(): void {
    this.stdout.off('resize', this.resize);
    this.stdin.off('data', this.forwardInput);
    if (this.stdin.isTTY && this.stdin.setRawMode) {
      this.stdin.setRawMode(false);
    }
    clearFooter(this.stdout, this.footerRows);
  }
}

function writeFooter(stdout: RuntimeOutput, lines: string[], columns: number, rows: number): void {
  const startRow = Math.max(1, rows - lines.length + 1);
  stdout.write('\u001b7');
  for (let index = 0; index < lines.length; index += 1) {
    stdout.write(`\u001b[${startRow + index};1H\u001b[2K${lines[index]}`);
    const width = visibleWidth(lines[index]);
    if (width < columns) {
      stdout.write(' '.repeat(columns - width));
    }
  }
  stdout.write('\u001b8');
}

function clearFooter(stdout: RuntimeOutput, rows: number): void {
  const terminalRows = stdout.rows ?? 24;
  const startRow = Math.max(1, terminalRows - rows + 1);
  stdout.write('\u001b7');
  for (let index = 0; index < rows; index += 1) {
    stdout.write(`\u001b[${startRow + index};1H\u001b[2K`);
  }
  stdout.write('\u001b8');
}

function colorEnabled(stdout: RuntimeOutput): boolean {
  return Boolean(stdout.isTTY && !process.env.NO_COLOR && process.env.TERM !== 'dumb');
}
