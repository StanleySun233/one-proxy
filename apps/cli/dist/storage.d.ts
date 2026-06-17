export type LocalOverrides = {
    direct: string[];
    proxy: string[];
};
export type OneProxyConfig = {
    schemaVersion: number;
    profileName?: string;
    controlPlaneUrl?: string;
    activeTenantId?: string;
    activeAccessPathId?: string;
    ignoredCliVersion?: string;
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
export type BootstrapNode = {
    id: string;
    name: string;
    mode: string;
    scopeKey: string;
    parentNodeId: string;
    enabled: boolean;
    status: string;
    publicHost?: string;
    publicPort?: number;
};
export type TopologyHop = {
    nodeId: string;
    nodeName: string;
    mode: string;
    scopeKey: string;
    publicHost?: string;
    publicPort?: number;
    transport: string;
};
export type AccessPathSnapshot = {
    id: string;
    name: string;
    chainId: string;
    mode: string;
    protocol: string;
    serviceType: string;
    targetNodeId: string;
    entryNodeId: string;
    relayNodeIds: string[];
    listenHost: string;
    listenPort: number;
    targetProtocol: string;
    targetHost: string;
    targetPort: number;
    targetSni: string;
    tlsMode: string;
    authMode: 'proxy_token';
    enabled: boolean;
    options: Record<string, string>;
    topology: TopologyHop[];
    health: {
        status: string;
        reason: string;
        checkedAt: string;
    };
};
export type RouteSnapshot = {
    id: string;
    priority: number;
    matchType: 'domain' | 'domain_suffix' | 'ip' | 'ip_cidr' | 'protocol' | 'default';
    matchValue: string;
    actionType: 'chain' | 'direct' | 'deny';
    chainId: string;
    accessPathId: string;
    destinationScope: string;
    enabled: boolean;
    topology: TopologyHop[];
};
export type RouteEvaluationContract = {
    defaultClientMode: 'direct';
    defaultNodeMode: 'deny';
    ruleOrder: 'priority_asc_then_id_asc';
    noMatchNodeDenyReason: 'route_not_found';
    supportedMatchTypes: RouteSnapshot['matchType'][];
    supportedActions: RouteSnapshot['actionType'][];
};
export type OneProxyState = {
    schemaVersion: number;
    bootstrap?: {
        tenantId?: string;
        accessPathId?: string;
        entryNodes?: EntryNode[];
    };
    policyRevision?: string;
    fetchedAt?: string;
    nodes?: BootstrapNode[];
    accessPaths?: AccessPathSnapshot[];
    routes?: RouteSnapshot[];
    routeEvaluation?: RouteEvaluationContract;
};
export type DaemonBindings = {
    host: string;
    httpPort: number;
    httpsPort: number;
    ipcPort?: number;
    proxyOnlyPort?: number;
};
export type DaemonMetadata = {
    schemaVersion: number;
    pid: number;
    startedAt: string;
    lastHeartbeatAt: string;
    controlPlaneUrl?: string;
    tenantId?: string;
    accessPathId?: string;
    policyRevision?: string;
    bindings: DaemonBindings;
    portSelection?: {
        candidatePorts: number[];
        selectedPair: [number, number];
        excludedCommonPorts: number[];
    };
    idleTimeoutSeconds?: number;
    daemonSecret: string;
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
