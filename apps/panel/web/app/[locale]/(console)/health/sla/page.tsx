'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import ReactECharts from 'echarts-for-react';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {ConsoleFilterBar, ConsoleFilterItem} from '@/components/console-template';
import {NameTag} from '@/components/common/name-tag';
import {PageHero} from '@/components/page-hero';
import {getNodeSLA} from '@/lib/api';
import {NodeSLAMinute} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

export default function SLAHealthPage() {
  const pageT = useTranslations('pages');
  const common = useTranslations('common');
  const healthT = useTranslations('health');
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const [resultFilter, setResultFilter] = useState('all');
  const [scenarioFilter, setScenarioFilter] = useState('all');

  const slaQuery = useQuery({
    queryKey: ['node-sla', accessToken, activeTenantId],
    queryFn: () => getNodeSLA(accessToken, activeTenantId, '24h'),
    enabled: !!accessToken,
    refetchInterval: 30000
  });

  const rows = slaQuery.data || [];
  const scenarioOptions = useMemo(() => {
    const byID = new Map<string, string>();
    rows.forEach((row) => byID.set(row.scenarioId, row.scenarioName));
    return Array.from(byID.entries())
      .map(([value, label]) => ({value, label}))
      .sort((a, b) => a.label.localeCompare(b.label));
  }, [rows]);

  const filteredRows = useMemo(() => rows.filter((row) => {
    if (resultFilter === 'pass' && row.success !== 1) return false;
    if (resultFilter === 'fail' && row.success === 1) return false;
    if (scenarioFilter !== 'all' && row.scenarioId !== scenarioFilter) return false;
    return true;
  }), [resultFilter, rows, scenarioFilter]);

  const summary = useMemo(() => {
    const passed = rows.filter((row) => row.success === 1).length;
    const failed = rows.length - passed;
    const affected = new Set(rows.filter((row) => row.success !== 1).map((row) => row.scenarioId)).size;
    const passRate = rows.length === 0 ? 0 : Math.round((passed / rows.length) * 1000) / 10;
    return {passed, failed, affected, passRate};
  }, [rows]);

  const trendOption = useMemo(() => {
    const byMinute = new Map<string, {passed: number; failed: number}>();
    rows.forEach((row) => {
      const current = byMinute.get(row.windowStart) || {passed: 0, failed: 0};
      if (row.success === 1) {
        current.passed += 1;
      } else {
        current.failed += 1;
      }
      byMinute.set(row.windowStart, current);
    });
    const minutes = Array.from(byMinute.entries()).sort((a, b) => a[0].localeCompare(b[0]));
    return {
      tooltip: {trigger: 'axis' as const},
      legend: {bottom: 0, textStyle: {color: '#94a3b8'}},
      grid: {left: 44, right: 20, top: 12, bottom: 54},
      xAxis: {
        type: 'category' as const,
        data: minutes.map(([minute]) => formatISODateTime(minute)),
        axisLabel: {color: '#94a3b8', rotate: 35},
        axisLine: {lineStyle: {color: '#334155'}}
      },
      yAxis: {
        type: 'value' as const,
        minInterval: 1,
        axisLabel: {color: '#94a3b8'}
      },
      series: [
        {
          name: healthT('pass'),
          type: 'bar' as const,
          stack: 'sla',
          data: minutes.map(([, value]) => value.passed),
          itemStyle: {color: '#88b04b'}
        },
        {
          name: healthT('fail'),
          type: 'bar' as const,
          stack: 'sla',
          data: minutes.map(([, value]) => value.failed),
          itemStyle: {color: '#c97b5a'}
        }
      ]
    };
  }, [healthT, rows]);

  const loading = slaQuery.isPending;
  const error = slaQuery.isError;
  const empty = !loading && !error && rows.length === 0;
  const filteredEmpty = !loading && !error && !empty && filteredRows.length === 0;

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow={healthT('eyebrow')} title={pageT('healthTitle')} />

        <section className="metrics-grid">
          <article className="metric-card panel-card">
            <span className="metric-label">{healthT('slaPassRate')}</span>
            <strong>{summary.passRate}%</strong>
          </article>
          <article className="metric-card panel-card soft-card">
            <span className="metric-label">{healthT('slaPassedMinutes')}</span>
            <strong>{summary.passed}</strong>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">{healthT('slaFailedMinutes')}</span>
            <strong>{summary.failed}</strong>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">{healthT('slaAffectedScenarios')}</span>
            <strong>{summary.affected}</strong>
          </article>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">{healthT('slaBoard')}</p>
              <h3>{healthT('slaMinuteTrend')}</h3>
            </div>
            <span className="badge">{healthT('sla24h')}</span>
          </div>
          {loading ? (
            <AsyncState detail={healthT('loadingSLADetail')} title={healthT('loadingSLA')} />
          ) : error ? (
            <AsyncState actionLabel={common('retry')} detail={formatControlPlaneError(slaQuery.error)} onAction={() => void slaQuery.refetch()} title={healthT('failedSLA')} />
          ) : empty ? (
            <AsyncState detail={healthT('emptySLADetail')} title={healthT('emptySLA')} />
          ) : (
            <ReactECharts option={trendOption} style={{height: 320}} />
          )}
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">{healthT('scenario')}</p>
              <h3>{healthT('slaScenarioRecords')}</h3>
            </div>
            <div className="inline-cluster">
              <span className="badge">{filteredRows.length} {common('shown')}</span>
              <span className="badge">{rows.length} {common('total')}</span>
            </div>
          </div>
          <ConsoleFilterBar title={common('filter')}>
            <ConsoleFilterItem label={common('status')} match={common('equals')}>
              <select className="field-select" onChange={(event) => setResultFilter(event.target.value)} value={resultFilter}>
                <option value="all">{healthT('allResults')}</option>
                <option value="pass">{healthT('pass')}</option>
                <option value="fail">{healthT('fail')}</option>
              </select>
            </ConsoleFilterItem>
            <ConsoleFilterItem label={healthT('scenario')} match={common('equals')}>
              <select className="field-select" onChange={(event) => setScenarioFilter(event.target.value)} value={scenarioFilter}>
                <option value="all">{healthT('allScenarios')}</option>
                {scenarioOptions.map((option) => (
                  <option key={option.value} value={option.value}>{option.label}</option>
                ))}
              </select>
            </ConsoleFilterItem>
          </ConsoleFilterBar>
          {filteredEmpty ? (
            <AsyncState detail={healthT('noMatchingSLADetail')} title={healthT('noMatchingSLA')} />
          ) : empty ? (
            <AsyncState detail={healthT('emptySLADetail')} title={healthT('emptySLA')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{healthT('scenario')}</th>
                    <th>{common('status')}</th>
                    <th>{healthT('minute')}</th>
                    <th>{healthT('heartbeats')}</th>
                    <th>{healthT('node')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredRows.map((row) => (
                    <SLARow key={`${row.scenarioId}:${row.windowStart}`} row={row} />
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}

function SLARow({row}: {row: NodeSLAMinute}) {
  const common = useTranslations('common');
  const healthT = useTranslations('health');
  const passed = row.success === 1;
  return (
    <tr>
      <td>
        <div className="registry-name-cell">
          <NameTag kind="node">{row.scenarioName}</NameTag>
          <span className="muted-text">{row.scenarioId}</span>
        </div>
      </td>
      <td>
        <span className={`badge ${passed ? 'is-good' : 'is-danger'}`}>{passed ? healthT('pass') : healthT('fail')}</span>
      </td>
      <td className="mono">{formatISODateTime(row.windowStart)}</td>
      <td>{row.receivedHeartbeats}/{row.expectedHeartbeats}</td>
      <td>{row.nodeName || <span className="muted-text">{common('unknown')}</span>}</td>
    </tr>
  );
}
