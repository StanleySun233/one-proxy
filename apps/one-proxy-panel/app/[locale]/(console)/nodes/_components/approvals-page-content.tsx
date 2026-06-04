'use client';

import {useMemo, useState} from 'react';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {NameTag} from '@/components/common/name-tag';
import {Node, UnconsumedBootstrapToken} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {useNodeConsole} from './use-node-console';
import {statusBadgeClassName} from './node-utils';

export function NodeApprovalsPageContent() {
  const t = useTranslations();
  const nodesT = useTranslations('nodesConsole');
  const nodeConsole = useNodeConsole();
  const [nameFilter, setNameFilter] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [targetFilter, setTargetFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const pendingNodes = nodeConsole.pendingNodesQuery.data || [];
  const unconsumedTokens = nodeConsole.unconsumedTokensQuery.data || [];
  const scopes = nodeConsole.scopesQuery.data || [];
  const scopeNameById = useMemo(() => new Map(scopes.map((scope) => [scope.id, scope.name])), [scopes]);
  const allItems: Array<{kind: 'pending'; data: Node} | {kind: 'unconsumed'; data: UnconsumedBootstrapToken}> = [
    ...pendingNodes.map((node) => ({kind: 'pending' as const, data: node})),
    ...unconsumedTokens.map((token) => ({kind: 'unconsumed' as const, data: token}))
  ];
  const combinedCount = allItems.length;
  const filteredItems = useMemo(() => {
    return allItems.filter((item) => {
      if (item.kind === 'pending') {
        const node = item.data;
        const scopeName = scopeNameById.get(node.scopeKey) || '';
        return (!nameFilter.trim() || String(node.name || '').toLowerCase().includes(nameFilter.trim().toLowerCase())) &&
          (!typeFilter.trim() || node.mode.toLowerCase().includes(typeFilter.trim().toLowerCase())) &&
          (!targetFilter.trim() || scopeName.toLowerCase().includes(targetFilter.trim().toLowerCase())) &&
          (!statusFilter.trim() || node.status.toLowerCase().includes(statusFilter.trim().toLowerCase()));
      }
      const token = item.data;
      const scopeName = scopeNameById.get(token.scopeKey) || '';
      return (!nameFilter.trim() || String(token.nodeName || '').toLowerCase().includes(nameFilter.trim().toLowerCase())) &&
        (!typeFilter.trim() || nodesT('unconnected').toLowerCase().includes(typeFilter.trim().toLowerCase())) &&
        (!targetFilter.trim() || scopeName.toLowerCase().includes(targetFilter.trim().toLowerCase())) &&
        (!statusFilter.trim() || t('common.unused').toLowerCase().includes(statusFilter.trim().toLowerCase()));
    });
  }, [allItems, nameFilter, nodesT, scopeNameById, statusFilter, t, targetFilter, typeFilter]);

  return (
    <AuthGate>
      <ConsolePage title={t('shell.nodeApprovals')}>
        <ConsoleFilterBar title={t('common.filter')}>
          <ConsoleFilterItem label={t('common.name')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setNameFilter(event.target.value)} placeholder={t('common.name')} value={nameFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.type')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setTypeFilter(event.target.value)} placeholder={t('common.type')} value={typeFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.target')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setTargetFilter(event.target.value)} placeholder={t('common.target')} value={targetFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.status')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setStatusFilter(event.target.value)} placeholder={t('common.status')} value={statusFilter} />
          </ConsoleFilterItem>
        </ConsoleFilterBar>

        <ConsoleList count={filteredItems.length} title={nodesT('pendingEnrollments')}>
          {nodeConsole.pendingNodesQuery.isPending || nodeConsole.unconsumedTokensQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={nodesT('loadingPending')} />
          ) : nodeConsole.pendingNodesQuery.error ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(nodeConsole.pendingNodesQuery.error)}
              onAction={() => void nodeConsole.pendingNodesQuery.refetch()}
              title={nodesT('failedPending')}
            />
          ) : combinedCount === 0 ? (
            <AsyncState detail={nodesT('emptyPending')} title={t('common.empty')} />
          ) : filteredItems.length === 0 ? (
            <AsyncState detail={t('common.noMatching')} title={t('common.empty')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('common.name')}</th>
                    <th>{t('common.type')}</th>
                    <th>{t('common.target')}</th>
                    <th>{t('common.status')}</th>
                    <th>{t('common.createdExpires')}</th>
                    <th>{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredItems.map((item) => {
                    if (item.kind === 'pending') {
                      const node = item.data;
                      const scopeName = scopeNameById.get(node.scopeKey) || '';
                      return (
                        <tr key={node.id}>
                          <td>{node.name ? <NameTag kind="node">{node.name}</NameTag> : <span className="muted-text">{t('common.notSpecified')}</span>}</td>
                          <td>{node.mode}</td>
                          <td>{scopeName ? <NameTag kind="scope">{scopeName}</NameTag> : <span className="muted-text">{t('common.unknown')}</span>}</td>
                          <td>
                            <span className={statusBadgeClassName(node.status)}>{node.status}</span>
                          </td>
                          <td className="muted-text">—</td>
                          <td>
                            <div className="registry-actions">
                              <button
                                className="secondary-button"
                                disabled={nodeConsole.approve.isPending}
                                onClick={() => nodeConsole.approve.mutate(node.id)}
                                type="button"
                              >
                                {t('common.approve')}
                              </button>
                              <button
                                className="danger-button"
                                disabled={nodeConsole.deleteNode.isPending}
                                onClick={() => {
                                  if (!window.confirm(nodesT('deletePendingConfirm', {name: node.name || t('common.unknown')}))) {
                                    return;
                                  }
                                  nodeConsole.deleteNode.mutate(node.id);
                                }}
                                type="button"
                              >
                                {t('common.delete')}
                              </button>
                              <button
                                className="danger-button"
                                disabled={nodeConsole.rejectNode.isPending}
                                onClick={() => {
                                  if (!window.confirm(nodesT('rejectEnrollmentConfirm', {name: node.name || t('common.unknown')}))) {
                                    return;
                                  }
                                  nodeConsole.rejectNode.mutate({nodeId: node.id});
                                }}
                                type="button"
                              >
                                {t('common.reject')}
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    }
                    const token = item.data;
                    const scopeName = scopeNameById.get(token.scopeKey) || '';
                    return (
                      <tr key={token.id}>
                        <td>{token.nodeName || <span className="muted-text">{t('common.notSpecified')}</span>}</td>
                        <td><span className="badge is-neutral">{nodesT('unconnected')}</span></td>
                        <td>{scopeName ? <NameTag kind="scope">{scopeName}</NameTag> : <span className="muted-text">{t('common.newNode')}</span>}</td>
                        <td>
                          <span className="badge is-neutral">{t('common.unused')}</span>
                        </td>
                        <td>
                          <span className="muted-text">{formatISODateTime(token.createdAt)}</span>
                          <br />
                          <span className="muted-text">{t('common.expires')} {formatISODateTime(token.expiresAt)}</span>
                        </td>
                        <td>
                          <button
                            className="danger-button"
                            disabled={nodeConsole.deleteBootstrapToken.isPending}
                            onClick={() => {
                              if (!window.confirm(nodesT('deleteBootstrapTokenConfirm', {name: token.nodeName || t('common.unknown')}))) {
                                return;
                              }
                              nodeConsole.deleteBootstrapToken.mutate(token.id);
                            }}
                            type="button"
                          >
                            {t('common.delete')}
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </ConsoleList>
      </ConsolePage>
    </AuthGate>
  );
}
