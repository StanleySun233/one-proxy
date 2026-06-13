export declare class ChildOutputFilter {
    private pending;
    write(data: string, mainRows: number): string;
    clear(): void;
}
export declare function constrainTuiChildOutput(data: string, mainRows: number): string;
export declare function isTuiControlOutput(data: string): boolean;
export declare function scrollRegionSequence(bottom: number, top?: number): string;
