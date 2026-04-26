'use client';

import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import {Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {getCertificates, getNodeHealth} from '@/lib/control-plane-api';
import {formatControlPlaneError} from '@/lib/presentation';

export default function HealthPage() {
  const t = useTranslations('pages');
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';
  const healthQuery = useQuery({
    queryKey: ['node-health', accessToken],
    queryFn: () => getNodeHealth(accessToken),
    enabled: !!accessToken,
    refetchInterval: 5000
  });
  const certificatesQuery = useQuery({
    queryKey: ['certificates', accessToken],
    queryFn: () => getCertificates(accessToken),
    enabled: !!accessToken
  });

  const chartData = (healthQuery.data || []).map((item) => ({
    node: item.nodeId,
    listeners: Object.keys(item.listenerStatus || {}).length,
    certs: Object.keys(item.certStatus || {}).length
  }));

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Health" title={t('healthTitle')} description={t('healthDesc')} />
        <section className="forms-grid">
          <article className="panel-card">
            <h3>Node health</h3>
            {healthQuery.isPending ? (
              <AsyncState detail="Heartbeat and listener state are loading." title="Loading node health" />
            ) : healthQuery.isError ? (
              <AsyncState actionLabel="Retry" detail={formatControlPlaneError(healthQuery.error)} onAction={() => void healthQuery.refetch()} title="Failed to load node health" />
            ) : chartData.length === 0 ? (
              <AsyncState detail="Heartbeat rows will appear after the first node-agent reports status." title="No health data yet" />
            ) : (
              <div className="chart-card">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="node" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="listeners" fill="#88b04b" />
                    <Bar dataKey="certs" fill="#c97b5a" />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            )}
          </article>

          <article className="panel-card soft-card">
            <h3>Certificate pressure</h3>
            {certificatesQuery.isPending ? (
              <AsyncState detail="Certificate status is loading." title="Loading certificates" />
            ) : certificatesQuery.isError ? (
              <AsyncState actionLabel="Retry" detail={formatControlPlaneError(certificatesQuery.error)} onAction={() => void certificatesQuery.refetch()} title="Failed to load certificates" />
            ) : (certificatesQuery.data || []).length === 0 ? (
              <AsyncState detail="Public and internal certificates will appear here once registered." title="No certificates yet" />
            ) : (
              <div className="stack-list">
                {(certificatesQuery.data || []).map((cert) => (
                  <div className="stack-item" key={cert.id}>
                    <div className="stack-head">
                      <strong>{cert.ownerId}</strong>
                      <span className={`badge ${cert.status === 'healthy' || cert.status === 'renewed' ? 'is-good' : 'is-warn'}`}>{cert.status}</span>
                    </div>
                    <span className="muted-text">
                      {cert.certType} · {cert.provider}
                    </span>
                    <span className="mono">{cert.notAfter || '-'}</span>
                  </div>
                ))}
              </div>
            )}
          </article>
        </section>

        <article className="panel-card">
          <h3>Heartbeats</h3>
          {healthQuery.isPending ? (
            <AsyncState detail="Heartbeat detail is loading." title="Loading heartbeat table" />
          ) : healthQuery.isError ? (
            <AsyncState actionLabel="Retry" detail={formatControlPlaneError(healthQuery.error)} onAction={() => void healthQuery.refetch()} title="Failed to load heartbeat table" />
          ) : (healthQuery.data || []).length === 0 ? (
            <AsyncState detail="No node has reported heartbeat information yet." title="No heartbeats yet" />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Node</th>
                    <th>Heartbeat</th>
                    <th>Policy</th>
                    <th>Listeners</th>
                    <th>Certs</th>
                  </tr>
                </thead>
                <tbody>
                  {(healthQuery.data || []).map((item) => (
                    <tr key={item.nodeId}>
                      <td>{item.nodeId}</td>
                      <td className="mono">{item.heartbeatAt}</td>
                      <td>{item.policyRevisionId || '-'}</td>
                      <td>{Object.entries(item.listenerStatus || {}).map(([key, value]) => `${key}:${value}`).join(', ') || '-'}</td>
                      <td>{Object.entries(item.certStatus || {}).map(([key, value]) => `${key}:${value}`).join(', ') || '-'}</td>
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
