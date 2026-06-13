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
export declare function footerRowsForTerminal(terminalRows: number): number;
export type FormatFooterOptions = {
    columns: number;
    footerRows: number;
    color: boolean;
};
export type FormattedFooter = {
    lines: string[];
};
export declare function formatFooter(snapshot: TuiStatusSnapshot, options: FormatFooterOptions): FormattedFooter;
export declare function planFooter(terminalColumns: number, terminalRows: number): FooterPlan;
export declare function renderFooterLines(snapshot: TuiStatusSnapshot, options: RenderFooterOptions): string[];
export declare function renderStatusLine(snapshot: TuiStatusSnapshot, color: boolean): string;
export declare function renderTotalsLine(snapshot: TuiStatusSnapshot, color: boolean): string;
export declare function renderPathLine(snapshot: TuiStatusSnapshot, color: boolean): string;
export declare function formatPathText(path: Pick<TuiStatusSnapshot['path'], 'nodes'>): string;
export declare function latencyStyleName(pingMs: number | null): 'gray' | 'mint' | 'butter' | 'coral';
export declare function visibleWidth(value: string): number;
export declare function stripAnsi(value: string): string;
export declare function truncateVisible(value: string, columns: number): string;
export declare function truncateMiddleVisible(value: string, columns: number): string;
export declare function rightAlign(value: string, columns: number): string;
