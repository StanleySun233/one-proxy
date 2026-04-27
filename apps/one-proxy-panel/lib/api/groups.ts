import { request } from './client';
import type { Group, GroupDetail } from '@/lib/types';

export function listGroups(accessToken: string) {
  return request<Group[]>('/groups', {accessToken});
}

export function createGroup(
  accessToken: string,
  payload: {name: string; description: string; enabled: boolean}
) {
  return request<Group>('/groups', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function getGroup(accessToken: string, groupID: string) {
  return request<GroupDetail>(`/groups/${groupID}`, {accessToken});
}

export function updateGroup(
  accessToken: string,
  groupID: string,
  payload: {name?: string; description?: string; enabled?: boolean}
) {
  return request<Group>(`/groups/${groupID}`, {
    method: 'PUT',
    accessToken,
    body: payload
  });
}

export function deleteGroup(accessToken: string, groupID: string) {
  return request<{status: string}>(`/groups/${groupID}`, {
    method: 'DELETE',
    accessToken
  });
}

export function setGroupAccounts(accessToken: string, groupID: string, accountIds: string[]) {
  return request<{status: string}>(`/groups/${groupID}/accounts`, {
    method: 'PUT',
    accessToken,
    body: {accountIds}
  });
}

export function setGroupScopes(accessToken: string, groupID: string, scopeKeys: string[]) {
  return request<{status: string}>(`/groups/${groupID}/scopes`, {
    method: 'PUT',
    accessToken,
    body: {scopeKeys}
  });
}
