'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {Plus, Trash2} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {useEffect, useMemo, useState} from 'react';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {useAuth} from '@/components/auth-provider';
import {ConsoleCrudModal} from '@/components/console-template';
import {deleteResourceBinding, getGrantTenants, getResourceBindings, upsertResourceBinding} from '@/lib/api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';
import type {ResourceBinding, ResourceBindingPermission, ResourceBindingType, Tenant} from '@/lib/types';

type ResourceGrantModalProps = {
  open: boolean;
  resourceType: ResourceBindingType;
  resourceId: string;
  resourceName?: string;
  onClose: () => void;
  onChanged?: () => void;
};

const permissionOptions: {value: ResourceBindingPermission; label: string}[] = [
  {value: 'use', label: 'Use'},
  {value: 'manage', label: 'Manage'}
];

function isOnlyManageBinding(binding: ResourceBinding, bindings: ResourceBinding[]) {
  return binding.permission === 'manage' && bindings.filter((item) => item.permission === 'manage').length === 1;
}

export function ResourceGrantModal({open, resourceType, resourceId, resourceName, onClose, onChanged}: ResourceGrantModalProps) {
  const t = useTranslations();
  const queryClient = useQueryClient();
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const [tenantId, setTenantId] = useState('');
  const [permission, setPermission] = useState<ResourceBindingPermission>('use');
  const bindingsQueryKey = useMemo(
    () => ['grants', accessToken, activeTenantId, resourceType, resourceId],
    [accessToken, activeTenantId, resourceType, resourceId]
  );

  const bindingsQuery = useQuery({
    queryKey: bindingsQueryKey,
    queryFn: () => getResourceBindings(accessToken, activeTenantId, resourceType, resourceId),
    enabled: open && !!accessToken && !!resourceId
  });
  const tenantsQuery = useQuery({
    queryKey: ['grantTenants', accessToken, activeTenantId],
    queryFn: () => getGrantTenants(accessToken, activeTenantId),
    enabled: open && !!accessToken
  });

  const bindings = bindingsQuery.data || [];
  const tenants = tenantsQuery.data || [];
  const boundTenantIds = useMemo(() => new Set(bindings.map((binding) => binding.tenantId)), [bindings]);
  const availableTenants = useMemo(() => tenants.filter((tenant) => !boundTenantIds.has(tenant.id)), [boundTenantIds, tenants]);

  useEffect(() => {
    if (!open) {
      setTenantId('');
      setPermission('use');
      return;
    }
    if (availableTenants.length > 0 && !availableTenants.some((tenant) => tenant.id === tenantId)) {
      setTenantId(availableTenants[0].id);
    }
  }, [availableTenants, open, tenantId]);

  const afterChanged = () => {
    queryClient.invalidateQueries({queryKey: bindingsQueryKey});
    onChanged?.();
  };
  const upsertMutation = useMutation({
    mutationFn: (payload: {targetTenantId: string; permission: ResourceBindingPermission}) =>
      upsertResourceBinding(accessToken, activeTenantId, resourceType, resourceId, payload.targetTenantId, {permission: payload.permission}),
    onSuccess: () => {
      toast.success('Resource grant saved');
      afterChanged();
      setTenantId('');
      setPermission('use');
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });
  const deleteMutation = useMutation({
    mutationFn: (targetTenantId: string) => deleteResourceBinding(accessToken, activeTenantId, resourceType, resourceId, targetTenantId),
    onSuccess: () => {
      toast.success('Resource grant removed');
      afterChanged();
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });

  const loading = bindingsQuery.isPending || tenantsQuery.isPending;
  const error = bindingsQuery.error || tenantsQuery.error;

  return (
    <ConsoleCrudModal
      footer={<button className="secondary-button" onClick={onClose} type="button">{t('common.close')}</button>}
      onClose={onClose}
      open={open}
      subtitle={resourceName || resourceId}
      title="Resource grants"
    >
      {loading ? (
        <AsyncState detail={t('common.loading')} title={t('common.loadingTitle')} />
      ) : error ? (
        <AsyncState actionLabel={t('common.retry')} detail={formatControlPlaneError(error)} onAction={() => { void bindingsQuery.refetch(); void tenantsQuery.refetch(); }} title="Failed to load resource grants" />
      ) : (
        <div className="sub-grid">
          <div className="forms-grid">
            <label className="field-stack">
              <span>Tenant</span>
              <select className="field-select" onChange={(event) => setTenantId(event.target.value)} value={tenantId}>
                <option value="">{availableTenants.length === 0 ? 'No available tenant' : 'Select tenant'}</option>
                {availableTenants.map((tenant: Tenant) => <option key={tenant.id} value={tenant.id}>{tenant.name}</option>)}
              </select>
            </label>
            <label className="field-stack">
              <span>Permission</span>
              <select className="field-select" onChange={(event) => setPermission(event.target.value as ResourceBindingPermission)} value={permission}>
                {permissionOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <div className="field-stack" style={{alignSelf: 'end'}}>
              <button
                className="secondary-button"
                disabled={!tenantId || upsertMutation.isPending}
                onClick={() => upsertMutation.mutate({targetTenantId: tenantId, permission})}
                type="button"
              >
                <Plus size={14} />
                Add grant
              </button>
            </div>
          </div>

          <div className="table-card">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Tenant</th>
                  <th>Permission</th>
                  <th>Created</th>
                  <th>{t('common.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {bindings.map((binding) => {
                  const onlyManage = isOnlyManageBinding(binding, bindings);

                  return (
                    <tr key={binding.tenantId}>
                      <td>{binding.tenantName}</td>
                      <td>
                        <select
                          className="field-select"
                          disabled={upsertMutation.isPending}
                          onChange={(event) => upsertMutation.mutate({targetTenantId: binding.tenantId, permission: event.target.value as ResourceBindingPermission})}
                          value={binding.permission}
                        >
                          {permissionOptions.map((option) => (
                            <option disabled={onlyManage && option.value === 'use'} key={option.value} value={option.value}>{option.label}</option>
                          ))}
                        </select>
                      </td>
                      <td className="mono">{formatISODateTime(binding.createdAt, binding.createdAt)}</td>
                      <td>
                        <button
                          className="danger-button"
                          disabled={deleteMutation.isPending || onlyManage}
                          onClick={() => deleteMutation.mutate(binding.tenantId)}
                          type="button"
                        >
                          <Trash2 size={14} />
                          {t('common.delete')}
                        </button>
                      </td>
                    </tr>
                  );
                })}
                {bindings.length === 0 ? (
                  <tr>
                    <td colSpan={4}><span className="muted-text">{t('common.empty')}</span></td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </ConsoleCrudModal>
  );
}
