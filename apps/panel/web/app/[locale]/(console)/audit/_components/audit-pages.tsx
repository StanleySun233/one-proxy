'use client';

import {Suspense, useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import {useSearchParams} from 'next/navigation';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {useAuth} from '@/components/auth-provider';
import {Link, usePathname, useRouter} from '@/i18n/navigation';
import {getAuditBusinessEvents, getAuditDashboard, getAuditNetworkSessions} from '@/lib/api';
import type {AuditDecisionCount, AuditEvent, AuditGroup, NetworkAuditSummary, NetworkSession} from '@/lib/types/audit';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

const LIMIT = 100;

function defaultFrom() {
  const date = new Date();
  date.setDate(date.getDate() - 7);
  return date.toISOString().slice(0, 10);
}

function todayDate() {
  return new Date().toISOString().slice(0, 10);
}

function bytesLabel(value: number | undefined) {
  const bytes = value || 0;
  if (bytes >= 1024 * 1024 * 1024) {
    return `${(bytes / 1024 / 1024 / 1024).toFixed(1)} GB`;
  }
  if (bytes >= 1024 * 1024) {
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  }
  if (bytes >= 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${bytes} B`;
}

function isoRangeValue(value: string, endOfDay = false) {
  if (!value) {
    return '';
  }
  return `${value}T${endOfDay ? '23:59:59' : '00:00:00'}Z`;
}

function networkAuditHref(params: Record<string, string>) {
  const values = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value) {
      values.set(key, value);
    }
  });
  const query = values.toString();
  return query ? `/audit/network?${query}` : '/audit/network';
}

function groupName(item: AuditGroup) {
  return item.name || item.id || '-';
}

function groupCount(item: AuditGroup) {
  return item.count || item.sessions || 0;
}

function sumGroups(groups: AuditGroup[], key: 'bytesIn' | 'bytesOut' | 'count') {
  return groups.reduce((total, item) => total + (key === 'count' ? groupCount(item) : item[key] || 0), 0);
}

function decisionCountsFromMap(values: Record<string, number> | undefined): AuditDecisionCount[] {
  return Object.entries(values || {}).map(([decision, count]) => ({decision, count}));
}

function tenantGroups(summary: Pick<NetworkAuditSummary, 'tenantTraffic'> | null | undefined): AuditGroup[] {
  return (summary?.tenantTraffic || []).map((item) => ({
    id: item.tenantId,
    name: item.tenantId,
    count: item.count,
    bytesIn: item.bytesIn,
    bytesOut: item.bytesOut
  }));
}

function userGroups(summary: Pick<NetworkAuditSummary, 'userTraffic'> | null | undefined): AuditGroup[] {
  return (summary?.userTraffic || []).map((item) => ({
    id: item.actorId,
    name: item.actorId || '-',
    count: item.count,
    bytesIn: item.bytesIn,
    bytesOut: item.bytesOut
  }));
}

function nodeGroups(summary: Pick<NetworkAuditSummary, 'nodeTraffic'> | null | undefined): AuditGroup[] {
  return (summary?.nodeTraffic || []).map((item) => ({
    id: item.nodeId,
    name: item.nodeId || '-',
    count: item.count,
    bytesIn: item.bytesIn,
    bytesOut: item.bytesOut
  }));
}

function targetGroups(summary: Pick<NetworkAuditSummary, 'topTargets'> | null | undefined): AuditGroup[] {
  return (summary?.topTargets || []).map((item) => ({
    id: item.targetHost,
    name: item.targetHost || '-',
    count: item.count,
    bytesIn: item.bytesIn,
    bytesOut: item.bytesOut
  }));
}

function useAuditRange() {
  const {session} = useAuth();
  const [from, setFrom] = useState(defaultFrom);
  const [to, setTo] = useState(() => new Date().toISOString().slice(0, 10));
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const rangeQuery = useMemo(() => ({
    from: isoRangeValue(from),
    to: isoRangeValue(to, true),
    tenantId: activeTenantId || undefined
  }), [activeTenantId, from, to]);

  return {accessToken, activeTenantId, from, rangeQuery, setFrom, setTo, to};
}

function DateRangeFilter({from, onFromChange, onToChange, to}: {
  from: string;
  onFromChange: (value: string) => void;
  onToChange: (value: string) => void;
  to: string;
}) {
  const t = useTranslations();
  const auditT = useTranslations('pages.audit');

  return (
    <ConsoleFilterBar title={t('common.filter')}>
      <ConsoleFilterItem label={auditT('from')} match={t('common.equals')}>
        <input className="field-input" onChange={(event) => onFromChange(event.target.value)} type="date" value={from} />
      </ConsoleFilterItem>
      <ConsoleFilterItem label={auditT('to')} match={t('common.equals')}>
        <input className="field-input" onChange={(event) => onToChange(event.target.value)} type="date" value={to} />
      </ConsoleFilterItem>
    </ConsoleFilterBar>
  );
}

export function AuditDashboardPage() {
  const t = useTranslations();
  const auditT = useTranslations('pages.audit');
  const {accessToken, activeTenantId, from, rangeQuery, setFrom, setTo, to} = useAuditRange();

  const dashboardQuery = useQuery({
    queryKey: ['audit-dashboard', accessToken, activeTenantId, rangeQuery.from, rangeQuery.to],
    queryFn: () => getAuditDashboard(accessToken, activeTenantId, rangeQuery),
    enabled: !!accessToken
  });

  const dashboard = dashboardQuery.data;
  const tenantTraffic = tenantGroups(dashboard);
  const userTraffic = userGroups(dashboard);
  const nodeTraffic = nodeGroups(dashboard);
  const targetTraffic = targetGroups(dashboard);
  const decisionCounts = decisionCountsFromMap(dashboard?.decisionCount);
  const totalSessions = dashboard?.total || sumGroups(tenantTraffic, 'count');
  const totalBytes = (dashboard?.bytesIn || sumGroups(tenantTraffic, 'bytesIn')) + (dashboard?.bytesOut || sumGroups(tenantTraffic, 'bytesOut'));

  return (
    <AuthGate>
      <ConsolePage title={auditT('dashboardTitle')}>
        <DateRangeFilter from={from} onFromChange={setFrom} onToChange={setTo} to={to} />

        <section className="metrics-grid">
          <article className="metric-card panel-card">
            <span>{auditT('businessEvents')}</span>
            <strong>{dashboardQuery.isPending ? '-' : dashboard?.recentBusinessEvents?.length ?? 0}</strong>
          </article>
          <article className="metric-card panel-card soft-card">
            <span>{auditT('networkSessions')}</span>
            <strong>{dashboardQuery.isPending ? '-' : totalSessions}</strong>
          </article>
          <Link className="metric-card panel-card warm-card audit-card-link" href={networkAuditHref({from, to, decision: 'deny'})}>
            <span>{auditT('deniedTraffic')}</span>
            <strong>{dashboardQuery.isPending ? '-' : decisionCounts.find((item) => item.decision === 'deny')?.count ?? 0}</strong>
          </Link>
          <article className="metric-card panel-card">
            <span>{auditT('totalTraffic')}</span>
            <strong>{dashboardQuery.isPending ? '-' : bytesLabel(totalBytes)}</strong>
          </article>
        </section>

        {dashboardQuery.isError ? (
          <AsyncState
            actionLabel={t('common.retry')}
            detail={formatControlPlaneError(dashboardQuery.error)}
            onAction={() => void dashboardQuery.refetch()}
            title={auditT('dashboardFailed')}
          />
        ) : null}

        <section className="two-column-grid">
          <AuditGroupPanel groups={tenantTraffic} title={auditT('tenantTraffic')} />
          <AuditGroupPanel groups={userTraffic} title={auditT('userTraffic')} />
          <AuditGroupPanel groups={nodeTraffic} hrefForGroup={(item) => networkAuditHref({from, to, nodeId: item.id})} title={auditT('nodeTraffic')} />
          <AuditGroupPanel groups={targetTraffic} hrefForGroup={(item) => networkAuditHref({from, to, targetHost: item.id})} title={auditT('topTargets')} />
        </section>
      </ConsolePage>
    </AuthGate>
  );
}

export function AuditBusinessPage() {
  const auditT = useTranslations('pages.audit');
  const {accessToken, activeTenantId, from, rangeQuery, setFrom, setTo, to} = useAuditRange();
  const [businessActor, setBusinessActor] = useState('');
  const [businessResource, setBusinessResource] = useState('');
  const [businessOutcome, setBusinessOutcome] = useState('');

  const businessQuery = useQuery({
    queryKey: ['audit-business', accessToken, activeTenantId, rangeQuery.from, rangeQuery.to, businessActor, businessResource, businessOutcome],
    queryFn: () => getAuditBusinessEvents(accessToken, activeTenantId, {
      ...rangeQuery,
      actorId: businessActor.trim(),
      resourceType: businessResource.trim(),
      outcome: businessOutcome,
      limit: LIMIT
    }),
    enabled: !!accessToken
  });

  const business = businessQuery.data;

  return (
    <AuthGate>
      <ConsolePage title={auditT('businessTitle')}>
        <BusinessAuditList
          actor={businessActor}
          events={business?.items || []}
          from={from}
          isError={businessQuery.isError}
          isPending={businessQuery.isPending}
          onActorChange={setBusinessActor}
          onFromChange={setFrom}
          onOutcomeChange={setBusinessOutcome}
          onRefetch={() => void businessQuery.refetch()}
          onResourceChange={setBusinessResource}
          onToChange={setTo}
          outcome={businessOutcome}
          queryError={businessQuery.error}
          resource={businessResource}
          to={to}
        />
      </ConsolePage>
    </AuthGate>
  );
}

export function AuditNetworkPage() {
  const auditT = useTranslations('pages.audit');

  return (
    <Suspense fallback={<AsyncState detail="" title={auditT('loadingNetwork')} />}>
      <AuditNetworkPageContent />
    </Suspense>
  );
}

function AuditNetworkPageContent() {
  const auditT = useTranslations('pages.audit');
  const {session} = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const from = searchParams.get('from') || defaultFrom();
  const to = searchParams.get('to') || todayDate();
  const actor = searchParams.get('actorId') || '';
  const node = searchParams.get('nodeId') || '';
  const target = searchParams.get('targetHost') || '';
  const decision = searchParams.get('decision') || '';
  const routeId = searchParams.get('routeId') || '';
  const chainId = searchParams.get('chainId') || '';
  const denyReason = searchParams.get('denyReason') || '';
  const matchedRuleId = searchParams.get('matchedRuleId') || '';
  const policyRevision = searchParams.get('policyRevision') || '';
  const decisionSource = searchParams.get('decisionSource') || '';
  const rangeQuery = useMemo(() => ({
    from: isoRangeValue(from),
    to: isoRangeValue(to, true),
    tenantId: activeTenantId || undefined
  }), [activeTenantId, from, to]);

  const setFilter = (key: string, value: string) => {
    const params = new URLSearchParams(searchParams.toString());
    if (value) {
      params.set(key, value);
    } else {
      params.delete(key);
    }
    const next = params.toString();
    router.replace(next ? `${pathname}?${next}` : pathname);
  };

  const applyPreset = (preset: 'all' | 'denied' | 'policy') => {
    const params = new URLSearchParams(searchParams.toString());
    if (preset === 'all') {
      ['decision', 'denyReason', 'decisionSource'].forEach((key) => params.delete(key));
    }
    if (preset === 'denied') {
      params.set('decision', 'deny');
      params.delete('decisionSource');
    }
    if (preset === 'policy') {
      params.set('decisionSource', 'policy');
    }
    const next = params.toString();
    router.replace(next ? `${pathname}?${next}` : pathname);
  };

  const networkQuery = useQuery({
    queryKey: ['audit-network', accessToken, activeTenantId, rangeQuery.from, rangeQuery.to, actor, node, target, decision, routeId, chainId, denyReason, matchedRuleId, policyRevision, decisionSource],
    queryFn: () => getAuditNetworkSessions(accessToken, activeTenantId, {
      ...rangeQuery,
      actorId: actor.trim(),
      nodeId: node.trim(),
      targetHost: target.trim(),
      decision,
      routeId: routeId.trim(),
      chainId: chainId.trim(),
      denyReason: denyReason.trim(),
      matchedRuleId: matchedRuleId.trim(),
      policyRevision: policyRevision.trim(),
      decisionSource,
      limit: LIMIT
    }),
    enabled: !!accessToken
  });

  const network = networkQuery.data;

  return (
    <AuthGate>
      <ConsolePage title={auditT('networkTitle')}>
        <NetworkAuditList
          actor={actor}
          chainId={chainId}
          decision={decision}
          decisionSource={decisionSource}
          denyReason={denyReason}
          from={from}
          isError={networkQuery.isError}
          isPending={networkQuery.isPending}
          matchedRuleId={matchedRuleId}
          networkNode={node}
          onApplyPreset={applyPreset}
          onFilterChange={setFilter}
          onRefetch={() => void networkQuery.refetch()}
          policyRevision={policyRevision}
          queryError={networkQuery.error}
          routeId={routeId}
          sessions={network?.items || []}
          summary={network?.summary || null}
          target={target}
          to={to}
        />
      </ConsolePage>
    </AuthGate>
  );
}

function AuditGroupPanel({groups, hrefForGroup, title}: {groups: AuditGroup[]; hrefForGroup?: (item: AuditGroup) => string; title: string}) {
  const common = useTranslations('common');

  return (
    <article className="panel-card">
      <div className="panel-toolbar">
        <div>
          <p className="section-kicker">{common('shown')}</p>
          <h3>{title}</h3>
        </div>
        <span className="badge">{groups.length}</span>
      </div>
      <div className="queue-list">
        {groups.length === 0 ? (
          <div className="queue-item">
            <strong>{common('empty')}</strong>
          </div>
        ) : groups.slice(0, 5).map((item) => {
          const content = (
            <>
              <strong>{groupName(item)}</strong>
              <span className="muted-text">
                {groupCount(item)} · {bytesLabel((item.bytesIn || 0) + (item.bytesOut || 0))}
              </span>
            </>
          );
          return hrefForGroup ? (
            <Link className="queue-item audit-card-link" href={hrefForGroup(item)} key={item.id || item.name}>
              {content}
            </Link>
          ) : (
            <div className="queue-item" key={item.id || item.name}>
              {content}
            </div>
          );
        })}
      </div>
    </article>
  );
}

function BusinessAuditList(props: {
  actor: string;
  events: AuditEvent[];
  from: string;
  isError: boolean;
  isPending: boolean;
  onActorChange: (value: string) => void;
  onFromChange: (value: string) => void;
  onOutcomeChange: (value: string) => void;
  onRefetch: () => void;
  onResourceChange: (value: string) => void;
  onToChange: (value: string) => void;
  outcome: string;
  queryError: unknown;
  resource: string;
  to: string;
}) {
  const t = useTranslations();
  const auditT = useTranslations('pages.audit');

  return (
    <>
      <ConsoleFilterBar title={auditT('businessFilters')}>
        <ConsoleFilterItem label={auditT('from')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onFromChange(event.target.value)} type="date" value={props.from} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('to')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onToChange(event.target.value)} type="date" value={props.to} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('actor')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => props.onActorChange(event.target.value)} placeholder={auditT('actor')} value={props.actor} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('resource')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onResourceChange(event.target.value)} placeholder={auditT('resource')} value={props.resource} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('outcome')} match={t('common.equals')}>
          <select className="field-select" onChange={(event) => props.onOutcomeChange(event.target.value)} value={props.outcome}>
            <option value="">{t('common.all')}</option>
            <option value="success">{auditT('success')}</option>
            <option value="failure">{auditT('failure')}</option>
            <option value="denied">{auditT('denied')}</option>
          </select>
        </ConsoleFilterItem>
      </ConsoleFilterBar>

      <ConsoleList count={props.events.length} title={auditT('businessEvents')}>
        {props.isPending ? (
          <AsyncState detail={t('common.loading')} title={auditT('loadingBusiness')} />
        ) : props.isError ? (
          <AsyncState actionLabel={t('common.retry')} detail={formatControlPlaneError(props.queryError)} onAction={props.onRefetch} title={auditT('businessFailed')} />
        ) : props.events.length === 0 ? (
          <AsyncState detail={auditT('emptyBusinessDetail')} title={t('common.empty')} />
        ) : (
          <div className="table-card">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{auditT('time')}</th>
                  <th>{auditT('actor')}</th>
                  <th>{auditT('action')}</th>
                  <th>{auditT('resource')}</th>
                  <th>{auditT('outcome')}</th>
                  <th>{auditT('reason')}</th>
                </tr>
              </thead>
              <tbody>
                {props.events.map((event) => (
                  <tr key={event.id}>
                    <td>{formatISODateTime(event.occurredAt, event.occurredAt)}</td>
                    <td>{event.actorName || event.actorId || event.actorType || '-'}</td>
                    <td>{event.action}</td>
                    <td>{event.resourceName || event.resourceId || event.resourceType}</td>
                    <td><span className={event.outcome === 'success' ? 'badge is-good' : 'badge is-warn'}>{event.outcome}</span></td>
                    <td>{event.reason || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ConsoleList>
    </>
  );
}

function NetworkAuditList(props: {
  actor: string;
  chainId: string;
  decision: string;
  decisionSource: string;
  denyReason: string;
  from: string;
  isError: boolean;
  isPending: boolean;
  matchedRuleId: string;
  networkNode: string;
  onApplyPreset: (preset: 'all' | 'denied' | 'policy') => void;
  onFilterChange: (key: string, value: string) => void;
  onRefetch: () => void;
  policyRevision: string;
  queryError: unknown;
  routeId: string;
  sessions: NetworkSession[];
  summary: NetworkAuditSummary | null;
  target: string;
  to: string;
}) {
  const t = useTranslations();
  const auditT = useTranslations('pages.audit');
  const [selected, setSelected] = useState<NetworkSession | null>(null);
  const deniedCount = props.summary?.decisionCount?.deny || 0;
  const totalTraffic = (props.summary?.bytesIn || 0) + (props.summary?.bytesOut || 0);
  const denyReasons = Object.entries(props.summary?.denyReasonCount || {}).filter(([reason]) => reason);
  const topTargets = props.summary?.topTargets || [];

  return (
    <>
      <section className="metrics-grid">
        <article className="metric-card panel-card">
          <span>{auditT('networkSessions')}</span>
          <strong>{props.isPending ? '-' : props.summary?.total ?? props.sessions.length}</strong>
        </article>
        <article className="metric-card panel-card warm-card">
          <span>{auditT('deniedTraffic')}</span>
          <strong>{props.isPending ? '-' : deniedCount}</strong>
        </article>
        <article className="metric-card panel-card soft-card">
          <span>{auditT('uniqueTargets')}</span>
          <strong>{props.isPending ? '-' : props.summary?.topTargets?.length ?? 0}</strong>
        </article>
        <article className="metric-card panel-card">
          <span>{auditT('totalTraffic')}</span>
          <strong>{props.isPending ? '-' : bytesLabel(totalTraffic)}</strong>
        </article>
      </section>

      <div className="audit-preset-bar">
        <button className="ghost-button" onClick={() => props.onApplyPreset('all')} type="button">{auditT('allTraffic')}</button>
        <button className="ghost-button" onClick={() => props.onApplyPreset('denied')} type="button">{auditT('deniedOnly')}</button>
        <button className="ghost-button" onClick={() => props.onApplyPreset('policy')} type="button">{auditT('policyMatched')}</button>
      </div>

      <ConsoleFilterBar title={auditT('networkFilters')}>
        <ConsoleFilterItem label={auditT('from')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('from', event.target.value)} type="date" value={props.from} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('to')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('to', event.target.value)} type="date" value={props.to} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('actor')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('actorId', event.target.value)} placeholder={auditT('actor')} value={props.actor} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('node')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('nodeId', event.target.value)} placeholder={auditT('node')} value={props.networkNode} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('targetHost')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('targetHost', event.target.value)} placeholder={auditT('targetHost')} value={props.target} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('decision')} match={t('common.equals')}>
          <select className="field-select" onChange={(event) => props.onFilterChange('decision', event.target.value)} value={props.decision}>
            <option value="">{t('common.all')}</option>
            <option value="allow">{auditT('allow')}</option>
            <option value="deny">{auditT('deny')}</option>
          </select>
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('routeId')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('routeId', event.target.value)} placeholder={auditT('routeId')} value={props.routeId} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('chainId')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('chainId', event.target.value)} placeholder={auditT('chainId')} value={props.chainId} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('denyReason')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('denyReason', event.target.value)} placeholder={auditT('denyReason')} value={props.denyReason} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('matchedRule')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('matchedRuleId', event.target.value)} placeholder={auditT('matchedRule')} value={props.matchedRuleId} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('policyRevision')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onFilterChange('policyRevision', event.target.value)} placeholder={auditT('policyRevision')} value={props.policyRevision} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('decisionSource')} match={t('common.equals')}>
          <select className="field-select" onChange={(event) => props.onFilterChange('decisionSource', event.target.value)} value={props.decisionSource}>
            <option value="">{t('common.all')}</option>
            <option value="policy">{auditT('policy')}</option>
            <option value="unknown">{auditT('unknown')}</option>
          </select>
        </ConsoleFilterItem>
      </ConsoleFilterBar>

      {denyReasons.length > 0 ? (
        <div className="audit-preset-bar">
          {denyReasons.slice(0, 6).map(([reason, count]) => (
            <button className="ghost-button" key={reason} onClick={() => props.onFilterChange('denyReason', reason)} type="button">
              {reason} · {count}
            </button>
          ))}
        </div>
      ) : null}

      {topTargets.length > 0 ? (
        <div className="audit-preset-bar">
          {topTargets.slice(0, 6).map((target) => (
            <button className="ghost-button" key={target.targetHost} onClick={() => props.onFilterChange('targetHost', target.targetHost)} type="button">
              {target.targetHost} · {bytesLabel((target.bytesIn || 0) + (target.bytesOut || 0))}
            </button>
          ))}
        </div>
      ) : null}

      <ConsoleList count={props.sessions.length} title={auditT('networkSessions')}>
        {props.isPending ? (
          <AsyncState detail={t('common.loading')} title={auditT('loadingNetwork')} />
        ) : props.isError ? (
          <AsyncState actionLabel={t('common.retry')} detail={formatControlPlaneError(props.queryError)} onAction={props.onRefetch} title={auditT('networkFailed')} />
        ) : props.sessions.length === 0 ? (
          <AsyncState detail={auditT('emptyNetworkDetail')} title={t('common.empty')} />
        ) : (
          <div className="table-card">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{auditT('time')}</th>
                  <th>{auditT('path')}</th>
                  <th>{auditT('target')}</th>
                  <th>{auditT('matchedRule')}</th>
                  <th>{auditT('decision')}</th>
                  <th>{auditT('traffic')}</th>
                  <th>{auditT('duration')}</th>
                </tr>
              </thead>
              <tbody>
                {props.sessions.map((session) => (
                  <tr className="audit-clickable-row" key={session.id} onClick={() => setSelected(session)}>
                    <td>{formatISODateTime(session.endedAt || session.startedAt, session.endedAt || session.startedAt)}</td>
                    <td>{session.entryNodeId || '-'} -&gt; {session.exitNodeId || '-'}</td>
                    <td>{session.targetHost}:{session.targetPort}</td>
                    <td>{session.matchedRuleId || session.routeId || '-'}</td>
                    <td><span className={session.decision === 'allow' ? 'badge is-good' : 'badge is-warn'}>{session.decision}</span></td>
                    <td>{bytesLabel((session.bytesIn || 0) + (session.bytesOut || 0))}</td>
                    <td>{session.durationMs || 0} ms</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ConsoleList>

      <NetworkSessionDetail session={selected} onClose={() => setSelected(null)} />
    </>
  );
}

function NetworkSessionDetail({session, onClose}: {session: NetworkSession | null; onClose: () => void}) {
  const auditT = useTranslations('pages.audit');
  if (!session) {
    return null;
  }
  return (
    <ConsoleCrudModal onClose={onClose} open={true} subtitle={session.id} title={auditT('sessionDetail')}>
      <div className="audit-detail-grid">
        <AuditDetailSection
          items={[
            [auditT('time'), formatISODateTime(session.endedAt || session.startedAt, session.endedAt || session.startedAt)],
            [auditT('actor'), session.actorId || '-'],
            [auditT('token'), session.tokenId || '-'],
            [auditT('sourceIp'), session.sourceIp || '-']
          ]}
          title={auditT('request')}
        />
        <AuditDetailSection
          items={[
            [auditT('target'), `${session.targetHost}:${session.targetPort}`],
            [auditT('method'), session.method || '-'],
            [auditT('scheme'), session.scheme || '-'],
            [auditT('statusCode'), session.statusCode ? String(session.statusCode) : '-']
          ]}
          title={auditT('target')}
        />
        <AuditDetailSection
          items={[
            [auditT('entryNode'), session.entryNodeId || '-'],
            [auditT('exitNode'), session.exitNodeId || '-'],
            [auditT('chainId'), session.chainId || '-'],
            [auditT('routeId'), session.routeId || '-']
          ]}
          title={auditT('path')}
        />
        <AuditDetailSection
          items={[
            [auditT('governanceMode'), session.governanceMode || '-'],
            [auditT('policyRevision'), session.policyRevision || '-'],
            [auditT('matchedRule'), session.matchedRuleId || '-'],
            [auditT('rulePattern'), [session.matchedRuleType, session.matchedRulePattern].filter(Boolean).join(': ') || '-'],
            [auditT('action'), session.matchedAction || '-'],
            [auditT('decisionSource'), session.decisionSource || '-']
          ]}
          title={auditT('evidence')}
        />
        <AuditDetailSection
          items={[
            [auditT('decision'), session.decision],
            [auditT('denyReason'), session.denyReason || '-'],
            [auditT('errorCode'), session.errorCode || '-'],
            [auditT('duration'), `${session.durationMs || 0} ms`],
            [auditT('bytesIn'), bytesLabel(session.bytesIn)],
            [auditT('bytesOut'), bytesLabel(session.bytesOut)]
          ]}
          title={auditT('result')}
        />
      </div>
    </ConsoleCrudModal>
  );
}

function AuditDetailSection({items, title}: {items: [string, string][]; title: string}) {
  return (
    <section className="audit-detail-section">
      <h4>{title}</h4>
      <dl>
        {items.map(([label, value]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{value}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
