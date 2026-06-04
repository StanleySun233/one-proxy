'use client';

import {useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {Share2} from 'lucide-react';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {ResourceGrantModal} from '@/components/resource-grant-modal';
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
  const [grantLinkID, setGrantLinkID] = useState<string | null>(null);
  const [sourceFilter, setSourceFilter] = useState('');
  const [targetFilter, setTargetFilter] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [trustFilter, setTrustFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [addressFilter, setAddressFilter] = useState('');
  const editingLink = links.find((link) => link.id === editingLinkID) || null;
  const grantLink = links.find((link) => link.id === grantLinkID) || null;
  const transportTypeKeys = Object.keys(enums?.transport_type || {});
  const LINK_TYPE_RELAY = Object.keys(enums?.link_type || {}).find(k => k === 'relay') || 'relay';
  const TRUST_STATE_TRUSTED = Object.keys(enums?.trust_state || {}).find(k => k === 'trusted') || 'trusted';
  const REVERSE_WS_PARENT = transportTypeKeys.find(k => k === 'reverse_ws_parent') || 'reverse_ws_parent';
  const linkTransportReady = (sourceNodeId: string, targetNodeId: string) => transports.some((transport) =>
    transport.nodeId === targetNodeId &&
    transport.parentNodeId === sourceNodeId &&
    transport.transportType === REVERSE_WS_PARENT &&
    transport.status === 'connected'
  );
  const filteredLinks = useMemo(() => {
    return links.filter((link) => {
      const transportReady = linkTransportReady(link.sourceNodeId, link.targetNodeId);
      return (!sourceFilter.trim() || [link.sourceNodeId, describeNodeName(link.sourceNodeId, nodesByID)].some((value) => String(value).toLowerCase().includes(sourceFilter.trim().toLowerCase()))) &&
        (!targetFilter.trim() || [link.targetNodeId, describeNodeName(link.targetNodeId, nodesByID)].some((value) => String(value).toLowerCase().includes(targetFilter.trim().toLowerCase()))) &&
        (!typeFilter.trim() || link.linkType.toLowerCase().includes(typeFilter.trim().toLowerCase())) &&
        (!trustFilter.trim() || link.trustState.toLowerCase().includes(trustFilter.trim().toLowerCase())) &&
        (!statusFilter || (statusFilter === 'connected' ? transportReady : !transportReady));
    });
  }, [links, nodesByID, REVERSE_WS_PARENT, sourceFilter, statusFilter, targetFilter, transports, trustFilter, typeFilter]);
  const filteredTransports = useMemo(() => {
    return transports.filter((transport) =>
      (!targetFilter.trim() || [transport.nodeId, describeNodeName(transport.nodeId, nodesByID)].some((value) => String(value).toLowerCase().includes(targetFilter.trim().toLowerCase()))) &&
      (!sourceFilter.trim() || [transport.parentNodeId, describeNodeName(transport.parentNodeId, nodesByID)].some((value) => String(value).toLowerCase().includes(sourceFilter.trim().toLowerCase()))) &&
      (!typeFilter.trim() || transport.transportType.toLowerCase().includes(typeFilter.trim().toLowerCase())) &&
      (!statusFilter || transport.status === statusFilter) &&
      (!addressFilter.trim() || transport.address.toLowerCase().includes(addressFilter.trim().toLowerCase()))
    );
  }, [addressFilter, nodesByID, sourceFilter, statusFilter, targetFilter, transports, typeFilter]);
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
          <ConsoleFilterItem label={t('common.source')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setSourceFilter(event.target.value)} placeholder={t('common.source')} value={sourceFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.target')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setTargetFilter(event.target.value)} placeholder={t('common.target')} value={targetFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.type')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setTypeFilter(event.target.value)} placeholder={t('common.type')} value={typeFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={nodesT('trustState')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setTrustFilter(event.target.value)} placeholder={nodesT('trustState')} value={trustFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.status')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
              <option value="">{t('common.all')}</option>
              <option value="connected">{nodesT('reverseTunnelsUp')}</option>
              <option value="blocked">{nodesT('reverseTunnelsBlocked')}</option>
            </select>
          </ConsoleFilterItem>
          <ConsoleFilterItem label={nodesT('address')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setAddressFilter(event.target.value)} placeholder={nodesT('address')} value={addressFilter} />
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
              <table className="data-table runtime-transport-table">
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
                    const transportReady = linkTransportReady(link.sourceNodeId, link.targetNodeId);
                    const canManage = nodeConsole.globalSuperAdmin || link.permission === 'manage';
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
                              {canManage ? (
                                <button className="secondary-button" onClick={() => setGrantLinkID(link.id)} type="button">
                                  <Share2 size={14} />
                                  {t('common.grant')}
                                </button>
                              ) : null}
                              <button className="secondary-button" disabled={!canManage} onClick={() => setEditingLinkID(link.id)} type="button">
                                {t('common.edit')}
                              </button>
                              <button
                                className="danger-button"
                                disabled={nodeConsole.deleteNodeLink.isPending || !canManage}
                                onClick={() => {
                                  if (window.confirm(nodesT('deleteLinkConfirm'))) {
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
            <AsyncState detail={sourceFilter || targetFilter || typeFilter || statusFilter || addressFilter ? t('common.noMatching') : nodesT('noRuntimeTransports')} title={t('common.empty')} />
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
                      <td>{describeNodeName(transport.nodeId, nodesByID) || t('common.unknown')}</td>
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

        {grantLink ? (
          <ResourceGrantModal
            onChanged={() => void nodeConsole.linksQuery.refetch()}
            onClose={() => setGrantLinkID(null)}
            open={Boolean(grantLink)}
            resourceId={grantLink.id}
            resourceName={`${describeNodeName(grantLink.sourceNodeId, nodesByID)} -> ${describeNodeName(grantLink.targetNodeId, nodesByID)}`}
            resourceType="node_link"
          />
        ) : null}
      </ConsolePage>
    </AuthGate>
  );
}
