import {request} from './client';
import type {ResourceBinding, ResourceBindingPayload, ResourceBindingType, Tenant} from '@/lib/types';

function bindingPath(resourceType: ResourceBindingType, resourceId: string, tenantId: string) {
  return `/grants/${encodeURIComponent(resourceType)}/${encodeURIComponent(resourceId)}/${encodeURIComponent(tenantId)}`;
}

export function getResourceBindings(accessToken: string, tenantId: string | null, resourceType: ResourceBindingType, resourceId: string) {
  const params = new URLSearchParams({resourceType, resourceId});
  return request<{bindings: ResourceBinding[]}>(`/grants?${params.toString()}`, {
    accessToken,
    tenantId
  }).then((result) => result.bindings);
}

export function getGrantTenants(accessToken: string, tenantId: string | null) {
  return request<{tenants: Tenant[]}>('/grants/tenants', {accessToken, tenantId}).then((result) => result.tenants);
}

export function upsertResourceBinding(
  accessToken: string,
  tenantId: string | null,
  resourceType: ResourceBindingType,
  resourceId: string,
  targetTenantId: string,
  payload: ResourceBindingPayload
) {
  return request<{binding: ResourceBinding}>(bindingPath(resourceType, resourceId, targetTenantId), {
    method: 'PUT',
    accessToken,
    tenantId,
    body: payload
  }).then((result) => result.binding);
}

export function deleteResourceBinding(accessToken: string, tenantId: string | null, resourceType: ResourceBindingType, resourceId: string, targetTenantId: string) {
  return request<null>(bindingPath(resourceType, resourceId, targetTenantId), {
    method: 'DELETE',
    accessToken,
    tenantId
  });
}
