export async function isNodePtyAvailable() {
    return (await loadNodePty()) !== null;
}
export async function loadPtyAdapter() {
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
async function loadNodePty() {
    try {
        const dynamicImport = new Function('specifier', 'return import(specifier)');
        return await dynamicImport('node-pty');
    }
    catch {
        return null;
    }
}
//# sourceMappingURL=pty.js.map