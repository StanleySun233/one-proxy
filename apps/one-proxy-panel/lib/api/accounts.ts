import { request } from './client';
import type { Account } from '@/lib/types';

export function getAccounts(accessToken: string) {
  return request<Account[]>('/accounts', {accessToken});
}

export function createAccount(accessToken: string, payload: {account: string; password: string; role: string}) {
  return request<Account>('/accounts', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateAccount(
  accessToken: string,
  accountID: string,
  payload: {password?: string; role?: string; status?: string}
) {
  return request<Account>(`/accounts/${accountID}`, {
    method: 'PATCH',
    accessToken,
    body: payload
  });
}

export function deleteAccount(accessToken: string, accountID: string) {
  return request<{status: string}>(`/accounts/${accountID}`, {
    method: 'DELETE',
    accessToken
  });
}
