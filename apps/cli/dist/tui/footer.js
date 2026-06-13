const ansiPattern = /\u001b\[[0-9;]*m/g;
const minimumMainRows = 8;
const reset = '\u001b[0m';
const darkForeground = '\u001b[38;2;26;32;44m';
const styles = {
    mint: segmentStyle(186, 242, 214),
    lavender: segmentStyle(218, 207, 255),
    butter: segmentStyle(255, 236, 169),
    coral: segmentStyle(255, 190, 171),
    gray: segmentStyle(217, 222, 226),
    blue: foregroundStyle(93, 148, 255),
    cyan: foregroundStyle(62, 194, 211),
    muted: foregroundStyle(132, 140, 148),
    failure: foregroundStyle(220, 72, 77),
    fallback: foregroundStyle(180, 132, 0)
};
export function footerRowsForTerminal(terminalRows) {
    if (terminalRows >= 18) {
        return 3;
    }
    if (terminalRows >= 14) {
        return 2;
    }
    if (terminalRows < 10) {
        return 0;
    }
    return 1;
}
export function formatFooter(snapshot, options) {
    const lines = [];
    if (options.footerRows >= 1) {
        lines.push(truncateVisible(renderStatusLine(snapshot, options.color), options.columns));
    }
    if (options.footerRows >= 2) {
        lines.push(rightAlign(renderTotalsLine(snapshot, options.color), options.columns));
    }
    if (options.footerRows >= 3) {
        lines.push(truncateMiddleVisible(renderPathLine(snapshot, options.color), options.columns));
    }
    return { lines };
}
export function planFooter(terminalColumns, terminalRows) {
    const rows = footerRowsForTerminal(terminalRows);
    const childRows = Math.max(terminalRows - rows, minimumMainRows);
    return {
        rows,
        startRow: Math.max(1, terminalRows - rows + 1),
        terminalColumns,
        terminalRows,
        childColumns: terminalColumns,
        childRows
    };
}
export function renderFooterLines(snapshot, options) {
    return formatFooter(snapshot, {
        columns: options.columns,
        footerRows: footerRowsForTerminal(options.rows),
        color: options.color
    }).lines;
}
export function renderStatusLine(snapshot, color) {
    return [
        styleSegment(snapshot.account || 'not logged in', styles.mint, color),
        styleSegment(snapshot.tenant || 'none', styles.lavender, color),
        styleSegment(formatPing(snapshot.pingMs), pingStyle(snapshot.pingMs), color)
    ].join('  ');
}
export function renderTotalsLine(snapshot, color) {
    return [
        'Total',
        colorize('↑', styles.blue, color),
        formatBytes(snapshot.uploadBytes),
        '|',
        colorize('↓', styles.cyan, color),
        formatBytes(snapshot.downloadBytes)
    ].join(' ');
}
export function renderPathLine(snapshot, color) {
    const nodes = snapshot.path.nodes
        .map((node, index) => {
        const label = (node.name || node.id).trim();
        return label ? styleSegment(label, pathNodeStyle(node, index), color) : '';
    })
        .filter(Boolean);
    if (nodes.length === 0) {
        return '';
    }
    const separator = colorize('-', styles.muted, color);
    const line = nodes.join(separator);
    if (!snapshot.path.fallbackReason) {
        return line;
    }
    return `${line}${separator}${colorize(snapshot.path.fallbackReason, styles.fallback, color)}`;
}
export function formatPathText(path) {
    return path.nodes
        .map((node) => (node.name || node.id).trim())
        .filter(Boolean)
        .join('-');
}
export function latencyStyleName(pingMs) {
    if (pingMs === null) {
        return 'gray';
    }
    if (pingMs < 100) {
        return 'mint';
    }
    if (pingMs < 300) {
        return 'butter';
    }
    return 'coral';
}
export function visibleWidth(value) {
    return stripAnsi(value).length;
}
export function stripAnsi(value) {
    return value.replace(ansiPattern, '');
}
export function truncateVisible(value, columns) {
    if (columns <= 0 || visibleWidth(value) <= columns) {
        return columns <= 0 ? '' : value;
    }
    const styled = ansiPattern.test(value);
    ansiPattern.lastIndex = 0;
    let output = '';
    let width = 0;
    for (let index = 0; index < value.length && width < columns;) {
        const ansi = value.slice(index).match(/^\u001b\[[0-9;]*m/);
        if (ansi) {
            output += ansi[0];
            index += ansi[0].length;
            continue;
        }
        output += value[index];
        width += 1;
        index += 1;
    }
    return styled && !output.endsWith(reset) ? `${output}${reset}` : output;
}
export function truncateMiddleVisible(value, columns) {
    if (columns <= 0) {
        return '';
    }
    if (visibleWidth(value) <= columns) {
        return value;
    }
    if (columns <= 1) {
        return '…';
    }
    const plain = stripAnsi(value);
    const left = Math.ceil((columns - 1) / 2);
    const right = Math.floor((columns - 1) / 2);
    return `${plain.slice(0, left)}…${plain.slice(plain.length - right)}`;
}
export function rightAlign(value, columns) {
    const width = visibleWidth(value);
    if (width >= columns) {
        return truncateVisible(value, columns);
    }
    return `${' '.repeat(columns - width)}${value}`;
}
function formatPing(pingMs) {
    return pingMs === null ? '--' : `${Math.round(pingMs)}ms`;
}
function formatBytes(bytes) {
    if (bytes === null) {
        return '--';
    }
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let value = bytes;
    let unit = 0;
    while (value >= 1000 && unit < units.length - 1) {
        value /= 1000;
        unit += 1;
    }
    if (unit === 0) {
        return `${value} ${units[unit]}`;
    }
    return `${value.toFixed(value >= 10 ? 1 : 2)} ${units[unit]}`;
}
function pingStyle(pingMs) {
    return styles[latencyStyleName(pingMs)];
}
function pathNodeStyle(node, index) {
    if (node.kind === 'user') {
        return styles.mint;
    }
    if (node.kind === 'web') {
        return styles.coral;
    }
    return index % 2 === 0 ? styles.lavender : styles.butter;
}
function styleSegment(value, style, color) {
    return color ? colorize(` ${value} `, style, true) : value;
}
function colorize(value, style, color) {
    return color ? `${style.open}${value}${style.close}` : value;
}
function segmentStyle(red, green, blue) {
    return {
        open: `\u001b[48;2;${red};${green};${blue}m${darkForeground}`,
        close: reset
    };
}
function foregroundStyle(red, green, blue) {
    return {
        open: `\u001b[38;2;${red};${green};${blue}m`,
        close: reset
    };
}
//# sourceMappingURL=footer.js.map