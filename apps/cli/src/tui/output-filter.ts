export class ChildOutputFilter {
  private pending = '';

  write(data: string, mainRows: number): string {
    const [ready, pending] = splitPendingEscape(`${this.pending}${data}`);
    this.pending = pending;
    return constrainTuiChildOutput(ready, mainRows);
  }

  clear(): void {
    this.pending = '';
  }
}

export function constrainTuiChildOutput(data: string, mainRows: number): string {
  return data
    .replace(/\u001bc/g, () => `${clearMainRowsSequence(mainRows)}${scrollRegionSequence(mainRows)}`)
    .replace(/\u001b\[!p/g, () => scrollRegionSequence(mainRows))
    .replace(/\u001b\[([0-9;]*)J/g, (_match, params: string) => constrainEraseDisplay(params, mainRows))
    .replace(/\u001b\[([0-9;]*)r/g, (_match, params: string) => constrainScrollRegion(params, mainRows))
    .replace(/\u001b\[\?([0-9;]+)([hl])/g, (_match, params: string, mode: string) => constrainPrivateMode(params, mode, mainRows))
    .replace(/\u001b\[([0-9;]*)([Hf])/g, (_match, params: string, mode: string) => constrainCursorPosition(params, mode, mainRows))
    .replace(/\u001b\[([0-9]*)d/g, (_match, params: string) => constrainVerticalPosition(params, mainRows));
}

export function isTuiControlOutput(data: string): boolean {
  return /\u001bc|\u001b\[(?:![p]|[0-9;]*[JrHfd]|\?[0-9;]+[hl])/.test(data);
}

export function scrollRegionSequence(bottom: number, top = 1): string {
  return `\u001b[${top};${bottom}r`;
}

function constrainEraseDisplay(params: string, mainRows: number): string {
  return clearMainRowsSequence(mainRows);
}

function constrainScrollRegion(params: string, mainRows: number): string {
  const [rawTop = '', rawBottom = ''] = params.split(';');
  const top = clampScrollRegionValue(rawTop, 1, mainRows);
  const bottom = clampScrollRegionValue(rawBottom, mainRows, mainRows);
  return scrollRegionSequence(Math.max(top, bottom), Math.min(top, bottom));
}

function constrainCursorPosition(params: string, mode: string, mainRows: number): string {
  const [rawRow = '', rawColumn = ''] = params.split(';');
  return `\u001b[${clampScrollRegionValue(rawRow, 1, mainRows)};${cursorColumn(rawColumn)}${mode}`;
}

function constrainVerticalPosition(params: string, mainRows: number): string {
  return `\u001b[${clampScrollRegionValue(params, 1, mainRows)}d`;
}

function constrainPrivateMode(params: string, mode: string, mainRows: number): string {
  const sequence = privateModeSequence(params, mode);
  return params.split(';').some((value) => value === '47' || value === '1047' || value === '1049')
    ? `${sequence}${scrollRegionSequence(mainRows)}`
    : sequence;
}

function clampScrollRegionValue(value: string, fallback: number, mainRows: number): number {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isFinite(parsed) || parsed < 1) {
    return fallback;
  }
  return Math.min(parsed, mainRows);
}

function cursorColumn(value: string): number {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isFinite(parsed) || parsed < 1) {
    return 1;
  }
  return parsed;
}

function privateModeSequence(params: string, mode: string): string {
  return `\u001b[?${params}${mode}`;
}

function clearMainRowsSequence(mainRows: number): string {
  let output = '\u001b7';
  for (let row = 1; row <= mainRows; row += 1) {
    output += `\u001b[${row};1H\u001b[2K`;
  }
  return `${output}\u001b8`;
}

function splitPendingEscape(data: string): [string, string] {
  const index = data.lastIndexOf('\u001b');
  if (index === -1) {
    return [data, ''];
  }
  const suffix = data.slice(index);
  if (isCompleteEscape(suffix)) {
    return [data, ''];
  }
  return [data.slice(0, index), suffix];
}

function isCompleteEscape(sequence: string): boolean {
  if (sequence.length === 1) {
    return false;
  }
  if (sequence[1] !== '[') {
    return true;
  }
  for (let index = 2; index < sequence.length; index += 1) {
    const code = sequence.charCodeAt(index);
    if (code >= 0x40 && code <= 0x7e) {
      return true;
    }
  }
  return false;
}
