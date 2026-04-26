'use client';

import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {TopologyPreview} from '@/components/topology-preview';
import {getNodeAccessPaths, getNodeOnboardingTasks, getNodes, getOverview} from '@/lib/control-plane-api';
import {formatControlPlaneError} from '@/lib/presentation';

export default function OverviewPage() {
  const t = useTranslations();
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';

  const overviewQuery = useQuery({
    queryKey: ['overview', accessToken],
    queryFn: () => getOverview(accessToken),
    enabled: !!accessToken
  });
  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });
  const tasksQuery = useQuery({
    queryKey: ['onboarding-tasks', accessToken],
    queryFn: () => getNodeOnboardingTasks(accessToken),
    enabled: !!accessToken
  });
  const pathsQuery = useQuery({
    queryKey: ['node-access-paths', accessToken],
    queryFn: () => getNodeAccessPaths(accessToken),
    enabled: !!accessToken
  });

  const overview = overviewQuery.data;
  const nodes = nodesQuery.data || [];
  const tasks = tasksQuery.data || [];
  const paths = pathsQuery.data || [];
  const pendingTasks = tasks.slice(0, 3);

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero
          aside={
            <div className="metrics-grid">
              <article className="metric-card">
                <span>{t('overview.healthyNodes')}</span>
                <strong>{overviewQuery.isPending ? '-' : overview?.nodes.healthy ?? '-'}</strong>
              </article>
              <article className="metric-card">
                <span>{t('overview.degradedNodes')}</span>
                <strong>{overviewQuery.isPending ? '-' : overview?.nodes.degraded ?? '-'}</strong>
              </article>
              <article className="metric-card">
                <span>{t('overview.activeTasks')}</span>
                <strong>{tasksQuery.isPending ? '-' : tasks.length}</strong>
              </article>
              <article className="metric-card">
                <span>{t('overview.policyRevision')}</span>
                <strong>{overviewQuery.isPending ? '-' : overview?.policies.activeRevision || '-'}</strong>
              </article>
            </div>
          }
          description={t('overview.subtitle')}
          eyebrow={t('overview.eyebrow')}
          title={t('overview.title')}
        />

        <section className="two-column-grid">
          <article className="panel-card">
            <div className="panel-toolbar">
              <div>
                <p className="section-kicker">{t('overview.topology')}</p>
                <h3>{t('overview.pathDesigner')}</h3>
                <p className="section-copy">{t('overview.topologyHint')}</p>
              </div>
              <span className="badge">{nodes.length} nodes</span>
            </div>
            {nodesQuery.isPending || pathsQuery.isPending ? (
              <AsyncState detail={t('common.loading')} title="Loading topology" />
            ) : nodesQuery.isError || pathsQuery.isError ? (
              <AsyncState
                actionLabel={t('common.retry')}
                detail={formatControlPlaneError(nodesQuery.error || pathsQuery.error)}
                onAction={() => {
                  void nodesQuery.refetch();
                  void pathsQuery.refetch();
                }}
                title="Failed to load topology"
              />
            ) : (
              <TopologyPreview nodes={nodes} paths={paths} />
            )}
          </article>

          <article className="panel-card soft-card">
            <div>
              <p className="section-kicker">{t('overview.tasks')}</p>
              <h3>{t('overview.queueTitle')}</h3>
              <p className="section-copy">{t('overview.tasksHint')}</p>
            </div>
            {tasksQuery.isPending ? (
              <AsyncState detail={t('common.loading')} title="Loading onboarding queue" />
            ) : tasksQuery.isError ? (
              <AsyncState
                actionLabel={t('common.retry')}
                detail={formatControlPlaneError(tasksQuery.error)}
                onAction={() => void tasksQuery.refetch()}
                title="Failed to load onboarding queue"
              />
            ) : pendingTasks.length === 0 ? (
              <div className="queue-list">
                <div className="queue-item">
                  <strong>{t('common.empty')}</strong>
                  <span className="section-copy">{t('overview.queueMuted')}</span>
                </div>
              </div>
            ) : (
              <div className="queue-list">
                {pendingTasks.map((task) => (
                  <div className="queue-item" key={task.id}>
                    <strong>{task.mode}</strong>
                    <span className="section-copy">
                      {task.targetNodeId || task.targetHost || 'target'} · {task.status} · {task.statusMessage}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </article>
        </section>

        <section className="signal-strip">
          <article className="signal-card">
            <strong>{t('overview.health')}</strong>
            <p>{overview ? `${overview.nodes.healthy} healthy / ${overview.nodes.degraded} degraded` : t('common.loading')}</p>
          </article>
          <article className="signal-card">
            <strong>{t('overview.tasks')}</strong>
            <p>{tasksQuery.isPending ? t('common.loading') : `${tasks.length} task(s) tracked by control plane`}</p>
          </article>
          <article className="signal-card">
            <strong>{t('overview.pathDesigner')}</strong>
            <p>{pathsQuery.isPending ? t('common.loading') : `${paths.length} access path(s) ready for onboarding flows`}</p>
          </article>
        </section>
      </div>
    </AuthGate>
  );
}
