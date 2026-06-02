'use client';

import {UseQueryResult} from '@tanstack/react-query';

import {AsyncState} from '@/components/async-state';
import {PolicyRevision} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

type PolicyPanelProps = {
  policies: PolicyRevision[];
  policiesQuery: UseQueryResult<PolicyRevision[], Error>;
  publishPending: boolean;
  t: (key: string) => string;
  routesT: (key: string, values?: Record<string, string | number>) => string;
  onPublish: () => void;
};

export function PolicyPanel({policies, policiesQuery, publishPending, t, routesT, onPublish}: PolicyPanelProps) {
  return (
    <article className="panel-card soft-card">
      <div className="panel-toolbar">
        <h3>{routesT('policies')}</h3>
        <button className="primary-button" disabled={publishPending} onClick={onPublish} type="button">
          {publishPending ? t('common.submitting') : routesT('publishPolicy')}
        </button>
      </div>
      {policiesQuery.isPending ? (
        <AsyncState detail={t('common.loading')} title={routesT('loadingPolicies')} />
      ) : policiesQuery.isError ? (
        <AsyncState
          actionLabel={t('common.retry')}
          detail={formatControlPlaneError(policiesQuery.error)}
          onAction={() => void policiesQuery.refetch()}
          title={routesT('failedPolicies')}
        />
      ) : policies.length === 0 ? (
        <AsyncState detail={routesT('emptyPolicies')} title={t('common.empty')} />
      ) : (
        <div className="stack-list">
          {policies.map((policy) => (
            <div className="stack-item" key={policy.id}>
              <strong>{policy.version}</strong>
              <span className="muted-text">
                {policy.status} · {routesT('nodesCount', {count: policy.assignedNodes})}
              </span>
              <span className="mono">{formatISODateTime(policy.createdAt)}</span>
            </div>
          ))}
        </div>
      )}
    </article>
  );
}
