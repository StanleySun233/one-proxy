import type { OneProxyConfig, OneProxyState, RouteRule } from './lifecycle.ts';
export type RouteMode = 'direct' | 'proxy';
export type RouteResolverInput = {
    config: OneProxyConfig;
    state: OneProxyState;
    target: string;
    protocol?: string;
    proxyOnly?: boolean;
};
export type RouteResult = {
    target: string;
    host: string;
    port: number;
    mode: RouteMode;
    matched: {
        source: 'local_override_direct' | 'local_override_proxy' | 'policy' | 'default_direct' | 'proxy_only';
        ruleId?: string;
        ruleType?: RouteRule['type'];
        pattern?: string;
    };
    tenant: {
        id?: string;
        name?: string;
    };
    group: {
        id?: string;
        name?: string;
    };
    topology: {
        entryNodeId: string;
        entryHost: string;
        entryPort: number;
        protocol: string;
    } | null;
};
export declare function resolveRoute(input: RouteResolverInput): RouteResult;
