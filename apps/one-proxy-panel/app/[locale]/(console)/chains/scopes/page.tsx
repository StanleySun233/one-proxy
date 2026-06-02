'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {Edit, Trash2} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {useState} from 'react';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {NameTag} from '@/components/common/name-tag';
import {PageHero} from '@/components/page-hero';
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
  const pageT = useTranslations('pages');
  const scopesT = useTranslations('chainsScopes');
  const {session, activeTenant} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';
  const [editingScope, setEditingScope] = useState<Scope | null>(null);
  const [formState, setFormState] = useState<ScopeFormState>(emptyForm);

  const scopesQuery = useQuery({
    queryKey: ['chains-scopes', accessToken, activeTenantId],
    queryFn: () => getScopes(accessToken),
    enabled: !!accessToken
  });

  const createScopeMutation = useMutation({
    mutationFn: (payload: ScopeFormState) => createScope(accessToken, payload),
    onSuccess: () => {
      toast.success(scopesT('createSuccess'));
      setFormState(emptyForm);
      queryClient.invalidateQueries({queryKey: ['chains-scopes']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const updateScopeMutation = useMutation({
    mutationFn: (payload: ScopeFormState) => updateScope(accessToken, editingScope!.id, {name: payload.name, description: payload.description}),
    onSuccess: () => {
      toast.success(scopesT('updateSuccess'));
      setEditingScope(null);
      setFormState(emptyForm);
      queryClient.invalidateQueries({queryKey: ['chains-scopes']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const deleteScopeMutation = useMutation({
    mutationFn: (scopeID: string) => deleteScope(accessToken, scopeID),
    onSuccess: () => {
      toast.success(scopesT('deleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['chains-scopes']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const handleEdit = (scope: Scope) => {
    setEditingScope(scope);
    setFormState({
      name: scope.name,
      description: scope.description || ''
    });
  };

  const handleCancelEdit = () => {
    setEditingScope(null);
    setFormState(emptyForm);
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
  const saving = createScopeMutation.isPending || updateScopeMutation.isPending;

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow={scopesT('eyebrow')} title={pageT('scopesTitle')} />

        {canWrite ? (
          <section className="panel-card">
            <div className="panel-toolbar">
              <div>
                <p className="section-kicker">{scopesT('formEyebrow')}</p>
                <h3>{editingScope ? scopesT('editTitle') : scopesT('createTitle')}</h3>
              </div>
              {editingScope ? (
                <button className="secondary-button" onClick={handleCancelEdit} type="button">
                  {t('common.cancel')}
                </button>
              ) : null}
            </div>

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

            <div className="submit-row">
              <button className="primary-button" disabled={saving} onClick={handleSubmit} type="button">
                {editingScope ? scopesT('save') : scopesT('create')}
              </button>
            </div>
          </section>
        ) : null}

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">{scopesT('listEyebrow')}</p>
              <h3>{scopesT('listTitle')}</h3>
            </div>
            <span className="badge is-neutral">{scopes.length}</span>
          </div>

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
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('common.id')}</th>
                    <th>{t('common.name')}</th>
                    <th>{scopesT('description')}</th>
                    <th>{scopesT('updatedAt')}</th>
                    {canWrite ? <th>{t('common.actions')}</th> : null}
                  </tr>
                </thead>
                <tbody>
                  {scopes.map((scope) => (
                    <tr key={scope.id}>
                      <td className="mono">{scope.id}</td>
                      <td><NameTag kind="scope">{scope.name}</NameTag></td>
                      <td>{scope.description || <span className="muted-text">-</span>}</td>
                      <td className="mono">{scope.updatedAt}</td>
                      {canWrite ? (
                        <td>
                          <div className="chain-list-actions">
                            <button className="secondary-button" onClick={() => handleEdit(scope)} type="button">
                              <Edit size={14} />
                              {t('common.edit')}
                            </button>
                            <button
                              className="danger-button"
                              disabled={deleteScopeMutation.isPending}
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
        </section>
      </div>
    </AuthGate>
  );
}
