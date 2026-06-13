export declare function missingNativeBuildTools(): string[];
export declare function nativeBuildToolsHint(missing: string[]): string;
export declare function nodePtyReady(): Promise<boolean>;
export declare function printNativeDependencyHint(): Promise<void>;
