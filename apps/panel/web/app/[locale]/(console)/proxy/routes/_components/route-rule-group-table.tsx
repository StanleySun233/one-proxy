'use client';

import {UseQueryResult} from '@tanstack/react-query';
import {Share2} from 'lucide-react';

import {AsyncState} from '@/components/async-state';
import {RouteRuleGroup} from '@/lib/types';
import {formatControlPlaneError} from '@/lib/presentation';

type RouteRuleGroupTableProps = {
  groups: RouteRuleGroup[];
  selectedGroupId: string;
  groupsQuery: UseQueryResult<RouteRuleGroup[], Error>;
  deletePending: boolean;
  globalSuperAdmin: boolean;
  t: (key: string) => string;
  routesT: (key: string, values?: Record<string, string | number>) => string;
  onSelect: (groupId: string) => void;
  onGrant?: (group: RouteRuleGroup) => void;
  onEdit?: (group: RouteRuleGroup) => void;
  onDelete?: (group: RouteRuleGroup) => void;
};

export function RouteRuleGroupTable({groups, selectedGroupId, groupsQuery, deletePending, globalSuperAdmin, t, routesT, onSelect, onGrant, onEdit, onDelete}: RouteRuleGroupTableProps) {
  return (
    <>
      {groupsQuery.isPending ? (
        <AsyncState detail={t('common.loading')} title={routesT('loadingGroups')} />
      ) : groupsQuery.isError ? (
        <AsyncState
          actionLabel={t('common.retry')}
          detail={formatControlPlaneError(groupsQuery.error)}
          onAction={() => void groupsQuery.refetch()}
          title={routesT('failedGroups')}
        />
      ) : groups.length === 0 ? (
        <AsyncState detail={routesT('emptyGroups')} title={t('common.empty')} />
      ) : (
        <div className="table-card">
          <table className="data-table">
            <thead>
              <tr>
                <th>{t('common.name')}</th>
                <th>{routesT('rules')}</th>
                <th>{routesT('status')}</th>
                <th>{routesT('routeGroupDescription')}</th>
                <th>{t('common.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {groups.map((group) => {
                const canManage = globalSuperAdmin || group.permission === 'manage';
                const selected = group.id === selectedGroupId;
                return (
                  <tr key={group.id}>
                    <td>
                      <strong>{group.name}</strong>
                      {selected ? <div><span className="badge is-neutral">{routesT('selectedGroup')}</span></div> : null}
                    </td>
                    <td>{routesT('rulesCount', {count: group.ruleCount})}</td>
                    <td>
                      <span className={group.enabled ? 'badge is-good' : 'badge'}>
                        {group.enabled ? t('common.enabled') : t('common.disabled')}
                      </span>
                    </td>
                    <td>{group.description || '-'}</td>
                    <td>
                      <div className="inline-cluster">
                        <button className="secondary-button" onClick={() => onSelect(group.id)} type="button">
                          {routesT('selectGroup')}
                        </button>
                        {onGrant && canManage ? (
                          <button className="secondary-button" onClick={() => onGrant(group)} type="button">
                            <Share2 size={14} />
                            {t('common.grant')}
                          </button>
                        ) : null}
                        {onEdit ? (
                          <button className="secondary-button" disabled={!canManage} onClick={() => onEdit(group)} type="button">
                            {t('common.edit')}
                          </button>
                        ) : null}
                        {onDelete ? (
                          <button className="danger-button" disabled={deletePending || !canManage} onClick={() => onDelete(group)} type="button">
                            {t('common.delete')}
                          </button>
                        ) : null}
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}
