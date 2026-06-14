'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import {AlertTriangle, Check, Info, X} from 'lucide-react';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {ConsoleFilterBar, ConsoleFilterItem} from '@/components/console-template';
import {NameTag} from '@/components/common/name-tag';
import {PageHero} from '@/components/page-hero';
import {getNodeSLA} from '@/lib/api';
import {NodeSLAMinute} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

type HourState = 'good' | 'warn' | 'danger' | 'empty';

type SLAHourBucket = {
  startedAt: number;
  total: number;
  passed: number;
  failed: number;
  expected: number;
  received: number;
  state: HourState;
};

type SLAScenarioRow = {
  scenarioId: string;
  scenarioName: string;
  nodeName: string;
  total: number;
  passed: number;
  failed: number;
  availability: number;
  state: HourState;
  buckets: SLAHourBucket[];
};

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
  const hourStarts = useMemo(() => buildHourStarts(), [rows]);
  const scenarioRows = useMemo(() => buildScenarioRows(rows, hourStarts), [hourStarts, rows]);
  const scenarioOptions = useMemo(() => scenarioRows.map((row) => ({
    value: row.scenarioId,
    label: row.scenarioName
  })), [scenarioRows]);

  const filteredScenarios = useMemo(() => scenarioRows.filter((row) => {
    if (resultFilter === 'pass' && row.failed > 0) return false;
    if (resultFilter === 'fail' && row.failed === 0) return false;
    if (scenarioFilter !== 'all' && row.scenarioId !== scenarioFilter) return false;
    return true;
  }), [resultFilter, scenarioFilter, scenarioRows]);

  const summary = useMemo(() => {
    const passed = rows.filter((row) => row.success === 1).length;
    const failed = rows.length - passed;
    const affected = new Set(rows.filter((row) => row.success !== 1).map((row) => row.scenarioId)).size;
    const passRate = rows.length === 0 ? 0 : Math.round((passed / rows.length) * 1000) / 10;
    return {passed, failed, affected, passRate};
  }, [rows]);

  const loading = slaQuery.isPending;
  const error = slaQuery.isError;
  const empty = !loading && !error && rows.length === 0;
  const filteredEmpty = !loading && !error && !empty && filteredScenarios.length === 0;

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
              <h3>{healthT('slaScenarioRecords')}</h3>
            </div>
            <div className="inline-cluster">
              <span className="badge">{healthT('sla24h')}</span>
              <span className="badge">{filteredScenarios.length} {common('shown')}</span>
              <span className="badge">{scenarioRows.length} {common('total')}</span>
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
            <div className="sla-status-list">
              {filteredScenarios.map((row) => (
                <SLAScenarioStatus key={row.scenarioId} row={row} />
              ))}
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}

function SLAScenarioStatus({row}: {row: SLAScenarioRow}) {
  const healthT = useTranslations('health');
  const StateIcon = row.state === 'danger' ? X : row.state === 'warn' ? AlertTriangle : Check;
  return (
    <article className="sla-status-row">
      <div className="sla-status-head">
        <div className="sla-service-meta">
          <span className={`sla-status-dot is-${row.state}`}>
            <StateIcon size={10} strokeWidth={3} />
          </span>
          <div className="sla-service-title">
            <div className="sla-service-name-line">
              <NameTag kind="node">{row.scenarioName}</NameTag>
              <span className="sla-info-icon" title={`${row.nodeName} · ${row.scenarioId}`}>
                <Info size={14} />
              </span>
            </div>
            <span className="muted-text">{row.nodeName}</span>
          </div>
        </div>
        <div className="sla-availability">
          {row.total === 0 ? healthT('noRecords') : `${row.availability}% ${healthT('availability')}`}
        </div>
      </div>
      <div className="sla-hour-grid">
        {row.buckets.map((bucket) => (
          <span
            aria-label={hourBucketLabel(bucket, healthT)}
            className={`sla-hour-cell is-${bucket.state}`}
            key={bucket.startedAt}
            title={hourBucketLabel(bucket, healthT)}
          />
        ))}
      </div>
      <div className="sla-hour-axis">
        {row.buckets.map((bucket, index) => (
          <span key={bucket.startedAt}>{index % 3 === 0 || index === row.buckets.length - 1 ? formatHour(bucket.startedAt) : ''}</span>
        ))}
      </div>
    </article>
  );
}

function buildHourStarts() {
  const currentHour = startOfHour(new Date()).getTime();
  return Array.from({length: 24}, (_, index) => currentHour - (23 - index) * 60 * 60 * 1000);
}

function buildScenarioRows(rows: NodeSLAMinute[], hourStarts: number[]): SLAScenarioRow[] {
  const grouped = new Map<string, NodeSLAMinute[]>();
  rows.forEach((row) => {
    const current = grouped.get(row.scenarioId) || [];
    current.push(row);
    grouped.set(row.scenarioId, current);
  });
  return Array.from(grouped.entries()).map(([scenarioId, items]) => {
    const first = items[0];
    const byHour = new Map<number, NodeSLAMinute[]>();
    items.forEach((item) => {
      const hour = startOfHour(new Date(item.windowStart)).getTime();
      const current = byHour.get(hour) || [];
      current.push(item);
      byHour.set(hour, current);
    });
    const buckets = hourStarts.map((startedAt) => buildHourBucket(startedAt, byHour.get(startedAt) || []));
    const passed = items.filter((item) => item.success === 1).length;
    const failed = items.length - passed;
    return {
      scenarioId,
      scenarioName: first.scenarioName,
      nodeName: first.nodeName,
      total: items.length,
      passed,
      failed,
      availability: availability(passed, items.length),
      state: stateFor(passed, items.length),
      buckets
    };
  }).sort((a, b) => severity(b.state) - severity(a.state) || a.scenarioName.localeCompare(b.scenarioName));
}

function buildHourBucket(startedAt: number, rows: NodeSLAMinute[]): SLAHourBucket {
  const passed = rows.filter((row) => row.success === 1).length;
  const expected = rows.reduce((sum, row) => sum + row.expectedHeartbeats, 0);
  const received = rows.reduce((sum, row) => sum + row.receivedHeartbeats, 0);
  return {
    startedAt,
    total: rows.length,
    passed,
    failed: rows.length - passed,
    expected,
    received,
    state: stateFor(passed, rows.length)
  };
}

function stateFor(passed: number, total: number): HourState {
  if (total === 0) return 'empty';
  if (passed === total) return 'good';
  if (passed / total >= 0.95) return 'warn';
  return 'danger';
}

function severity(state: HourState) {
  return ({danger: 3, warn: 2, empty: 1, good: 0} as Record<HourState, number>)[state];
}

function availability(passed: number, total: number) {
  return total === 0 ? 0 : Math.round((passed / total) * 10000) / 100;
}

function startOfHour(date: Date) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), date.getHours(), 0, 0, 0);
}

function formatHour(value: number) {
  return new Intl.DateTimeFormat(undefined, {hour: '2-digit', hour12: false}).format(new Date(value));
}

function hourBucketLabel(bucket: SLAHourBucket, healthT: ReturnType<typeof useTranslations>) {
  if (bucket.total === 0) {
    return `${formatISODateTime(new Date(bucket.startedAt).toISOString())} · ${healthT('noRecords')}`;
  }
  return `${formatISODateTime(new Date(bucket.startedAt).toISOString())} · ${availability(bucket.passed, bucket.total)}% · ${bucket.passed}/${bucket.total} ${healthT('pass')} · ${bucket.received}/${bucket.expected} ${healthT('heartbeats')}`;
}
