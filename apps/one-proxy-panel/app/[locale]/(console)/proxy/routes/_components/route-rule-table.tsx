'use client';

import {UseQueryResult} from '@tanstack/react-query';

import {AsyncState} from '@/components/async-state';
import {NameTag} from '@/components/common/name-tag';
import {Chain, RouteRule, Scope} from '@/lib/types';
import {formatControlPlaneError} from '@/lib/presentation';

type RouteRuleTableProps = {
  routeRules: RouteRule[];
  chains: Chain[];
  scopes: Scope[];
  routeRulesQuery: UseQueryResult<RouteRule[], Error>;
  deletePending: boolean;
  t: (key: string) => string;
  routesT: (key: string) => string;
  onEdit?: (rule: RouteRule) => void;
  onDelete?: (ruleId: string) => void;
};

export function RouteRuleTable({routeRules, chains, scopes, routeRulesQuery, deletePending, t, routesT, onEdit, onDelete}: RouteRuleTableProps) {
  const scopeById = new Map(scopes.map((scope) => [scope.id, scope]));

  return (
    <>
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
                {onEdit && onDelete ? <th>{t('common.actions')}</th> : null}
              </tr>
            </thead>
            <tbody>
              {routeRules.map((rule) => {
                const chain = chains.find((c) => c.id === rule.chainId);
                const chainName = chain?.name || '';
                const scopeName = rule.destinationScope ? scopeById.get(rule.destinationScope)?.name || t('common.unknown') : '';
                return (
                  <tr key={rule.id}>
                    <td>{rule.priority}</td>
                    <td>
                      <strong>{rule.matchType}</strong>
                      <div className="muted-text mono">{rule.matchValue}</div>
                    </td>
                    <td>{rule.actionType}</td>
                    <td>{chainName ? <NameTag kind="chain">{chainName}</NameTag> : '-'}</td>
                    <td>{scopeName ? <NameTag kind="scope">{scopeName}</NameTag> : '-'}</td>
                    <td>
                      <span className={rule.enabled ? 'badge is-good' : 'badge'}>
                        {rule.enabled ? t('common.enabled') : t('common.disabled')}
                      </span>
                    </td>
                    {onEdit && onDelete ? (
                      <td>
                        <div className="inline-cluster">
                          <button className="secondary-button" onClick={() => onEdit(rule)} type="button">
                            {t('common.edit')}
                          </button>
                          <button
                            className="danger-button"
                            disabled={deletePending}
                            onClick={() => onDelete(rule.id)}
                            type="button"
                          >
                            {t('common.delete')}
                          </button>
                        </div>
                      </td>
                    ) : null}
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
