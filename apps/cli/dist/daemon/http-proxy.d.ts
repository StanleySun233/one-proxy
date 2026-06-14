import * as http from 'node:http';
import type { DaemonBindings } from './lifecycle.ts';
import type { RouteResolverInput } from './router.ts';
export type ProxyRouteContext = Omit<RouteResolverInput, 'target' | 'protocol'>;
export type ProxyServers = {
    httpServer: http.Server;
    httpsServer: http.Server;
    proxyOnlyServer?: http.Server;
    close: () => Promise<void>;
};
export declare function startHttpProxyListeners(input: ProxyRouteContext, bindings: DaemonBindings, liveState?: boolean, onProxyActivity?: () => void): Promise<ProxyServers>;
export declare function createHttpProxyServer(input: ProxyRouteContext, liveState?: boolean, onProxyActivity?: () => void, proxyOnly?: boolean): http.Server<typeof http.IncomingMessage, typeof http.ServerResponse>;
