'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';
import {useCallback, useEffect, useMemo, useState} from 'react';

import {AuthGate} from '@/components/auth-gate';
import {AsyncState} from '@/components/async-state';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {DeleteConfirmationModal, DeleteImpactSection} from '@/components/delete-confirmation-modal';
import {ResourceGrantModal} from '@/components/resource-grant-modal';
import {useAuth} from '@/components/auth-provider';
import {createRouteRule, deleteRouteRule, fetchEnums, getChains, getPolicyRevisions, getRouteRules, getScopes, publishPolicy, updateRouteRule} from '@/lib/api';
import {RouteRule} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {RegexTesterModal} from './_components/regex-tester-modal';
import {RouteRuleForm} from './_components/route-rule-form';
import {RouteRuleTable} from './_components/route-rule-table';
import {useRouteRuleValidation} from './_hooks/use-route-rule-validation';
import {defaultRouteRuleFormValues, routeRuleFormValues, routeRuleSubmitPayload, RouteRuleFormValues, RouteRuleSubmitPayload} from './_lib/form';
import {validateMatchValue} from './_lib/validation';

export default function RoutesPage() {
  const t = useTranslations();
  const routesT = useTranslations('proxyRoutes');
  const {session, activeTenant} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';
  const globalSuperAdmin = session?.account.role === 'super_admin';
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const matchTypeKeys = Object.keys(enums?.match_type || {});
  const actionTypeKeys = Object.keys(enums?.action_type || {});
  const DEFAULT_MATCH_TYPE = matchTypeKeys.find(k => k === 'domain') || 'domain';
  const DEFAULT_ACTION_TYPE = actionTypeKeys.find(k => k === 'chain') || 'chain';
  const form = useForm<RouteRuleFormValues>({
    defaultValues: {...defaultRouteRuleFormValues(), matchType: DEFAULT_MATCH_TYPE, actionType: DEFAULT_ACTION_TYPE}
  });
  const actionType = form.watch('actionType');
  const matchType = form.watch('matchType');
  const matchTypeOptions = enums?.match_type ? Object.entries(enums.match_type).map(([value, item]) => ({value, label: item.name})) : [];
  const actionTypeOptions = enums?.action_type ? Object.entries(enums.action_type).map(([value, item]) => ({value, label: item.name})) : [];
  const selectedChainId = form.watch('chainId');
  const [regexTesterOpen, setRegexTesterOpen] = useState(false);
  const [editingRuleId, setEditingRuleId] = useState('');
  const [grantRule, setGrantRule] = useState<RouteRule | null>(null);
  const [deletingRule, setDeletingRule] = useState<RouteRule | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [matchFilter, setMatchFilter] = useState('');
  const [chainFilter, setChainFilter] = useState('');
  const [scopeFilter, setScopeFilter] = useState('');
  const [actionFilter, setActionFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const formValues = form.watch();
  const {validationPending, validationResult} = useRouteRuleValidation({accessToken, activeTenantId, editingRuleId, formValues});

  const routeRulesQuery = useQuery({
    queryKey: ['route-rules', accessToken, activeTenantId],
    queryFn: () => getRouteRules(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const chainsQuery = useQuery({
    queryKey: ['proxy-chains', accessToken, activeTenantId],
    queryFn: () => getChains(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const scopesQuery = useQuery({
    queryKey: ['proxy-scopes', accessToken, activeTenantId],
    queryFn: () => getScopes(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const policyRevisionsQuery = useQuery({
    queryKey: ['policy-revisions', accessToken, activeTenantId],
    queryFn: () => getPolicyRevisions(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const createRuleMutation = useMutation({
    mutationFn: (payload: RouteRuleSubmitPayload) => createRouteRule(accessToken, activeTenantId, payload),
    onSuccess: () => {
      toast.success(routesT('createSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      setCreateOpen(false);
      form.reset(defaultRouteRuleFormValues());
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const updateRuleMutation = useMutation({
    mutationFn: (payload: RouteRuleSubmitPayload & {id: string}) => updateRouteRule(accessToken, activeTenantId, payload.id, {
      priority: payload.priority,
      matchType: payload.matchType,
      matchValue: payload.matchValue,
      actionType: payload.actionType,
      chainId: payload.chainId,
      destinationScope: payload.destinationScope,
      enabled: payload.enabled
    }),
    onSuccess: () => {
      toast.success(routesT('updateSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      setEditingRuleId('');
      form.reset(defaultRouteRuleFormValues());
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const deleteRuleMutation = useMutation({
    mutationFn: (ruleId: string) => deleteRouteRule(accessToken, activeTenantId, ruleId),
    onSuccess: () => {
      toast.success(routesT('deleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      if (editingRuleId) {
        setEditingRuleId('');
        form.reset(defaultRouteRuleFormValues());
      }
      setDeletingRule(null);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const publishMutation = useMutation({
    mutationFn: () => publishPolicy(accessToken, activeTenantId),
    onSuccess: () => {
      toast.success(routesT('publishSuccess'));
      queryClient.invalidateQueries({queryKey: ['policy-revisions']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const routeRules = routeRulesQuery.data || [];
  const policyRevisions = policyRevisionsQuery.data || [];
  const chains = chainsQuery.data || [];
  const scopes = scopesQuery.data || [];
  const filteredRouteRules = useMemo(() => {
    return routeRules.filter((rule) => {
      if (actionFilter && rule.actionType !== actionFilter) {
        return false;
      }
      if (chainFilter && rule.chainId !== chainFilter) {
        return false;
      }
      if (scopeFilter && rule.destinationScope !== scopeFilter) {
        return false;
      }
      if (statusFilter === 'enabled' && !rule.enabled) {
        return false;
      }
      if (statusFilter === 'disabled' && rule.enabled) {
        return false;
      }
      if (!matchFilter.trim()) {
        return true;
      }
      const keyword = matchFilter.trim().toLowerCase();
      return [rule.matchType, rule.matchValue]
        .some((value) => String(value || '').toLowerCase().includes(keyword));
    });
  }, [actionFilter, chainFilter, matchFilter, routeRules, scopeFilter, statusFilter]);

  const selectedChain = chains.find((c) => c.id === selectedChainId);
  const matchValuePlaceholder = matchType === 'default' ? '*' : routesT('matchValuePlaceholder', {type: matchType || routesT('value')});
  const editingRule = routeRules.find((rule) => rule.id === editingRuleId);

  const resetForm = useCallback(() => {
    setEditingRuleId('');
    setCreateOpen(false);
    form.reset(defaultRouteRuleFormValues());
  }, [form]);

  const startEdit = useCallback((rule: RouteRule) => {
    setCreateOpen(false);
    setEditingRuleId(rule.id);
    form.reset(routeRuleFormValues(rule));
  }, [form]);

  const startCreate = useCallback(() => {
    setEditingRuleId('');
    setCreateOpen(true);
    form.reset(defaultRouteRuleFormValues());
  }, [form]);

  const submitRouteRule = useCallback((values: RouteRuleFormValues) => {
    const payload = routeRuleSubmitPayload(values, chains);
    if (editingRuleId) {
      updateRuleMutation.mutate({id: editingRuleId, ...payload});
    } else {
      createRuleMutation.mutate(payload);
    }
  }, [chains, createRuleMutation, editingRuleId, updateRuleMutation]);

  const deleteRoute = useCallback((ruleId: string) => {
    setDeletingRule(routeRules.find((rule) => rule.id === ruleId) || null);
  }, [routeRules]);

  useEffect(() => {
    if (actionType !== 'chain') {
      return;
    }
    const nextScope = selectedChain?.destinationScope || '';
    if (form.getValues('destinationScope') !== nextScope) {
      form.setValue('destinationScope', nextScope, {shouldDirty: false, shouldValidate: false});
    }
  }, [actionType, form, selectedChain?.destinationScope]);
  const modalOpen = createOpen || Boolean(editingRule);
  const routeDeleteSections: DeleteImpactSection[] = deletingRule ? [
    {
      id: 'routeRule',
      label: routesT('deleteImpactRouteRule'),
      items: [{
        id: deletingRule.id,
        name: deletingRule.matchValue || String(deletingRule.priority),
        detail: `${deletingRule.matchType} / ${deletingRule.actionType}`
      }]
    },
    {
      id: 'tenantBindings',
      label: routesT('deleteImpactTenantBindings'),
      count: deletingRule.permission ? 1 : 0
    }
  ] : [];

  return (
    <AuthGate>
      <ConsolePage
        actions={canWrite ? (
          <>
            <button className="secondary-button" disabled={publishMutation.isPending} onClick={() => publishMutation.mutate()} type="button">
              {publishMutation.isPending ? t('common.submitting') : routesT('publishRevision')}
            </button>
            <button className="primary-button" onClick={startCreate} type="button">
              {routesT('createRule')}
            </button>
          </>
        ) : null}
        title={t('shell.routeBoard')}
      >
        <ConsoleFilterBar title={t('common.filter')}>
          <ConsoleFilterItem label={routesT('match')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setMatchFilter(event.target.value)} placeholder={routesT('match')} value={matchFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={routesT('actionType')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setActionFilter(event.target.value)} value={actionFilter}>
              <option value="">{t('common.all')}</option>
              {actionTypeOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
          </ConsoleFilterItem>
          <ConsoleFilterItem label={routesT('chain')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setChainFilter(event.target.value)} value={chainFilter}>
              <option value="">{t('common.all')}</option>
              {chains.map((chain) => <option key={chain.id} value={chain.id}>{chain.name}</option>)}
            </select>
          </ConsoleFilterItem>
          <ConsoleFilterItem label={routesT('scope')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setScopeFilter(event.target.value)} value={scopeFilter}>
              <option value="">{t('common.all')}</option>
              {scopes.map((scope) => <option key={scope.id} value={scope.id}>{scope.name}</option>)}
            </select>
          </ConsoleFilterItem>
          <ConsoleFilterItem label={routesT('status')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
              <option value="">{t('common.all')}</option>
              <option value="enabled">{t('common.enabled')}</option>
              <option value="disabled">{t('common.disabled')}</option>
            </select>
          </ConsoleFilterItem>
        </ConsoleFilterBar>

        <ConsoleList count={filteredRouteRules.length} title={routesT('routeRules')}>
          <RouteRuleTable
            chains={chains}
            deletePending={deleteRuleMutation.isPending}
            globalSuperAdmin={globalSuperAdmin}
            onGrant={canWrite ? setGrantRule : undefined}
            onDelete={canWrite ? deleteRoute : undefined}
            onEdit={canWrite ? startEdit : undefined}
            routeRules={filteredRouteRules}
            routeRulesQuery={routeRulesQuery}
            routesT={routesT}
            scopes={scopes}
            t={t}
          />
        </ConsoleList>

        <ConsoleList count={policyRevisions.length} title={routesT('publishHistory')}>
          {policyRevisionsQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={routesT('loadingPublishHistory')} />
          ) : policyRevisionsQuery.isError ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(policyRevisionsQuery.error)}
              onAction={() => void policyRevisionsQuery.refetch()}
              title={routesT('failedPublishHistory')}
            />
          ) : policyRevisions.length === 0 ? (
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
                  {policyRevisions.map((revision) => (
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

        {canWrite ? (
          <ConsoleCrudModal
            onClose={resetForm}
            open={modalOpen}
            subtitle={routesT('validation')}
            title={editingRule ? routesT('editRule') : routesT('createRule')}
          >
            <RouteRuleForm
              actionType={actionType}
              actionTypeOptions={actionTypeOptions}
              chains={chains}
              createPending={createRuleMutation.isPending}
              editingRule={!!editingRule}
              form={form}
              matchType={matchType}
              matchTypeOptions={matchTypeOptions}
              matchValuePlaceholder={matchValuePlaceholder}
              onCancel={resetForm}
              onOpenRegexTester={() => setRegexTesterOpen(true)}
              onSubmit={submitRouteRule}
              routesT={routesT}
              scopes={scopes}
              selectedChain={selectedChain}
              t={t}
              updatePending={updateRuleMutation.isPending}
              validateMatchValue={(type, value) => validateMatchValue(type, value, routesT)}
              validationPending={validationPending}
              validationResult={validationResult}
            />
          </ConsoleCrudModal>
        ) : null}

        {regexTesterOpen && (
          <RegexTesterModal initialPattern={form.getValues('matchValue')} onClose={() => setRegexTesterOpen(false)} />
        )}

        {grantRule ? (
          <ResourceGrantModal
            onChanged={() => queryClient.invalidateQueries({queryKey: ['route-rules']})}
            onClose={() => setGrantRule(null)}
            open={Boolean(grantRule)}
            resourceId={grantRule.id}
            resourceName={grantRule.matchValue || String(grantRule.priority)}
            resourceType="route_rule"
          />
        ) : null}

        <DeleteConfirmationModal
          onClose={() => setDeletingRule(null)}
          onConfirm={() => {
            if (deletingRule) {
              deleteRuleMutation.mutate(deletingRule.id);
            }
          }}
          open={Boolean(deletingRule)}
          pending={deleteRuleMutation.isPending}
          sections={routeDeleteSections}
          targetName={deletingRule?.matchValue || ''}
          title={routesT('deleteConfirmTitle')}
        />
      </ConsolePage>
    </AuthGate>
  );
}
