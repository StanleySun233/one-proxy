'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';
import {useCallback, useEffect, useRef, useState} from 'react';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {createRouteRule, fetchEnums, getChains, getNodes, getPolicyRevisions, getRouteRules, publishPolicy, validateRouteRule} from '@/lib/api';
import {RouteRuleValidationResult} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {RegexTesterModal} from './_components/regex-tester-modal';

type RouteRuleFormValues = {
  priority: string;
  matchType: string;
  matchValue: string;
  actionType: string;
  chainId: string;
  destinationScope: string;
};

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
      destinationScope: ''
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
  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken],
    queryFn: () => getNodes(accessToken),
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
        destinationScope: ''
      });
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
  const nodes = nodesQuery.data || [];


  const selectedChain = chains.find((c) => c.id === selectedChainId);
  const availableScopes = Array.from(new Set([...nodes.map((n) => n.scopeKey).filter(Boolean), ...chains.map((c) => c.destinationScope)]));
  const matchValuePlaceholder = matchType === 'default' ? '*' : routesT('matchValuePlaceholder', {type: matchType || routesT('value')});

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
        <PageHero eyebrow={t('nav.routes')} title={pageT('routesTitle')} description={pageT('routesDesc')} />

        <section className="forms-grid">
          <article className="panel-card">
            <div className="inline-cluster" style={{gap: 8}}>
              <h3>{routesT('createRule')}</h3>
              {validationPending && <span className="badge is-neutral">{t('common.validating')}</span>}
              {!validationPending && validationResult && (
                <span className={`badge ${validationResult.valid ? 'is-good' : 'is-danger'}`}>
                  {validationResult.valid ? t('common.valid') : t('common.invalid')}
                </span>
              )}
            </div>
            <form
              className="sub-grid"
              onSubmit={(e) => { form.handleSubmit((values) => {
                const chainDestinationScope = values.actionType === 'chain'
                  ? chains.find((chain) => chain.id === values.chainId)?.destinationScope || ''
                  : values.destinationScope.trim();
                createRuleMutation.mutate({
                  priority: Number(values.priority),
                  matchType: values.matchType.trim(),
                  matchValue: values.matchValue.trim(),
                  actionType: values.actionType,
                  chainId: values.chainId.trim(),
                  destinationScope: chainDestinationScope
                });
              })(e); }}
            >
              <div className="field-stack">
                <span>{routesT('priority')}</span>
                <input
                  aria-invalid={form.formState.errors.priority ? 'true' : 'false'}
                  className="field-input"
                  type="number"
                  {...form.register('priority', {
                    required: routesT('priorityRequired'),
                    validate: (value) => Number(value) > 0 || routesT('priorityPositive')
                  })}
                />
                {form.formState.errors.priority ? <p className="error-text">{form.formState.errors.priority.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>{routesT('matchType')}</span>
                <select
                  aria-invalid={form.formState.errors.matchType ? 'true' : 'false'}
                  className="field-select"
                  {...form.register('matchType', {required: routesT('matchTypeRequired')})}
                >
                  {matchTypeOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
                {form.formState.errors.matchType ? <p className="error-text">{form.formState.errors.matchType.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>{routesT('matchValue')}</span>
                <div className="inline-cluster">
                  <input
                    aria-invalid={form.formState.errors.matchValue ? 'true' : 'false'}
                    className="field-input"
                    placeholder={matchValuePlaceholder}
                    style={{flex: 1}}
                    {...form.register('matchValue', {
                      required: routesT('matchValueRequired'),
                      validate: (value) => validateMatchValue(matchType, value, routesT)
                    })}
                  />
                  {matchType === 'url_regex' && (
                    <button className="secondary-button" onClick={() => setRegexTesterOpen(true)} type="button">
                      {routesT('testRegex')}
                    </button>
                  )}
                </div>
                {form.formState.errors.matchValue ? <p className="error-text">{form.formState.errors.matchValue.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>{routesT('actionType')}</span>
                <select className="field-select" {...form.register('actionType', {required: true})}>
                  {actionTypeOptions.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
              </div>
              <div className="field-stack">
                <span>{routesT('chain')}</span>
                <select
                  aria-invalid={form.formState.errors.chainId ? 'true' : 'false'}
                  className="field-select"
                  disabled={actionType !== 'chain'}
                  {...form.register('chainId', {
                    validate: (value) => (actionType !== 'chain' || value.trim() !== '' ? true : routesT('chainRequired'))
                  })}
                >
                  <option value="">{t('common.selectChain')}</option>
                  {chains.map((chain) => {
                    const hopCount = Array.isArray(chain.hops) ? chain.hops.length : 0;
                    const hopDisplay = hopCount > 0 ? ` (${Array.from({length: hopCount}, (_, i) => i + 1).join(' → ')})` : '';
                    return (
                      <option key={chain.id} value={chain.id}>
                        {chain.name}
                        {hopDisplay}
                      </option>
                    );
                  })}
                </select>
                {form.formState.errors.chainId ? <p className="error-text">{form.formState.errors.chainId.message}</p> : null}
                {selectedChain && actionType === 'chain' ? (
                  <div className="field-hint">
                    <span className="muted-text">
                      {routesT('destinationScope')}: <strong>{selectedChain.destinationScope}</strong>
                    </span>
                  </div>
                ) : null}
              </div>
              <div className="field-stack">
                <span>{routesT('destinationScope')}</span>
                <input
                  aria-invalid={form.formState.errors.destinationScope ? 'true' : 'false'}
                  className="field-input"
                  disabled={actionType !== 'direct'}
                  list="scope-options"
                  placeholder={routesT('destinationScopePlaceholder')}
                  {...form.register('destinationScope', {
                    validate: (value) => (actionType !== 'direct' || value.trim() !== '' ? true : routesT('destinationScopeRequired'))
                  })}
                />
                <datalist id="scope-options">
                  {availableScopes.map((scope) => (
                    <option key={scope} value={scope} />
                  ))}
                </datalist>
                {form.formState.errors.destinationScope ? <p className="error-text">{form.formState.errors.destinationScope.message}</p> : null}
              </div>

              {validationResult && (
                <div className="probe-results-section">
                  <div className="section-header">
                    <h4>{routesT('validation')}</h4>
                    {validationPending ? <span className="badge is-neutral">{t('common.validating')}</span> : (
                      <span className={`badge ${validationResult.valid ? 'is-good' : 'is-danger'}`}>
                        {validationResult.valid ? t('common.valid') : t('common.invalid')}
                      </span>
                    )}
                  </div>
                  {validationResult.errors.map((msg, i) => (
                    <div className="token-box" key={`err-${i}`} style={{borderColor: 'var(--danger)'}}>
                      <span className="field-hint" style={{color: 'var(--danger)'}}>{msg}</span>
                    </div>
                  ))}
                  {validationResult.warnings.map((msg, i) => (
                    <div className="token-box" key={`warn-${i}`} style={{borderColor: 'var(--accent)'}}>
                      <span className="field-hint" style={{color: 'var(--accent)'}}>{msg}</span>
                    </div>
                  ))}
                  {validationResult.matchValueValidation && !validationResult.matchValueValidation.valid && (
                    <div className="token-box" style={{borderColor: 'var(--danger)'}}>
                      <span className="field-hint" style={{color: 'var(--danger)'}}>{validationResult.matchValueValidation.message}</span>
                    </div>
                  )}
                  {validationResult.chainValidation && !validationResult.chainValidation.valid && (
                    <div className="token-box" style={{borderColor: 'var(--danger)'}}>
                      <span className="field-hint" style={{color: 'var(--danger)'}}>{routesT('chainNotFound')}</span>
                    </div>
                  )}
                  {validationResult.scopeValidation && !validationResult.scopeValidation.valid && (
                    <div className="token-box" style={{borderColor: 'var(--danger)'}}>
                      <span className="field-hint" style={{color: 'var(--danger)'}}>{routesT('scopeNotFound')}</span>
                    </div>
                  )}
                  {validationResult.scopeValidation && validationResult.scopeValidation.valid && !validationResult.scopeValidation.matchesChainFinalHop && (
                    <div className="token-box" style={{borderColor: 'var(--accent)'}}>
                      <span className="field-hint" style={{color: 'var(--accent)'}}>{routesT('scopeChainMismatch')}</span>
                    </div>
                  )}
                </div>
              )}

              <div className="submit-row">
                <button className="primary-button" disabled={createRuleMutation.isPending} type="submit">
                  {createRuleMutation.isPending ? t('common.submitting') : routesT('createRule')}
                </button>
              </div>
            </form>
          </article>

          <article className="panel-card soft-card">
            <div className="panel-toolbar">
              <h3>{routesT('policies')}</h3>
              <button className="primary-button" disabled={publishMutation.isPending} onClick={() => publishMutation.mutate()} type="button">
                {publishMutation.isPending ? t('common.submitting') : routesT('publishPolicy')}
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
        </section>

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
                        <td>{chainName || '-'}</td>
                        <td>{rule.destinationScope || '-'}</td>
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

        {regexTesterOpen && (
          <RegexTesterModal initialPattern={form.getValues('matchValue')} onClose={() => setRegexTesterOpen(false)} />
        )}
      </div>
    </AuthGate>
  );
}
