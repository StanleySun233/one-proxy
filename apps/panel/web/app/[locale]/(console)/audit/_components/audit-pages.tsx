'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {useAuth} from '@/components/auth-provider';
import {getAuditBusinessEvents, getAuditDashboard, getAuditNetworkSessions} from '@/lib/api';
import type {AuditDecisionCount, AuditEvent, AuditGroup, NetworkAuditSummary, NetworkSession} from '@/lib/types/audit';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

const LIMIT = 100;

function defaultFrom() {
  const date = new Date();
  date.setDate(date.getDate() - 7);
  return date.toISOString().slice(0, 10);
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
          <article className="metric-card panel-card warm-card">
            <span>{auditT('deniedTraffic')}</span>
            <strong>{dashboardQuery.isPending ? '-' : decisionCounts.find((item) => item.decision === 'deny')?.count ?? 0}</strong>
          </article>
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
          <AuditGroupPanel groups={nodeTraffic} title={auditT('nodeTraffic')} />
          <AuditGroupPanel groups={targetTraffic} title={auditT('topTargets')} />
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
  const {accessToken, activeTenantId, from, rangeQuery, setFrom, setTo, to} = useAuditRange();
  const [networkActor, setNetworkActor] = useState('');
  const [networkNode, setNetworkNode] = useState('');
  const [networkTarget, setNetworkTarget] = useState('');
  const [networkDecision, setNetworkDecision] = useState('');

  const networkQuery = useQuery({
    queryKey: ['audit-network', accessToken, activeTenantId, rangeQuery.from, rangeQuery.to, networkActor, networkNode, networkTarget, networkDecision],
    queryFn: () => getAuditNetworkSessions(accessToken, activeTenantId, {
      ...rangeQuery,
      actorId: networkActor.trim(),
      nodeId: networkNode.trim(),
      targetHost: networkTarget.trim(),
      decision: networkDecision,
      limit: LIMIT
    }),
    enabled: !!accessToken
  });

  const network = networkQuery.data;

  return (
    <AuthGate>
      <ConsolePage title={auditT('networkTitle')}>
        <NetworkAuditList
          actor={networkActor}
          decision={networkDecision}
          from={from}
          isError={networkQuery.isError}
          isPending={networkQuery.isPending}
          networkNode={networkNode}
          onActorChange={setNetworkActor}
          onDecisionChange={setNetworkDecision}
          onFromChange={setFrom}
          onNodeChange={setNetworkNode}
          onRefetch={() => void networkQuery.refetch()}
          onTargetChange={setNetworkTarget}
          onToChange={setTo}
          queryError={networkQuery.error}
          sessions={network?.items || []}
          target={networkTarget}
          to={to}
        />
      </ConsolePage>
    </AuthGate>
  );
}

function AuditGroupPanel({groups, title}: {groups: AuditGroup[]; title: string}) {
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
        ) : groups.slice(0, 5).map((item) => (
          <div className="queue-item" key={item.id || item.name}>
            <strong>{groupName(item)}</strong>
            <span className="muted-text">
              {groupCount(item)} · {bytesLabel((item.bytesIn || 0) + (item.bytesOut || 0))}
            </span>
          </div>
        ))}
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
  decision: string;
  from: string;
  isError: boolean;
  isPending: boolean;
  networkNode: string;
  onActorChange: (value: string) => void;
  onDecisionChange: (value: string) => void;
  onFromChange: (value: string) => void;
  onNodeChange: (value: string) => void;
  onRefetch: () => void;
  onTargetChange: (value: string) => void;
  onToChange: (value: string) => void;
  queryError: unknown;
  sessions: NetworkSession[];
  target: string;
  to: string;
}) {
  const t = useTranslations();
  const auditT = useTranslations('pages.audit');

  return (
    <>
      <ConsoleFilterBar title={auditT('networkFilters')}>
        <ConsoleFilterItem label={auditT('from')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onFromChange(event.target.value)} type="date" value={props.from} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('to')} match={t('common.equals')}>
          <input className="field-input" onChange={(event) => props.onToChange(event.target.value)} type="date" value={props.to} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('actor')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => props.onActorChange(event.target.value)} placeholder={auditT('actor')} value={props.actor} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('node')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => props.onNodeChange(event.target.value)} placeholder={auditT('node')} value={props.networkNode} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('targetHost')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => props.onTargetChange(event.target.value)} placeholder={auditT('targetHost')} value={props.target} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={auditT('decision')} match={t('common.equals')}>
          <select className="field-select" onChange={(event) => props.onDecisionChange(event.target.value)} value={props.decision}>
            <option value="">{t('common.all')}</option>
            <option value="allow">{auditT('allow')}</option>
            <option value="deny">{auditT('deny')}</option>
          </select>
        </ConsoleFilterItem>
      </ConsoleFilterBar>

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
                  <th>{auditT('actor')}</th>
                  <th>{auditT('node')}</th>
                  <th>{auditT('target')}</th>
                  <th>{auditT('decision')}</th>
                  <th>{auditT('traffic')}</th>
                  <th>{auditT('duration')}</th>
                </tr>
              </thead>
              <tbody>
                {props.sessions.map((session) => (
                  <tr key={session.id}>
                    <td>{formatISODateTime(session.endedAt || session.startedAt, session.endedAt || session.startedAt)}</td>
                    <td>{session.actorId || session.tokenId || '-'}</td>
                    <td>{session.exitNodeId || session.entryNodeId || '-'}</td>
                    <td>{session.targetHost}:{session.targetPort}</td>
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
    </>
  );
}
