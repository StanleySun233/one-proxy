import { currentTuiCapabilityInput, detectTuiCapability, tuiUnavailableWarning } from "./capability.js";
import { planFooter, renderFooterLines, visibleWidth } from "./footer.js";
import { isNodePtyAvailable, loadPtyAdapter } from "./pty.js";
export async function runTuiRuntime(options) {
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
export async function runTuiCommand(options) {
    const result = await runTuiRuntime({
        command: {
            executable: options.executable,
            args: options.args,
            cwd: options.cwd,
            env: options.env
        },
        snapshot: options.status,
        requested: true
    });
    return {
        available: result.ran,
        exitCode: result.exitCode
    };
}
export function runTuiSession(options) {
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
class ActiveTuiRuntime {
    child = null;
    footerRows = 0;
    options;
    ptyAdapter;
    stdin;
    stdout;
    constructor(options, ptyAdapter) {
        this.options = options;
        this.ptyAdapter = ptyAdapter;
        this.stdin = options.stdin ?? process.stdin;
        this.stdout = options.stdout ?? process.stdout;
    }
    async run() {
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
    wireOutput() {
        this.child?.onData((data) => {
            this.stdout.write(data);
        });
    }
    wireInput() {
        if (this.stdin.isTTY && this.stdin.setRawMode) {
            this.stdin.setRawMode(true);
        }
        this.stdin.resume();
        this.stdin.on('data', this.forwardInput);
    }
    wireResize() {
        this.stdout.on('resize', this.resize);
    }
    async waitForExit() {
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
    forwardInput = (data) => {
        this.child?.write(data);
    };
    resize = () => {
        const plan = planFooter(this.stdout.columns ?? 80, this.stdout.rows ?? 24);
        this.footerRows = plan.rows;
        this.child?.resize(plan.childColumns, plan.childRows);
        void this.redrawFooter();
    };
    async redrawFooter() {
        const rows = this.stdout.rows ?? 24;
        const columns = this.stdout.columns ?? 80;
        const lines = renderFooterLines(await this.snapshot(), {
            columns,
            rows,
            color: colorEnabled(this.stdout)
        });
        writeFooter(this.stdout, lines, columns, rows);
    }
    async snapshot() {
        return typeof this.options.snapshot === 'function' ? await this.options.snapshot() : this.options.snapshot;
    }
    cleanup() {
        this.stdout.off('resize', this.resize);
        this.stdin.off('data', this.forwardInput);
        if (this.stdin.isTTY && this.stdin.setRawMode) {
            this.stdin.setRawMode(false);
        }
        clearFooter(this.stdout, this.footerRows);
    }
}
function writeFooter(stdout, lines, columns, rows) {
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
function clearFooter(stdout, rows) {
    const terminalRows = stdout.rows ?? 24;
    const startRow = Math.max(1, terminalRows - rows + 1);
    stdout.write('\u001b7');
    for (let index = 0; index < rows; index += 1) {
        stdout.write(`\u001b[${startRow + index};1H\u001b[2K`);
    }
    stdout.write('\u001b8');
}
function colorEnabled(stdout) {
    return Boolean(stdout.isTTY && !process.env.NO_COLOR && process.env.TERM !== 'dumb');
}
//# sourceMappingURL=runtime.js.map