'use client';

import {UseQueryResult} from '@tanstack/react-query';

import {AsyncState} from '@/components/async-state';
import {ConsoleList} from '@/components/console-template';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';
import {PolicyRevision} from '@/lib/types';

type RoutePolicyHistoryProps = {
  revisions: PolicyRevision[];
  query: UseQueryResult<PolicyRevision[], Error>;
  t: (key: string) => string;
  routesT: (key: string, values?: Record<string, string | number>) => string;
};

export function RoutePolicyHistory({revisions, query, t, routesT}: RoutePolicyHistoryProps) {
  return (
    <ConsoleList count={revisions.length} title={routesT('publishHistory')}>
      {query.isPending ? (
        <AsyncState detail={t('common.loading')} title={routesT('loadingPublishHistory')} />
      ) : query.isError ? (
        <AsyncState
          actionLabel={t('common.retry')}
          detail={formatControlPlaneError(query.error)}
          onAction={() => void query.refetch()}
          title={routesT('failedPublishHistory')}
        />
      ) : revisions.length === 0 ? (
        <AsyncState detail={routesT('emptyPublishHistory')} title={t('common.empty')} />
      ) : (
        <div className="table-card">
          <table className="data-table">
            <thead>
              <tr>
                <th>{routesT('revision')}</th>
                <th>{routesT('status')}</th>
                <th>{t('common.target')}</th>
                <th>{t('common.updated')}</th>
              </tr>
            </thead>
            <tbody>
              {revisions.map((revision) => (
                <tr key={revision.id}>
                  <td className="mono">{revision.version}</td>
                  <td>{revision.status}</td>
                  <td>{routesT('nodesCount', {count: revision.assignedNodes})}</td>
                  <td className="mono">{formatISODateTime(revision.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </ConsoleList>
  );
}
