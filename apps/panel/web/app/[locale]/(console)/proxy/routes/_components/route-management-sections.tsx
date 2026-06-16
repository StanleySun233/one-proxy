'use client';

import {UseQueryResult} from '@tanstack/react-query';

import {ConsoleList} from '@/components/console-template';
import {Chain, RouteRule, RouteRuleGroup, Scope} from '@/lib/types';

import {RouteRuleFilterBar} from './route-rule-filter-bar';
import {RouteRuleGroupTable} from './route-rule-group-table';
import {RouteRuleTable} from './route-rule-table';

type SelectOption = {
  value: string;
  label: string;
};

type Translator = (key: string) => string;

type RouteRulesSectionProps = {
  actionFilter: string;
  actionTypeOptions: SelectOption[];
  chainFilter: string;
  chains: Chain[];
  deletePending: boolean;
  globalSuperAdmin: boolean;
  groupFilter: string;
  groups: RouteRuleGroup[];
  matchFilter: string;
  routeRules: RouteRule[];
  routeRulesQuery: UseQueryResult<RouteRule[], Error>;
  routesT: Translator;
  scopeFilter: string;
  scopes: Scope[];
  statusFilter: string;
  t: Translator;
  onActionFilterChange: (value: string) => void;
  onChainFilterChange: (value: string) => void;
  onDelete?: (ruleId: string) => void;
  onEdit?: (rule: RouteRule) => void;
  onGroupFilterChange: (value: string) => void;
  onMatchFilterChange: (value: string) => void;
  onScopeFilterChange: (value: string) => void;
  onStatusFilterChange: (value: string) => void;
};

export function RouteRulesSection({
  actionFilter,
  actionTypeOptions,
  chainFilter,
  chains,
  deletePending,
  globalSuperAdmin,
  groupFilter,
  groups,
  matchFilter,
  routeRules,
  routeRulesQuery,
  routesT,
  scopeFilter,
  scopes,
  statusFilter,
  t,
  onActionFilterChange,
  onChainFilterChange,
  onDelete,
  onEdit,
  onGroupFilterChange,
  onMatchFilterChange,
  onScopeFilterChange,
  onStatusFilterChange
}: RouteRulesSectionProps) {
  return (
    <>
      <RouteRuleFilterBar
        actionFilter={actionFilter}
        actionTypeOptions={actionTypeOptions}
        chainFilter={chainFilter}
        chains={chains}
        groupFilter={groupFilter}
        groups={groups}
        matchFilter={matchFilter}
        onActionFilterChange={onActionFilterChange}
        onChainFilterChange={onChainFilterChange}
        onGroupFilterChange={onGroupFilterChange}
        onMatchFilterChange={onMatchFilterChange}
        onScopeFilterChange={onScopeFilterChange}
        onStatusFilterChange={onStatusFilterChange}
        routesT={routesT}
        scopeFilter={scopeFilter}
        scopes={scopes}
        statusFilter={statusFilter}
        t={t}
      />
      <ConsoleList count={routeRules.length} title={routesT('routeRules')}>
        <RouteRuleTable
          chains={chains}
          deletePending={deletePending}
          globalSuperAdmin={globalSuperAdmin}
          groups={groups}
          onDelete={onDelete}
          onEdit={onEdit}
          routeRules={routeRules}
          routeRulesQuery={routeRulesQuery}
          routesT={routesT}
          scopes={scopes}
          t={t}
        />
      </ConsoleList>
    </>
  );
}

type RouteGroupsSectionProps = {
  deletePending: boolean;
  globalSuperAdmin: boolean;
  groups: RouteRuleGroup[];
  groupsQuery: UseQueryResult<RouteRuleGroup[], Error>;
  routesT: Translator;
  selectedGroupId: string;
  t: Translator;
  onDelete?: (group: RouteRuleGroup) => void;
  onEdit?: (group: RouteRuleGroup) => void;
  onGrant?: (group: RouteRuleGroup) => void;
  onSelect: (groupId: string) => void;
};

export function RouteGroupsSection({deletePending, globalSuperAdmin, groups, groupsQuery, routesT, selectedGroupId, t, onDelete, onEdit, onGrant, onSelect}: RouteGroupsSectionProps) {
  return (
    <ConsoleList count={groups.length} title={routesT('routeGroups')}>
      <RouteRuleGroupTable
        deletePending={deletePending}
        globalSuperAdmin={globalSuperAdmin}
        groups={groups}
        groupsQuery={groupsQuery}
        onDelete={onDelete}
        onEdit={onEdit}
        onGrant={onGrant}
        onSelect={onSelect}
        routesT={routesT}
        selectedGroupId={selectedGroupId}
        t={t}
      />
    </ConsoleList>
  );
}
