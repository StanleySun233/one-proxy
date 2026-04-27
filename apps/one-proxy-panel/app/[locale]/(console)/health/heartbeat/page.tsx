'use client';

import {useCallback, useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import {ChevronDown, ChevronRight} from 'lucide-react';
import ReactECharts from 'echarts-for-react';

import {AsyncState} from '@/components/async-state';
import {AuthGate, useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {getNodeHealth, getNodeHealthHistory, getNodes} from '@/lib/control-plane-api';
import {NodeHealth} from '@/lib/control-plane-types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

const staleThresholdMs = 2 * 60 * 1000;

export default function HeartbeatPage() {
  const t = useTranslations('pages');
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';

  const [query, setQuery] = useState('');
  const [healthFilter, setHealthFilter] = useState('all');
  const [parentFilter, setParentFilter] = useState('all');
  const [expandedNodeId, setExpandedNodeId] = useState<string | null>(null);

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

  const nodes = nodesQuery.data || [];
  const health = healthQuery.data || [];

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
          parentNodeId: node.parentNodeId,
          derivedStatus: 'unreported',
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
        parentNodeId: node.parentNodeId,
        derivedStatus: derived.status,
        derivedLabel: derived.label,
        listenerSummary: joinMap(item.listenerStatus),
        certSummary: joinMap(item.certStatus)
      };
    });
  }, [health, nodes]);

  const parentNodeOptions = useMemo(() => {
    const parentIds = new Set<string>();
    const childCountMap = new Map<string, number>();
    nodes.forEach((node) => {
      if (node.parentNodeId) {
        parentIds.add(node.parentNodeId);
        childCountMap.set(node.parentNodeId, (childCountMap.get(node.parentNodeId) || 0) + 1);
      }
    });
    return Array.from(parentIds).map((id) => {
      const parent = nodesByID.get(id);
      return {
        value: id,
        label: `${parent?.name || id} (${childCountMap.get(id) || 0})`
      };
    }).sort((a, b) => a.label.localeCompare(b.label));
  }, [nodes, nodesByID]);

  const filteredHealth = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return healthRows.filter((item) => {
      if (healthFilter !== 'all' && item.derivedStatus !== healthFilter) {
        return false;
      }
      if (parentFilter !== 'all' && item.parentNodeId !== parentFilter) {
        return false;
      }
      if (!normalized) {
        return true;
      }
      return [item.nodeId, item.name, item.mode, item.scopeKey, item.policyRevisionId, item.listenerSummary, item.certSummary]
        .join(' ')
        .toLowerCase()
        .includes(normalized);
    });
  }, [healthFilter, healthRows, parentFilter, query]);

  const handleToggleExpand = useCallback((nodeId: string) => {
    setExpandedNodeId((prev) => (prev === nodeId ? null : nodeId));
  }, []);

  const loading = healthQuery.isPending || nodesQuery.isPending;
  const error = healthQuery.isError || nodesQuery.isError;
  const empty = !loading && !error && healthRows.length === 0;
  const filteredEmpty = !loading && !error && !empty && filteredHealth.length === 0;

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Health" title={t('healthTitle')} description={t('healthDesc')} />

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Heartbeat Registry</p>
              <h3>Node health</h3>
              <p className="section-copy">Filter by derived state, inspect listener and certificate summaries, and watch policy attachment drift per node.</p>
            </div>
            <div className="inline-cluster">
              <span className="badge">{filteredHealth.length} shown</span>
              <span className="badge">{healthRows.length} total</span>
            </div>
          </div>
          {loading ? (
            <AsyncState detail="Heartbeat and node registry data are loading." title="Loading node health" />
          ) : error ? (
            <AsyncState actionLabel="Retry" detail={formatControlPlaneError(healthQuery.error || nodesQuery.error)} onAction={() => { void healthQuery.refetch(); void nodesQuery.refetch(); }} title="Failed to load node health" />
          ) : empty ? (
            <AsyncState detail="Heartbeat rows will appear after the first node-agent reports status." title="No health data yet" />
          ) : (
            <div className="registry-stack">
              <div className="registry-toolbar">
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Parent node</span>
                  <select className="field-select" onChange={(event) => setParentFilter(event.target.value)} value={parentFilter}>
                    <option value="all">All nodes</option>
                    {parentNodeOptions.map((opt) => (
                      <option key={opt.value} value={opt.value}>
                        {opt.label}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field-stack registry-filter">
                  <span>Search</span>
                  <input
                    className="field-input"
                    onChange={(event) => setQuery(event.target.value)}
                    placeholder="Search by name, id, mode, scope, policy, or listener"
                    type="search"
                    value={query}
                  />
                </label>
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Derived state</span>
                  <select className="field-select" onChange={(event) => setHealthFilter(event.target.value)} value={healthFilter}>
                    <option value="all">All states</option>
                    <option value="healthy">healthy</option>
                    <option value="degraded">degraded</option>
                    <option value="stale">stale</option>
                    <option value="unreported">unreported</option>
                  </select>
                </label>
              </div>
              {filteredEmpty ? (
                <AsyncState detail="Adjust the current query or filters to see matching heartbeat rows." title="No matching health rows" />
              ) : (
                <div className="table-card">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th className="registry-expand-col" />
                        <th>Node</th>
                        <th>State</th>
                        <th>Mode</th>
                        <th>Heartbeat</th>
                        <th>Policy</th>
                        <th>Listeners</th>
                        <th>Certificates</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredHealth.map((item) => (
                        <ExpandableRow
                          key={item.nodeId}
                          item={item}
                          expanded={expandedNodeId === item.nodeId}
                          accessToken={accessToken}
                          onToggle={() => handleToggleExpand(item.nodeId)}
                        />
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}

type HealthRow = {
  nodeId: string;
  heartbeatAt: string;
  policyRevisionId: string;
  listenerStatus: Record<string, string>;
  certStatus: Record<string, string>;
  name: string;
  mode: string;
  scopeKey: string;
  parentNodeId: string;
  derivedStatus: string;
  derivedLabel: string;
  listenerSummary: string;
  certSummary: string;
};

function ExpandableRow({
  item,
  expanded,
  accessToken,
  onToggle
}: {
  item: HealthRow;
  expanded: boolean;
  accessToken: string;
  onToggle: () => void;
}) {
  const historyQuery = useQuery({
    queryKey: ['node-health-history', accessToken, item.nodeId],
    queryFn: () => getNodeHealthHistory(accessToken, item.nodeId, '24h'),
    enabled: expanded && !!accessToken
  });

  const historyData = useMemo(() => {
    if (!historyQuery.data) return [];
    return historyQuery.data
      .filter((h) => Date.parse(h.heartbeatAt))
      .map((h) => [Date.parse(h.heartbeatAt), deriveTrendState(h)]);
  }, [historyQuery.data]);

  return (
    <>
      <tr className="data-row-clickable" onClick={onToggle}>
        <td className="registry-expand-col">
          <span className="registry-expand-icon">
            {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
          </span>
        </td>
        <td>
          <div className="registry-name-cell">
            <strong>{item.name}</strong>
            <span className="muted-text">{item.scopeKey || item.nodeId}</span>
          </div>
        </td>
        <td>
          <span className={healthBadgeClassName(item.derivedStatus)}>{item.derivedLabel}</span>
        </td>
        <td>{item.mode || <span className="muted-text">unknown</span>}</td>
        <td className="mono">{item.heartbeatAt ? formatISODateTime(item.heartbeatAt) : <span className="muted-text">never</span>}</td>
        <td>{item.policyRevisionId || <span className="muted-text">unassigned</span>}</td>
        <td>{item.listenerSummary || <span className="muted-text">none</span>}</td>
        <td>{item.certSummary || <span className="muted-text">none</span>}</td>
      </tr>
      {expanded ? (
        <tr className="detail-row">
          <td colSpan={8}>
            <div className="detail-panel">
              <div className="detail-section">
                <h4>Listener status</h4>
                <div className="detail-badge-grid">
                  {Object.entries(item.listenerStatus || {}).length > 0 ? (
                    Object.entries(item.listenerStatus).map(([key, value]) => (
                      <span key={key} className={`badge ${value === 'up' ? 'is-good' : 'is-danger'}`}>{key}: {value}</span>
                    ))
                  ) : (
                    <span className="muted-text">No listener data</span>
                  )}
                </div>
              </div>
              <div className="detail-section">
                <h4>Certificate status</h4>
                <div className="detail-badge-grid">
                  {Object.entries(item.certStatus || {}).length > 0 ? (
                    Object.entries(item.certStatus).map(([key, value]) => (
                      <span key={key} className={`badge ${value === 'healthy' || value === 'renewed' ? 'is-good' : value === 'renew-soon' || value === 'rotate' ? 'is-warn' : 'is-danger'}`}>{key}: {value}</span>
                    ))
                  ) : (
                    <span className="muted-text">No certificate data</span>
                  )}
                </div>
              </div>
              <div className="detail-section">
                <h4>Health trend (24h)</h4>
                {historyQuery.isPending ? (
                  <AsyncState detail="Loading health history for this node." title="" />
                ) : historyQuery.isError ? (
                  <AsyncState actionLabel="Retry" detail={formatControlPlaneError(historyQuery.error)} onAction={() => void historyQuery.refetch()} title="Failed to load history" />
                ) : historyData.length === 0 ? (
                  <AsyncState detail="No historical health data available for this node." title="" />
                ) : (
                  <ReactECharts
                    option={{
                      grid: {top: 8, right: 8, bottom: 8, left: 40},
                      xAxis: {type: 'time', axisLabel: {fontSize: 10}},
                      yAxis: {type: 'category', data: ['healthy', 'degraded', 'stale'], axisLabel: {fontSize: 10}},
                      series: [{type: 'line', step: 'end', data: historyData, smooth: false, symbol: 'circle', symbolSize: 4}],
                      tooltip: {trigger: 'axis'},
                      backgroundColor: 'transparent'
                    }}
                    style={{height: 180}}
                  />
                )}
              </div>
            </div>
          </td>
        </tr>
      ) : null}
    </>
  );
}

function deriveHealthState(item: NodeHealth) {
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

function deriveTrendState(item: NodeHealth): string {
  const listenerValues = Object.values(item.listenerStatus || {});
  const certValues = Object.values(item.certStatus || {});
  const hasDegradedSignal = [...listenerValues, ...certValues].some((value) => value !== 'up' && value !== 'healthy' && value !== 'renewed');
  return hasDegradedSignal ? 'degraded' : 'healthy';
}

function joinMap(value: Record<string, string>) {
  return Object.entries(value || {})
    .map(([key, item]) => `${key}:${item}`)
    .join(', ');
}

function healthBadgeClassName(status: string) {
  if (status === 'healthy') {
    return 'badge is-good';
  }
  if (status === 'stale') {
    return 'badge is-warn';
  }
  if (status === 'unreported') {
    return 'badge is-neutral';
  }
  return 'badge is-danger';
}
