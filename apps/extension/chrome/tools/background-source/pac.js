import { appendLog } from './diagnostics.js';
import { accessPathById, uniqueStrings } from './state.js';
import { accessPathProxyTarget, isUsableAccessPath, sortedEnabledRoutes, urlHostname } from './routing.js';

function denyProxyTarget() {
  return 'PROXY 127.0.0.1:9';
}

function localHelperTarget(state) {
  const helper = state.localHelper || {};
  return helper.enabled && helper.host && helper.port ? `${helper.scheme || 'SOCKS5'} ${helper.host}:${helper.port}` : '';
}

function compiledRules(state) {
  const helperTarget = localHelperTarget(state);
  return sortedEnabledRoutes(state).map((route) => {
    const accessPath = accessPathById(state, route.accessPathId);
    const proxyTarget = route.actionType === 'chain' && isUsableAccessPath(accessPath)
      ? helperTarget || accessPathProxyTarget(accessPath)
      : '';
    return {
      id: route.id,
      matchType: route.matchType,
      matchValue: route.matchValue,
      actionType: route.actionType,
      chainId: route.chainId,
      accessPathId: route.accessPathId,
      proxyTarget
    };
  });
}

export function buildPacScript(state) {
  const helper = state.localHelper || {};
  return `
const enabled = ${state.enabled ? 'true' : 'false'};
const rules = ${JSON.stringify(compiledRules(state))};
const controlPlaneHost = ${JSON.stringify(urlHostname(state.controlPlaneUrl))};
const helperHost = ${JSON.stringify(helper.enabled ? String(helper.host || '').toLowerCase() : '')};
const helperPort = ${Number(helper.enabled ? helper.port || 0 : 0)};
const denyTarget = ${JSON.stringify(denyProxyTarget())};

function protocolFromUrl(url) {
  const index = String(url || '').indexOf(':');
  return index > 0 ? String(url).slice(0, index).toLowerCase() : 'http';
}

function portFromUrl(url, protocol) {
  const match = String(url || '').match(/^[a-z][a-z0-9+.-]*:\\/\\/(?:[^@/]*@)?(?:\\[[^\\]]+\\]|[^/:?#]+)(?::(\\d+))?/i);
  if (match && match[1]) {
    return Number(match[1]);
  }
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

function sanitizeHost(host) {
  return String(host || '').toLowerCase();
}

function domainSuffixMatches(value, host) {
  const suffix = String(value || '').replace(/^\\*\\./, '').replace(/^\\./, '');
  return Boolean(suffix) && (host === suffix || dnsDomainIs(host, '.' + suffix));
}

function ipv4ToNumber(value) {
  const parts = String(value || '').split('.');
  if (parts.length !== 4) {
    return null;
  }
  let result = 0;
  for (let index = 0; index < parts.length; index += 1) {
    const octet = Number(parts[index]);
    if (!Number.isInteger(octet) || octet < 0 || octet > 255) {
      return null;
    }
    result = (result * 256) + octet;
  }
  return result;
}

function cidrMatches(pattern, host) {
  const ip = ipv4ToNumber(host);
  if (ip === null) {
    return false;
  }
  const parts = String(pattern || '').split('/');
  const networkIp = ipv4ToNumber(parts[0]);
  const prefix = Number(parts[1]);
  if (networkIp === null || !Number.isInteger(prefix) || prefix < 0 || prefix > 32) {
    return false;
  }
  const mask = prefix === 0 ? 0 : (0xffffffff << (32 - prefix)) >>> 0;
  return (ip & mask) === (networkIp & mask);
}

function isLoopbackHost(host) {
  return host === 'localhost' ||
    dnsDomainIs(host, '.localhost') ||
    host === '::1' ||
    host.indexOf('127.') === 0;
}

function isLocalSafetyDirect(host, port) {
  if (host && host === controlPlaneHost) {
    return true;
  }
  if (isLoopbackHost(host)) {
    return true;
  }
  return Boolean(helperHost && host === helperHost && Number(port || 0) === helperPort);
}

function routeMatches(rule, target) {
  switch (rule.matchType) {
    case 'domain':
      return target.host === String(rule.matchValue || '').toLowerCase();
    case 'domain_suffix':
      return domainSuffixMatches(rule.matchValue, target.host);
    case 'ip':
      return target.host === String(rule.matchValue || '').toLowerCase();
    case 'ip_cidr':
      return cidrMatches(rule.matchValue, target.host);
    case 'protocol':
      return target.protocol === String(rule.matchValue || '').toLowerCase();
    case 'default':
      return true;
    default:
      return false;
  }
}

function FindProxyForURL(url, host) {
  if (!enabled) {
    return 'DIRECT';
  }
  const protocol = protocolFromUrl(url);
  const port = portFromUrl(url, protocol);
  const target = {
    host: sanitizeHost(host),
    protocol,
    port
  };
  if (isLocalSafetyDirect(target.host, target.port)) {
    return 'DIRECT';
  }
  for (let index = 0; index < rules.length; index += 1) {
    const rule = rules[index];
    if (!routeMatches(rule, target)) {
      continue;
    }
    if (rule.actionType === 'direct') {
      return 'DIRECT';
    }
    if (rule.actionType === 'chain') {
      return rule.proxyTarget || denyTarget;
    }
    return denyTarget;
  }
  return 'DIRECT';
}
`;
}

export function pacSummary(state) {
  const helperTarget = localHelperTarget(state);
  const rules = compiledRules(state);
  const proxyTargets = uniqueStrings(rules.map((rule) => rule.proxyTarget).filter(Boolean));
  return {
    enabled: Boolean(state.enabled),
    proxyTarget: proxyTargets.length === 1 ? proxyTargets[0] : '',
    localHelper: helperTarget,
    accessPaths: state.remote.accessPaths.length,
    enabledAccessPaths: state.remote.accessPaths.filter((path) => isUsableAccessPath(accessPathById(state, path.id))).length,
    routes: state.remote.routes.length,
    enabledRoutes: rules.length,
    chainRoutes: rules.filter((rule) => rule.actionType === 'chain').length,
    directRoutes: rules.filter((rule) => rule.actionType === 'direct').length,
    denyRoutes: rules.filter((rule) => rule.actionType === 'deny').length,
    proxyTargets: proxyTargets.length
  };
}

export function applyProxy(state) {
  return chrome.proxy.settings.set({
    value: {
      mode: 'pac_script',
      pacScript: {
        data: buildPacScript(state)
      }
    },
    scope: 'regular'
  }).then(() => appendLog('info', 'proxy_applied', pacSummary(state)));
}
