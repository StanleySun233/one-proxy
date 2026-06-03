'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {fetchEnums} from '@/lib/api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {useNodeConsole} from '../../../nodes/_components/use-node-console';
import {describeNodeName, transportBadgeClassName} from '../../../nodes/_components/node-utils';
import {CreateNodeLinkForm} from './create-node-link-form';

export function NodeTopologyPageContent() {
  const t = useTranslations();
  const nodesT = useTranslations('nodesConsole');
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const links = nodeConsole.linksQuery.data || [];
  const transports = nodeConsole.transportsQuery.data || [];
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  const [createOpen, setCreateOpen] = useState(false);
  const [editingLinkID, setEditingLinkID] = useState<string | null>(null);
  const [search, setSearch] = useState('');
  const editingLink = links.find((link) => link.id === editingLinkID) || null;
  const transportTypeKeys = Object.keys(enums?.transport_type || {});
  const LINK_TYPE_RELAY = Object.keys(enums?.link_type || {}).find(k => k === 'relay') || 'relay';
  const TRUST_STATE_TRUSTED = Object.keys(enums?.trust_state || {}).find(k => k === 'trusted') || 'trusted';
  const REVERSE_WS_PARENT = transportTypeKeys.find(k => k === 'reverse_ws_parent') || 'reverse_ws_parent';
  const filteredLinks = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    if (!keyword) {
      return links;
    }
    return links.filter((link) =>
      [link.id, link.sourceNodeId, link.targetNodeId, link.linkType, link.trustState, describeNodeName(link.sourceNodeId, nodesByID), describeNodeName(link.targetNodeId, nodesByID)]
        .some((value) => String(value || '').toLowerCase().includes(keyword))
    );
  }, [links, nodesByID, search]);
  const filteredTransports = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    if (!keyword) {
      return transports;
    }
    return transports.filter((transport) =>
      [transport.id, transport.nodeId, transport.transportType, transport.direction, transport.status, transport.address, transport.parentNodeId, describeNodeName(transport.nodeId, nodesByID)]
        .some((value) => String(value || '').toLowerCase().includes(keyword))
    );
  }, [nodesByID, search, transports]);
  const modalOpen = createOpen || Boolean(editingLink);
  const closeModal = () => {
    setCreateOpen(false);
    setEditingLinkID(null);
  };

  return (
    <AuthGate>
      <ConsolePage
        actions={nodeConsole.canWrite ? (
          <button className="primary-button" onClick={() => setCreateOpen(true)} type="button">
            {nodesT('addLink')}
          </button>
        ) : null}
        title={t('shell.nodeTopology')}
      >
        <ConsoleFilterBar title={t('common.filter')}>
          <ConsoleFilterItem label={`${t('common.source')} / ${t('common.target')} / ${t('common.type')} / ${nodesT('trustState')}`} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setSearch(event.target.value)} placeholder={t('common.searchPlaceholder')} value={search} />
          </ConsoleFilterItem>
        </ConsoleFilterBar>

        <ConsoleList count={filteredLinks.length} title={nodesT('topologyTitle')}>
          {nodeConsole.linksQuery.isPending || nodeConsole.nodesQuery.isPending ? (
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
          ) : links.length === 0 ? (
            <AsyncState detail={nodesT('emptyTopology')} title={t('common.empty')} />
          ) : filteredLinks.length === 0 ? (
            <AsyncState detail={t('common.noMatching')} title={t('common.empty')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('common.source')}</th>
                    <th>{t('common.target')}</th>
                    <th>{t('common.type')}</th>
                    <th>{nodesT('trustState')}</th>
                    <th>{t('common.status')}</th>
                    {nodeConsole.canWrite ? <th>{t('common.actions')}</th> : null}
                  </tr>
                </thead>
                <tbody>
                  {filteredLinks.map((link) => {
                    const transportReady = transports.some((transport) =>
                      transport.nodeId === link.targetNodeId &&
                      transport.parentNodeId === link.sourceNodeId &&
                      transport.transportType === REVERSE_WS_PARENT
                    );
                    return (
                      <tr key={link.id}>
                        <td>{describeNodeName(link.sourceNodeId, nodesByID)}</td>
                        <td>{describeNodeName(link.targetNodeId, nodesByID)}</td>
                        <td className="mono">{link.linkType}</td>
                        <td><span className="badge">{link.trustState}</span></td>
                        <td><span className={`badge ${transportReady ? 'is-good' : 'is-warn'}`}>{transportReady ? nodesT('reverseTunnelsUp') : nodesT('reverseTunnelsBlocked')}</span></td>
                        {nodeConsole.canWrite ? (
                          <td>
                            <div className="chain-list-actions">
                              <button className="secondary-button" onClick={() => setEditingLinkID(link.id)} type="button">
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
                          </td>
                        ) : null}
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </ConsoleList>

        <ConsoleList count={filteredTransports.length} title={nodesT('runtimeTransports')}>
          {nodeConsole.transportsQuery.isPending || nodeConsole.nodesQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={nodesT('loadingTransport')} />
          ) : nodeConsole.transportsQuery.error ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(nodeConsole.transportsQuery.error)}
              onAction={() => void nodeConsole.transportsQuery.refetch()}
              title={nodesT('failedTransport')}
            />
          ) : filteredTransports.length === 0 ? (
            <AsyncState detail={search ? t('common.noMatching') : nodesT('noRuntimeTransports')} title={t('common.empty')} />
          ) : (
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
                  {filteredTransports.map((transport) => (
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
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </ConsoleList>

        {nodeConsole.canWrite ? (
          <ConsoleCrudModal
            onClose={closeModal}
            open={modalOpen}
            title={editingLink ? nodesT('saveLink') : nodesT('addLink')}
          >
            <CreateNodeLinkForm
              nodes={nodes}
              pending={nodeConsole.createNodeLink.isPending || nodeConsole.updateNodeLink.isPending}
              editingLink={editingLink}
              onCancelEdit={closeModal}
              onSubmit={(payload) => {
                if (editingLink) {
                  nodeConsole.updateNodeLink.mutate(
                    {linkID: editingLink.id, ...payload},
                    {onSuccess: closeModal}
                  );
                  return;
                }
                nodeConsole.createNodeLink.mutate(payload, {onSuccess: closeModal});
              }}
              defaultLinkType={LINK_TYPE_RELAY}
              defaultTrustState={TRUST_STATE_TRUSTED}
            />
          </ConsoleCrudModal>
        ) : null}
      </ConsolePage>
    </AuthGate>
  );
}
