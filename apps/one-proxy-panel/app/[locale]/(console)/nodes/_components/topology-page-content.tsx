'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {fetchEnums} from '@/lib/api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {useNodeConsole} from './use-node-console';
import {describeNodeName, transportBadgeClassName} from './node-utils';
import {CreateNodeLinkForm} from './create-node-link-form';
import {NodeLinkCard} from './node-link-card';

export function NodeTopologyPageContent() {
  const t = useTranslations();
  const nodesT = useTranslations('nodesConsole');
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const links = nodeConsole.linksQuery.data || [];
  const transports = nodeConsole.transportsQuery.data || [];
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  const [editingLinkID, setEditingLinkID] = useState<string | null>(null);
  const editingLink = links.find((link) => link.id === editingLinkID) || null;
  // Derive enum value references from the enums object
  const transportTypeKeys = Object.keys(enums?.transport_type || {});
  const PUBLIC_HTTP = transportTypeKeys.find(k => k === 'public_http') || 'public_http';
  const PUBLIC_HTTPS = transportTypeKeys.find(k => k === 'public_https') || 'public_https';
  const REVERSE_WS_PARENT = transportTypeKeys.find(k => k === 'reverse_ws_parent') || 'reverse_ws_parent';
  const CONNECTED = Object.keys(enums?.transport_status || {}).find(k => k === 'connected') || 'connected';
  const LINK_TYPE_RELAY = Object.keys(enums?.link_type || {}).find(k => k === 'relay') || 'relay';
  const TRUST_STATE_TRUSTED = Object.keys(enums?.trust_state || {}).find(k => k === 'trusted') || 'trusted';
  const transportSummary = useMemo(
    () => ({
      publicEndpoints: transports.filter((item) => item.transportType === PUBLIC_HTTP || item.transportType === PUBLIC_HTTPS).length,
      reverseConnected: transports.filter((item) => item.transportType === REVERSE_WS_PARENT && item.status === CONNECTED).length,
      reverseBlocked: transports.filter((item) => item.transportType === REVERSE_WS_PARENT && item.status !== CONNECTED).length
    }),
    [transports]
  );

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="metrics-grid">
          <article className="metric-card panel-card">
            <span className="metric-label">{nodesT('publicTransports')}</span>
            <strong>{transportSummary.publicEndpoints}</strong>
          </article>
          <article className="metric-card panel-card soft-card">
            <span className="metric-label">{nodesT('reverseTunnelsUp')}</span>
            <strong>{transportSummary.reverseConnected}</strong>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">{nodesT('reverseTunnelsBlocked')}</span>
            <strong>{transportSummary.reverseBlocked}</strong>
          </article>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">{nodesT('topology')}</p>
              <h3>{nodesT('topologyTitle')}</h3>
            </div>
            <span className="badge">{links.length}</span>
          </div>
          <CreateNodeLinkForm
            nodes={nodes}
            pending={nodeConsole.createNodeLink.isPending || nodeConsole.updateNodeLink.isPending}
            editingLink={editingLink}
            onCancelEdit={() => setEditingLinkID(null)}
            onSubmit={(payload) => {
              if (editingLink) {
                nodeConsole.updateNodeLink.mutate(
                  {linkID: editingLink.id, ...payload},
                  {onSuccess: () => setEditingLinkID(null)}
                );
                return;
              }
              nodeConsole.createNodeLink.mutate(payload);
            }}
            defaultLinkType={LINK_TYPE_RELAY}
            defaultTrustState={TRUST_STATE_TRUSTED}
          />
          {nodeConsole.linksQuery.isPending || nodeConsole.nodesQuery.isPending || nodeConsole.transportsQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={nodesT('loadingTopology')} />
          ) : nodeConsole.nodesQuery.error ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(nodeConsole.nodesQuery.error)}
              onAction={() => void nodeConsole.nodesQuery.refetch()}
              title={nodesT('failedRegistry')}
            />
          ) : nodeConsole.linksQuery.error ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(nodeConsole.linksQuery.error)}
              onAction={() => void nodeConsole.linksQuery.refetch()}
              title={nodesT('failedTopology')}
            />
          ) : nodeConsole.transportsQuery.error ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(nodeConsole.transportsQuery.error)}
              onAction={() => void nodeConsole.transportsQuery.refetch()}
              title={nodesT('failedTransport')}
            />
          ) : links.length === 0 ? (
            <AsyncState detail={nodesT('emptyTopology')} title={t('common.empty')} />
          ) : (
            <div className="topology-stack">
              <div className="nodes-link-grid">
                {links.map((link) => (
                  <div key={link.id} className="topology-link-item">
                    <NodeLinkCard link={link} nodesByID={nodesByID} transports={transports} reverseWsType={REVERSE_WS_PARENT} />
                    <div className="inline-actions">
                      <button className="ghost-button" onClick={() => setEditingLinkID(link.id)} type="button">
                        {t('common.edit')}
                      </button>
                      <button
                        className="danger-button"
                        disabled={nodeConsole.deleteNodeLink.isPending}
                        onClick={() => {
                          if (window.confirm(nodesT('deleteLinkConfirm', {id: link.id}))) {
                            nodeConsole.deleteNodeLink.mutate(link.id);
                          }
                        }}
                        type="button"
                      >
                        {t('common.delete')}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
              <div className="table-card">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>{t('common.name')}</th>
                      <th>{t('common.type')}</th>
                      <th>{nodesT('direction')}</th>
                      <th>{t('common.status')}</th>
                      <th>{nodesT('address')}</th>
                      <th>{t('common.parent')}</th>
                      <th>{t('common.heartbeat')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {transports.length === 0 ? (
                      <tr>
                        <td className="muted-text" colSpan={7}>
                          {nodesT('noRuntimeTransports')}
                        </td>
                      </tr>
                    ) : (
                      transports.map((transport) => (
                        <tr key={transport.id}>
                          <td>{describeNodeName(transport.nodeId, nodesByID) || transport.nodeId}</td>
                          <td className="mono">{transport.transportType}</td>
                          <td>{transport.direction}</td>
                          <td>
                            <span className={transportBadgeClassName(transport.status, enums)}>{transport.status}</span>
                          </td>
                          <td className="mono">{transport.address}</td>
                          <td>{describeNodeName(transport.parentNodeId, nodesByID) || <span className="muted-text">{t('common.root')}</span>}</td>
                          <td className="mono">{transport.lastHeartbeatAt ? formatISODateTime(transport.lastHeartbeatAt) : <span className="muted-text">{t('common.never')}</span>}</td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}
