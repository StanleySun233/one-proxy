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

class ActiveTuiRuntime {
  private child: PtyProcess | null = null;
  private footerRows = 0;

  constructor(private readonly options: TuiRuntimeOptions, private readonly ptyAdapter: PtyAdapter) {}

  async run(): Promise<number> {
    const plan = planFooter(process.stdout.columns ?? 80, process.stdout.rows ?? 24);
    this.footerRows = plan.rows;
    this.child = this.ptyAdapter.spawn(this.options.command.executable, this.options.command.args, {
      columns: plan.childColumns,
      rows: plan.childRows,
      cwd: this.options.command.cwd,
      env: this.options.command.env
    });
    this.wireOutput();
    this.wireInput();
    this.wireResize();
    await this.redrawFooter();
    return await this.waitForExit();
  }

  private wireOutput(): void {
    this.child?.onData((data) => {
      process.stdout.write(data);
    });
  }

  private wireInput(): void {
    if (process.stdin.isTTY) {
      process.stdin.setRawMode(true);
    }
    process.stdin.resume();
    process.stdin.on('data', this.forwardInput);
  }

  private wireResize(): void {
    process.stdout.on('resize', this.resize);
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
    const plan = planFooter(process.stdout.columns ?? 80, process.stdout.rows ?? 24);
    this.footerRows = plan.rows;
    this.child?.resize(plan.childColumns, plan.childRows);
    void this.redrawFooter();
  };

  private async redrawFooter(): Promise<void> {
    const rows = process.stdout.rows ?? 24;
    const columns = process.stdout.columns ?? 80;
    const lines = renderFooterLines(await this.snapshot(), {
      columns,
      rows,
      color: colorEnabled()
    });
    writeFooter(lines, columns, rows);
  }

  private async snapshot(): Promise<TuiStatusSnapshot> {
    return typeof this.options.snapshot === 'function' ? await this.options.snapshot() : this.options.snapshot;
  }

  private cleanup(): void {
    process.stdout.off('resize', this.resize);
    process.stdin.off('data', this.forwardInput);
    if (process.stdin.isTTY) {
      process.stdin.setRawMode(false);
    }
    clearFooter(this.footerRows);
  }
}

function writeFooter(lines: string[], columns: number, rows: number): void {
  const startRow = Math.max(1, rows - lines.length + 1);
  process.stdout.write('\u001b7');
  for (let index = 0; index < lines.length; index += 1) {
    process.stdout.write(`\u001b[${startRow + index};1H\u001b[2K${lines[index]}`);
    const width = visibleWidth(lines[index]);
    if (width < columns) {
      process.stdout.write(' '.repeat(columns - width));
    }
  }
  process.stdout.write('\u001b8');
}

function clearFooter(rows: number): void {
  const terminalRows = process.stdout.rows ?? 24;
  const startRow = Math.max(1, terminalRows - rows + 1);
  process.stdout.write('\u001b7');
  for (let index = 0; index < rows; index += 1) {
    process.stdout.write(`\u001b[${startRow + index};1H\u001b[2K`);
  }
  process.stdout.write('\u001b8');
}

function colorEnabled(): boolean {
  return Boolean(process.stdout.isTTY && !process.env.NO_COLOR && process.env.TERM !== 'dumb');
}
