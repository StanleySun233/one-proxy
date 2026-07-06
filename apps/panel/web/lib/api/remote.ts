import { request } from './client';
import type {
  RemoteCredential,
  RemoteCredentialPayload,
  RemoteCredentialUpdatePayload,
  RemoteProtocol,
  RemoteSession,
  RemoteSessionPayload
} from '@/lib/types/remote';

export function getRemoteCredentials(accessToken: string, tenantId: string | null, protocol: RemoteProtocol) {
  return request<RemoteCredential[]>(`/remote/credentials?protocol=${encodeURIComponent(protocol)}`, {accessToken, tenantId});
}

export function createRemoteCredential(accessToken: string, tenantId: string | null, payload: RemoteCredentialPayload) {
  return request<RemoteCredential>('/remote/credentials', {
    method: 'POST',
    accessToken,
    tenantId,
    body: payload
  });
}

export function updateRemoteCredential(accessToken: string, tenantId: string | null, credentialId: string, payload: RemoteCredentialUpdatePayload) {
  return request<RemoteCredential>(`/remote/credentials/${credentialId}`, {
    method: 'PATCH',
    accessToken,
    tenantId,
    body: payload
  });
}

export function deleteRemoteCredential(accessToken: string, tenantId: string | null, credentialId: string) {
  return request<{id: string}>(`/remote/credentials/${credentialId}`, {
    method: 'DELETE',
    accessToken,
    tenantId
  });
}

export function createRemoteSession(accessToken: string, tenantId: string | null, payload: RemoteSessionPayload) {
  return request<RemoteSession>('/remote/sessions', {
    method: 'POST',
    accessToken,
    tenantId,
    body: payload
  });
}
