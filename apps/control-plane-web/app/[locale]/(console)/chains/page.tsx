'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {createChain, getChains} from '@/lib/control-plane-api';
import {formatControlPlaneError, splitList} from '@/lib/presentation';

type ChainFormValues = {
  name: string;
  destinationScope: string;
  hops: string;
};

export default function ChainsPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const form = useForm<ChainFormValues>({
    defaultValues: {
      name: '',
      destinationScope: '',
      hops: ''
    }
  });
  const chainsQuery = useQuery({
    queryKey: ['chains', accessToken],
    queryFn: () => getChains(accessToken),
    enabled: !!accessToken
  });
  const createChainMutation = useMutation({
    mutationFn: (payload: {name: string; destinationScope: string; hops: string[]}) => createChain(accessToken, payload),
    onSuccess: () => {
      toast.success('chain created');
      queryClient.invalidateQueries({queryKey: ['chains']});
      form.reset();
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const chains = chainsQuery.data || [];

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Chains" title={pageT('chainsTitle')} description={pageT('chainsDesc')} />
        <section className="forms-grid">
          <article className="panel-card">
            <h3>Create chain</h3>
            <form
              className="sub-grid"
              onSubmit={form.handleSubmit((values) => {
                createChainMutation.mutate({
                  name: values.name.trim(),
                  destinationScope: values.destinationScope.trim(),
                  hops: splitList(values.hops)
                });
              })}
            >
              <div className="field-stack">
                <span>Name</span>
                <input
                  aria-invalid={form.formState.errors.name ? 'true' : 'false'}
                  className="field-input"
                  placeholder="panel-to-node2"
                  {...form.register('name', {required: 'name is required'})}
                />
                {form.formState.errors.name ? <p className="error-text">{form.formState.errors.name.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Destination scope</span>
                <input
                  aria-invalid={form.formState.errors.destinationScope ? 'true' : 'false'}
                  className="field-input"
                  placeholder="cn-hz-b"
                  {...form.register('destinationScope', {required: 'destination scope is required'})}
                />
                {form.formState.errors.destinationScope ? <p className="error-text">{form.formState.errors.destinationScope.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>Hops</span>
                <input
                  aria-invalid={form.formState.errors.hops ? 'true' : 'false'}
                  className="field-input"
                  placeholder="node-a, node-b"
                  {...form.register('hops', {
                    validate: (value) => splitList(value).length > 0 || 'at least one hop is required'
                  })}
                />
                {form.formState.errors.hops ? <p className="error-text">{form.formState.errors.hops.message}</p> : null}
              </div>
              <div className="submit-row">
                <button className="primary-button" disabled={createChainMutation.isPending} type="submit">
                  {createChainMutation.isPending ? t('common.submitting') : 'Create chain'}
                </button>
              </div>
            </form>
          </article>

          <article className="panel-card soft-card">
            <h3>Chain list</h3>
            {chainsQuery.isPending ? (
              <AsyncState detail={t('common.loading')} title="Loading chains" />
            ) : chainsQuery.isError ? (
              <AsyncState
                actionLabel={t('common.retry')}
                detail={formatControlPlaneError(chainsQuery.error)}
                onAction={() => void chainsQuery.refetch()}
                title="Failed to load chains"
              />
            ) : chains.length === 0 ? (
              <AsyncState detail="Create relay chains before binding route rules to them." title={t('common.empty')} />
            ) : (
              <div className="stack-list">
                {chains.map((chain) => (
                  <div className="stack-item" key={chain.id}>
                    <div className="stack-head">
                      <strong>{chain.name}</strong>
                      <span className={`badge ${chain.enabled ? 'is-good' : 'is-warn'}`}>{chain.enabled ? 'enabled' : 'disabled'}</span>
                    </div>
                    <span className="muted-text">{chain.destinationScope}</span>
                    <span className="mono">{chain.hops.join(' -> ')}</span>
                  </div>
                ))}
              </div>
            )}
          </article>
        </section>
      </div>
    </AuthGate>
  );
}
