'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {fetchEnums, getCertificates, getNodes} from '@/lib/api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

export default function CertificatesPage() {
  const t = useTranslations('pages');
  const {session} = useAuth();
  const accessToken = session?.accessToken || '';

  const [certFilter, setCertFilter] = useState('all');
  const [query, setQuery] = useState('');
  const [expiryRange, setExpiryRange] = useState('all');

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });
  const certificatesQuery = useQuery({
    queryKey: ['certificates', accessToken],
    queryFn: () => getCertificates(accessToken),
    enabled: !!accessToken,
    refetchInterval: 10000
  });

  const enumsQuery = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const enums = enumsQuery.data;
  const nodes = nodesQuery.data || [];
  const certificates = certificatesQuery.data || [];
  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);

  const certificateRows = useMemo(() => {
    return certificates.map((item) => ({
      ...item,
      ownerName: nodesByID.get(item.ownerId)?.name || item.ownerId,
      daysRemaining: daysUntil(item.notAfter)
    }));
  }, [certificates, nodesByID]);

  const availableCertStatuses = useMemo(
    () => Array.from(new Set(certificateRows.map((item) => item.status))).sort(),
    [certificateRows]
  );

  const filteredCertificates = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return certificateRows.filter((item) => {
      if (certFilter !== 'all' && item.status !== certFilter) {
        return false;
      }
      if (expiryRange === '7d' && (item.daysRemaining === null || item.daysRemaining > 7)) {
        return false;
      }
      if (expiryRange === '30d' && (item.daysRemaining === null || item.daysRemaining > 30)) {
        return false;
      }
      if (expiryRange === '90d' && (item.daysRemaining === null || item.daysRemaining > 90)) {
        return false;
      }
      if (expiryRange === 'expired' && (item.daysRemaining === null || item.daysRemaining > 0)) {
        return false;
      }
      if (!normalized) {
        return true;
      }
      return [item.ownerName, item.certType, item.provider, item.id].join(' ').toLowerCase().includes(normalized);
    });
  }, [certFilter, certificateRows, expiryRange, query]);

  const loading = certificatesQuery.isPending || nodesQuery.isPending;
  const error = certificatesQuery.isError || nodesQuery.isError;
  const empty = !loading && !error && certificateRows.length === 0;
  const filteredEmpty = !loading && !error && !empty && filteredCertificates.length === 0;

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Health" title={t('healthTitle')} description={t('healthDesc')} />

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
          {loading ? (
            <AsyncState detail="Certificate status is loading." title="Loading certificates" />
          ) : error ? (
            <AsyncState actionLabel="Retry" detail={formatControlPlaneError(certificatesQuery.error || nodesQuery.error)} onAction={() => { void certificatesQuery.refetch(); void nodesQuery.refetch(); }} title="Failed to load certificates" />
          ) : empty ? (
            <AsyncState detail="Public and internal certificates will appear here once registered." title="No certificates yet" />
          ) : (
            <div className="registry-stack">
              <div className="registry-toolbar">
                <label className="field-stack registry-filter">
                  <span>Search</span>
                  <input
                    className="field-input"
                    onChange={(event) => setQuery(event.target.value)}
                    placeholder="Search by owner, type, provider, or ID"
                    type="search"
                    value={query}
                  />
                </label>
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
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Expiring within</span>
                  <select className="field-select" onChange={(event) => setExpiryRange(event.target.value)} value={expiryRange}>
                    <option value="all">All</option>
                    <option value="expired">Expired</option>
                    <option value="7d">7 days</option>
                    <option value="30d">30 days</option>
                    <option value="90d">90 days</option>
                  </select>
                </label>
              </div>
              {filteredEmpty ? (
                <AsyncState detail="Adjust the current search or filters to see matching certificates." title="No matching certificates" />
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
                            <span className={certBadgeClassName(item.status, enums?.cert_status)}>{item.status}</span>
                          </td>
                          <td>{item.certType}</td>
                          <td>{item.provider}</td>
                          <td className="mono">
                            <span className="expiry-indicator">
                              <span className={`expiry-dot ${expiryDotColor(item.daysRemaining)}`} />
                              {formatISODateTime(item.notAfter, '-')}
                              {item.daysRemaining !== null && (
                                <span className={`expiry-days ${expiryDotColor(item.daysRemaining)}`}>
                                  {item.daysRemaining > 0 ? `${item.daysRemaining}d` : 'expired'}
                                </span>
                              )}
                            </span>
                          </td>
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

function daysUntil(notAfter: string): number | null {
  if (!notAfter) return null;
  const target = Date.parse(notAfter);
  if (!Number.isFinite(target)) return null;
  return Math.ceil((target - Date.now()) / (1000 * 60 * 60 * 24));
}

function expiryDotColor(daysRemaining: number | null): string {
  if (daysRemaining === null) return '';
  if (daysRemaining <= 0 || daysRemaining < 7) return 'is-danger';
  if (daysRemaining <= 30) return 'is-warn';
  return 'is-good';
}

function certBadgeClassName(status: string, certStatusEnum?: Record<string, {name: string; meta?: {className?: string}}>): string {
  const className = certStatusEnum?.[status]?.meta?.className;
  return `badge ${className || 'is-neutral'}`;
}
