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

type AccessPathSnapshot = {
  id: string;
  name: string;
  chainId: string;
  protocol: string;
  entryNodeId: string;
  listenHost: string;
  listenPort: number;
  enabled: boolean;
  topology: TopologyHop[];
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

type LatestConfig = OneProxyConfig & {
  activeAccessPathId?: string;
};

type LatestState = OneProxyState & {
  accessPaths?: AccessPathSnapshot[];
  routes?: RouteSnapshot[];
  bootstrap?: {
    tenantId?: string;
    accessPathId?: string;
  };
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

export function resolveRoute(input: RouteResolverInput): RouteResult {
  const target = parseTarget(input.target, input.protocol);
  const host = target.host.toLowerCase();

  if (isSafetyDirectTarget(host, input.config)) {
    return routeResult(input, target, 'direct', 'local_safety_direct');
  }

  if (input.proxyOnly) {
    return proxyOnlyRoute(input, target);
  }

  if (matchesOverride(host, input.config.overrides?.direct ?? [])) {
    return routeResult(input, target, 'direct', 'local_override_direct');
  }

  if (matchesOverride(host, input.config.overrides?.proxy ?? [])) {
    return routeForAccessPath(input, target, activeAccessPath(input), 'local_override_proxy');
  }

  const route = enabledRoutes(input.state)
    .sort((left, right) => left.priority - right.priority || left.id.localeCompare(right.id))
    .find((candidate) => matchesRoute(target, candidate));

  if (!route) {
    return routeResult(input, target, 'direct', 'default_direct');
  }

  if (route.actionType === 'direct') {
    return routeResult(input, target, 'direct', 'policy', route);
  }

  if (route.actionType === 'deny') {
    return routeResult(input, target, 'deny', 'policy', route, undefined, 'route_denied');
  }

  const accessPath = accessPaths(input.state).find((candidate) => candidate.enabled && candidate.id === route.accessPathId);
  if (!accessPath) {
    return routeResult(input, target, 'deny', 'policy', route, undefined, 'access_path_unavailable');
  }
  return routeResult(input, target, 'proxy', 'policy', route, accessPath);
}

function proxyOnlyRoute(input: RouteResolverInput, target: ReturnType<typeof parseTarget>) {
  const accessPath = activeAccessPath(input);
  return accessPath
    ? routeForAccessPath(input, target, accessPath, 'proxy_only')
    : routeResult(input, target, 'deny', 'proxy_only', undefined, undefined, 'access_path_unavailable');
}

function routeForAccessPath(
  input: RouteResolverInput,
  target: ReturnType<typeof parseTarget>,
  accessPath: AccessPathSnapshot | undefined,
  source: RouteSource
) {
  return accessPath
    ? routeResult(input, target, 'proxy', source, undefined, accessPath)
    : routeResult(input, target, 'deny', source, undefined, undefined, 'access_path_unavailable');
}

function routeResult(
  input: RouteResolverInput,
  target: ReturnType<typeof parseTarget>,
  mode: RouteMode,
  source: RouteSource,
  route?: RouteSnapshot,
  accessPath?: AccessPathSnapshot,
  denyReason: RouteDenyReason = ''
): RouteResult {
  return {
    target: target.original,
    host: target.host,
    port: target.port,
    targetHost: target.host,
    targetPort: target.port,
    protocol: target.protocol,
    mode,
    source,
    routeId: route?.id ?? '',
    chainId: route?.chainId ?? accessPath?.chainId ?? '',
    accessPathId: route?.accessPathId ?? accessPath?.id ?? '',
    denyReason,
    matched: {
      source,
      ruleId: route?.id,
      ruleType: route?.matchType,
      pattern: route?.matchValue
    },
    tenant: { id: input.config.activeTenantId },
    group: { id: route?.chainId ?? accessPath?.chainId },
    topology: mode === 'proxy' && accessPath ? topologyFromAccessPath(accessPath) : null
  };
}

function accessPaths(state: OneProxyState): AccessPathSnapshot[] {
  return ((state as LatestState).accessPaths ?? []);
}

function enabledRoutes(state: OneProxyState): RouteSnapshot[] {
  return ((state as LatestState).routes ?? []).filter((route) => route.enabled);
}

function activeAccessPath(input: RouteResolverInput): AccessPathSnapshot | undefined {
  const config = input.config as LatestConfig;
  const state = input.state as LatestState;
  const selectedId = config.activeAccessPathId ?? state.bootstrap?.accessPathId;
  const paths = accessPaths(input.state);
  return paths.find((accessPath) => accessPath.enabled && accessPath.id === selectedId) || paths.find((accessPath) => accessPath.enabled);
}

function topologyFromAccessPath(accessPath: AccessPathSnapshot): NonNullable<RouteResult['topology']> {
  return {
    entryNodeId: accessPath.entryNodeId,
    entryHost: accessPath.listenHost,
    entryPort: accessPath.listenPort,
    protocol: accessPath.protocol,
    hops: accessPath.topology
  };
}

function parseTarget(target: string, protocol = 'https') {
  const normalizedProtocol = protocol.toLowerCase().replace(/:$/, '');
  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(target) ? target : `${normalizedProtocol}://${target}`;
  const url = new URL(withScheme);
  const parsedProtocol = url.protocol.replace(':', '').toLowerCase();
  const port = url.port ? Number(url.port) : defaultPort(parsedProtocol);
  return {
    original: target,
    host: url.hostname.toLowerCase(),
    port,
    protocol: parsedProtocol
  };
}

function defaultPort(protocol: string) {
  if (protocol === 'http') {
    return 80;
  }
  if (protocol === 'ssh') {
    return 22;
  }
  return 443;
}

function isSafetyDirectTarget(host: string, config: OneProxyConfig) {
  if (isLoopbackHost(host)) {
    return true;
  }
  return Boolean(config.controlPlaneUrl && new URL(config.controlPlaneUrl).hostname.toLowerCase() === host);
}

function isLoopbackHost(host: string) {
  const normalized = host.replace(/^\[|\]$/g, '');
  return normalized === 'localhost' || normalized === '::1' || normalized.startsWith('127.') || normalized === '0.0.0.0';
}

function matchesOverride(host: string, overrides: string[]) {
  return overrides.some((override) => hostMatchesPattern(host, override.toLowerCase()));
}

function matchesRoute(target: ReturnType<typeof parseTarget>, route: RouteSnapshot) {
  const pattern = route.matchValue.toLowerCase();
  if (route.matchType === 'default') {
    return true;
  }
  if (route.matchType === 'protocol') {
    return target.protocol === pattern;
  }
  if (route.matchType === 'domain_suffix') {
    return hostMatchesSuffix(target.host, pattern);
  }
  if (route.matchType === 'domain') {
    return target.host === pattern;
  }
  if (route.matchType === 'ip') {
    return target.host === pattern;
  }
  if (route.matchType === 'ip_cidr') {
    return matchesIpv4Cidr(target.host, pattern);
  }
  return false;
}

function hostMatchesPattern(host: string, pattern: string) {
  if (pattern.startsWith('*.')) {
    const suffix = pattern.slice(2);
    return hostMatchesSuffix(host, suffix);
  }
  if (pattern.startsWith('.')) {
    return hostMatchesSuffix(host, pattern);
  }
  return host === pattern || host.endsWith(`.${pattern}`);
}

function hostMatchesSuffix(host: string, pattern: string) {
  const suffix = pattern.replace(/^\*\./, '').replace(/^\./, '');
  return host === suffix || host.endsWith(`.${suffix}`);
}

function matchesIpv4Cidr(host: string, cidr: string) {
  const [range, prefixText] = cidr.split('/');
  const prefix = Number(prefixText);
  const hostValue = ipv4ToNumber(host);
  const rangeValue = ipv4ToNumber(range);
  if (hostValue === null || rangeValue === null || !Number.isInteger(prefix) || prefix < 0 || prefix > 32) {
    return false;
  }
  const mask = prefix === 0 ? 0 : (0xffffffff << (32 - prefix)) >>> 0;
  return (hostValue & mask) === (rangeValue & mask);
}

function ipv4ToNumber(value: string) {
  const parts = value.split('.').map(Number);
  if (parts.length !== 4 || parts.some((part) => !Number.isInteger(part) || part < 0 || part > 255)) {
    return null;
  }
  return (((parts[0] << 24) >>> 0) + (parts[1] << 16) + (parts[2] << 8) + parts[3]) >>> 0;
}
