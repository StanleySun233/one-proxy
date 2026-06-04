'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {Edit, Share2, Trash2} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {useMemo, useState} from 'react';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {ResourceGrantModal} from '@/components/resource-grant-modal';
import {useAuth} from '@/components/auth-provider';
import {NameTag} from '@/components/common/name-tag';
import {createScope, deleteScope, getScopes, updateScope} from '@/lib/api';
import {Scope} from '@/lib/types';
import {formatControlPlaneError} from '@/lib/presentation';

type ScopeFormState = {
  name: string;
  description: string;
};

const emptyForm: ScopeFormState = {
  name: '',
  description: ''
};

export default function ScopesPage() {
  const t = useTranslations();
  const scopesT = useTranslations('proxyScopes');
  const {session, activeTenant} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';
  const globalSuperAdmin = session?.account.role === 'super_admin';
  const [editingScope, setEditingScope] = useState<Scope | null>(null);
  const [grantScope, setGrantScope] = useState<Scope | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [nameFilter, setNameFilter] = useState('');
  const [descriptionFilter, setDescriptionFilter] = useState('');
  const [formState, setFormState] = useState<ScopeFormState>(emptyForm);

  const scopesQuery = useQuery({
    queryKey: ['proxy-scopes', accessToken, activeTenantId],
    queryFn: () => getScopes(accessToken, activeTenantId),
    enabled: !!accessToken
  });

  const createScopeMutation = useMutation({
    mutationFn: (payload: ScopeFormState) => createScope(accessToken, activeTenantId, payload),
    onSuccess: () => {
      toast.success(scopesT('createSuccess'));
      setFormState(emptyForm);
      setCreateOpen(false);
      queryClient.invalidateQueries({queryKey: ['proxy-scopes']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const updateScopeMutation = useMutation({
    mutationFn: (payload: ScopeFormState) => updateScope(accessToken, activeTenantId, editingScope!.id, {name: payload.name, description: payload.description}),
    onSuccess: () => {
      toast.success(scopesT('updateSuccess'));
      setEditingScope(null);
      setFormState(emptyForm);
      queryClient.invalidateQueries({queryKey: ['proxy-scopes']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const deleteScopeMutation = useMutation({
    mutationFn: (scopeID: string) => deleteScope(accessToken, activeTenantId, scopeID),
    onSuccess: () => {
      toast.success(scopesT('deleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['proxy-scopes']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const handleEdit = (scope: Scope) => {
    setEditingScope(scope);
    setCreateOpen(false);
    setFormState({
      name: scope.name,
      description: scope.description || ''
    });
  };

  const handleCancelEdit = () => {
    setEditingScope(null);
    setCreateOpen(false);
    setFormState(emptyForm);
  };

  const handleOpenCreate = () => {
    setEditingScope(null);
    setFormState(emptyForm);
    setCreateOpen(true);
  };

  const handleSubmit = () => {
    const payload = {
      name: formState.name.trim(),
      description: formState.description.trim()
    };
    if (!payload.name) {
      toast.error(scopesT('required'));
      return;
    }
    if (editingScope) {
      updateScopeMutation.mutate(payload);
      return;
    }
    createScopeMutation.mutate(payload);
  };

  const scopes = scopesQuery.data || [];
  const filteredScopes = useMemo(() => {
    return scopes.filter((scope) =>
      (!nameFilter.trim() || scope.name.toLowerCase().includes(nameFilter.trim().toLowerCase())) &&
      (!descriptionFilter.trim() || String(scope.description || '').toLowerCase().includes(descriptionFilter.trim().toLowerCase()))
    );
  }, [descriptionFilter, nameFilter, scopes]);
  const saving = createScopeMutation.isPending || updateScopeMutation.isPending;
  const modalOpen = createOpen || Boolean(editingScope);

  return (
    <AuthGate>
      <ConsolePage
        actions={canWrite ? (
          <button className="primary-button" onClick={handleOpenCreate} type="button">
            {scopesT('create')}
          </button>
        ) : null}
        title={t('shell.scopeBoard')}
      >
        <ConsoleFilterBar title={t('common.filter')}>
          <ConsoleFilterItem label={t('common.name')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setNameFilter(event.target.value)} placeholder={t('common.name')} value={nameFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={scopesT('description')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setDescriptionFilter(event.target.value)} placeholder={scopesT('description')} value={descriptionFilter} />
          </ConsoleFilterItem>
        </ConsoleFilterBar>

        <ConsoleList count={filteredScopes.length} title={scopesT('listTitle')}>
          {scopesQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={scopesT('loading')} />
          ) : scopesQuery.isError ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(scopesQuery.error)}
              onAction={() => void scopesQuery.refetch()}
              title={scopesT('failed')}
            />
          ) : scopes.length === 0 ? (
            <AsyncState detail={scopesT('empty')} title={t('common.empty')} />
          ) : filteredScopes.length === 0 ? (
            <AsyncState detail={t('common.noMatching')} title={t('common.empty')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('common.name')}</th>
                    <th>{scopesT('description')}</th>
                    <th>{scopesT('updatedAt')}</th>
                    {canWrite ? <th>{t('common.actions')}</th> : null}
                  </tr>
                </thead>
                <tbody>
                  {filteredScopes.map((scope) => (
                    <tr key={scope.id}>
                      <td><NameTag kind="scope">{scope.name}</NameTag></td>
                      <td>{scope.description || <span className="muted-text">-</span>}</td>
                      <td className="mono">{scope.updatedAt}</td>
                      {canWrite ? (
                        <td>
                          <div className="chain-list-actions">
                            {globalSuperAdmin || scope.permission === 'manage' ? (
                              <button className="secondary-button" onClick={() => setGrantScope(scope)} type="button">
                                <Share2 size={14} />
                                {t('common.grant')}
                              </button>
                            ) : null}
                            <button className="secondary-button" disabled={!globalSuperAdmin && scope.permission !== 'manage'} onClick={() => handleEdit(scope)} type="button">
                              <Edit size={14} />
                              {t('common.edit')}
                            </button>
                            <button
                              className="danger-button"
                              disabled={deleteScopeMutation.isPending || (!globalSuperAdmin && scope.permission !== 'manage')}
                              onClick={() => {
                                if (window.confirm(scopesT('deleteConfirm'))) {
                                  deleteScopeMutation.mutate(scope.id);
                                }
                              }}
                              type="button"
                            >
                              <Trash2 size={14} />
                              {t('common.delete')}
                            </button>
                          </div>
                        </td>
                      ) : null}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </ConsoleList>

        <ConsoleCrudModal
          footer={(
            <>
              <button className="secondary-button" onClick={handleCancelEdit} type="button">
                {t('common.cancel')}
              </button>
              <button className="primary-button" disabled={saving} onClick={handleSubmit} type="button">
                {editingScope ? scopesT('save') : scopesT('create')}
              </button>
            </>
          )}
          onClose={handleCancelEdit}
          open={modalOpen}
          subtitle={scopesT('formEyebrow')}
          title={editingScope ? scopesT('editTitle') : scopesT('createTitle')}
        >
          <div className="forms-grid">
              <label className="field-stack">
                <span>{t('common.name')}</span>
                <input
                  className="field-input"
                  onChange={(event) => setFormState((current) => ({...current, name: event.target.value}))}
                  placeholder={scopesT('namePlaceholder')}
                  value={formState.name}
                />
              </label>
              <label className="field-stack" style={{gridColumn: '1 / -1'}}>
                <span>{scopesT('description')}</span>
                <textarea
                  className="field-textarea"
                  onChange={(event) => setFormState((current) => ({...current, description: event.target.value}))}
                  placeholder={scopesT('descriptionPlaceholder')}
                  rows={3}
                  value={formState.description}
                />
              </label>
            </div>
        </ConsoleCrudModal>

        {grantScope ? (
          <ResourceGrantModal
            onChanged={() => queryClient.invalidateQueries({queryKey: ['proxy-scopes']})}
            onClose={() => setGrantScope(null)}
            open={Boolean(grantScope)}
            resourceId={grantScope.id}
            resourceName={grantScope.name}
            resourceType="scope"
          />
        ) : null}
      </ConsolePage>
    </AuthGate>
  );
}
