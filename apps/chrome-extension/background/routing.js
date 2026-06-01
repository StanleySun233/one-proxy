import { activeGroupFrom, uniqueStrings } from './state.js';

export function wildcardToRegExp(pattern) {
  return new RegExp(`^${String(pattern || '').replace(/[.+?^${}()|[\]\\]/g, '\\$&').replaceAll('*', '.*')}$`, 'i');
}

export function sanitizeHost(value) {
  return String(value || '').trim().toLowerCase();
}

export function hostMatches(patterns, host) {
  const cleanHost = sanitizeHost(host);
  return uniqueStrings(patterns).some((pattern) => wildcardToRegExp(pattern).test(cleanHost));
}

export function ipv4ToNumber(value) {
  const parts = String(value || '').split('.');
  if (parts.length !== 4) {
    return null;
  }
  let result = 0;
  for (const part of parts) {
    const octet = Number(part);
    if (!Number.isInteger(octet) || octet < 0 || octet > 255) {
      return null;
    }
    result = (result * 256) + octet;
  }
  return result;
}

export function cidrMatches(patterns, host) {
  const ip = ipv4ToNumber(host);
  if (ip === null) {
    return false;
  }
  return uniqueStrings(patterns).some((pattern) => {
    const [network, prefixValue] = pattern.split('/');
    const networkIp = ipv4ToNumber(network);
    const prefix = Number(prefixValue);
    if (networkIp === null || !Number.isInteger(prefix) || prefix < 0 || prefix > 32) {
      return false;
    }
    const mask = prefix === 0 ? 0 : (0xffffffff << (32 - prefix)) >>> 0;
    return (ip & mask) === (networkIp & mask);
  });
}

export function routePreviewForHost(state, host) {
  return routePreviewForUrl(state, host ? `http://${host}` : '');
}

export function routePreviewForUrl(state, value) {
  const parsed = parseTargetUrl(value);
  const cleanHost = sanitizeHost(parsed.host);
  const group = activeGroupFrom(state);
  if (!cleanHost) {
    return { mode: 'unknown', source: 'no_site', host: '', topology: [] };
  }
  if (!state.enabled) {
    return { mode: 'direct', source: 'proxy_off', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
  }
  if (!group || !group.proxyHost || !group.proxyPort) {
    return { mode: 'direct', source: 'no_proxy_target', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
  }
  if (hostMatches(['localhost', '*.local', '*.lan', urlHostname(state.controlPlaneUrl), group.proxyHost, ...(group.directHosts || []), ...(state.localOverrides.directHosts || [])], cleanHost)) {
    const local = hostMatches(state.localOverrides.directHosts, cleanHost);
    return { mode: 'direct', source: local ? 'local_direct' : 'remote_direct', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
  }
  if (cidrMatches(group.directCidrs || [], cleanHost)) {
    return { mode: 'direct', source: 'remote_direct', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
  }
  const matchedRoute = matchGroupRoute(group, cleanHost, parsed.protocol);
  if (matchedRoute) {
    const mode = matchedRoute.actionType === 'direct' ? 'direct' : 'proxy';
    return {
      mode,
      source: routeSource(matchedRoute),
      host: cleanHost,
      protocol: parsed.protocol,
      port: parsed.port,
      rule: matchedRoute,
      topology: mode === 'proxy' ? (matchedRoute.topology && matchedRoute.topology.length ? matchedRoute.topology : group.topology) : []
    };
  }
  if (hostMatches([...(group.proxyHosts || []), ...(state.localOverrides.proxyHosts || [])], cleanHost)) {
    const local = hostMatches(state.localOverrides.proxyHosts, cleanHost);
    return { mode: 'proxy', source: local ? 'local_proxy' : 'remote_proxy', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: group.topology || [] };
  }
  if (cidrMatches(group.proxyCidrs || [], cleanHost)) {
    return { mode: 'proxy', source: 'remote_proxy', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: group.topology || [] };
  }
  if (group.proxyDefault) {
    return { mode: 'proxy', source: 'proxy_default', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: group.topology || [] };
  }
  return { mode: 'direct', source: 'default_direct', host: cleanHost, protocol: parsed.protocol, port: parsed.port, topology: [] };
}

export function parseTargetUrl(value) {
  const raw = String(value || '').trim();
  if (!raw) {
    return { url: '', host: '', protocol: 'http', port: 80 };
  }
  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(raw) ? raw : `http://${raw}`;
  try {
    const parsed = new URL(withScheme);
    const protocol = parsed.protocol.replace(':', '').toLowerCase() || 'http';
    const port = Number(parsed.port) || defaultPort(protocol);
    return { url: parsed.href, host: parsed.hostname, protocol, port };
  } catch (_error) {
    return { url: raw, host: raw.split('/')[0], protocol: 'http', port: 80 };
  }
}

function defaultPort(protocol) {
  if (protocol === 'https' || protocol === 'wss') {
    return 443;
  }
  return 80;
}

function matchGroupRoute(group, host, protocol) {
  const routes = [...(group.routes || [])].sort((left, right) => Number(left.priority || 0) - Number(right.priority || 0));
  for (const route of routes) {
    if (!routeMatches(route, host, protocol)) {
      continue;
    }
    return route;
  }
  return null;
}

export function routeMatches(route, host, protocol) {
  const value = String(route.matchValue || '').toLowerCase();
  const cleanHost = sanitizeHost(host);
  switch (route.matchType) {
    case 'domain':
      return cleanHost === value;
    case 'domain_suffix':
      return cleanHost.endsWith(value);
    case 'ip':
      return cleanHost === value;
    case 'ip_cidr':
      return cidrMatches([route.matchValue], cleanHost);
    case 'protocol':
      return String(protocol || '').toLowerCase() === value;
    case 'default':
      return true;
    default:
      return false;
  }
}

function routeSource(route) {
  if (route.actionType === 'direct') {
    return 'remote_direct';
  }
  if (route.matchType === 'default') {
    return 'proxy_default';
  }
  if (route.actionType === 'chain') {
    return 'remote_proxy';
  }
  return 'remote_proxy';
}

export function urlHostname(value) {
  try {
    return new URL(value).hostname;
  } catch (_error) {
    return '';
  }
}
