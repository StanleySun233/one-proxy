'use client';

import {Share2} from 'lucide-react';

import {AsyncState} from '@/components/async-state';
import {NameTag} from '@/components/common/name-tag';
import {FieldEnumMap, Node, Scope} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {describeNodeName, statusBadgeClassName} from './node-utils';
import {RegistryNodeRow} from './types';

type RegistryNodeTableProps = {
  nodes: Node[];
  nodeRows: RegistryNodeRow[];
  filteredNodes: RegistryNodeRow[];
  nodesByID: Map<string, Node>;
  scopes: Scope[];
  enums: FieldEnumMap | undefined;
  editingNodeID: string;
  nodesPending: boolean;
  healthPending: boolean;
  nodesError: Error | null;
  healthError: Error | null;
  canWrite: boolean;
  globalSuperAdmin: boolean;
  deletePending: boolean;
  t: (key: string) => string;
  nodesT: (key: string, values?: Record<string, string | number>) => string;
  onGrant: (nodeID: string) => void;
  onToggleEdit: (nodeID: string) => void;
  onDelete: (node: RegistryNodeRow) => void;
  onRetryNodes: () => void;
  onRetryHealth: () => void;
};

export function RegistryNodeTable({
  nodes,
  nodeRows,
  filteredNodes,
  nodesByID,
  scopes,
  enums,
  editingNodeID,
  nodesPending,
  healthPending,
  nodesError,
  healthError,
  canWrite,
  globalSuperAdmin,
  deletePending,
  t,
  nodesT,
  onGrant,
  onToggleEdit,
  onDelete,
  onRetryNodes,
  onRetryHealth
}: RegistryNodeTableProps) {
  const scopeById = new Map(scopes.map((scope) => [scope.id, scope]));

  return (
    <>
      {nodesPending || healthPending ? (
        <AsyncState detail={t('common.loading')} title={nodesT('loadingRegistry')} />
      ) : nodesError ? (
        <AsyncState
          actionLabel={t('common.retry')}
          detail={formatControlPlaneError(nodesError)}
          onAction={onRetryNodes}
          title={nodesT('failedRegistry')}
        />
      ) : healthError ? (
        <AsyncState
          actionLabel={t('common.retry')}
          detail={formatControlPlaneError(healthError)}
          onAction={onRetryHealth}
          title={nodesT('failedHealth')}
        />
      ) : nodes.length === 0 ? (
        <AsyncState detail={nodesT('emptyRegistry')} title={t('common.empty')} />
      ) : filteredNodes.length === 0 ? (
        <AsyncState detail={nodesT('noMatchingNodesDetail')} title={nodesT('noMatchingNodes')} />
      ) : (
        <div className="table-card">
          <table className="data-table registry-table">
            <thead>
              <tr>
                <th>{t('common.name')}</th>
                <th>{t('common.status')}</th>
                <th>{t('common.mode')}</th>
                <th>{t('common.scope')}</th>
                <th>{t('common.heartbeat')}</th>
                <th>{t('common.policy')}</th>
                <th>{nodesT('publicEndpoint')}</th>
                <th>{t('common.parent')}</th>
                {canWrite ? <th>{t('common.actions')}</th> : null}
              </tr>
            </thead>
            <tbody>
              {filteredNodes.map((node) => (
                <RegistryNodeTableRow
                  deletePending={deletePending}
                  editing={node.id === editingNodeID}
                  enums={enums}
                  canWrite={canWrite}
                  globalSuperAdmin={globalSuperAdmin}
                  key={node.id}
                  node={node}
                  nodesByID={nodesByID}
                  nodesT={nodesT}
                  onGrant={onGrant}
                  onDelete={onDelete}
                  scopeName={node.scopeKey ? scopeById.get(node.scopeKey)?.name || t('common.unknown') : ''}
                  onToggleEdit={onToggleEdit}
                  t={t}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

function RegistryNodeTableRow({
  node,
  nodesByID,
  enums,
  editing,
  canWrite,
  globalSuperAdmin,
  deletePending,
  t,
  nodesT,
  scopeName,
  onGrant,
  onToggleEdit,
  onDelete
}: {
  node: RegistryNodeRow;
  nodesByID: Map<string, Node>;
  enums: FieldEnumMap | undefined;
  editing: boolean;
  canWrite: boolean;
  globalSuperAdmin: boolean;
  deletePending: boolean;
  t: (key: string) => string;
  nodesT: (key: string, values?: Record<string, string | number>) => string;
  scopeName: string;
  onGrant: (nodeID: string) => void;
  onToggleEdit: (nodeID: string) => void;
  onDelete: (node: RegistryNodeRow) => void;
}) {
  const canManage = globalSuperAdmin || node.permission === 'manage';
  return (
    <tr className={editing ? 'is-active-row' : ''}>
      <td>
        <div className="registry-name-cell">
          <NameTag kind="node">{node.name}</NameTag>
          <span className={`badge ${node.enabled ? 'is-good-soft' : 'is-neutral'}`}>
            {node.enabled ? t('common.enabled') : t('common.disabled')}
          </span>
        </div>
      </td>
      <td>
        <span className={statusBadgeClassName(node.status, enums)}>{node.status}</span>
      </td>
      <td>{node.mode}</td>
      <td>{scopeName ? <NameTag kind="scope">{scopeName}</NameTag> : <span className="muted-text">{t('common.noScope')}</span>}</td>
      <td className="mono">{node.heartbeatAt ? formatISODateTime(node.heartbeatAt) : <span className="muted-text">{t('common.never')}</span>}</td>
      <td>{node.policyRevisionId || <span className="muted-text">{t('common.unassigned')}</span>}</td>
      <td>{node.publicHost ? `${node.publicHost}:${node.publicPort}` : <span className="muted-text">{nodesT('noPublicEndpoint')}</span>}</td>
      <td>{describeNodeName(node.parentNodeId, nodesByID) || <span className="muted-text">{t('common.root')}</span>}</td>
      {canWrite ? (
        <td>
          <div className="registry-actions">
            {canManage ? (
              <button className="secondary-button" onClick={() => onGrant(node.id)} type="button">
                <Share2 size={14} />
                {t('common.grant')}
              </button>
            ) : null}
            <button className="secondary-button" disabled={!canManage} onClick={() => onToggleEdit(node.id)} type="button">
              {editing ? t('common.cancel') : t('common.edit')}
            </button>
            <button className="danger-button" disabled={deletePending || !canManage} onClick={() => onDelete(node)} type="button">
              {t('common.delete')}
            </button>
          </div>
        </td>
      ) : null}
    </tr>
  );
}
