import { request } from './client';
import type {Tenant, TenantCreatedResult, TenantMembershipAccount} from '@/lib/types';

export function getTenants(accessToken: string) {
  return request<{tenants: Tenant[]}>('/tenants', {accessToken}).then((result) => result.tenants);
}

export function createTenant(accessToken: string, payload: {name: string; initialAdminAccountId: string}) {
  return request<TenantCreatedResult>('/tenants', {
    method: 'POST',
    accessToken,
    body: payload
  });
}

export function updateTenant(accessToken: string, tenantId: string, payload: {name: string}) {
  return request<{tenant: Tenant}>(`/tenants/${tenantId}`, {
    method: 'PATCH',
    accessToken,
    tenantId,
    body: payload
  }).then((result) => result.tenant);
}

export function deleteTenant(accessToken: string, tenantId: string) {
  return request<null>(`/tenants/${tenantId}`, {
    method: 'DELETE',
    accessToken,
    tenantId
  });
}

export function getTenantMembers(accessToken: string, tenantId: string) {
  return request<{memberships: TenantMembershipAccount[]}>(`/tenants/${tenantId}/memberships`, {
    accessToken,
    tenantId
  }).then((result) => result.memberships);
}

export function upsertTenantMember(accessToken: string, tenantId: string, accountId: string, payload: {role: string}) {
  return request<{membership: TenantMembershipAccount}>(`/tenants/${tenantId}/memberships/${accountId}`, {
    method: 'PUT',
    accessToken,
    tenantId,
    body: payload
  }).then((result) => result.membership);
}

export function deleteTenantMember(accessToken: string, tenantId: string, accountId: string) {
  return request<null>(`/tenants/${tenantId}/memberships/${accountId}`, {
    method: 'DELETE',
    accessToken,
    tenantId
  });
}
