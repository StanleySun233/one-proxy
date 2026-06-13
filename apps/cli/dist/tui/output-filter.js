export class ChildOutputFilter {
    pending = '';
    write(data, mainRows) {
        const [ready, pending] = splitPendingEscape(`${this.pending}${data}`);
        this.pending = pending;
        return constrainTuiChildOutput(ready, mainRows);
    }
    clear() {
        this.pending = '';
    }
}
export function constrainTuiChildOutput(data, mainRows) {
    return data
        .replace(/\u001bc/g, () => `${clearMainRowsSequence(mainRows)}${scrollRegionSequence(mainRows)}`)
        .replace(/\u001b\[!p/g, () => scrollRegionSequence(mainRows))
        .replace(/\u001b\[([0-9;]*)J/g, (_match, params) => constrainEraseDisplay(params, mainRows))
        .replace(/\u001b\[([0-9;]*)r/g, (_match, params) => constrainScrollRegion(params, mainRows))
        .replace(/\u001b\[\?([0-9;]+)([hl])/g, (_match, params, mode) => constrainPrivateMode(params, mode, mainRows))
        .replace(/\u001b\[([0-9;]*)([Hf])/g, (_match, params, mode) => constrainCursorPosition(params, mode, mainRows))
        .replace(/\u001b\[([0-9]*)d/g, (_match, params) => constrainVerticalPosition(params, mainRows));
}
export function isTuiControlOutput(data) {
    return /\u001bc|\u001b\[(?:![p]|[0-9;]*[JrHfd]|\?[0-9;]+[hl])/.test(data);
}
export function scrollRegionSequence(bottom, top = 1) {
    return `\u001b[${top};${bottom}r`;
}
function constrainEraseDisplay(params, mainRows) {
    return clearMainRowsSequence(mainRows);
}
function constrainScrollRegion(params, mainRows) {
    const [rawTop = '', rawBottom = ''] = params.split(';');
    const top = clampScrollRegionValue(rawTop, 1, mainRows);
    const bottom = clampScrollRegionValue(rawBottom, mainRows, mainRows);
    return scrollRegionSequence(Math.max(top, bottom), Math.min(top, bottom));
}
function constrainCursorPosition(params, mode, mainRows) {
    const [rawRow = '', rawColumn = ''] = params.split(';');
    return `\u001b[${clampScrollRegionValue(rawRow, 1, mainRows)};${cursorColumn(rawColumn)}${mode}`;
}
function constrainVerticalPosition(params, mainRows) {
    return `\u001b[${clampScrollRegionValue(params, 1, mainRows)}d`;
}
function constrainPrivateMode(params, mode, mainRows) {
    const sequence = privateModeSequence(params, mode);
    return params.split(';').some((value) => value === '47' || value === '1047' || value === '1049')
        ? `${sequence}${scrollRegionSequence(mainRows)}`
        : sequence;
}
function clampScrollRegionValue(value, fallback, mainRows) {
    const parsed = Number.parseInt(value, 10);
    if (!Number.isFinite(parsed) || parsed < 1) {
        return fallback;
    }
    return Math.min(parsed, mainRows);
}
function cursorColumn(value) {
    const parsed = Number.parseInt(value, 10);
    if (!Number.isFinite(parsed) || parsed < 1) {
        return 1;
    }
    return parsed;
}
function privateModeSequence(params, mode) {
    return `\u001b[?${params}${mode}`;
}
function clearMainRowsSequence(mainRows) {
    let output = '\u001b7';
    for (let row = 1; row <= mainRows; row += 1) {
        output += `\u001b[${row};1H\u001b[2K`;
    }
    return `${output}\u001b8`;
}
function splitPendingEscape(data) {
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
function isCompleteEscape(sequence) {
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
//# sourceMappingURL=output-filter.js.map