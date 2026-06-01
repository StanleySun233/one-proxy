import { appendLog } from './diagnostics.js';
import { activeGroupFrom, uniqueStrings } from './state.js';
import { urlHostname } from './routing.js';

function escapePacString(value) {
  return String(value || '').replaceAll('\\', '\\\\').replaceAll("'", "\\'");
}

function cidrToMask(prefix) {
  const bits = Number(prefix);
  if (!Number.isInteger(bits) || bits < 0 || bits > 32) {
    return null;
  }
  const octets = [];
  let remaining = bits;
  for (let index = 0; index < 4; index += 1) {
    const value = remaining >= 8 ? 255 : remaining <= 0 ? 0 : 256 - 2 ** (8 - remaining);
    octets.push(value);
    remaining -= 8;
  }
  return octets.join('.');
}

function cidrEntries(items) {
  return uniqueStrings(items)
    .map((item) => {
      const [network, prefix] = item.split('/');
      const mask = cidrToMask(prefix);
      if (!network || !mask) {
        return null;
      }
      return { network, mask };
    })
    .filter(Boolean);
}

export function buildPacScript(state) {
  const group = activeGroupFrom(state);
  const proxyTarget = group && group.proxyHost && group.proxyPort ? `${group.proxyScheme || 'PROXY'} ${group.proxyHost}:${group.proxyPort}` : 'DIRECT';
  const directHosts = uniqueStrings([
    'localhost',
    '*.local',
    '*.lan',
    urlHostname(state.controlPlaneUrl),
    group ? group.proxyHost : '',
    ...(group ? group.directHosts : []),
    ...(state.localOverrides.directHosts || [])
  ]);
  const proxyHosts = uniqueStrings([
    ...(group ? group.proxyHosts : []),
    ...(state.localOverrides.proxyHosts || [])
  ]);
  const directCidrs = cidrEntries(group ? group.directCidrs : []);
  const proxyCidrs = cidrEntries(group ? group.proxyCidrs : []);
  return `
const enabled = ${state.enabled ? 'true' : 'false'};
const proxyTarget = '${escapePacString(proxyTarget)}';
const proxyDefault = ${group && group.proxyDefault ? 'true' : 'false'};
const directHosts = ${JSON.stringify(directHosts)};
const proxyHosts = ${JSON.stringify(proxyHosts)};
const directCidrs = ${JSON.stringify(directCidrs)};
const proxyCidrs = ${JSON.stringify(proxyCidrs)};

function hostMatches(patterns, host) {
  for (const pattern of patterns) {
    if (shExpMatch(host, pattern)) {
      return true;
    }
  }
  return false;
}

function inCidrs(cidrs, ip) {
  if (!ip) {
    return false;
  }
  for (const item of cidrs) {
    if (isInNet(ip, item.network, item.mask)) {
      return true;
    }
  }
  return false;
}

function isLocalOnly(host, ip) {
  if (isPlainHostName(host) || dnsDomainIs(host, '.local')) {
    return true;
  }
  if (!ip) {
    return false;
  }
  return isInNet(ip, '127.0.0.0', '255.0.0.0') ||
    isInNet(ip, '169.254.0.0', '255.255.0.0');
}

function FindProxyForURL(url, host) {
  if (!enabled || proxyTarget === 'DIRECT') {
    return 'DIRECT';
  }
  const resolved = dnsResolve(host);
  if (hostMatches(directHosts, host)) {
    return 'DIRECT';
  }
  if (inCidrs(directCidrs, resolved)) {
    return 'DIRECT';
  }
  if (hostMatches(proxyHosts, host)) {
    return proxyTarget;
  }
  if (inCidrs(proxyCidrs, resolved)) {
    return proxyTarget;
  }
  if (isLocalOnly(host, resolved)) {
    return 'DIRECT';
  }
  if (proxyDefault) {
    return proxyTarget;
  }
  return 'DIRECT';
}
`;
}

export function pacSummary(state) {
  const group = activeGroupFrom(state);
  return {
    enabled: Boolean(state.enabled),
    activeGroupId: group ? group.id : '',
    activeGroupName: group ? group.name : '',
    proxyTarget: group && group.proxyHost && group.proxyPort ? `${group.proxyScheme || 'PROXY'} ${group.proxyHost}:${group.proxyPort}` : 'DIRECT',
    proxyDefault: Boolean(group && group.proxyDefault),
    remoteProxyHosts: group ? uniqueStrings(group.proxyHosts).length : 0,
    remoteProxyCidrs: group ? uniqueStrings(group.proxyCidrs).length : 0,
    remoteDirectHosts: group ? uniqueStrings(group.directHosts).length : 0,
    remoteDirectCidrs: group ? uniqueStrings(group.directCidrs).length : 0,
    localProxyHosts: uniqueStrings(state.localOverrides.proxyHosts).length,
    localDirectHosts: uniqueStrings(state.localOverrides.directHosts).length
  };
}

export async function applyProxy(state) {
  await chrome.proxy.settings.set({
    value: {
      mode: 'pac_script',
      pacScript: {
        data: buildPacScript(state)
      }
    },
    scope: 'regular'
  });
  await appendLog('info', 'proxy_applied', pacSummary(state));
}
