import type { OneProxyConfig, OneProxyState } from './lifecycle.ts';
export type RouteMode = 'direct' | 'proxy' | 'deny';
type RouteSource = 'policy' | 'default_direct' | 'default_deny' | 'local_safety_direct' | 'local_override_direct' | 'local_override_proxy' | 'proxy_only';
type RouteDenyReason = '' | 'route_not_found' | 'route_denied' | 'access_path_unavailable' | 'node_unavailable';
type TopologyHop = {
    nodeId: string;
    nodeName: string;
    mode: string;
    scopeKey: string;
    publicHost?: string;
    publicPort?: number;
    transport: string;
};
type RouteSnapshot = {
    id: string;
    priority: number;
    matchType: 'domain' | 'domain_suffix' | 'ip' | 'ip_cidr' | 'protocol' | 'default';
    matchValue: string;
    actionType: 'chain' | 'direct' | 'deny';
    chainId: string;
    accessPathId: string;
    enabled: boolean;
    topology: TopologyHop[];
};
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
    targetHost: string;
    targetPort: number;
    protocol: string;
    mode: RouteMode;
    source: RouteSource;
    routeId: string;
    chainId: string;
    accessPathId: string;
    denyReason: RouteDenyReason;
    matched: {
        source: RouteSource;
        ruleId?: string;
        ruleType?: RouteSnapshot['matchType'];
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
        hops: TopologyHop[];
    } | null;
};
export declare function resolveRoute(input: RouteResolverInput): RouteResult;
export {};
