'use client';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {NameTag} from '@/components/common/name-tag';
import {Node, UnconsumedBootstrapToken} from '@/lib/types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';
import {useTranslations} from 'next-intl';

import {useNodeConsole} from './use-node-console';
import {statusBadgeClassName} from './node-utils';

export function NodeApprovalsPageContent() {
  const t = useTranslations();
  const nodesT = useTranslations('nodesConsole');
  const nodeConsole = useNodeConsole();
  const pendingNodes = nodeConsole.pendingNodesQuery.data || [];
  const unconsumedTokens = nodeConsole.unconsumedTokensQuery.data || [];
  const allItems: Array<{kind: 'pending'; data: Node} | {kind: 'unconsumed'; data: UnconsumedBootstrapToken}> = [
    ...pendingNodes.map((node) => ({kind: 'pending' as const, data: node})),
    ...unconsumedTokens.map((token) => ({kind: 'unconsumed' as const, data: token}))
  ];
  const combinedCount = allItems.length;

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">{nodesT('approvals')}</p>
              <h3>{nodesT('pendingEnrollments')}</h3>
            </div>
            <span className="badge">{combinedCount}</span>
          </div>
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
                  {allItems.map((item) => {
                    if (item.kind === 'pending') {
                      const node = item.data;
                      return (
                        <tr key={node.id}>
                          <td>{node.name ? <NameTag kind="node">{node.name}</NameTag> : <span className="muted-text">{t('common.notSpecified')}</span>}</td>
                          <td>{node.mode}</td>
                          <td className="mono">{node.id.substring(0, 12)}</td>
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
                                  if (!window.confirm(nodesT('deletePendingConfirm', {name: node.name || node.id}))) {
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
                                  if (!window.confirm(nodesT('rejectEnrollmentConfirm', {name: node.name || node.id}))) {
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
                    return (
                      <tr key={token.id}>
                        <td>{token.nodeName || <span className="muted-text">{t('common.notSpecified')}</span>}</td>
                        <td><span className="badge is-neutral">{nodesT('unconnected')}</span></td>
                        <td className="mono">{token.targetId || <span className="muted-text">{t('common.newNode')}</span>}</td>
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
                              if (!window.confirm(nodesT('deleteBootstrapTokenConfirm', {name: token.nodeName || token.id}))) {
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
        </section>
      </div>
    </AuthGate>
  );
}
