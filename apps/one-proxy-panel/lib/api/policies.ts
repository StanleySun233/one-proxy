import { request } from './client';
import type { PolicyRevision } from '@/lib/types';

export function getPolicyRevisions(accessToken: string) {
  return request<PolicyRevision[]>('/policies/revisions', {accessToken});
}

export function publishPolicy(accessToken: string) {
  return request<PolicyRevision>('/policies/publish', {
    method: 'POST',
    accessToken
  });
}
