import { appendLog } from './diagnostics.js';
import { DEFAULT_STATE, getState, mergeState, persistState } from './state.js';

function authHeaders(token) {
  return token ? { Authorization: `Bearer ${token}` } : {};
}

function tenantHeaders(state) {
  return state.session && state.session.activeTenantId ? { 'X-Tenant-ID': state.session.activeTenantId } : {};
}

async function apiRequest(state, path, options = {}) {
  const controlPlaneUrl = String(state.controlPlaneUrl || '').trim().replace(/\/$/, '');
  if (!controlPlaneUrl) {
    throw new Error('missing_control_plane_url');
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
  const response = await fetch(`${controlPlaneUrl}${path}`, {
    method: options.method || 'GET',
    headers,
    body: options.body ? JSON.stringify(options.body) : undefined
  });
  if (response.status === 401 && options.allowRefresh !== false && state.session.refreshToken) {
    const refreshed = await refreshSession(state);
    return apiRequest(refreshed, path, { ...options, allowRefresh: false });
  }
  let payload = null;
  try {
    payload = await response.json();
  } catch (_error) {
  }
  if (!response.ok) {
    await appendLog('error', 'api_request_failed', { path, status: response.status, message: (payload && payload.message) || 'request_failed' });
    throw new Error((payload && payload.message) || 'request_failed');
  }
  await appendLog('info', 'api_request_ok', { path, status: response.status });
  return payload ? payload.data : null;
}

export async function login(controlPlaneUrl, account, password) {
  const response = await fetch(`${String(controlPlaneUrl || '').trim().replace(/\/$/, '')}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ account, password })
  });
  const payload = await response.json();
  if (!response.ok) {
    await appendLog('error', 'login_failed', { status: response.status, message: (payload && payload.message) || 'login_failed' });
    throw new Error((payload && payload.message) || 'login_failed');
  }
  const memberships = Array.isArray(payload.data.tenantMemberships) ? payload.data.tenantMemberships : [];
  const activeTenantId = payload.data.activeTenantId || (memberships.length === 1 ? memberships[0].tenantId : '');
  const nextState = mergeState({
    ...(await getState()),
    controlPlaneUrl: String(controlPlaneUrl || '').trim(),
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
  await persistState(nextState);
  await appendLog('info', 'login_ok', { account: nextState.session.account, controlPlaneUrl: nextState.controlPlaneUrl });
  return nextState.session.activeTenantId ? syncRemoteConfig(nextState) : null;
}

export async function testConnection(controlPlaneUrl) {
  const baseUrl = String(controlPlaneUrl || '').trim().replace(/\/$/, '');
  if (!baseUrl) {
    throw new Error('missing_control_plane_url');
  }
  let response;
  try {
    response = await fetch(`${baseUrl}/healthz`);
  } catch (error) {
    await appendLog('error', 'test_connection_failed', { controlPlaneUrl: baseUrl, message: error.message || 'connection_failed' });
    throw new Error('connection_failed');
  }
  let payload = null;
  try {
    payload = await response.json();
  } catch (_error) {
  }
  if (!response.ok) {
    await appendLog('error', 'test_connection_failed', { controlPlaneUrl: baseUrl, status: response.status, message: (payload && payload.message) || 'connection_failed' });
    throw new Error((payload && payload.message) || 'connection_failed');
  }
  await appendLog('info', 'test_connection_ok', { controlPlaneUrl: baseUrl });
  return payload ? payload.data : null;
}

export async function refreshSession(sourceState) {
  const state = mergeState(sourceState || (await getState()));
  if (!state.controlPlaneUrl || !state.session.refreshToken) {
    throw new Error('missing_refresh_token');
  }
  const response = await fetch(`${state.controlPlaneUrl.replace(/\/$/, '')}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refreshToken: state.session.refreshToken })
  });
  const payload = await response.json();
  if (!response.ok) {
    await appendLog('error', 'refresh_failed', { status: response.status, message: (payload && payload.message) || 'refresh_failed' });
    throw new Error((payload && payload.message) || 'refresh_failed');
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
  await persistState(nextState);
  await appendLog('info', 'refresh_ok', { account: nextState.session.account });
  return nextState;
}

export async function syncRemoteConfig(sourceState) {
  const state = mergeState(sourceState || (await getState()));
  if (!state.session.activeTenantId) {
    throw new Error('tenant_required');
  }
  const data = await apiRequest(state, '/api/v1/extension/bootstrap');
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
  await persistState(nextState);
  await appendLog('info', 'remote_config_synced', {
    policyRevision: nextState.remote.policyRevision,
    groups: nextState.remote.groups.length
  });
  return null;
}

export async function selectTenant(tenantId) {
  const state = await getState();
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
  await persistState(nextState);
  await appendLog('info', 'tenant_selected', { tenantId: activeTenantId });
  return syncRemoteConfig(nextState);
}

export async function logout() {
  const state = await getState();
  if (state.controlPlaneUrl && state.session.accessToken) {
    try {
      await fetch(`${state.controlPlaneUrl.replace(/\/$/, '')}/api/v1/auth/logout`, {
        method: 'POST',
        headers: authHeaders(state.session.accessToken)
      });
    } catch (_error) {
    }
  }
  const nextState = mergeState({
    ...state,
    enabled: false,
    session: DEFAULT_STATE.session,
    remote: DEFAULT_STATE.remote,
    selection: DEFAULT_STATE.selection
  });
  await persistState(nextState);
  await appendLog('info', 'logout_ok');
}
