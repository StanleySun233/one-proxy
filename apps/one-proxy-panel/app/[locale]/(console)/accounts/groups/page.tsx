'use client';

import {useMutation, useQueries, useQuery, useQueryClient} from '@tanstack/react-query';
import {useState} from 'react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {deleteGroup, getGroup, listGroups} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';
import type {Group} from '@/lib/types';

import GroupDialog from '../_components/group-dialog';

export default function GroupListPage() {
  const t = useTranslations();
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';

  const [dialogGroup, setDialogGroup] = useState<Group | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  const groupsQuery = useQuery({
    queryKey: ['groups', accessToken],
    queryFn: () => listGroups(accessToken),
    enabled: !!accessToken
  });

  const groups = groupsQuery.data || [];

  const detailQueries = useQueries({
    queries: groups.map((group) => ({
      queryKey: ['groups', group.id, accessToken],
      queryFn: () => getGroup(accessToken, group.id),
      enabled: !!accessToken && groups.length > 0
    }))
  });

  const deleteMutation = useMutation({
    mutationFn: (groupID: string) => deleteGroup(accessToken, groupID),
    onSuccess: () => {
      toast.success(t('shell.groupDeleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['groups']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const detailsMap = new Map(
    detailQueries
      .filter((q) => q.data)
      .map((q) => [q.data!.id, q.data!])
  );

  const isPending = groupsQuery.isPending || (groups.length > 0 && detailQueries.some((q) => q.isPending));

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow={t('accounts.listTitle')} title={t('shell.groupList')} description={t('shell.groupList')} />
        <article className="panel-card soft-card">
          <div className="stack-head">
            <h3>{t('shell.groupList')}</h3>
            <button className="primary-button" onClick={() => setShowCreate(true)} type="button">
              {t('shell.groupCreate')}
            </button>
          </div>
          {isPending ? (
            <AsyncState detail={t('common.loading')} title={t('shell.groupList')} />
          ) : groupsQuery.isError ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(groupsQuery.error)}
              onAction={() => void groupsQuery.refetch()}
              title={t('accounts.failedToLoadGroups')}
            />
          ) : groups.length === 0 ? (
            <AsyncState detail={t('accounts.emptyGroupsList')} title={t('common.empty')} />
          ) : (
            <div className="stack-list">
              {groups.map((group) => {
                const detail = detailsMap.get(group.id);
                const accountCount = detail?.accounts?.length ?? 0;
                const scopeCount = detail?.scopes?.length ?? 0;

                return (
                  <div className="stack-item" key={group.id}>
                    <div className="stack-head">
                      <strong>{group.name}</strong>
                      <span className={`badge${group.enabled ? ' is-good' : ' is-neutral'}`}>
                        {group.enabled ? t('shell.groupEnabled') : t('common.disabled')}
                      </span>
                    </div>
                    {group.description ? <span className="muted-text">{group.description}</span> : null}
                    <div className="inline-cluster">
                      <span className="badge is-neutral">{t('shell.groupAccounts')}: {accountCount}</span>
                      <span className="badge is-neutral">{t('shell.groupScopes')}: {scopeCount}</span>
                    </div>
                    <div className="stack-actions">
                      <button
                        className="secondary-button"
                        onClick={() => setDialogGroup(group)}
                        type="button"
                      >
                        {t('common.edit')}
                      </button>
                      <button
                        className="secondary-button"
                        disabled={deleteMutation.isPending}
                        onClick={() => {
                          if (window.confirm(t('shell.groupDeleteConfirm'))) {
                            deleteMutation.mutate(group.id);
                          }
                        }}
                        type="button"
                      >
                        {t('common.delete')}
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </article>
      </div>

      <GroupDialog
        accessToken={accessToken}
        group={dialogGroup}
        onClose={() => setDialogGroup(null)}
        onSaved={() => {
          queryClient.invalidateQueries({queryKey: ['groups']});
        }}
        open={!!dialogGroup}
      />

      <GroupDialog
        accessToken={accessToken}
        group={null}
        onClose={() => setShowCreate(false)}
        onSaved={() => {
          queryClient.invalidateQueries({queryKey: ['groups']});
        }}
        open={showCreate}
      />
    </AuthGate>
  );
}
