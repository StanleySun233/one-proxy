'use client';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {Node, UnconsumedBootstrapToken} from '@/lib/control-plane-types';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {useNodeConsole} from './use-node-console';
import {statusBadgeClassName} from './node-utils';

export function NodeApprovalsPageContent() {
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
              <p className="section-kicker">Approvals</p>
              <h3>Pending node enrollments</h3>
              <p className="section-copy">Review and approve pending node enrollment requests created via bootstrap tokens.</p>
            </div>
            <span className="badge">{combinedCount}</span>
          </div>
          {nodeConsole.pendingNodesQuery.isPending || nodeConsole.unconsumedTokensQuery.isPending ? (
            <AsyncState detail="Loading" title="Loading pending enrollments" />
          ) : nodeConsole.pendingNodesQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.pendingNodesQuery.error)}
              onAction={() => void nodeConsole.pendingNodesQuery.refetch()}
              title="Failed to load pending enrollments"
            />
          ) : combinedCount === 0 ? (
            <AsyncState detail="No pending enrollment requests. Create a bootstrap token to start." title="Empty" />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Type</th>
                    <th>Target</th>
                    <th>Status</th>
                    <th>Created / Expires</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {allItems.map((item) => {
                    if (item.kind === 'pending') {
                      const node = item.data;
                      return (
                        <tr key={node.id}>
                          <td>{node.name || <span className="muted-text">not specified</span>}</td>
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
                                Approve
                              </button>
                              <button
                                className="danger-button"
                                disabled={nodeConsole.rejectNode.isPending}
                                onClick={() => {
                                  if (!window.confirm(`Reject enrollment for ${node.name || 'this node'}?`)) {
                                    return;
                                  }
                                  nodeConsole.rejectNode.mutate({nodeId: node.id});
                                }}
                                type="button"
                              >
                                Reject
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    }
                    const token = item.data;
                    return (
                      <tr key={token.id}>
                        <td>{token.nodeName || <span className="muted-text">not specified</span>}</td>
                        <td><span className="badge is-neutral">unconnected</span></td>
                        <td className="mono">{token.targetId || <span className="muted-text">new node</span>}</td>
                        <td>
                          <span className="badge is-neutral">unused</span>
                        </td>
                        <td>
                          <span className="muted-text">{formatISODateTime(token.createdAt)}</span>
                          <br />
                          <span className="muted-text">expires {formatISODateTime(token.expiresAt)}</span>
                        </td>
                        <td>
                          <span className="muted-text">awaiting connection</span>
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
