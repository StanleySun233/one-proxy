'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {getCertificates, getNodeHealth, getNodes} from '@/lib/control-plane-api';
import {Certificate, NodeHealth} from '@/lib/control-plane-types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

const staleThresholdMs = 2 * 60 * 1000;

export default function HealthPage() {
  const t = useTranslations('pages');
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';

  const [query, setQuery] = useState('');
  const [healthFilter, setHealthFilter] = useState('all');
  const [certFilter, setCertFilter] = useState('all');

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

  const filteredHealth = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return healthRows.filter((item) => {
      if (healthFilter !== 'all' && item.derivedStatus !== healthFilter) {
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
  }, [healthFilter, healthRows, query]);

  const filteredCertificates = useMemo(() => {
    return certificateRows.filter((item) => (certFilter === 'all' ? true : item.status === certFilter));
  }, [certFilter, certificateRows]);

  const summary = useMemo(() => {
    const healthy = healthRows.filter((item) => item.derivedStatus === 'healthy').length;
    const stale = healthRows.filter((item) => item.derivedStatus === 'stale').length;
    const degraded = healthRows.filter((item) => item.derivedStatus === 'degraded').length;
    const unreported = healthRows.filter((item) => item.derivedStatus === 'unreported').length;
    const certPressure = certificateRows.filter((item) => item.status !== 'healthy' && item.status !== 'renewed').length;
    return {healthy, stale, degraded, unreported, certPressure};
  }, [certificateRows, healthRows]);

  const availableCertStatuses = useMemo(
    () => Array.from(new Set(certificateRows.map((item) => item.status))).sort(),
    [certificateRows]
  );

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
          {healthQuery.isPending || nodesQuery.isPending ? (
            <AsyncState detail="Heartbeat and node registry data are loading." title="Loading node health" />
          ) : healthQuery.isError ? (
            <AsyncState actionLabel="Retry" detail={formatControlPlaneError(healthQuery.error)} onAction={() => void healthQuery.refetch()} title="Failed to load node health" />
          ) : nodesQuery.isError ? (
            <AsyncState actionLabel="Retry" detail={formatControlPlaneError(nodesQuery.error)} onAction={() => void nodesQuery.refetch()} title="Failed to load node registry" />
          ) : healthRows.length === 0 ? (
            <AsyncState detail="Heartbeat rows will appear after the first node-agent reports status." title="No health data yet" />
          ) : (
            <div className="registry-stack">
              <div className="registry-toolbar">
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
              {filteredHealth.length === 0 ? (
                <AsyncState detail="Adjust the current query or filters to see matching heartbeat rows." title="No matching health rows" />
              ) : (
                <div className="table-card">
                  <table className="data-table">
                    <thead>
                      <tr>
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
                        <tr key={item.nodeId}>
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
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Certificate Registry</p>
              <h3>Certificate status</h3>
              <p className="section-copy">Track public and internal certificate pressure separately from heartbeat freshness so renewal risk remains visible.</p>
            </div>
            <div className="inline-cluster">
              <span className="badge">{filteredCertificates.length} shown</span>
              <span className="badge">{certificateRows.length} total</span>
            </div>
          </div>
          {certificatesQuery.isPending ? (
            <AsyncState detail="Certificate status is loading." title="Loading certificates" />
          ) : certificatesQuery.isError ? (
            <AsyncState actionLabel="Retry" detail={formatControlPlaneError(certificatesQuery.error)} onAction={() => void certificatesQuery.refetch()} title="Failed to load certificates" />
          ) : certificateRows.length === 0 ? (
            <AsyncState detail="Public and internal certificates will appear here once registered." title="No certificates yet" />
          ) : (
            <div className="registry-stack">
              <div className="registry-toolbar">
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Status</span>
                  <select className="field-select" onChange={(event) => setCertFilter(event.target.value)} value={certFilter}>
                    <option value="all">All statuses</option>
                    {availableCertStatuses.map((status) => (
                      <option key={status} value={status}>
                        {status}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
              {filteredCertificates.length === 0 ? (
                <AsyncState detail="Adjust the current status filter to see matching certificates." title="No matching certificates" />
              ) : (
                <div className="table-card">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>Owner</th>
                        <th>Status</th>
                        <th>Type</th>
                        <th>Provider</th>
                        <th>Valid to</th>
                        <th>ID</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredCertificates.map((item) => (
                        <tr key={item.id}>
                          <td>{item.ownerName}</td>
                          <td>
                            <span className={certificateBadgeClassName(item.status)}>{item.status}</span>
                          </td>
                          <td>{item.certType}</td>
                          <td>{item.provider}</td>
                          <td className="mono">{formatISODateTime(item.notAfter, '-')}</td>
                          <td className="mono registry-id-cell">{item.id}</td>
                        </tr>
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

function certificateBadgeClassName(status: string) {
  if (status === 'healthy' || status === 'renewed') {
    return 'badge is-good';
  }
  if (status === 'renew-soon' || status === 'rotate') {
    return 'badge is-warn';
  }
  return 'badge is-danger';
}
