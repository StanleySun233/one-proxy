import type { CliContext } from './main.ts';
import { type OneProxyTokens, type RouteRule } from './storage.ts';
type ExtensionBootstrap = {
    policyRevision?: string;
    fetchedAt?: string;
    proxyToken?: string;
    proxyTokenExpiresAt?: string;
    groups?: Array<{
        id: string;
        name: string;
        entryNodeId?: string;
        proxyHost?: string;
        proxyPort?: number;
        proxyScheme?: string;
        proxyDefault?: boolean;
        proxyHosts?: string[];
        directHosts?: string[];
        routes?: Array<{
            id: string;
            matchType: string;
            matchValue: string;
            actionType: string;
            topology?: Array<{
                id: string;
                publicHost?: string;
                publicPort?: number;
                mode?: string;
            }>;
        }>;
    }>;
};
export declare function login(args: string[], context: CliContext): Promise<void>;
export declare function logout(_args: string[], context: CliContext): Promise<void>;
export declare function refreshSession(): Promise<OneProxyTokens>;
export declare function tenantList(_args: string[], context: CliContext): Promise<void>;
export declare function tenantUse(args: string[], context: CliContext): Promise<void>;
export declare function groupList(_args: string[], context: CliContext): Promise<void>;
export declare function groupUse(args: string[], context: CliContext): Promise<void>;
export declare function routeRulesFromBootstrap(group: NonNullable<ExtensionBootstrap['groups']>[number]): RouteRule[];
export declare function sync(_args: string[], context: CliContext): Promise<void>;
export {};
