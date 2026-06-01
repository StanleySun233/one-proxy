'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';
import {useCallback, useEffect, useRef, useState} from 'react';

import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {createRouteRule, deleteRouteRule, fetchEnums, getChains, getPolicyRevisions, getRouteRules, getScopes, publishPolicy, updateRouteRule, validateRouteRule} from '@/lib/api';
import {RouteRule, RouteRuleValidationResult} from '@/lib/types';
import {formatControlPlaneError} from '@/lib/presentation';

import {PolicyPanel} from './_components/policy-panel';
import {RegexTesterModal} from './_components/regex-tester-modal';
import {RouteRuleForm, RouteRuleFormValues} from './_components/route-rule-form';
import {RouteRuleTable} from './_components/route-rule-table';

type RouteRuleValidationPayload = {
  priority: number;
  matchType: string;
  matchValue: string;
  actionType: string;
  chainId: string;
  destinationScope: string;
};

function routeRuleValidationPayload(values: RouteRuleFormValues): RouteRuleValidationPayload {
  return {
    priority: Number(values.priority) || 0,
    matchType: values.matchType,
    matchValue: values.matchValue.trim(),
    actionType: values.actionType,
    chainId: values.chainId.trim(),
    destinationScope: values.destinationScope.trim()
  };
}

function validateMatchValue(matchType: string, value: string, t: (key: string) => string): string | true {
  const trimmed = value.trim();
  if (!trimmed) return t('matchValueRequired');

  switch (matchType) {
    case 'domain':
      if (!/^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/i.test(trimmed)) {
        return t('invalidDomain');
      }
      break;
    case 'domain_suffix':
      if (!/^\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$/i.test(trimmed)) {
        return t('invalidDomainSuffix');
      }
      break;
    case 'ip_cidr':
      if (!/^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/.test(trimmed)) {
        return t('invalidCidr');
      }
      break;
    case 'ip_range':
      if (!/^(\d{1,3}\.){3}\d{1,3}-(\d{1,3}\.){3}\d{1,3}$/.test(trimmed)) {
        return t('invalidIpRange');
      }
      break;
    case 'port':
      const port = Number(trimmed);
      if (isNaN(port) || port < 1 || port > 65535) {
        return t('invalidPort');
      }
      break;
    case 'url_regex':
      try {
        new RegExp(trimmed);
      } catch {
        return t('invalidRegex');
      }
      break;
  }
  return true;
}

export default function RoutesPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const routesT = useTranslations('routes');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const matchTypeKeys = Object.keys(enums?.match_type || {});
  const actionTypeKeys = Object.keys(enums?.action_type || {});
  const DEFAULT_MATCH_TYPE = matchTypeKeys.find(k => k === 'domain') || 'domain';
  const DEFAULT_ACTION_TYPE = actionTypeKeys.find(k => k === 'chain') || 'chain';
  const form = useForm<RouteRuleFormValues>({
    defaultValues: {
      priority: '100',
      matchType: DEFAULT_MATCH_TYPE,
      matchValue: '',
      actionType: DEFAULT_ACTION_TYPE,
      chainId: '',
      destinationScope: '',
      enabled: true
    }
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
    const validationKey = `${accessToken}:${JSON.stringify(payload)}`;

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
  }, [accessToken, formValues, runValidation]);

  const routeRulesQuery = useQuery({
    queryKey: ['route-rules', accessToken],
    queryFn: () => getRouteRules(accessToken),
    enabled: !!accessToken
  });
  const chainsQuery = useQuery({
    queryKey: ['chains', accessToken],
    queryFn: () => getChains(accessToken),
    enabled: !!accessToken
  });
  const scopesQuery = useQuery({
    queryKey: ['scopes', accessToken],
    queryFn: () => getScopes(accessToken),
    enabled: !!accessToken
  });
  const policiesQuery = useQuery({
    queryKey: ['policies-revisions', accessToken],
    queryFn: () => getPolicyRevisions(accessToken),
    enabled: !!accessToken
  });
  const createRuleMutation = useMutation({
    mutationFn: (payload: {
      priority: number;
      matchType: string;
      matchValue: string;
      actionType: string;
      chainId: string;
      destinationScope: string;
    }) => createRouteRule(accessToken, payload),
    onSuccess: () => {
      toast.success(routesT('createSuccess'));
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      form.reset({
        priority: '100',
        matchType: 'domain',
        matchValue: '',
        actionType: 'chain',
        chainId: '',
        destinationScope: '',
        enabled: true
      });
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const updateRuleMutation = useMutation({
    mutationFn: (payload: {
      id: string;
      priority: number;
      matchType: string;
      matchValue: string;
      actionType: string;
      chainId: string;
      destinationScope: string;
      enabled: boolean;
    }) => updateRouteRule(accessToken, payload.id, {
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
      form.reset({
        priority: '100',
        matchType: 'domain',
        matchValue: '',
        actionType: 'chain',
        chainId: '',
        destinationScope: '',
        enabled: true
      });
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
        form.reset({
          priority: '100',
          matchType: 'domain',
          matchValue: '',
          actionType: 'chain',
          chainId: '',
          destinationScope: '',
          enabled: true
        });
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
    form.reset({
      priority: '100',
      matchType: 'domain',
      matchValue: '',
      actionType: 'chain',
      chainId: '',
      destinationScope: '',
      enabled: true
    });
  }, [form]);

  const startEdit = useCallback((rule: RouteRule) => {
    setEditingRuleId(rule.id);
    form.reset({
      priority: String(rule.priority),
      matchType: rule.matchType,
      matchValue: rule.matchValue,
      actionType: rule.actionType,
      chainId: rule.chainId || '',
      destinationScope: rule.destinationScope || '',
      enabled: rule.enabled
    });
  }, [form]);

  const submitRouteRule = useCallback((values: RouteRuleFormValues) => {
    const chainDestinationScope = values.actionType === 'chain'
      ? chains.find((chain) => chain.id === values.chainId)?.destinationScope || ''
      : values.destinationScope.trim();
    const payload = {
      priority: Number(values.priority),
      matchType: values.matchType.trim(),
      matchValue: values.matchValue.trim(),
      actionType: values.actionType,
      chainId: values.chainId.trim(),
      destinationScope: chainDestinationScope,
      enabled: values.enabled
    };
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
        <PageHero eyebrow={t('nav.routes')} title={pageT('routesTitle')} />

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

        {regexTesterOpen && (
          <RegexTesterModal initialPattern={form.getValues('matchValue')} onClose={() => setRegexTesterOpen(false)} />
        )}
      </div>
    </AuthGate>
  );
}
