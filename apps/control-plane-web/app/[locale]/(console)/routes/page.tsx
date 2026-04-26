'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {createRouteRule, getChains, getPolicyRevisions, getRouteRules, publishPolicy} from '@/lib/control-plane-api';
import {formatControlPlaneError} from '@/lib/presentation';

type RouteRuleFormValues = {
  priority: string;
  matchType: string;
  matchValue: string;
  actionType: string;
  chainId: string;
  destinationScope: string;
};

export default function RoutesPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const form = useForm<RouteRuleFormValues>({
    defaultValues: {
      priority: '100',
      matchType: 'domain',
      matchValue: '',
      actionType: 'chain',
      chainId: '',
      destinationScope: ''
    }
  });
  const actionType = form.watch('actionType');

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
      toast.success('route rule created');
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
      toast.success('policy published');
      queryClient.invalidateQueries({queryKey: ['policies-revisions']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const routeRules = routeRulesQuery.data || [];
  const policies = policiesQuery.data || [];
  const chains = chainsQuery.data || [];

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Routes" title={pageT('routesTitle')} description={pageT('routesDesc')} />

        <section className="forms-grid">
          <article className="panel-card">
            <h3>Create route rule</h3>
            <form
              className="sub-grid"
              onSubmit={form.handleSubmit((values) => {
                createRuleMutation.mutate({
                  priority: Number(values.priority),
                  matchType: values.matchType.trim(),
                  matchValue: values.matchValue.trim(),
                  actionType: values.actionType,
                  chainId: values.chainId.trim(),
                  destinationScope: values.destinationScope.trim()
                });
              })}
            >
              <div className="field-stack">
                <span>Priority</span>
                <input
                  aria-invalid={form.formState.errors.priority ? 'true' : 'false'}
                  className="field-input"
                  type="number"
                  {...form.register('priority', {
                    required: 'priority is required',
                    validate: (value) => Number(value) > 0 || 'priority must be greater than 0'
                  })}
                />
                {form.formState.errors.priority ? <p className="error-text">{form.formState.errors.priority.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Match type</span>
                <input
                  aria-invalid={form.formState.errors.matchType ? 'true' : 'false'}
                  className="field-input"
                  placeholder="domain"
                  {...form.register('matchType', {required: 'match type is required'})}
                />
                {form.formState.errors.matchType ? <p className="error-text">{form.formState.errors.matchType.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Match value</span>
                <input
                  aria-invalid={form.formState.errors.matchValue ? 'true' : 'false'}
                  className="field-input"
                  placeholder="internal.example.com"
                  {...form.register('matchValue', {required: 'match value is required'})}
                />
                {form.formState.errors.matchValue ? <p className="error-text">{form.formState.errors.matchValue.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Action type</span>
                <select className="field-select" {...form.register('actionType', {required: true})}>
                  <option value="chain">chain</option>
                  <option value="direct">direct</option>
                </select>
              </div>
              <div className="field-stack">
                <span>Chain</span>
                <select
                  aria-invalid={form.formState.errors.chainId ? 'true' : 'false'}
                  className="field-select"
                  {...form.register('chainId', {
                    validate: (value) => (actionType !== 'chain' || value.trim() !== '' ? true : 'chain id is required for chain action')
                  })}
                >
                  <option value="">Select chain</option>
                  {chains.map((chain) => (
                    <option key={chain.id} value={chain.id}>
                      {chain.name}
                    </option>
                  ))}
                </select>
                {form.formState.errors.chainId ? <p className="error-text">{form.formState.errors.chainId.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Destination scope</span>
                <input
                  aria-invalid={form.formState.errors.destinationScope ? 'true' : 'false'}
                  className="field-input"
                  placeholder="cn-hz-b"
                  {...form.register('destinationScope', {
                    validate: (value) => (actionType !== 'direct' || value.trim() !== '' ? true : 'destination scope is required for direct action')
                  })}
                />
                {form.formState.errors.destinationScope ? <p className="error-text">{form.formState.errors.destinationScope.message}</p> : null}
              </div>
              <div className="submit-row">
                <button className="primary-button" disabled={createRuleMutation.isPending} type="submit">
                  {createRuleMutation.isPending ? t('common.submitting') : 'Create rule'}
                </button>
              </div>
            </form>
          </article>

          <article className="panel-card soft-card">
            <div className="panel-toolbar">
              <h3>Policies</h3>
              <button className="primary-button" disabled={publishMutation.isPending} onClick={() => publishMutation.mutate()} type="button">
                {publishMutation.isPending ? t('common.submitting') : 'Publish policy'}
              </button>
            </div>
            {policiesQuery.isPending ? (
              <AsyncState detail={t('common.loading')} title="Loading policy revisions" />
            ) : policiesQuery.isError ? (
              <AsyncState
                actionLabel={t('common.retry')}
                detail={formatControlPlaneError(policiesQuery.error)}
                onAction={() => void policiesQuery.refetch()}
                title="Failed to load policy revisions"
              />
            ) : policies.length === 0 ? (
              <AsyncState detail="The first publish will create a visible revision here." title={t('common.empty')} />
            ) : (
              <div className="stack-list">
                {policies.map((policy) => (
                  <div className="stack-item" key={policy.id}>
                    <strong>{policy.version}</strong>
                    <span className="muted-text">
                      {policy.status} · {policy.assignedNodes} nodes
                    </span>
                    <span className="mono">{policy.createdAt}</span>
                  </div>
                ))}
              </div>
            )}
          </article>
        </section>

        <article className="panel-card">
          <div className="panel-toolbar">
            <h3>Route rules</h3>
            <span className="badge">{routeRules.length}</span>
          </div>
          {routeRulesQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title="Loading route rules" />
          ) : routeRulesQuery.isError ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(routeRulesQuery.error)}
              onAction={() => void routeRulesQuery.refetch()}
              title="Failed to load route rules"
            />
          ) : routeRules.length === 0 ? (
            <AsyncState detail="Whitelist rules will appear here after the first rule is created." title={t('common.empty')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Priority</th>
                    <th>Match</th>
                    <th>Action</th>
                    <th>Chain</th>
                    <th>Scope</th>
                  </tr>
                </thead>
                <tbody>
                  {routeRules.map((rule) => (
                    <tr key={rule.id}>
                      <td>{rule.priority}</td>
                      <td>
                        <strong>{rule.matchType}</strong>
                        <div className="muted-text mono">{rule.matchValue}</div>
                      </td>
                      <td>{rule.actionType}</td>
                      <td>{rule.chainId || '-'}</td>
                      <td>{rule.destinationScope || '-'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </article>
      </div>
    </AuthGate>
  );
}
