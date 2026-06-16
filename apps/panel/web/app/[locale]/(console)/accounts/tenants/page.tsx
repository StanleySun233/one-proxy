'use client';

import {useMutation, useQueries, useQuery, useQueryClient} from '@tanstack/react-query';
import {useMemo, useState} from 'react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {NameTag} from '@/components/common/name-tag';
import {useAuth} from '@/components/auth-provider';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {DeleteConfirmationModal, DeleteImpactSection} from '@/components/delete-confirmation-modal';
import {createTenant, deleteTenant, deleteTenantMember, fetchEnums, getAccounts, getTenantMembers, getTenants, updateTenant, upsertTenantMember} from '@/lib/api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';
import type {Tenant, TenantMembershipAccount} from '@/lib/types';

type TenantFormState = {
  name: string;
  initialAdminAccountId: string;
};

export default function TenantsPage() {
  const t = useTranslations();
  const accountsT = useTranslations('accounts');
  const queryClient = useQueryClient();
  const [nameFilter, setNameFilter] = useState('');
  const [createOpen, setCreateOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);
  const [memberTenant, setMemberTenant] = useState<Tenant | null>(null);
  const [deletingTenant, setDeletingTenant] = useState<Tenant | null>(null);
  const [formState, setFormState] = useState<TenantFormState>({name: '', initialAdminAccountId: ''});

  const {session, tenantMemberships} = useAuth();
  const accessToken = session?.accessToken || '';
  const isSuperAdmin = session?.account.role === 'super_admin';
  const tenantRoleById = new Map(tenantMemberships.map((membership) => [membership.tenantId, membership.role]));
  const canManageAnyTenant = isSuperAdmin || tenantMemberships.some((membership) => membership.role === 'tenant_admin');
  const canManageTenant = (tenantId: string) => isSuperAdmin || tenantRoleById.get(tenantId) === 'tenant_admin';
  const accountListTenantId = tenantMemberships.find((membership) => membership.role === 'tenant_admin')?.tenantId || null;

  const tenantsQuery = useQuery({
    queryKey: ['tenants', accessToken],
    queryFn: () => getTenants(accessToken),
    enabled: !!accessToken
  });
  const accountsQuery = useQuery({
    queryKey: ['accounts', accessToken, accountListTenantId],
    queryFn: () => getAccounts(accessToken, accountListTenantId),
    enabled: !!accessToken && (isSuperAdmin || !!accountListTenantId)
  });
  const memberQueries = useQueries({
    queries: (tenantsQuery.data || []).map((tenant) => ({
      queryKey: ['tenant-members', tenant.id, accessToken],
      queryFn: () => getTenantMembers(accessToken, tenant.id),
      enabled: !!accessToken
    }))
  });

  const tenants = tenantsQuery.data || [];
  const accounts = accountsQuery.data || [];
  const membersByTenantId = new Map(tenants.map((tenant, index) => [tenant.id, memberQueries[index]?.data || []]));
  const filteredTenants = useMemo(() => tenants.filter((tenant) =>
    !nameFilter.trim() || tenant.name.toLowerCase().includes(nameFilter.trim().toLowerCase())
  ), [nameFilter, tenants]);

  const createMutation = useMutation({
    mutationFn: (payload: TenantFormState) => createTenant(accessToken, payload),
    onSuccess: () => {
      toast.success(accountsT('tenantCreateSuccess'));
      queryClient.invalidateQueries({queryKey: ['tenants']});
      setCreateOpen(false);
      setFormState({name: '', initialAdminAccountId: ''});
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });
  const updateMutation = useMutation({
    mutationFn: (payload: {tenantId: string; name: string}) => updateTenant(accessToken, payload.tenantId, {name: payload.name}),
    onSuccess: () => {
      toast.success(accountsT('tenantUpdateSuccess'));
      queryClient.invalidateQueries({queryKey: ['tenants']});
      setEditingTenant(null);
      setFormState({name: '', initialAdminAccountId: ''});
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });
  const deleteMutation = useMutation({
    mutationFn: (tenantId: string) => deleteTenant(accessToken, tenantId),
    onSuccess: () => {
      toast.success(accountsT('tenantDeleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['tenants']});
      setDeletingTenant(null);
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });

  const openCreate = () => {
    setEditingTenant(null);
    setFormState({name: '', initialAdminAccountId: accounts[0]?.id || ''});
    setCreateOpen(true);
  };
  const openEdit = (tenant: Tenant) => {
    setCreateOpen(false);
    setEditingTenant(tenant);
    setFormState({name: tenant.name, initialAdminAccountId: ''});
  };
  const closeForm = () => {
    setCreateOpen(false);
    setEditingTenant(null);
    setFormState({name: '', initialAdminAccountId: ''});
  };
  const submitForm = () => {
    const name = formState.name.trim();
    if (!name) {
      toast.error(accountsT('tenantNameRequired'));
      return;
    }
    if (editingTenant) {
      updateMutation.mutate({tenantId: editingTenant.id, name});
      return;
    }
    if (!formState.initialAdminAccountId) {
      toast.error(accountsT('tenantInitialAdminRequired'));
      return;
    }
    createMutation.mutate({name, initialAdminAccountId: formState.initialAdminAccountId});
  };
  const tenantDeleteSections: DeleteImpactSection[] = deletingTenant ? [
    {id: 'tenant', label: accountsT('tenantDeleteImpactTenant'), items: [{id: deletingTenant.id, name: deletingTenant.name}]},
    {id: 'memberships', label: accountsT('tenantDeleteImpactMembers'), count: membersByTenantId.get(deletingTenant.id)?.length || 0}
  ] : [];

  return (
    <AuthGate>
      <ConsolePage
        actions={isSuperAdmin ? <button className="primary-button" onClick={openCreate} type="button">{accountsT('tenantCreate')}</button> : null}
        title={accountsT('tenantListTitle')}
      >
        <ConsoleFilterBar title={t('common.filter')}>
          <ConsoleFilterItem label={t('common.name')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setNameFilter(event.target.value)} placeholder={t('common.name')} value={nameFilter} />
          </ConsoleFilterItem>
        </ConsoleFilterBar>

        <ConsoleList count={filteredTenants.length} title={accountsT('tenantListTitle')}>
          {tenantsQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={accountsT('tenantLoading')} />
          ) : tenantsQuery.isError ? (
            <AsyncState actionLabel={t('common.retry')} detail={formatControlPlaneError(tenantsQuery.error)} onAction={() => void tenantsQuery.refetch()} title={accountsT('tenantFailed')} />
          ) : tenants.length === 0 ? (
            <AsyncState detail={accountsT('tenantEmpty')} title={t('common.empty')} />
          ) : filteredTenants.length === 0 ? (
            <AsyncState detail={t('common.noMatching')} title={t('common.empty')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('common.name')}</th>
                    <th>{accountsT('tenantMembers')}</th>
                    <th>{t('common.updated')}</th>
                    {canManageAnyTenant ? <th>{t('common.actions')}</th> : null}
                  </tr>
                </thead>
                <tbody>
                  {filteredTenants.map((tenant) => {
                    const members = membersByTenantId.get(tenant.id) || [];
                    const canManageThisTenant = canManageTenant(tenant.id);
                    return (
                      <tr key={tenant.id}>
                        <td><NameTag kind="group">{tenant.name}</NameTag></td>
                        <td>{members.length}</td>
                        <td className="mono">{formatISODateTime(tenant.updatedAt)}</td>
                        {canManageAnyTenant ? (
                          <td>
                            {canManageThisTenant ? (
                              <div className="chain-list-actions">
                                <button className="secondary-button" onClick={() => setMemberTenant(tenant)} type="button">{accountsT('tenantMembers')}</button>
                                <button className="secondary-button" onClick={() => openEdit(tenant)} type="button">{t('common.edit')}</button>
                                <button
                                  className="danger-button"
                                  disabled={deleteMutation.isPending}
                                  onClick={() => setDeletingTenant(tenant)}
                                  type="button"
                                >
                                  {t('common.delete')}
                                </button>
                              </div>
                            ) : null}
                          </td>
                        ) : null}
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </ConsoleList>

        {canManageAnyTenant ? (
          <ConsoleCrudModal
          footer={(
            <>
              <button className="secondary-button" onClick={closeForm} type="button">{t('common.cancel')}</button>
              <button className="primary-button" disabled={createMutation.isPending || updateMutation.isPending} onClick={submitForm} type="button">
                {createMutation.isPending || updateMutation.isPending ? t('common.submitting') : editingTenant ? t('common.save') : accountsT('tenantCreate')}
              </button>
            </>
          )}
          onClose={closeForm}
          open={createOpen || Boolean(editingTenant)}
          title={editingTenant ? accountsT('tenantEdit') : accountsT('tenantCreate')}
        >
          <div className="sub-grid">
            <label className="field-stack">
              <span>{t('common.name')}</span>
              <input className="field-input" onChange={(event) => setFormState((current) => ({...current, name: event.target.value}))} value={formState.name} />
            </label>
            {!editingTenant ? (
              <label className="field-stack">
                <span>{accountsT('tenantInitialAdmin')}</span>
                <select className="field-select" onChange={(event) => setFormState((current) => ({...current, initialAdminAccountId: event.target.value}))} value={formState.initialAdminAccountId}>
                  <option value="">{accountsT('tenantSelectInitialAdmin')}</option>
                  {accounts.map((account) => <option key={account.id} value={account.id}>{account.account}</option>)}
                </select>
              </label>
            ) : null}
          </div>
          </ConsoleCrudModal>
        ) : null}

        {memberTenant && canManageTenant(memberTenant.id) ? (
          <TenantMembersModal
            accessToken={accessToken}
            accounts={accounts}
            members={membersByTenantId.get(memberTenant.id) || []}
            onClose={() => setMemberTenant(null)}
            tenant={memberTenant}
          />
        ) : null}

        <DeleteConfirmationModal
          onClose={() => setDeletingTenant(null)}
          onConfirm={() => {
            if (deletingTenant) {
              deleteMutation.mutate(deletingTenant.id);
            }
          }}
          open={Boolean(deletingTenant)}
          pending={deleteMutation.isPending}
          sections={tenantDeleteSections}
          targetName={deletingTenant?.name || ''}
          title={accountsT('tenantDeleteTitle')}
        />
      </ConsolePage>
    </AuthGate>
  );
}

function TenantMembersModal({
  accessToken,
  accounts,
  members,
  onClose,
  tenant
}: {
  accessToken: string;
  accounts: {id: string; account: string; role: string; status: string}[];
  members: TenantMembershipAccount[];
  onClose: () => void;
  tenant: Tenant;
}) {
  const t = useTranslations();
  const accountsT = useTranslations('accounts');
  const queryClient = useQueryClient();
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const [accountId, setAccountId] = useState('');
  const [role, setRole] = useState('user');
  const [deletingMember, setDeletingMember] = useState<TenantMembershipAccount | null>(null);
  const tenantRoleOptions = enums?.tenant_role ? Object.entries(enums.tenant_role).map(([value, item]) => ({value, label: item.name})) : [
    {value: 'tenant_admin', label: 'Tenant Admin'},
    {value: 'user', label: 'User'}
  ];
  const memberAccountIds = new Set(members.map((member) => member.accountId));
  const availableAccounts = accounts.filter((account) => !memberAccountIds.has(account.id));

  const upsertMutation = useMutation({
    mutationFn: (payload: {accountId: string; role: string}) => upsertTenantMember(accessToken, tenant.id, payload.accountId, {role: payload.role}),
    onSuccess: () => {
      toast.success(accountsT('tenantMemberSaved'));
      queryClient.invalidateQueries({queryKey: ['tenant-members', tenant.id]});
      setAccountId('');
      setRole('user');
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });
  const deleteMutation = useMutation({
    mutationFn: (targetAccountId: string) => deleteTenantMember(accessToken, tenant.id, targetAccountId),
    onSuccess: () => {
      toast.success(accountsT('tenantMemberDeleted'));
      queryClient.invalidateQueries({queryKey: ['tenant-members', tenant.id]});
      setDeletingMember(null);
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });
  const memberDeleteSections: DeleteImpactSection[] = deletingMember ? [
    {id: 'membership', label: accountsT('tenantMemberDeleteImpact'), items: [{id: deletingMember.accountId, name: deletingMember.account, detail: deletingMember.role}]}
  ] : [];

  return (
    <ConsoleCrudModal onClose={onClose} open={true} title={accountsT('tenantMembersTitle', {name: tenant.name})}>
      <div className="sub-grid">
        <div className="forms-grid">
          <label className="field-stack">
            <span>{accountsT('tenantMemberAccount')}</span>
            <select className="field-select" onChange={(event) => setAccountId(event.target.value)} value={accountId}>
              <option value="">{accountsT('tenantSelectAccount')}</option>
              {availableAccounts.map((account) => <option key={account.id} value={account.id}>{account.account}</option>)}
            </select>
          </label>
          <label className="field-stack">
            <span>{accountsT('tenantMemberRole')}</span>
            <select className="field-select" onChange={(event) => setRole(event.target.value)} value={role}>
              {tenantRoleOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
          </label>
          <div className="field-stack" style={{alignSelf: 'end'}}>
            <button className="secondary-button" disabled={!accountId || upsertMutation.isPending} onClick={() => upsertMutation.mutate({accountId, role})} type="button">
              {accountsT('tenantAddMember')}
            </button>
          </div>
        </div>

        <div className="table-card">
          <table className="data-table">
            <thead>
              <tr>
                <th>{accountsT('tenantMemberAccount')}</th>
                <th>{accountsT('tenantMemberRole')}</th>
                <th>{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {members.map((member) => (
                <tr key={member.accountId}>
                  <td>{member.account}</td>
                  <td>
                    <select className="field-select" onChange={(event) => upsertMutation.mutate({accountId: member.accountId, role: event.target.value})} value={member.role}>
                      {tenantRoleOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                    </select>
                  </td>
                  <td>
                    <button className="danger-button" disabled={deleteMutation.isPending} onClick={() => setDeletingMember(member)} type="button">
                      {t('common.delete')}
                    </button>
                  </td>
                </tr>
              ))}
              {members.length === 0 ? (
                <tr>
                  <td colSpan={3}><span className="muted-text">{t('common.empty')}</span></td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </div>
      <DeleteConfirmationModal
        onClose={() => setDeletingMember(null)}
        onConfirm={() => {
          if (deletingMember) {
            deleteMutation.mutate(deletingMember.accountId);
          }
        }}
        open={Boolean(deletingMember)}
        pending={deleteMutation.isPending}
        sections={memberDeleteSections}
        targetName={deletingMember?.account || ''}
        title={accountsT('tenantMemberDeleteTitle')}
      />
    </ConsoleCrudModal>
  );
}
