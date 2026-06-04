'use client';

import {useMutation, useQueries, useQuery, useQueryClient} from '@tanstack/react-query';
import {useMemo, useState} from 'react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {NameTag} from '@/components/common/name-tag';
import {useAuth} from '@/components/auth-provider';
import {ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {deleteGroup, getGroup, listGroups} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';
import type {Group} from '@/lib/types';

import GroupDialog from '../_components/group-dialog';

export default function GroupListPage() {
  const t = useTranslations();
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;

  const [dialogGroup, setDialogGroup] = useState<Group | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [nameFilter, setNameFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

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
  const filteredGroups = useMemo(() => groups.filter((group) =>
    (!nameFilter.trim() || group.name.toLowerCase().includes(nameFilter.trim().toLowerCase())) &&
    (!statusFilter || (statusFilter === 'enabled' ? group.enabled : !group.enabled))
  ), [groups, nameFilter, statusFilter]);

  return (
    <AuthGate>
      <ConsolePage
        actions={(
          <button className="primary-button" onClick={() => setShowCreate(true)} type="button">
            {t('shell.groupCreate')}
          </button>
        )}
        title={t('shell.groupList')}
      >
        <ConsoleFilterBar title={t('common.filter')}>
          <ConsoleFilterItem label={t('common.name')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setNameFilter(event.target.value)} placeholder={t('common.name')} value={nameFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.status')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
              <option value="">{t('common.all')}</option>
              <option value="enabled">{t('common.enabled')}</option>
              <option value="disabled">{t('common.disabled')}</option>
            </select>
          </ConsoleFilterItem>
        </ConsoleFilterBar>
        <ConsoleList count={filteredGroups.length} title={t('shell.groupList')}>
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
          ) : filteredGroups.length === 0 ? (
            <AsyncState detail={t('common.noMatching')} title={t('common.empty')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('common.name')}</th>
                    <th>{t('shell.groupDescription')}</th>
                    <th>{t('shell.groupAccounts')}</th>
                    <th>{t('shell.groupScopes')}</th>
                    <th>{t('common.status')}</th>
                    <th>{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredGroups.map((group) => {
                    const detail = detailsMap.get(group.id);
                    const accountCount = detail?.accounts?.length ?? 0;
                    const scopeCount = detail?.scopes?.length ?? 0;
                    return (
                      <tr key={group.id}>
                        <td><NameTag kind="group">{group.name}</NameTag></td>
                        <td>{group.description || <span className="muted-text">-</span>}</td>
                        <td>{accountCount}</td>
                        <td>{scopeCount}</td>
                        <td>
                          <span className={`badge${group.enabled ? ' is-good' : ' is-neutral'}`}>
                            {group.enabled ? t('shell.groupEnabled') : t('common.disabled')}
                          </span>
                        </td>
                        <td>
                          <div className="chain-list-actions">
                            <button className="secondary-button" onClick={() => setDialogGroup(group)} type="button">
                              {t('common.edit')}
                            </button>
                            <button
                              className="danger-button"
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
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </ConsoleList>

      <GroupDialog
        accessToken={accessToken}
        activeTenantId={activeTenantId}
        group={dialogGroup}
        onClose={() => setDialogGroup(null)}
        onSaved={() => {
          queryClient.invalidateQueries({queryKey: ['groups']});
        }}
        open={!!dialogGroup}
      />

      <GroupDialog
        accessToken={accessToken}
        activeTenantId={activeTenantId}
        group={null}
        onClose={() => setShowCreate(false)}
        onSaved={() => {
          queryClient.invalidateQueries({queryKey: ['groups']});
        }}
        open={showCreate}
      />
      </ConsolePage>
    </AuthGate>
  );
}
