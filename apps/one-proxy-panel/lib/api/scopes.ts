import type {Scope} from '@/lib/types';

import {request} from './client';

export function getScopes(accessToken: string) {
  return request<Scope[]>('/scopes', {accessToken});
}

export function createScope(accessToken: string, payload: {name: string; description: string}) {
  return request<Scope>('/scopes', {
    accessToken,
    method: 'POST',
    body: payload
  });
}

export function updateScope(accessToken: string, scopeID: string, payload: {name: string; description: string}) {
  return request<Scope>(`/scopes/${scopeID}`, {
    accessToken,
    method: 'PATCH',
    body: payload
  });
}

export function deleteScope(accessToken: string, scopeID: string) {
  return request<{status: string}>(`/scopes/${scopeID}`, {
    accessToken,
    method: 'DELETE'
  });
}
