'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';
import {useCallback, useEffect, useRef, useState} from 'react';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {NameTag} from '@/components/common/name-tag';
import {PageHero} from '@/components/page-hero';
import {createRouteRule, deleteRouteRule, fetchEnums, getChains, getPolicyRevisions, getRouteRules, getScopes, publishPolicy, updateRouteRule, validateRouteRule} from '@/lib/api';
import {RouteRule, RouteRuleValidationResult} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {PolicyPanel} from './_components/policy-panel';
import {RegexTesterModal} from './_components/regex-tester-modal';
import {RouteRuleForm} from './_components/route-rule-form';
import {RouteRuleTable} from './_components/route-rule-table';
import {defaultRouteRuleFormValues, routeRuleFormValues, routeRuleSubmitPayload, routeRuleValidationPayload, RouteRuleFormValues, RouteRuleSubmitPayload, RouteRuleValidationPayload} from './_lib/form';
import {validateMatchValue} from './_lib/validation';

export default function RoutesPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const routesT = useTranslations('chainsRoutes');
  const {session, activeTenant} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';
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
  const [validationResult, setValidationResult] = useState<RouteRuleValidationResult | null>(null);
  const [validationPending, setValidationPending] = useState(false);
  const [editingRuleId, setEditingRuleId] = useState('');
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const scheduledValidationKeyRef = useRef('');
  const inFlightValidationKeyRef = useRef('');
  const completedValidationKeyRef = useRef('');

  const formValues = form.watch();

  const runValidation = useCallback(async (payload: RouteRuleValidationPayload, validationKey: string) => {
    if (completedValidationKeyRef.current === validationKey || inFlightValidationKeyRef.current === validationKey) {
      return;
    }
    scheduledValidationKeyRef.current = scheduledValidationKeyRef.current === validationKey ? '' : scheduledValidationKeyRef.current;
    inFlightValidationKeyRef.current = validationKey;
    setValidationPending(true);
    try {
      const result = await validateRouteRule(accessToken, payload);
      if (inFlightValidationKeyRef.current === validationKey) {
        setValidationResult(result);
      }
    } catch {
      if (inFlightValidationKeyRef.current === validationKey) {
        setValidationResult(null);
      }
    } finally {
      if (inFlightValidationKeyRef.current === validationKey) {
        completedValidationKeyRef.current = validationKey;
        inFlightValidationKeyRef.current = '';
        setValidationPending(false);
      }
    }
  }, [accessToken]);

  useEffect(() => {
    const payload = routeRuleValidationPayload(formValues);
    const validationKey = `${accessToken}:${activeTenantId}:${JSON.stringify(payload)}`;

    if (!accessToken || !payload.matchValue) {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
        debounceRef.current = null;
      }
      scheduledValidationKeyRef.current = '';
      inFlightValidationKeyRef.current = '';
      completedValidationKeyRef.current = '';
      setValidationResult(null);
      setValidationPending(false);
      return;
    }

    if (
      completedValidationKeyRef.current === validationKey ||
      inFlightValidationKeyRef.current === validationKey ||
      scheduledValidationKeyRef.current === validationKey
    ) {
      return;
    }

    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
      debounceRef.current = null;
    }

    scheduledValidationKeyRef.current = validationKey;
    debounceRef.current = setTimeout(() => {
      runValidation(payload, validationKey);
    }, 500);
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
        debounceRef.current = null;
      }
    };
  }, [accessToken, activeTenantId, formValues, runValidation]);

  const routeRulesQuery = useQuery({
    queryKey: ['route-rules', accessToken, activeTenantId],
    queryFn: () => getRouteRules(accessToken),
    enabled: !!accessToken
  });
  const chainsQuery = useQuery({
    queryKey: ['chains', accessToken, activeTenantId],
    queryFn: () => getChains(accessToken),
    enabled: !!accessToken
  });
  const scopesQuery = useQuery({
    queryKey: ['chains-scopes', accessToken, activeTenantId],
    queryFn: () => getScopes(accessToken),
    enabled: !!accessToken
  });
  const policiesQuery = useQuery({
    queryKey: ['policies-revisions', accessToken, activeTenantId],
    queryFn: () => getPolicyRevisions(accessToken),
    enabled: !!accessToken
  });
  const createRuleMutation = useMutation({
    mutationFn: (payload: RouteRuleSubmitPayload) => createRouteRule(accessToken, payload),
    onSuccess: () => {
      toast.success(routesT('createSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      form.reset(defaultRouteRuleFormValues());
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const updateRuleMutation = useMutation({
    mutationFn: (payload: RouteRuleSubmitPayload & {id: string}) => updateRouteRule(accessToken, payload.id, {
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
    mutationFn: (ruleId: string) => deleteRouteRule(accessToken, ruleId),
    onSuccess: () => {
      toast.success(routesT('deleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      if (editingRuleId) {
        setEditingRuleId('');
        form.reset(defaultRouteRuleFormValues());
      }
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const publishMutation = useMutation({
    mutationFn: () => publishPolicy(accessToken),
    onSuccess: () => {
      toast.success(routesT('publishSuccess'));
      queryClient.invalidateQueries({queryKey: ['policies-revisions']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const routeRules = routeRulesQuery.data || [];
  const policies = policiesQuery.data || [];
  const chains = chainsQuery.data || [];
  const scopes = scopesQuery.data || [];


  const selectedChain = chains.find((c) => c.id === selectedChainId);
  const matchValuePlaceholder = matchType === 'default' ? '*' : routesT('matchValuePlaceholder', {type: matchType || routesT('value')});
  const editingRule = routeRules.find((rule) => rule.id === editingRuleId);

  const resetForm = useCallback(() => {
    setEditingRuleId('');
    form.reset(defaultRouteRuleFormValues());
  }, [form]);

  const startEdit = useCallback((rule: RouteRule) => {
    setEditingRuleId(rule.id);
    form.reset(routeRuleFormValues(rule));
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
    if (window.confirm(routesT('deleteConfirm'))) {
      deleteRuleMutation.mutate(ruleId);
    }
  }, [deleteRuleMutation, routesT]);

  useEffect(() => {
    if (actionType !== 'chain') {
      return;
    }
    const nextScope = selectedChain?.destinationScope || '';
    if (form.getValues('destinationScope') !== nextScope) {
      form.setValue('destinationScope', nextScope, {shouldDirty: false, shouldValidate: false});
    }
  }, [actionType, form, selectedChain?.destinationScope]);

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow={t('shell.routeBoard')} title={pageT('routesTitle')} />

        {canWrite ? (
          <section className="forms-grid">
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

            <PolicyPanel
              onPublish={() => publishMutation.mutate()}
              policies={policies}
              policiesQuery={policiesQuery}
              publishPending={publishMutation.isPending}
              routesT={routesT}
              t={t}
            />
          </section>
        ) : (
          <section className="panel-card soft-card">
            <div className="panel-toolbar">
              <h3>{routesT('policies')}</h3>
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
          </section>
        )}

        {canWrite ? (
          <RouteRuleTable
            chains={chains}
            deletePending={deleteRuleMutation.isPending}
            onDelete={deleteRoute}
            onEdit={startEdit}
            routeRules={routeRules}
            routeRulesQuery={routeRulesQuery}
            routesT={routesT}
            t={t}
          />
        ) : (
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
        )}

        {regexTesterOpen && (
          <RegexTesterModal initialPattern={form.getValues('matchValue')} onClose={() => setRegexTesterOpen(false)} />
        )}
      </div>
    </AuthGate>
  );
}
