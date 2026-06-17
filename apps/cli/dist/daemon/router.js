export function resolveRoute(input) {
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
function proxyOnlyRoute(input, target) {
    const accessPath = activeAccessPath(input);
    return accessPath
        ? routeForAccessPath(input, target, accessPath, 'proxy_only')
        : routeResult(input, target, 'deny', 'proxy_only', undefined, undefined, 'access_path_unavailable');
}
function routeForAccessPath(input, target, accessPath, source) {
    return accessPath
        ? routeResult(input, target, 'proxy', source, undefined, accessPath)
        : routeResult(input, target, 'deny', source, undefined, undefined, 'access_path_unavailable');
}
function routeResult(input, target, mode, source, route, accessPath, denyReason = '') {
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
        accessPath: {
            id: route?.accessPathId ?? accessPath?.id,
            name: accessPath?.name
        },
        topology: mode === 'proxy' && accessPath ? topologyFromAccessPath(accessPath) : null
    };
}
function accessPaths(state) {
    return (state.accessPaths ?? []);
}
function enabledRoutes(state) {
    return (state.routes ?? []).filter((route) => route.enabled);
}
function activeAccessPath(input) {
    const config = input.config;
    const state = input.state;
    const selectedId = config.activeAccessPathId ?? state.bootstrap?.accessPathId;
    const paths = accessPaths(input.state);
    return paths.find((accessPath) => accessPath.enabled && accessPath.id === selectedId) || paths.find((accessPath) => accessPath.enabled);
}
function topologyFromAccessPath(accessPath) {
    return {
        entryNodeId: accessPath.entryNodeId,
        entryHost: accessPath.listenHost,
        entryPort: accessPath.listenPort,
        protocol: accessPath.protocol,
        hops: accessPath.topology
    };
}
function parseTarget(target, protocol = 'https') {
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
function defaultPort(protocol) {
    if (protocol === 'http') {
        return 80;
    }
    if (protocol === 'ssh') {
        return 22;
    }
    return 443;
}
function isSafetyDirectTarget(host, config) {
    if (isLoopbackHost(host)) {
        return true;
    }
    return Boolean(config.controlPlaneUrl && new URL(config.controlPlaneUrl).hostname.toLowerCase() === host);
}
function isLoopbackHost(host) {
    const normalized = host.replace(/^\[|\]$/g, '');
    return normalized === 'localhost' || normalized === '::1' || normalized.startsWith('127.') || normalized === '0.0.0.0';
}
function matchesOverride(host, overrides) {
    return overrides.some((override) => hostMatchesPattern(host, override.toLowerCase()));
}
function matchesRoute(target, route) {
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
function hostMatchesPattern(host, pattern) {
    if (pattern.startsWith('*.')) {
        const suffix = pattern.slice(2);
        return hostMatchesSuffix(host, suffix);
    }
    if (pattern.startsWith('.')) {
        return hostMatchesSuffix(host, pattern);
    }
    return host === pattern || host.endsWith(`.${pattern}`);
}
function hostMatchesSuffix(host, pattern) {
    const suffix = pattern.replace(/^\*\./, '').replace(/^\./, '');
    return host === suffix || host.endsWith(`.${suffix}`);
}
function matchesIpv4Cidr(host, cidr) {
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
function ipv4ToNumber(value) {
    const parts = value.split('.').map(Number);
    if (parts.length !== 4 || parts.some((part) => !Number.isInteger(part) || part < 0 || part > 255)) {
        return null;
    }
    return (((parts[0] << 24) >>> 0) + (parts[1] << 16) + (parts[2] << 8) + parts[3]) >>> 0;
}
//# sourceMappingURL=router.js.map