'use client';

import {UseQueryResult} from '@tanstack/react-query';

import {AsyncState} from '@/components/async-state';
import {NameTag} from '@/components/common/name-tag';
import {Chain, RouteRule} from '@/lib/types';
import {formatControlPlaneError} from '@/lib/presentation';

type ReadOnlyRouteRuleTableProps = {
  routeRules: RouteRule[];
  chains: Chain[];
  routeRulesQuery: UseQueryResult<RouteRule[], Error>;
  t: (key: string) => string;
  routesT: (key: string) => string;
};

export function ReadOnlyRouteRuleTable({routeRules, chains, routeRulesQuery, t, routesT}: ReadOnlyRouteRuleTableProps) {
  return (
    <article className="panel-card">
      <div className="panel-toolbar">
        <h3>{routesT('routeRules')}</h3>
        <span className="badge">{routeRules.length}</span>
      </div>
      {routeRulesQuery.isPending ? (
        <AsyncState detail={t('common.loading')} title={routesT('loadingRules')} />
      ) : routeRulesQuery.isError ? (
        <AsyncState
          actionLabel={t('common.retry')}
          detail={formatControlPlaneError(routeRulesQuery.error)}
          onAction={() => void routeRulesQuery.refetch()}
          title={routesT('failedRules')}
        />
      ) : routeRules.length === 0 ? (
        <AsyncState detail={routesT('emptyRules')} title={t('common.empty')} />
      ) : (
        <div className="table-card">
          <table className="data-table">
            <thead>
              <tr>
                <th>{routesT('priority')}</th>
                <th>{routesT('match')}</th>
                <th>{routesT('action')}</th>
                <th>{routesT('chain')}</th>
                <th>{routesT('scope')}</th>
                <th>{routesT('status')}</th>
              </tr>
            </thead>
            <tbody>
              {routeRules.map((rule) => {
                const chain = chains.find((c) => c.id === rule.chainId);
                const chainName = chain?.name || rule.chainId;
                return (
                  <tr key={rule.id}>
                    <td>{rule.priority}</td>
                    <td>
                      <strong>{rule.matchType}</strong>
                      <div className="muted-text mono">{rule.matchValue}</div>
                    </td>
                    <td>{rule.actionType}</td>
                    <td>{chainName ? <NameTag kind="chain">{chainName}</NameTag> : '-'}</td>
                    <td>{rule.destinationScope ? <NameTag kind="scope">{rule.destinationScope}</NameTag> : '-'}</td>
                    <td>
                      <span className={rule.enabled ? 'badge is-good' : 'badge'}>
                        {rule.enabled ? t('common.enabled') : t('common.disabled')}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </article>
  );
}
