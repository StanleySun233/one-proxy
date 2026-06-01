'use client';

import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {NameTag} from '@/components/common/name-tag';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {TopologyPreview} from '@/components/topology-preview';
import {Link} from '@/i18n/navigation';
import {getNodes, getOverview, getPendingNodes} from '@/lib/api';
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
  const pendingQuery = useQuery({
    queryKey: ['pending-nodes', accessToken],
    queryFn: () => getPendingNodes(accessToken),
    enabled: !!accessToken,
    refetchInterval: 30000
  });

  const overview = overviewQuery.data;
  const nodes = nodesQuery.data || [];
  const pendingNodes = pendingQuery.data || [];

  return (
    <AuthGate>
      <div className="page-stack">
        {pendingNodes.length > 0 && (
          <div className="alert-banner">
            <div className="alert-content">
              <strong>{t('overview.pendingEnrollments')}</strong>
              <span>
                {t('overview.pendingApproval', {count: pendingNodes.length})}{' '}
                <Link href="/nodes/approvals">{t('overview.reviewNow')}</Link>
              </span>
            </div>
          </div>
        )}
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
                <span>{t('overview.pendingEnrollments')}</span>
                <strong>{pendingQuery.isPending ? '-' : pendingNodes.length}</strong>
              </article>
              <article className="metric-card">
                <span>{t('overview.policyRevision')}</span>
                <strong>{overviewQuery.isPending ? '-' : overview?.policies.activeRevision || '-'}</strong>
              </article>
            </div>
          }
          eyebrow={t('overview.eyebrow')}
          title={t('overview.title')}
        />

        <section className="two-column-grid">
          <article className="panel-card">
            <div className="panel-toolbar">
              <div>
                <p className="section-kicker">{t('overview.topology')}</p>
                <h3>{t('overview.pathDesigner')}</h3>
              </div>
              <span className="badge">{t('overview.nodesCount', {count: nodes.length})}</span>
            </div>
            {nodesQuery.isPending ? (
              <AsyncState detail={t('common.loading')} title={t('overview.loadingTopology')} />
            ) : nodesQuery.isError ? (
              <AsyncState
                actionLabel={t('common.retry')}
                detail={formatControlPlaneError(nodesQuery.error)}
                onAction={() => {
                  void nodesQuery.refetch();
                }}
                title={t('overview.failedTopology')}
              />
            ) : (
              <TopologyPreview nodes={nodes} />
            )}
          </article>

          <article className="panel-card soft-card">
            <div>
              <p className="section-kicker">{t('overview.pendingEnrollments')}</p>
              <h3>{t('shell.nodeApprovals')}</h3>
            </div>
            {pendingQuery.isPending ? (
              <AsyncState detail={t('common.loading')} title={t('overview.loadingQueue')} />
            ) : pendingQuery.isError ? (
              <AsyncState
                actionLabel={t('common.retry')}
                detail={formatControlPlaneError(pendingQuery.error)}
                onAction={() => void pendingQuery.refetch()}
                title={t('overview.failedQueue')}
              />
            ) : pendingNodes.length === 0 ? (
              <div className="queue-list">
                <div className="queue-item">
                  <strong>{t('common.empty')}</strong>
                </div>
              </div>
            ) : (
              <div className="queue-list">
                {pendingNodes.slice(0, 3).map((node) => (
                  <div className="queue-item" key={node.id}>
                    <NameTag kind="node">{node.name}</NameTag>
                    <span className="muted-text">
                      {node.mode} · {node.scopeKey} · {node.status}
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
            <p>{overview ? t('overview.healthSummary', {healthy: overview.nodes.healthy, degraded: overview.nodes.degraded}) : t('common.loading')}</p>
          </article>
          <article className="signal-card">
            <strong>{t('overview.pendingEnrollments')}</strong>
            <p>{pendingQuery.isPending ? t('common.loading') : t('overview.pendingApproval', {count: pendingNodes.length})}</p>
          </article>
          <article className="signal-card">
            <strong>{t('overview.pathDesigner')}</strong>
            <p>{t('overview.nodesCount', {count: nodes.length})}</p>
          </article>
        </section>
      </div>
    </AuthGate>
  );
}
