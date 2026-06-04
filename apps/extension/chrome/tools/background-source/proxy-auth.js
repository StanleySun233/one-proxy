import { appendLog } from './diagnostics.js';
import { activeGroupFrom } from './state.js';

let proxyAuthCache = {
  host: '',
  port: 0,
  token: ''
};

export function updateProxyAuthCache(state) {
  const group = activeGroupFrom(state);
  proxyAuthCache = {
    host: group && group.proxyHost ? String(group.proxyHost) : '',
    port: group && group.proxyPort ? Number(group.proxyPort) : 0,
    token: state.session && state.session.proxyToken ? String(state.session.proxyToken) : ''
  };
}

function matchesProxyChallenge(details) {
  if (!details || !details.isProxy || !proxyAuthCache.token) {
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
          username: 'token',
          password: proxyAuthCache.token
        }
      };
    },
    { urls: ['<all_urls>'] },
    ['blocking']
  );
}
