export type PortSelection = {
    candidatePorts: number[];
    selectedPair: [number, number];
    excludedCommonPorts: number[];
};
export declare const excludedCommonPorts: number[];
export declare function selectProxyPorts(): Promise<PortSelection>;
export declare function scanAvailableCandidatePorts(start?: number, end?: number): Promise<number[]>;
export declare function isUsablePort(port: number): Promise<boolean>;
