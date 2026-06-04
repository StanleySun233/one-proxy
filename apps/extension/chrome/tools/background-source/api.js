import { appendLog } from './diagnostics.js';
import { DEFAULT_STATE, getState, mergeState, persistState } from './state.js';

function authHeaders(token) {
  return token ? { 'X-One-Proxy-Access-Token': token } : {};
}

function tenantHeaders(state) {
  return state.session && state.session.activeTenantId ? { 'X-One-Proxy-Tenant-ID': state.session.activeTenantId } : {};
}

function readJSON(response) {
  return response.json().catch(() => null);
}

export function normalizeControlPlaneUrl(value) {
  const clean = String(value || '').trim().replace(/\/+$/, '');
  if (!clean) {
    return '';
  }
  return /^[a-z][a-z0-9+.-]*:\/\//i.test(clean) ? clean : `https://${clean}`;
}

function apiRequest(state, path, options = {}) {
  const controlPlaneUrl = normalizeControlPlaneUrl(state.controlPlaneUrl);
  if (!controlPlaneUrl) {
    return Promise.reject(new Error('missing_control_plane_url'));
  }
  const headers = {
    ...(options.body ? { 'Content-Type': 'application/json' } : {}),
    ...(options.headers || {})
  };
  if (options.auth !== false && state.session.accessToken) {
    Object.assign(headers, authHeaders(state.session.accessToken));
  }
  if (options.tenant !== false) {
    Object.assign(headers, tenantHeaders(state));
  }
  return fetch(`${controlPlaneUrl}${path}`, {
    method: options.method || 'GET',
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined
  }).then((response) => {
    if (response.status === 401 && options.allowRefresh !== false && state.session.refreshToken) {
      return refreshSession(state).then((refreshed) => apiRequest(refreshed, path, { ...options, allowRefresh: false }));
    }
    return readJSON(response).then((payload) => {
      if (!response.ok) {
        const message = (payload && payload.message) || 'request_failed';
        return appendLog('error', 'api_request_failed', { path, status: response.status, message })
          .then(() => { throw new Error(message); });
      }
      return appendLog('info', 'api_request_ok', { path, status: response.status })
        .then(() => payload ? payload.data : null);
    });
  });
}

