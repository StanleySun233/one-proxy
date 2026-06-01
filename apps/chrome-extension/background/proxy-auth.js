import { appendLog } from './diagnostics.js';
import { activeGroupFrom } from './state.js';

let proxyAuthCache = {
  host: '',
  port: 0,
  username: '',
  password: ''
};

export function updateProxyAuthCache(state) {
  const group = activeGroupFrom(state);
  proxyAuthCache = {
    host: group && group.proxyHost ? String(group.proxyHost) : '',
    port: group && group.proxyPort ? Number(group.proxyPort) : 0,
    username: state.proxyAuth && state.proxyAuth.username ? String(state.proxyAuth.username) : '',
    password: state.proxyAuth && state.proxyAuth.password ? String(state.proxyAuth.password) : ''
  };
}

function matchesProxyChallenge(details) {
  if (!details || !details.isProxy || !proxyAuthCache.username || !proxyAuthCache.password) {
    return false;
  }
  const challenger = details.challenger || {};
  return challenger.host === proxyAuthCache.host && Number(challenger.port || 0) === proxyAuthCache.port;
}

export function registerProxyAuthHandler() {
  if (!chrome.webRequest || !chrome.webRequest.onAuthRequired) {
    return;
  }
  chrome.webRequest.onAuthRequired.addListener(
    (details) => {
      if (!matchesProxyChallenge(details)) {
        return {};
      }
      appendLog('info', 'proxy_auth_supplied', {
        host: proxyAuthCache.host,
        port: proxyAuthCache.port
      }).catch(() => {});
      return {
        authCredentials: {
          username: proxyAuthCache.username,
          password: proxyAuthCache.password
        }
      };
    },
    { urls: ['<all_urls>'] },
    ['blocking']
  );
}
