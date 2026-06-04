import {request} from './client';
import type {AuditBusinessEventsResult, AuditDashboard, AuditQuery, NetworkAuditQuery, NetworkAuditSessionsResult} from '@/lib/types/audit';

function auditQueryString(query: AuditQuery | NetworkAuditQuery = {}) {
  const params = new URLSearchParams();
  Object.entries(query).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      params.set(key, String(value));
    }
  });
  const value = params.toString();
  return value ? `?${value}` : '';
}

export function getAuditBusinessEvents(accessToken: string, tenantId: string | null, query: AuditQuery = {}) {
  return request<AuditBusinessEventsResult>(`/audit/business/events${auditQueryString(query)}`, {accessToken, tenantId});
}

export function getAuditNetworkSessions(accessToken: string, tenantId: string | null, query: NetworkAuditQuery = {}) {
  return request<NetworkAuditSessionsResult>(`/audit/network/sessions${auditQueryString(query)}`, {accessToken, tenantId});
}

export function getAuditDashboard(accessToken: string, tenantId: string | null, query: Pick<AuditQuery, 'tenantId' | 'from' | 'to'> = {}) {
  return request<AuditDashboard>(`/audit/dashboard${auditQueryString(query)}`, {accessToken, tenantId});
}
