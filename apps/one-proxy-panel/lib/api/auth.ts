import { request } from './client';
import type { LoginResult } from '@/lib/types';

export function login(account: string, password: string) {
  return request<LoginResult>('/auth/login', {
    method: 'POST',
    body: {account, password}
  });
}

export function logout(accessToken: string) {
  return request<{status: string}>('/auth/logout', {
    method: 'POST',
    accessToken
  });
}
