export type TuiPathNode = {
  id: string;
  name: string;
  kind: 'user' | 'node' | 'web' | string;
  transport: string;
};

export type TuiStatusSnapshot = {
  account: string;
  tenant: string;
  pingMs: number | null;
  uploadBytes: number | null;
  downloadBytes: number | null;
  path: {
    mode: string;
    transport: string;
    fallbackReason: string;
    nodes: TuiPathNode[];
  };
};

export type FooterPlan = {
  rows: number;
  startRow: number;
  terminalColumns: number;
  terminalRows: number;
  childColumns: number;
  childRows: number;
};

export type RenderFooterOptions = {
  columns: number;
  rows: number;
  color: boolean;
};

type Style = {
  open: string;
  close: string;
};

const ansiPattern = /\u001b\[[0-9;]*m/g;
const minimumMainRows = 8;
const reset = '\u001b[0m';
const darkForeground = '\u001b[38;2;26;32;44m';
const styles = {
  mint: segmentStyle(186, 242, 214),
  lavender: segmentStyle(218, 207, 255),
  yellow: segmentStyle(255, 236, 169),
  coral: segmentStyle(255, 190, 171),
  gray: segmentStyle(217, 222, 226),
  blue: foregroundStyle(93, 148, 255),
  cyan: foregroundStyle(62, 194, 211),
  muted: foregroundStyle(132, 140, 148),
  failure: foregroundStyle(220, 72, 77),
  fallback: foregroundStyle(180, 132, 0)
};

export function footerRowsForTerminal(terminalRows: number): number {
  if (terminalRows >= 18) {
    return 3;
  }
  if (terminalRows >= 14) {
    return 2;
  }
  return 1;
}

export function planFooter(terminalColumns: number, terminalRows: number): FooterPlan {
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

export function renderFooterLines(snapshot: TuiStatusSnapshot, options: RenderFooterOptions): string[] {
  const rowCount = footerRowsForTerminal(options.rows);
  const lines = [truncateVisible(renderStatusLine(snapshot, options.color), options.columns)];
  if (rowCount >= 2) {
    lines.push(rightAlign(renderTotalsLine(snapshot, options.color), options.columns));
  }
  if (rowCount >= 3) {
    lines.push(truncateMiddleVisible(renderPathLine(snapshot, options.color), options.columns));
  }
  return lines;
}

export function renderStatusLine(snapshot: TuiStatusSnapshot, color: boolean): string {
  return [
    styleSegment(snapshot.account || 'not logged in', styles.mint, color),
    styleSegment(snapshot.tenant || 'none', styles.lavender, color),
    styleSegment(formatPing(snapshot.pingMs), pingStyle(snapshot.pingMs), color)
  ].join('  ');
}

export function renderTotalsLine(snapshot: TuiStatusSnapshot, color: boolean): string {
  return [
    'Total',
    colorize('↑', styles.blue, color),
    formatBytes(snapshot.uploadBytes),
    '|',
    colorize('↓', styles.cyan, color),
    formatBytes(snapshot.downloadBytes)
  ].join(' ');
}

export function renderPathLine(snapshot: TuiStatusSnapshot, color: boolean): string {
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

export function visibleWidth(value: string): number {
  return stripAnsi(value).length;
}

export function stripAnsi(value: string): string {
  return value.replace(ansiPattern, '');
}

export function truncateVisible(value: string, columns: number): string {
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

export function truncateMiddleVisible(value: string, columns: number): string {
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

export function rightAlign(value: string, columns: number): string {
  const width = visibleWidth(value);
  if (width >= columns) {
    return truncateVisible(value, columns);
  }
  return `${' '.repeat(columns - width)}${value}`;
}

function formatPing(pingMs: number | null): string {
  return pingMs === null ? '--' : `${Math.round(pingMs)}ms`;
}

function formatBytes(bytes: number | null): string {
  if (bytes === null) {
    return '--';
  }
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  if (unit === 0) {
    return `${value} ${units[unit]}`;
  }
  return `${value.toFixed(value >= 10 ? 1 : 2)} ${units[unit]}`;
}

function pingStyle(pingMs: number | null): Style {
  if (pingMs === null) {
    return styles.gray;
  }
  if (pingMs < 100) {
    return styles.mint;
  }
  if (pingMs < 300) {
    return styles.yellow;
  }
  return styles.coral;
}

function pathNodeStyle(node: TuiPathNode, index: number): Style {
  if (node.kind === 'user') {
    return styles.mint;
  }
  if (node.kind === 'web') {
    return styles.coral;
  }
  return index % 2 === 0 ? styles.lavender : styles.yellow;
}

function styleSegment(value: string, style: Style, color: boolean): string {
  return colorize(` ${value} `, style, color);
}

function colorize(value: string, style: Style, color: boolean): string {
  return color ? `${style.open}${value}${style.close}` : value;
}

function segmentStyle(red: number, green: number, blue: number): Style {
  return {
    open: `\u001b[48;2;${red};${green};${blue}m${darkForeground}`,
    close: reset
  };
}

function foregroundStyle(red: number, green: number, blue: number): Style {
  return {
    open: `\u001b[38;2;${red};${green};${blue}m`,
    close: reset
  };
}
