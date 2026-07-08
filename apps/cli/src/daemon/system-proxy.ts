import type { DaemonBindings } from './lifecycle.ts';
import type { RouteResult } from './router.ts';

export type SystemProxy = {
  host: string;
  port: number;
  authorization?: string;
};

const unsetMarker = '__ONEPROXY_UNSET__';
const proxyVariables = {
  http: ['HTTP_PROXY', 'http_proxy', 'ALL_PROXY', 'all_proxy'],
  https: ['HTTPS_PROXY', 'https_proxy', 'HTTP_PROXY', 'http_proxy', 'ALL_PROXY', 'all_proxy']
};

export function systemProxyForRoute(route: RouteResult, bindings: Partial<DaemonBindings>, env: NodeJS.ProcessEnv = process.env): SystemProxy | null {
  if (route.mode !== 'direct' || route.source !== 'default_direct') {
    return null;
  }
  if (noProxyMatches(route.host, route.port, env)) {
    return null;
  }
  const protocol = route.protocol === 'http' ? 'http' : 'https';
  for (const name of proxyVariables[protocol]) {
    const value = proxyEnvValue(name, env);
    if (!value) {
      continue;
    }
    const proxy = parseSystemProxy(value);
    if (!proxy || isOneProxyBinding(proxy, bindings)) {
      continue;
    }
    return proxy;
  }
  return null;
}

function proxyEnvValue(name: string, env: NodeJS.ProcessEnv) {
  const previous = env.ONEPROXY_ACTIVE === '1' ? env[`ONEPROXY_PREV_${name}`] : undefined;
  if (previous && previous !== unsetMarker) {
    return previous;
  }
  return env[name];
}

function noProxyValue(env: NodeJS.ProcessEnv) {
  if (env.ONEPROXY_ACTIVE === '1') {
    return env.ONEPROXY_PREV_NO_PROXY && env.ONEPROXY_PREV_NO_PROXY !== unsetMarker
      ? env.ONEPROXY_PREV_NO_PROXY
      : env.ONEPROXY_PREV_no_proxy && env.ONEPROXY_PREV_no_proxy !== unsetMarker
        ? env.ONEPROXY_PREV_no_proxy
        : '';
  }
  return env.NO_PROXY || env.no_proxy || '';
}

function parseSystemProxy(value: string): SystemProxy | null {
  try {
    const url = new URL(/^[a-z][a-z0-9+.-]*:\/\//i.test(value) ? value : `http://${value}`);
    if (url.protocol !== 'http:') {
      return null;
    }
    const port = url.port ? Number(url.port) : 80;
    if (!url.hostname || !Number.isInteger(port) || port < 1 || port > 65535) {
      return null;
    }
    const username = decodeURIComponent(url.username);
    const password = decodeURIComponent(url.password);
    const proxy: SystemProxy = {
      host: url.hostname.toLowerCase(),
      port
    };
    if (username || password) {
      proxy.authorization = `Basic ${Buffer.from(`${username}:${password}`).toString('base64')}`;
    }
    return proxy;
  } catch {
    return null;
  }
}

function isOneProxyBinding(proxy: SystemProxy, bindings: Partial<DaemonBindings>) {
  if (!isLoopback(proxy.host)) {
    return false;
  }
  return new Set([bindings.httpPort, bindings.httpsPort, bindings.proxyOnlyPort].filter((port): port is number => Number.isInteger(port))).has(proxy.port);
}

function noProxyMatches(host: string, port: number, env: NodeJS.ProcessEnv) {
  const value = noProxyValue(env);
  if (!value) {
    return false;
  }
  const normalizedHost = host.toLowerCase().replace(/^\[|\]$/g, '');
  return value.split(',').map((item) => item.trim().toLowerCase()).filter(Boolean).some((item) => {
    if (item === '*') {
      return true;
    }
    const token = noProxyToken(item);
    if (token.port !== null && token.port !== port) {
      return false;
    }
    if (token.host.startsWith('*.')) {
      return hostMatchesSuffix(normalizedHost, token.host.slice(2));
    }
    if (token.host.startsWith('.')) {
      return hostMatchesSuffix(normalizedHost, token.host.slice(1));
    }
    return normalizedHost === token.host || normalizedHost.endsWith(`.${token.host}`);
  });
}

function noProxyToken(value: string) {
  const ipv6 = value.startsWith('[');
  const separator = ipv6 ? value.lastIndexOf(']:') : value.lastIndexOf(':');
  if (separator <= 0) {
    return { host: value.replace(/^\[|\]$/g, ''), port: null };
  }
  const portText = value.slice(separator + (ipv6 ? 2 : 1));
  const port = Number(portText);
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    return { host: value.replace(/^\[|\]$/g, ''), port: null };
  }
  return { host: value.slice(0, separator + (ipv6 ? 1 : 0)).replace(/^\[|\]$/g, ''), port };
}

function hostMatchesSuffix(host: string, suffix: string) {
  return host === suffix || host.endsWith(`.${suffix}`);
}

function isLoopback(host: string) {
  const normalized = host.replace(/^\[|\]$/g, '');
  return normalized === 'localhost' || normalized === '::1' || normalized.startsWith('127.');
}
