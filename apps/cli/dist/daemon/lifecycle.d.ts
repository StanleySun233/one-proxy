import * as http from 'node:http';
import * as net from 'node:net';
import { type PortSelection } from './port-selection.ts';
export type LocalOverrides = {
    direct: string[];
    proxy: string[];
};
export type OneProxyConfig = {
    schemaVersion: number;
    controlPlaneUrl?: string;
    activeTenantId?: string;
    activeGroupId?: string;
    overrides?: Partial<LocalOverrides>;
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
    routeGroups?: RouteGroup[];
};
export type OneProxyTokens = {
    schemaVersion: number;
    account?: {
        id: string;
        email: string;
    };
    accessToken?: string;
    refreshToken?: string;
    proxyToken?: string;
    accessTokenExpiresAt?: string;
    refreshTokenExpiresAt?: string;
    proxyTokenExpiresAt?: string;
};
export type DaemonBindings = {
    host: string;
    httpPort: number;
    httpsPort: number;
    ipcPort: number;
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
    portSelection: PortSelection;
    idleTimeoutSeconds: number;
};
export type DaemonHealth = {
    ok: boolean;
    pid: number;
    startedAt: string;
    lastHeartbeatAt: string;
    bindings: DaemonBindings;
    portSelection: PortSelection;
    policyRevision?: string;
};
export type DaemonRuntime = {
    metadata: DaemonMetadata;
    ipcServer: http.Server;
    proxyServers?: {
        close: () => Promise<void>;
    };
    close: () => Promise<void>;
};
export declare const loopbackHost = "127.0.0.1";
export declare const defaultIdleTimeoutSeconds = 300;
export declare function dataRoot(): string;
export declare function storagePath(name: 'config' | 'state' | 'tokens' | 'daemon' | 'log'): string;
export declare function ensureDataRoot(): Promise<void>;
export declare function readJsonFile<T>(filePath: string): Promise<T | null>;
export declare function writeJsonFile(filePath: string, value: unknown, mode?: number): Promise<void>;
export declare function normalizeConfig(config: OneProxyConfig | null): OneProxyConfig;
export declare function readConfig(): Promise<OneProxyConfig>;
export declare function readState(): Promise<OneProxyState>;
export declare function readTokens(): Promise<OneProxyTokens | null>;
export declare function readDaemonMetadata(): Promise<DaemonMetadata | null>;
export declare function writeDaemonMetadata(metadata: DaemonMetadata): Promise<void>;
export declare function appendLog(message: string): Promise<void>;
export declare function allocateLoopbackPort(requestedPort?: number): Promise<number>;
export type ResolvedBindings = {
    bindings: DaemonBindings;
    portSelection: PortSelection;
};
export declare function resolveBindings(config?: OneProxyConfig): Promise<ResolvedBindings>;
export declare function buildDaemonMetadata(resolved: ResolvedBindings): Promise<DaemonMetadata>;
export declare function healthFromMetadata(metadata: DaemonMetadata): DaemonHealth;
export declare function listenHttpServer(server: http.Server, port: number): Promise<void>;
export declare function closeServer(server: http.Server | net.Server): Promise<void>;
export declare function createIpcServer(metadata: DaemonMetadata, handlers: Record<string, (body: unknown) => Promise<unknown>>): http.Server<typeof http.IncomingMessage, typeof http.ServerResponse>;
export declare function startDaemonRuntime(handlers?: Record<string, (body: unknown) => Promise<unknown>>): Promise<DaemonRuntime>;
export declare function defaultDaemonHandlers(): Record<string, (body: unknown) => Promise<unknown>>;
export declare function ensureDaemon(): Promise<{
    metadata: DaemonMetadata;
}>;
export declare function serveDaemon(): Promise<void>;
export declare function probeDaemon(metadata?: DaemonMetadata | null): Promise<DaemonHealth | null>;
