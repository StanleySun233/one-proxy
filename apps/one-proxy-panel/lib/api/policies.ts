import { request } from './client';
import type { PolicyRevision } from '@/lib/types';

export function getPolicyRevisions(accessToken: string, tenantId: string | null) {
  return request<PolicyRevision[]>('/policies/revisions', {accessToken, tenantId});
}

export function publishPolicy(accessToken: string, tenantId: string | null) {
  return request<PolicyRevision>('/policies/publish', {
    method: 'POST',
    accessToken,
    tenantId
  });
}
