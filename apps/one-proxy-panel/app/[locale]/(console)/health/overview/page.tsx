'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import ReactECharts from 'echarts-for-react';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {ConsoleFilterBar, ConsoleFilterItem} from '@/components/console-template';
import {PageHero} from '@/components/page-hero';
import {fetchEnums, getNodeHealth, getNodeHealthHistory, getNodes} from '@/lib/api';
import {NodeHealthHistory} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {buildHealthRows, deriveHealthState} from '../_lib/health-state';

export default function HealthOverviewPage() {
  const pageT = useTranslations('pages');
  const common = useTranslations('common');
  const healthT = useTranslations('health');
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;

  const [selectedNodeId, setSelectedNodeId] = useState('');

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken, activeTenantId],
    queryFn: () => getNodes(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const healthQuery = useQuery({
    queryKey: ['node-health', accessToken, activeTenantId],
    queryFn: () => getNodeHealth(accessToken, activeTenantId),
    enabled: !!accessToken,
    refetchInterval: 5000
  });
  const enumsQuery = useQuery({
    queryKey: ['enums'],
    queryFn: () => fetchEnums()
  });
  const enums = enumsQuery.data;

  const nodes = nodesQuery.data || [];
  const health = healthQuery.data || [];

  const healthRows = useMemo(() => buildHealthRows(nodes, health, enums), [health, nodes, enums]);

  const summary = useMemo(() => {
    const healthy = healthRows.filter((item) => item.derivedStatus === 'healthy').length;
    const stale = healthRows.filter((item) => item.derivedStatus === 'stale').length;
    const degraded = healthRows.filter((item) => item.derivedStatus === 'degraded').length;
    const unreported = healthRows.filter((item) => item.derivedStatus === 'unreported').length;
    const certPressure = healthRows.filter((item) =>
      Object.values(item.certStatus || {}).some((status) => {
        const entry = enums?.cert_status?.[status];
        return entry ? entry.meta?.className !== 'is-good' : (status !== 'healthy' && status !== 'renewed');
      })
    ).length;
    return {healthy, stale, degraded, unreported, certPressure};
  }, [healthRows, enums]);

  const historyQuery = useQuery({
    queryKey: ['node-health-history', accessToken, activeTenantId, selectedNodeId],
    queryFn: () => getNodeHealthHistory(accessToken, activeTenantId, selectedNodeId, '24h'),
    enabled: !!accessToken && !!selectedNodeId
  });

  const isLoading = healthQuery.isPending || nodesQuery.isPending;
  const isError = healthQuery.isError || nodesQuery.isError;

  const pieOption = useMemo(() => {
    const healthColors: Record<string, string> = {
      healthy: enums?.node_status?.healthy?.meta?.color || '#22c55e',
      degraded: enums?.node_status?.degraded?.meta?.color || '#ef4444',
      stale: '#f59e0b',
      unreported: '#6b7280'
    };
    return {
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
          {name: 'Healthy', value: summary.healthy, itemStyle: {color: healthColors.healthy}},
          {name: 'Stale', value: summary.stale, itemStyle: {color: healthColors.stale}},
          {name: 'Degraded', value: summary.degraded, itemStyle: {color: healthColors.degraded}},
          {name: 'Unreported', value: summary.unreported, itemStyle: {color: healthColors.unreported}}
        ].filter((d) => d.value > 0)
      }]
    };
  }, [summary, enums]);

  const trendChartData = useMemo(() => {
    if (!selectedNodeId) return null;
    const history = historyQuery.data || [];
    return history.map((item: NodeHealthHistory) => {
      const derived = deriveHealthState({
        heartbeatAt: item.heartbeatAt,
        listenerStatus: item.listenerStatus,
        certStatus: item.certStatus
      }, enums);
      return {
        time: item.heartbeatAt,
        status: derived.status,
        label: derived.label
      };
    });
  }, [historyQuery.data, selectedNodeId, enums]);

  const trendOption = useMemo(() => {
    if (!trendChartData || trendChartData.length === 0) return null;

    const statusOrder = ['healthy', 'degraded', 'stale', 'unreported'];
    const statusColor: Record<string, string> = {};
    statusOrder.forEach((s) => {
      statusColor[s] = enums?.node_status?.[s]?.meta?.color || ({
        healthy: '#22c55e',
        degraded: '#ef4444',
        stale: '#f59e0b',
        unreported: '#6b7280'
      } as Record<string, string>)[s] || '#6b7280';
    });

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
  }, [trendChartData, enums]);

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow={healthT('eyebrow')} title={pageT('healthTitle')} />

        <section className="metrics-grid">
          <article className="metric-card panel-card">
            <span className="metric-label">{healthT('healthyHeartbeats')}</span>
            <strong>{summary.healthy}</strong>
          </article>
          <article className="metric-card panel-card soft-card">
            <span className="metric-label">{healthT('staleHeartbeats')}</span>
            <strong>{summary.stale}</strong>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">{healthT('unreportedNodes')}</span>
            <strong>{summary.unreported}</strong>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">{healthT('certificatePressure')}</span>
            <strong>{summary.certPressure}</strong>
          </article>
        </section>

        <section>
          <article className="panel-card">
            <h3 className="section-title">{healthT('nodeHealthDistribution')}</h3>
            {isLoading ? (
              <AsyncState detail={healthT('loadingDistributionDetail')} title={common('loadingTitle')} />
            ) : isError ? (
              <AsyncState title={healthT('failedDistribution')} detail={formatControlPlaneError(healthQuery.error || nodesQuery.error)} />
            ) : healthRows.length === 0 ? (
              <AsyncState detail={healthT('emptyDistributionDetail')} title={common('noData')} />
            ) : (
              <ReactECharts option={pieOption} style={{height: 300}} />
            )}
          </article>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">{healthT('trend')}</p>
              <h3>{healthT('healthTrend')}</h3>
            </div>
          </div>
          <ConsoleFilterBar title={common('filter')}>
            <ConsoleFilterItem label={common('name')} match={common('equals')}>
              <select
                className="field-select"
                value={selectedNodeId}
                onChange={(e) => setSelectedNodeId(e.target.value)}
              >
                <option value="">{healthT('allNodes')}</option>
                {nodes.map((node) => (
                  <option key={node.id} value={node.id}>{node.name}</option>
                ))}
              </select>
            </ConsoleFilterItem>
          </ConsoleFilterBar>
          {!selectedNodeId ? (
            <div className="trend-summary">
              <div className="trend-summary-grid">
                <div className="trend-summary-item">
                  <strong style={{color: enums?.node_status?.healthy?.meta?.color || '#22c55e'}}>{summary.healthy}</strong>
                  <span>{healthT('healthy')}</span>
                </div>
                <div className="trend-summary-item">
                  <strong style={{color: enums?.node_status?.stale?.meta?.color || '#f59e0b'}}>{summary.stale}</strong>
                  <span>{healthT('stale')}</span>
                </div>
                <div className="trend-summary-item">
                  <strong style={{color: enums?.node_status?.degraded?.meta?.color || '#ef4444'}}>{summary.degraded}</strong>
                  <span>{healthT('degraded')}</span>
                </div>
                <div className="trend-summary-item">
                  <strong style={{color: enums?.node_status?.unreported?.meta?.color || '#6b7280'}}>{summary.unreported}</strong>
                  <span>{healthT('unreported')}</span>
                </div>
              </div>
            </div>
          ) : historyQuery.isPending ? (
            <AsyncState detail={healthT('loadingHistoryDetail')} title={healthT('loadingHistory')} />
          ) : historyQuery.isError ? (
            <AsyncState
              actionLabel={common('retry')}
              detail={formatControlPlaneError(historyQuery.error)}
              onAction={() => void historyQuery.refetch()}
              title={healthT('failedHistory')}
            />
          ) : !trendChartData || trendChartData.length === 0 ? (
            <AsyncState detail={healthT('emptyHistoryDetail')} title={healthT('emptyHistory')} />
          ) : trendOption ? (
            <ReactECharts option={trendOption} style={{height: 300}} />
          ) : null}
        </section>
      </div>
    </AuthGate>
  );
}
