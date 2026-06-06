import type { EntryNode, OneProxyConfig, OneProxyState, RouteRule } from './lifecycle';

export type RouteMode = 'direct' | 'proxy';

export type RouteResolverInput = {
  config: OneProxyConfig;
  state: OneProxyState;
  target: string;
  protocol?: string;
};

export type RouteResult = {
  target: string;
  host: string;
  port: number;
  mode: RouteMode;
  matched: {
    source: 'local_override_direct' | 'local_override_proxy' | 'policy' | 'default_direct';
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

export function resolveRoute(input: RouteResolverInput): RouteResult {
  const target = parseTarget(input.target, input.protocol);
  const host = target.host.toLowerCase();
  const group = activeRouteGroup(input);
  const entryNode = firstEntryNode(input.state);

  if (matchesOverride(host, input.config.overrides?.direct ?? [])) {
    return routeResult(input, target, 'direct', { source: 'local_override_direct' }, group, null);
  }
  if (matchesOverride(host, input.config.overrides?.proxy ?? [])) {
    return routeResult(input, target, 'proxy', { source: 'local_override_proxy' }, group, entryNode);
  }

  const rule = group?.rules.find((candidate) => matchesRule(host, candidate));
  if (rule) {
    return routeResult(
      input,
      target,
      rule.mode,
      { source: 'policy', ruleId: rule.id, ruleType: rule.type, pattern: rule.pattern },
      group,
      rule.mode === 'proxy' ? entryNode : null
    );
  }

  return routeResult(input, target, 'direct', { source: 'default_direct' }, group, null);
}

function activeRouteGroup(input: RouteResolverInput) {
  return (input.state.routeGroups ?? []).find((group) => group.id === input.config.activeGroupId);
}

function firstEntryNode(state: OneProxyState): EntryNode | null {
  const node = state.bootstrap?.entryNodes?.[0];
  return node ? node : null;
}

function routeResult(
  input: RouteResolverInput,
  target: { original: string; host: string; port: number },
  mode: RouteMode,
  matched: RouteResult['matched'],
  group: ReturnType<typeof activeRouteGroup>,
  entryNode: EntryNode | null
): RouteResult {
  return {
    target: target.original,
    host: target.host,
    port: target.port,
    mode,
    matched,
    tenant: { id: input.config.activeTenantId },
    group: { id: group?.id, name: group?.name },
    topology: mode === 'proxy' && entryNode ? {
      entryNodeId: entryNode.id,
      entryHost: entryNode.host,
      entryPort: entryNode.port,
      protocol: entryNode.protocol
    } : null
  };
}

function parseTarget(target: string, protocol = 'https') {
  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(target) ? target : `${protocol}://${target}`;
  const url = new URL(withScheme);
  const port = url.port ? Number(url.port) : defaultPort(url.protocol);
  return {
    original: target,
    host: url.hostname.toLowerCase(),
    port
  };
}

function defaultPort(protocol: string) {
  if (protocol === 'http:') {
    return 80;
  }
  if (protocol === 'ssh:') {
    return 22;
  }
  return 443;
}

function matchesOverride(host: string, overrides: string[]) {
  return overrides.some((override) => hostMatchesPattern(host, override.toLowerCase()));
}

function matchesRule(host: string, rule: RouteRule) {
  const pattern = rule.pattern.toLowerCase();
  if (rule.type === 'wildcard') {
    return pattern === '*' || hostMatchesPattern(host, pattern);
  }
  if (rule.type === 'suffix') {
    return host === pattern || host.endsWith(`.${pattern.replace(/^\./, '')}`);
  }
  if (rule.type === 'domain') {
    return host === pattern;
  }
  if (rule.type === 'cidr') {
    return matchesIpv4Cidr(host, pattern);
  }
  return false;
}

function hostMatchesPattern(host: string, pattern: string) {
  if (pattern.startsWith('*.')) {
    const suffix = pattern.slice(2);
    return host.endsWith(`.${suffix}`);
  }
  return host === pattern || host.endsWith(`.${pattern}`);
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
