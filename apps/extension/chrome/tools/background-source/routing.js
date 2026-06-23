import { accessPathById, uniqueStrings } from './state.js';

export function wildcardToRegExp(pattern) {
  return new RegExp(`^${String(pattern || '').replace(/[.+?^${}()|[\]\\]/g, '\\$&').replaceAll('*', '.*')}$`, 'i');
}

export function sanitizeHost(value) {
  return String(value || '').trim().toLowerCase();
}

export function hostMatches(patterns, host) {
  const cleanHost = sanitizeHost(host);
  return uniqueStrings(patterns).some((pattern) => hostMatchesPattern(pattern, cleanHost));
}

function hostMatchesPattern(pattern, host) {
  const cleanPattern = sanitizeHost(pattern);
  if (cleanPattern.startsWith('*.') || cleanPattern.startsWith('.')) {
    return domainSuffixMatches(cleanPattern, host);
  }
  return wildcardToRegExp(cleanPattern).test(host);
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
  if (!cleanHost) {
    return { mode: 'unknown', source: 'no_site', host: '', topology: [] };
  }
  if (!state.enabled) {
    return { ...emptyResult(parsed), mode: 'direct', source: 'proxy_off', host: cleanHost };
  }
  return evaluateClientRoute(state, parsed);
}

export function evaluateClientRoute(state, input) {
  const parsed = normalizeRouteInput(input);
  const cleanHost = sanitizeHost(parsed.host);
  if (isLocalSafetyDirect(state, parsed)) {
    return {
      ...emptyResult(parsed),
      mode: 'direct',
      source: 'local_safety_direct',
      host: cleanHost
    };
  }
  const route = firstMatchingRoute(state, parsed);
  if (!route) {
    return {
      ...emptyResult(parsed),
      mode: 'direct',
      source: 'default_direct',
      host: cleanHost
    };
  }
  return applyRouteAction(state, route, parsed);
}

function emptyResult(parsed) {
  return {
    routeId: '',
    chainId: '',
    accessPathId: '',
    targetHost: sanitizeHost(parsed.host),
    targetPort: parsed.port,
    protocol: parsed.protocol,
    topology: [],
    denyReason: '',
    host: sanitizeHost(parsed.host),
    port: parsed.port,
    rule: null
  };
}

function applyRouteAction(state, route, parsed) {
  const base = {
    ...emptyResult(parsed),
    source: 'policy',
    routeId: route.id,
    chainId: route.chainId,
    accessPathId: route.accessPathId,
    rule: route
  };
  if (route.actionType === 'direct') {
    return { ...base, mode: 'direct' };
  }
  if (route.actionType === 'deny') {
    return { ...base, mode: 'deny', denyReason: 'route_denied' };
  }
  if (route.actionType === 'chain') {
    const accessPath = accessPathById(state, route.accessPathId);
    if (!isUsableAccessPath(accessPath)) {
      return { ...base, mode: 'deny', denyReason: 'access_path_unavailable' };
    }
    return {
      ...base,
      mode: 'proxy',
      topology: route.topology.length > 0 ? route.topology : accessPath.topology
    };
  }
  return { ...base, mode: 'deny', denyReason: 'route_denied' };
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
    return normalizeRouteInput({ url: raw, host: raw.split('/')[0], protocol: 'http', port: 80 });
  }
}

function normalizeRouteInput(input) {
  const parsed = typeof input === 'object' && input ? input : parseTargetUrl(input);
  const protocol = String(parsed.protocol || 'http').replace(':', '').toLowerCase();
  return {
    url: String(parsed.url || ''),
    host: sanitizeHost(parsed.host),
    port: Number(parsed.port || defaultPort(protocol)),
    protocol,
    accessPathId: String(parsed.accessPathId || '')
  };
}

function defaultPort(protocol) {
  if (protocol === 'http' || protocol === 'ws') {
    return 80;
  }
  if (protocol === 'https' || protocol === 'wss' || protocol === 'connect') {
    return 443;
  }
  if (protocol === 'ssh') {
    return 22;
  }
  return 0;
}

export function sortedEnabledRoutes(state) {
  return [...(state.remote.routes || [])]
    .filter((route) => route.enabled)
    .sort((left, right) => {
      const priority = Number(left.priority || 0) - Number(right.priority || 0);
      return priority || String(left.id || '').localeCompare(String(right.id || ''));
    });
}

export function firstMatchingRoute(state, parsed) {
  return sortedEnabledRoutes(state).find((route) => routeMatches(route, parsed)) || null;
}

export function routeMatches(route, target) {
  const parsed = normalizeRouteInput(target);
  const value = String(route.matchValue || '').toLowerCase();
  const cleanHost = sanitizeHost(parsed.host);
  switch (route.matchType) {
    case 'domain':
      return cleanHost === value;
    case 'domain_suffix':
      return domainSuffixMatches(value, cleanHost);
    case 'ip':
      return cleanHost === value;
    case 'ip_cidr':
      return cidrMatches([route.matchValue], cleanHost);
    case 'protocol':
      return parsed.protocol === value;
    case 'default':
      return true;
    default:
      return false;
  }
}

function domainSuffixMatches(value, host) {
  const suffix = value.replace(/^\*\./, '').replace(/^\./, '');
  return Boolean(suffix) && (host === suffix || host.endsWith(`.${suffix}`));
}

export function isLocalSafetyDirect(state, target) {
  const parsed = normalizeRouteInput(target);
  const controlPlaneHost = urlHostname(state.controlPlaneUrl);
  if (parsed.host && parsed.host === controlPlaneHost) {
    return true;
  }
  if (isLoopbackHost(parsed.host)) {
    return true;
  }
  const helper = state.localHelper || {};
  return Boolean(helper.enabled && parsed.host === sanitizeHost(helper.host) && Number(parsed.port || 0) === Number(helper.port || 0));
}

function isLoopbackHost(host) {
  const cleanHost = sanitizeHost(host).replace(/^\[/, '').replace(/\]$/, '');
  return cleanHost === 'localhost' ||
    cleanHost.endsWith('.localhost') ||
    cleanHost === '::1' ||
    cleanHost.startsWith('127.');
}

export function isUsableAccessPath(accessPath) {
  return Boolean(accessPath &&
    accessPath.enabled &&
    accessPath.effectiveEnabled !== false &&
    accessPath.authMode === 'proxy_token' &&
    accessPath.serviceType === 'http_forward_proxy' &&
    accessPath.listenHost &&
    accessPath.listenPort > 0 &&
    (!accessPath.health || accessPath.health.status !== 'unavailable'));
}

export function accessPathProxyTarget(accessPath) {
  if (!isUsableAccessPath(accessPath)) {
    return '';
  }
  const scheme = accessPath.protocol === 'https' ? 'HTTPS' : 'PROXY';
  return `${scheme} ${accessPath.listenHost}:${accessPath.listenPort}`;
}

export function urlHostname(value) {
  try {
    return new URL(value).hostname.toLowerCase();
  } catch (_error) {
    return '';
  }
}
