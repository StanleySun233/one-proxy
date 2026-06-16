'use client';

import {ConsoleFilterBar, ConsoleFilterItem} from '@/components/console-template';
import {Chain, Scope} from '@/lib/types';

type SelectOption = {
  value: string;
  label: string;
};

type RouteRuleFilterBarProps = {
  actionTypeOptions: SelectOption[];
  chains: Chain[];
  scopes: Scope[];
  matchFilter: string;
  actionFilter: string;
  chainFilter: string;
  scopeFilter: string;
  statusFilter: string;
  t: (key: string) => string;
  routesT: (key: string) => string;
  onMatchFilterChange: (value: string) => void;
  onActionFilterChange: (value: string) => void;
  onChainFilterChange: (value: string) => void;
  onScopeFilterChange: (value: string) => void;
  onStatusFilterChange: (value: string) => void;
};

export function RouteRuleFilterBar({
  actionTypeOptions,
  chains,
  scopes,
  matchFilter,
  actionFilter,
  chainFilter,
  scopeFilter,
  statusFilter,
  t,
  routesT,
  onMatchFilterChange,
  onActionFilterChange,
  onChainFilterChange,
  onScopeFilterChange,
  onStatusFilterChange
}: RouteRuleFilterBarProps) {
  return (
    <ConsoleFilterBar title={t('common.filter')}>
      <ConsoleFilterItem label={routesT('match')} match={t('common.contains')}>
        <input className="field-input" onChange={(event) => onMatchFilterChange(event.target.value)} placeholder={routesT('match')} value={matchFilter} />
      </ConsoleFilterItem>
      <ConsoleFilterItem label={routesT('actionType')} match={t('common.equals')}>
        <select className="field-select" onChange={(event) => onActionFilterChange(event.target.value)} value={actionFilter}>
          <option value="">{t('common.all')}</option>
          {actionTypeOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select>
      </ConsoleFilterItem>
      <ConsoleFilterItem label={routesT('chain')} match={t('common.equals')}>
        <select className="field-select" onChange={(event) => onChainFilterChange(event.target.value)} value={chainFilter}>
          <option value="">{t('common.all')}</option>
          {chains.map((chain) => <option key={chain.id} value={chain.id}>{chain.name}</option>)}
        </select>
      </ConsoleFilterItem>
      <ConsoleFilterItem label={routesT('scope')} match={t('common.equals')}>
        <select className="field-select" onChange={(event) => onScopeFilterChange(event.target.value)} value={scopeFilter}>
          <option value="">{t('common.all')}</option>
          {scopes.map((scope) => <option key={scope.id} value={scope.id}>{scope.name}</option>)}
        </select>
      </ConsoleFilterItem>
      <ConsoleFilterItem label={routesT('status')} match={t('common.equals')}>
        <select className="field-select" onChange={(event) => onStatusFilterChange(event.target.value)} value={statusFilter}>
          <option value="">{t('common.all')}</option>
          <option value="enabled">{t('common.enabled')}</option>
          <option value="disabled">{t('common.disabled')}</option>
        </select>
      </ConsoleFilterItem>
    </ConsoleFilterBar>
  );
}