export function login(controlPlaneUrl, account, password) {
  const normalizedControlPlaneUrl = normalizeControlPlaneUrl(controlPlaneUrl);
  if (!normalizedControlPlaneUrl) {
    return Promise.reject(new Error('missing_control_plane_url'));
  }
  return fetch(`${normalizedControlPlaneUrl}/api/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ account, password })
  }).then((response) => readJSON(response).then((payload) => {
    if (!response.ok) {
      const message = (payload && payload.message) || 'login_failed';
      return appendLog('error', 'login_failed', { status: response.status, message })
        .then(() => { throw new Error(message); });
    }
    const memberships = Array.isArray(payload.data.tenantMemberships) ? payload.data.tenantMemberships : [];
    const activeTenantId = payload.data.activeTenantId || (memberships.length === 1 ? memberships[0].tenantId : '');
    return getState()
      .then((state) => {
        const nextState = mergeState({
          ...state,
          controlPlaneUrl: normalizedControlPlaneUrl,
          session: {
            account: payload.data.account.account,
            accessToken: payload.data.accessToken,
            refreshToken: payload.data.refreshToken,
            expiresAt: payload.data.expiresAt,
            proxyToken: '',
            proxyTokenExpiresAt: '',
            mustRotatePassword: Boolean(payload.data.mustRotatePassword),
            tenantMemberships: memberships,
            activeTenantId
          }
        });
        return persistState(nextState).then(() => nextState);
      })
      .then((nextState) => appendLog('info', 'login_ok', { account: nextState.session.account, controlPlaneUrl: nextState.controlPlaneUrl })
        .then(() => nextState.session.activeTenantId ? syncRemoteConfig(nextState) : null));
  }));
}

export function testConnection(controlPlaneUrl) {
  const baseUrl = normalizeControlPlaneUrl(controlPlaneUrl);
  if (!baseUrl) {
    return Promise.reject(new Error('missing_control_plane_url'));
  }
  return fetch(`${baseUrl}/healthz`)
    .catch((error) => appendLog('error', 'test_connection_failed', { controlPlaneUrl: baseUrl, message: error.message || 'connection_failed' })
      .then(() => { throw new Error('connection_failed'); }))
    .then((response) => readJSON(response).then((payload) => {
      if (!response.ok) {
        const message = (payload && payload.message) || 'connection_failed';
        return appendLog('error', 'test_connection_failed', { controlPlaneUrl: baseUrl, status: response.status, message })
          .then(() => { throw new Error(message); });
      }
      return appendLog('info', 'test_connection_ok', { controlPlaneUrl: baseUrl })
        .then(() => payload ? payload.data : null);
    }));
}

export function refreshSession(sourceState) {
  return (sourceState ? Promise.resolve(sourceState) : getState()).then((source) => {
    const state = mergeState(source);
    if (!state.controlPlaneUrl || !state.session.refreshToken) {
      throw new Error('missing_refresh_token');
    }
    return fetch(`${normalizeControlPlaneUrl(state.controlPlaneUrl)}/api/auth/refresh`, {
      method: 'POST',
      headers: { 'X-One-Proxy-Refresh-Token': state.session.refreshToken }
    }).then((response) => readJSON(response).then((payload) => {
      if (!response.ok) {
        const message = (payload && payload.message) || 'refresh_failed';
        return appendLog('error', 'refresh_failed', { status: response.status, message })
          .then(() => { throw new Error(message); });
      }
      const memberships = Array.isArray(payload.data.tenantMemberships) ? payload.data.tenantMemberships : state.session.tenantMemberships;
      const activeTenantId = state.session.activeTenantId || payload.data.activeTenantId || (memberships.length === 1 ? memberships[0].tenantId : '');
      const nextState = mergeState({
        ...state,
        session: {
          account: payload.data.account.account,
          accessToken: payload.data.accessToken,
          refreshToken: payload.data.refreshToken,
          expiresAt: payload.data.expiresAt,
          proxyToken: state.session.proxyToken,
          proxyTokenExpiresAt: state.session.proxyTokenExpiresAt,
          mustRotatePassword: Boolean(payload.data.mustRotatePassword),
          tenantMemberships: memberships,
          activeTenantId
        }
      });
      return persistState(nextState)
        .then(() => appendLog('info', 'refresh_ok', { account: nextState.session.account }))
        .then(() => nextState);
    }));
  });
}

export function syncRemoteConfig(sourceState) {
  return (sourceState ? Promise.resolve(sourceState) : getState()).then((source) => {
    const state = mergeState(source);
    if (!state.session.activeTenantId) {
      throw new Error('tenant_required');
    }
    return apiRequest(state, '/api/proxy/extension/bootstrap')
      .then((data) => {
        const nextState = mergeState({
          ...state,
          remote: {
            policyRevision: data.policyRevision || '',
            fetchedAt: data.fetchedAt || '',
            groups: Array.isArray(data.groups) ? data.groups : []
          },
          session: {
            ...state.session,
            account: data.account ? data.account.account : state.session.account,
            mustRotatePassword: Boolean(data.account && data.account.mustRotatePassword),
            proxyToken: data.proxyToken || '',
            proxyTokenExpiresAt: data.proxyTokenExpiresAt || ''
          }
        });
        return persistState(nextState)
          .then(() => appendLog('info', 'remote_config_synced', {
            policyRevision: nextState.remote.policyRevision,
            groups: nextState.remote.groups.length
          }));
      })
      .then(() => null);
  });
}

export function getExtensionPageStatus(state, params) {
  const query = new URLSearchParams();
  if (params.host) {
    query.set('host', params.host);
  }
  if (params.routeId) {
    query.set('routeId', params.routeId);
  }
  if (params.chainId) {
    query.set('chainId', params.chainId);
  }
  return apiRequest(state, `/api/proxy/extension/page/status?${query.toString()}`);
}

export function selectTenant(tenantId) {
  return getState().then((state) => {
    const activeTenantId = String(tenantId || '').trim();
    const nextState = mergeState({
      ...state,
      enabled: false,
      session: {
        ...state.session,
        activeTenantId,
        proxyToken: '',
        proxyTokenExpiresAt: ''
      },
      remote: DEFAULT_STATE.remote,
      selection: DEFAULT_STATE.selection
    });
    return persistState(nextState)
      .then(() => appendLog('info', 'tenant_selected', { tenantId: activeTenantId }))
      .then(() => syncRemoteConfig(nextState));
  });
}

export function logout() {
  return getState().then((state) => {
    const logoutRequest = state.controlPlaneUrl && state.session.accessToken
      ? fetch(`${normalizeControlPlaneUrl(state.controlPlaneUrl)}/api/auth/logout`, {
        method: 'POST',
        headers: authHeaders(state.session.accessToken)
      }).catch(() => {})
      : Promise.resolve();
    const nextState = mergeState({
      ...state,
      enabled: false,
      session: DEFAULT_STATE.session,
      remote: DEFAULT_STATE.remote,
      selection: DEFAULT_STATE.selection
    });
    return logoutRequest
      .then(() => persistState(nextState))
      .then(() => appendLog('info', 'logout_ok'));
  });
}
