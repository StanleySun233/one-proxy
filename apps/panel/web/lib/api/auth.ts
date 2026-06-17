import { request } from './client';
import type { Account, LoginResult, TenantMembership } from '@/lib/types/auth';

type AuthResultResponse = {
  account: Account;
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
  mustRotatePassword: boolean;
  tenantMemberships: TenantMembership[];
  activeTenantId?: string | null;
};

function normalizeAuthResult(result: AuthResultResponse): LoginResult {
  return {
    account: result.account,
    accessToken: result.accessToken,
    refreshToken: result.refreshToken,
    expiresAt: result.expiresAt,
    mustRotatePassword: result.mustRotatePassword,
    tenantMemberships: result.tenantMemberships,
    activeTenantId: result.activeTenantId ?? null
  };
}

export function login(account: string, password: string) {
  return request<AuthResultResponse>('/auth/login', {
    method: 'POST',
    body: {account, password}
  }).then(normalizeAuthResult);
}

export function refreshSession(refreshToken: string) {
  return request<AuthResultResponse>('/auth/refresh', {
    method: 'POST',
    headers: {'X-One-Proxy-Refresh-Token': refreshToken}
  }).then(normalizeAuthResult);
}

export function logout(accessToken: string) {
  return request<{status: string}>('/auth/logout', {
    method: 'POST',
    accessToken
  });
}
