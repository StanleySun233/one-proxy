export type LocalOverrides = {
    direct: string[];
    proxy: string[];
};
export type OneProxyConfig = {
    schemaVersion: number;
    profileName?: string;
    controlPlaneUrl?: string;
    activeTenantId?: string;
    activeGroupId?: string;
    overrides: LocalOverrides;
};
export type Account = {
    id: string;
    email?: string;
    account?: string;
};
export type OneProxyTokens = {
    schemaVersion: number;
    account?: Account;
    accessToken?: string;
    refreshToken?: string;
    proxyToken?: string;
    accessTokenExpiresAt?: string;
    refreshTokenExpiresAt?: string;
    proxyTokenExpiresAt?: string;
};
export type EntryNode = {
    id: string;
    host: string;
    port: number;
    protocol: string;
};
export type RouteRule = {
    id: string;
    type: 'domain' | 'suffix' | 'cidr' | 'wildcard';
    pattern: string;
    mode: 'direct' | 'proxy';
};
export type RouteGroup = {
    id: string;
    tenantId: string;
    name?: string;
    rules: RouteRule[];
};
export type OneProxyState = {
    schemaVersion: number;
    bootstrap?: {
        tenantId?: string;
        groupId?: string;
        entryNodes?: EntryNode[];
    };
    policyRevision?: string;
    fetchedAt?: string;
    routeGroups: RouteGroup[];
};
export type DaemonBindings = {
    host: string;
    httpPort: number;
    httpsPort: number;
    ipcPort?: number;
};
export type DaemonMetadata = {
    schemaVersion: number;
    pid: number;
    startedAt: string;
    lastHeartbeatAt: string;
    controlPlaneUrl?: string;
    tenantId?: string;
    groupId?: string;
    policyRevision?: string;
    bindings: DaemonBindings;
    portSelection?: {
        candidatePorts: number[];
        selectedPair: [number, number];
        excludedCommonPorts: number[];
    };
    idleTimeoutSeconds?: number;
};
export declare const loopbackHost = "127.0.0.1";
export type ProfileRecord = {
    name: string;
    controlPlaneUrl: string;
};
export type ProfilesIndex = {
    schemaVersion: number;
    activeProfile?: string;
    profiles: Record<string, ProfileRecord>;
};
export declare function oneProxyHome(): string;
export declare function profilesFile(): string;
export declare function activeProfileName(): string;
export declare function profileRoot(name?: string): string;
export declare function storageFile(name: 'config' | 'state' | 'tokens' | 'daemon' | 'log'): string;
export declare function ensureStorageRoot(): Promise<void>;
export declare function ensureProfileRoot(): Promise<void>;
export declare function defaultConfig(): OneProxyConfig;
export declare function readConfig(): Promise<OneProxyConfig>;
export declare function writeConfig(config: OneProxyConfig): Promise<void>;
export declare function readProfilesIndex(): Promise<ProfilesIndex>;
export declare function writeProfilesIndex(index: ProfilesIndex): Promise<void>;
export declare function profileKey(name: string): string;
export declare function addProfile(name: string, controlPlaneUrl: string): Promise<ProfileRecord>;
export declare function useProfile(name: string): Promise<ProfileRecord>;
export declare function readTokens(): Promise<OneProxyTokens | null>;
export declare function writeTokens(tokens: OneProxyTokens): Promise<void>;
export declare function clearTokens(): Promise<void>;
export declare function readState(): Promise<OneProxyState>;
export declare function writeState(state: OneProxyState): Promise<void>;
export declare function readDaemonMetadata(): Promise<DaemonMetadata | null>;
export declare function writeDaemonMetadata(metadata: DaemonMetadata): Promise<void>;
export declare function appendLog(message: string): Promise<void>;
export declare function isLoopbackPortAvailable(port: number): Promise<boolean>;
export declare function scanAvailableProxyPortPairs(): Promise<Array<[number, number]>>;
export declare function processIsRunning(pid: number): boolean;
