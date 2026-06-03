'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';
import {useCallback, useEffect, useState} from 'react';

import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {createRouteRule, deleteRouteRule, fetchEnums, getChains, getPolicyRevisions, getRouteRules, getScopes, publishPolicy, updateRouteRule} from '@/lib/api';
import {RouteRule} from '@/lib/types';
import {formatControlPlaneError} from '@/lib/presentation';

import {PolicyPanel} from './_components/policy-panel';
import {ReadOnlyPolicyPanel} from './_components/read-only-policy-panel';
import {ReadOnlyRouteRuleTable} from './_components/read-only-route-rule-table';
import {RegexTesterModal} from './_components/regex-tester-modal';
import {RouteRuleForm} from './_components/route-rule-form';
import {RouteRuleTable} from './_components/route-rule-table';
import {useRouteRuleValidation} from './_hooks/use-route-rule-validation';
import {defaultRouteRuleFormValues, routeRuleFormValues, routeRuleSubmitPayload, RouteRuleFormValues, RouteRuleSubmitPayload} from './_lib/form';
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
  const [editingRuleId, setEditingRuleId] = useState('');
  const formValues = form.watch();
  const {validationPending, validationResult} = useRouteRuleValidation({accessToken, activeTenantId, formValues});

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
          <ReadOnlyPolicyPanel policies={policies} policiesQuery={policiesQuery} routesT={routesT} t={t} />
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
          <ReadOnlyRouteRuleTable
            chains={chains}
            routeRules={routeRules}
            routeRulesQuery={routeRulesQuery}
            routesT={routesT}
            t={t}
          />
        )}

        {regexTesterOpen && (
          <RegexTesterModal initialPattern={form.getValues('matchValue')} onClose={() => setRegexTesterOpen(false)} />
        )}
      </div>
    </AuthGate>
  );
}
