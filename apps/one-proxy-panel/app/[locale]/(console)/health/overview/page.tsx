'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import ReactECharts from 'echarts-for-react';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {getCertificates, getNodeHealth, getNodeHealthHistory, getNodes} from '@/lib/control-plane-api';
import {NodeHealthHistory} from '@/lib/control-plane-types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

const staleThresholdMs = 2 * 60 * 1000;

export default function HealthOverviewPage() {
  const t = useTranslations('pages');
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';

  const [selectedNodeId, setSelectedNodeId] = useState('');

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });
  const healthQuery = useQuery({
    queryKey: ['node-health', accessToken],
    queryFn: () => getNodeHealth(accessToken),
    enabled: !!accessToken,
    refetchInterval: 5000
  });
  const certificatesQuery = useQuery({
    queryKey: ['certificates', accessToken],
    queryFn: () => getCertificates(accessToken),
    enabled: !!accessToken,
    refetchInterval: 10000
  });

  const nodes = nodesQuery.data || [];
  const health = healthQuery.data || [];
  const certificates = certificatesQuery.data || [];

  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);

  const healthRows = useMemo(() => {
    const healthByNodeID = new Map(health.map((item) => [item.nodeId, item]));
    return nodes.map((node) => {
      const item = healthByNodeID.get(node.id);
      if (!item) {
        return {
          nodeId: node.id,
          heartbeatAt: '',
          policyRevisionId: '',
          listenerStatus: {},
          certStatus: {},
          name: node.name,
          mode: node.mode,
          scopeKey: node.scopeKey,
          derivedStatus: 'unreported' as const,
          derivedLabel: 'unreported',
          listenerSummary: '',
          certSummary: ''
        };
      }
      const derived = deriveHealthState(item);
      return {
        ...item,
        name: node.name,
        mode: node.mode,
        scopeKey: node.scopeKey,
        derivedStatus: derived.status,
        derivedLabel: derived.label,
        listenerSummary: joinMap(item.listenerStatus),
        certSummary: joinMap(item.certStatus)
      };
    });
  }, [health, nodes]);

  const certificateRows = useMemo(() => {
    return certificates.map((item) => ({
      ...item,
      ownerName: nodesByID.get(item.ownerId)?.name || item.ownerId
    }));
  }, [certificates, nodesByID]);

  const summary = useMemo(() => {
    const healthy = healthRows.filter((item) => item.derivedStatus === 'healthy').length;
    const stale = healthRows.filter((item) => item.derivedStatus === 'stale').length;
    const degraded = healthRows.filter((item) => item.derivedStatus === 'degraded').length;
    const unreported = healthRows.filter((item) => item.derivedStatus === 'unreported').length;
    const certPressure = certificateRows.filter((item) => item.status !== 'healthy' && item.status !== 'renewed').length;
    return {healthy, stale, degraded, unreported, certPressure};
  }, [certificateRows, healthRows]);

  const historyQuery = useQuery({
    queryKey: ['node-health-history', accessToken, selectedNodeId],
    queryFn: () => getNodeHealthHistory(accessToken, selectedNodeId, '24h'),
    enabled: !!accessToken && !!selectedNodeId
  });

  const isLoading = healthQuery.isPending || nodesQuery.isPending;
  const isError = healthQuery.isError || nodesQuery.isError;

  const pieOption = useMemo(() => ({
    tooltip: {trigger: 'item' as const, formatter: '{b}: {c} ({d}%)'},
    legend: {bottom: '0%', textStyle: {color: '#94a3b8'}},
    series: [{
      type: 'pie' as const,
      radius: ['40%', '70%'],
      center: ['50%', '45%'],
      avoidLabelOverlap: true,
      itemStyle: {borderRadius: 4, borderColor: 'transparent'},
      label: {color: '#e2e8f0'},
      labelLine: {lineStyle: {color: '#475569'}},
      data: [
        {name: 'Healthy', value: summary.healthy, itemStyle: {color: '#22c55e'}},
        {name: 'Stale', value: summary.stale, itemStyle: {color: '#f59e0b'}},
        {name: 'Degraded', value: summary.degraded, itemStyle: {color: '#ef4444'}},
        {name: 'Unreported', value: summary.unreported, itemStyle: {color: '#6b7280'}}
      ].filter((d) => d.value > 0)
    }]
  }), [summary]);

  const certStatusCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    certificateRows.forEach((cert) => {
      counts[cert.status] = (counts[cert.status] || 0) + 1;
    });
    return counts;
  }, [certificateRows]);

  const certBarColors: Record<string, string> = {
    healthy: '#22c55e',
    renewed: '#22c55e',
    'renew-soon': '#f59e0b',
    rotate: '#f97316',
    failed: '#ef4444',
    expired: '#dc2626'
  };

  const barOption = useMemo(() => {
    const statuses = Object.keys(certStatusCounts).sort();
    return {
      tooltip: {trigger: 'axis' as const},
      grid: {left: 40, right: 20, top: 10, bottom: 40},
      xAxis: {
        type: 'category' as const,
        data: statuses,
        axisLabel: {color: '#94a3b8'},
        axisLine: {lineStyle: {color: '#334155'}}
      },
      yAxis: {
        type: 'value' as const,
        minInterval: 1,
        axisLabel: {color: '#94a3b8'},
        splitLine: {lineStyle: {color: '#1e293b'}}
      },
      series: [{
        type: 'bar' as const,
        barWidth: '60%',
        itemStyle: {
          borderRadius: [4, 4, 0, 0],
          color: (params: {dataIndex: number}) => {
            const status = statuses[params.dataIndex];
            return certBarColors[status] || '#6b7280';
          }
        },
        data: statuses.map((s) => certStatusCounts[s])
      }]
    };
  }, [certStatusCounts]);

  const trendChartData = useMemo(() => {
    if (!selectedNodeId) return null;
    const history = historyQuery.data || [];
    return history.map((item: NodeHealthHistory) => {
      const derived = deriveHealthState({
        heartbeatAt: item.heartbeatAt,
        listenerStatus: item.listenerStatus,
        certStatus: item.certStatus
      });
      return {
        time: item.heartbeatAt,
        status: derived.status,
        label: derived.label
      };
    });
  }, [historyQuery.data, selectedNodeId]);

  const trendOption = useMemo(() => {
    if (!trendChartData || trendChartData.length === 0) return null;

    const statusOrder = ['healthy', 'degraded', 'stale', 'unreported'];
    const statusColor: Record<string, string> = {
      healthy: '#22c55e',
      degraded: '#ef4444',
      stale: '#f59e0b',
      unreported: '#6b7280'
    };

    const times = trendChartData.map((d) => formatISODateTime(d.time, d.time));
    const values = trendChartData.map((d) => statusOrder.indexOf(d.status));
    const colors = trendChartData.map((d) => statusColor[d.status] || '#6b7280');

    return {
      tooltip: {
        trigger: 'axis' as const,
        formatter: (params: {dataIndex: number; value: number}[]) => {
          const idx = params[0].dataIndex;
          return `${times[idx]}<br/>Status: ${trendChartData[idx].label}`;
        }
      },
      grid: {left: 50, right: 20, top: 10, bottom: 40},
      xAxis: {
        type: 'category' as const,
        data: times,
        axisLabel: {color: '#94a3b8', rotate: 30},
        axisLine: {lineStyle: {color: '#334155'}}
      },
      yAxis: {
        type: 'category' as const,
        data: statusOrder,
        axisLabel: {color: '#94a3b8'}
      },
      series: [{
        type: 'line' as const,
        data: values.map((v, i) => ({
          value: v,
          itemStyle: {color: colors[i]}
        })),
        step: 'end' as const,
        symbol: 'circle',
        symbolSize: 8,
        lineStyle: {width: 2, color: '#3b82f6'},
        areaStyle: {color: 'rgba(59, 130, 246, 0.1)'}
      }]
    };
  }, [trendChartData]);

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Health" title={t('healthTitle')} description={t('healthDesc')} />

        <section className="metrics-grid">
          <article className="metric-card panel-card">
            <span className="metric-label">Healthy heartbeats</span>
            <strong>{summary.healthy}</strong>
            <span className="metric-foot">Nodes reporting normal listener and cert state.</span>
          </article>
          <article className="metric-card panel-card soft-card">
            <span className="metric-label">Stale heartbeats</span>
            <strong>{summary.stale}</strong>
            <span className="metric-foot">Last heartbeat older than the configured freshness window.</span>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">Unreported nodes</span>
            <strong>{summary.unreported}</strong>
            <span className="metric-foot">Registered nodes that have never sent a heartbeat row.</span>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">Certificate pressure</span>
            <strong>{summary.certPressure}</strong>
            <span className="metric-foot">Certificates marked rotate, renew-soon, failed, or otherwise non-healthy.</span>
          </article>
        </section>

        <section className="charts-row">
          <article className="panel-card chart-col">
            <h3 className="section-title">Node Health Distribution</h3>
            {isLoading ? (
              <AsyncState detail="Loading health data for distribution chart." title="Loading" />
            ) : isError ? (
              <AsyncState title="Failed to load health data" detail={formatControlPlaneError(healthQuery.error || nodesQuery.error)} />
            ) : healthRows.length === 0 ? (
              <AsyncState detail="No health data available for distribution chart." title="No data" />
            ) : (
              <ReactECharts option={pieOption} style={{height: 300}} />
            )}
          </article>
          <article className="panel-card chart-col">
            <h3 className="section-title">Certificate Status</h3>
            {certificatesQuery.isPending ? (
              <AsyncState detail="Loading certificate data." title="Loading" />
            ) : certificatesQuery.isError ? (
              <AsyncState title="Failed to load certificates" detail={formatControlPlaneError(certificatesQuery.error)} />
            ) : certificateRows.length === 0 ? (
              <AsyncState detail="No certificate data available." title="No data" />
            ) : (
              <ReactECharts option={barOption} style={{height: 300}} />
            )}
          </article>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Trend</p>
              <h3>Health trend</h3>
              <p className="section-copy">
                {selectedNodeId
                  ? 'Heartbeat history for the selected node over the last 24 hours.'
                  : 'Select a node to view its heartbeat history.'}
              </p>
            </div>
            <label className="field-stack registry-filter registry-filter-short">
              <span>Node</span>
              <select
                className="field-select"
                value={selectedNodeId}
                onChange={(e) => setSelectedNodeId(e.target.value)}
              >
                <option value="">All nodes</option>
                {nodes.map((node) => (
                  <option key={node.id} value={node.id}>{node.name}</option>
                ))}
              </select>
            </label>
          </div>
          {!selectedNodeId ? (
            <div className="trend-summary">
              <div className="trend-summary-grid">
                <div className="trend-summary-item">
                  <strong style={{color: '#22c55e'}}>{summary.healthy}</strong>
                  <span>Healthy</span>
                </div>
                <div className="trend-summary-item">
                  <strong style={{color: '#f59e0b'}}>{summary.stale}</strong>
                  <span>Stale</span>
                </div>
                <div className="trend-summary-item">
                  <strong style={{color: '#ef4444'}}>{summary.degraded}</strong>
                  <span>Degraded</span>
                </div>
                <div className="trend-summary-item">
                  <strong style={{color: '#6b7280'}}>{summary.unreported}</strong>
                  <span>Unreported</span>
                </div>
              </div>
              <p className="muted-text" style={{textAlign: 'center', marginTop: 16}}>
                Select a specific node from the dropdown above to view its 24-hour heartbeat trend.
              </p>
            </div>
          ) : historyQuery.isPending ? (
            <AsyncState detail="Loading health history for selected node." title="Loading history" />
          ) : historyQuery.isError ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(historyQuery.error)}
              onAction={() => void historyQuery.refetch()}
              title="Failed to load health history"
            />
          ) : !trendChartData || trendChartData.length === 0 ? (
            <AsyncState detail="No heartbeat history available for this node in the last 24 hours." title="No history data" />
          ) : trendOption ? (
            <ReactECharts option={trendOption} style={{height: 300}} />
          ) : null}
        </section>
      </div>
    </AuthGate>
  );
}

function deriveHealthState(item: {heartbeatAt: string; listenerStatus: Record<string, string>; certStatus: Record<string, string>}) {
  const heartbeatTime = Date.parse(item.heartbeatAt);
  const isStale = Number.isFinite(heartbeatTime) ? Date.now() - heartbeatTime > staleThresholdMs : true;
  const listenerValues = Object.values(item.listenerStatus || {});
  const certValues = Object.values(item.certStatus || {});
  const hasDegradedSignal = [...listenerValues, ...certValues].some((value) => value !== 'up' && value !== 'healthy' && value !== 'renewed');
  if (isStale) {
    return {status: 'stale', label: 'stale'};
  }
  if (hasDegradedSignal) {
    return {status: 'degraded', label: 'degraded'};
  }
  return {status: 'healthy', label: 'healthy'};
}

function joinMap(value: Record<string, string>) {
  return Object.entries(value || {})
    .map(([key, item]) => `${key}:${item}`)
    .join(', ');
}
