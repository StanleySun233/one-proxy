'use client';

import {ReactNode, useEffect, useState} from 'react';

import {AuthGate} from '@/components/auth-gate';
import {AsyncState} from '@/components/async-state';
import {Node, NodeLink} from '@/lib/control-plane-types';
import {formatControlPlaneError} from '@/lib/presentation';

import {BootstrapTokenTab} from './bootstrap-token-tab';
import {ManualNodeTab} from './manual-node-tab';
import {QuickConnectTab} from './quick-connect-tab';
import {useNodeConsole} from './use-node-console';

export function NodeConnectPageContent() {
  const nodeConsole = useNodeConsole();

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card nodes-single-panel">
          <div>
            <p className="section-kicker">Node entry</p>
            <h3>Quick connect</h3>
            <p className="section-copy">Connect a running node directly with ip:port and join password.</p>
          </div>
          <QuickConnectTab
            form={nodeConsole.quickConnectForm}
            submitting={nodeConsole.quickConnect.isPending}
            onSubmit={() => {
              const values = nodeConsole.quickConnectForm.getValues();
              nodeConsole.quickConnect.mutate({
                address: values.address.trim(),
                password: values.password,
                newPassword: values.newPassword,
                name: values.name.trim(),
                mode: values.mode,
                scopeKey: values.scopeKey.trim(),
                parentNodeId: values.parentNodeId.trim(),
                publicHost: values.publicHost.trim(),
                publicPort: values.publicPort ? Number(values.publicPort) : 0,
                controlPlaneUrl: values.controlPlaneUrl.trim()
              });
            }}
          />
        </section>
      </div>
    </AuthGate>
  );
}

export function NodeManualPageContent() {
  const nodeConsole = useNodeConsole();

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card nodes-single-panel">
          <div>
            <p className="section-kicker">Node record</p>
            <h3>Manual record</h3>
            <p className="section-copy">Create node metadata first, then bind the real runtime later.</p>
          </div>
          <ManualNodeTab
            form={nodeConsole.nodeForm}
            submitting={nodeConsole.createNode.isPending}
            onSubmit={() => {
              const values = nodeConsole.nodeForm.getValues();
              nodeConsole.createNode.mutate({
                name: values.name.trim(),
                mode: values.mode,
                scopeKey: values.scopeKey.trim(),
                parentNodeId: values.parentNodeId.trim(),
                publicHost: values.publicHost.trim(),
                publicPort: values.publicPort ? Number(values.publicPort) : 0
              });
            }}
          />
        </section>
      </div>
    </AuthGate>
  );
}

export function NodeBootstrapPageContent() {
  const nodeConsole = useNodeConsole();

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card nodes-single-panel">
          <div>
            <p className="section-kicker">Bootstrap</p>
            <h3>Bootstrap token</h3>
            <p className="section-copy">Issue a one-time token for remote self-enroll flows.</p>
          </div>
          <BootstrapTokenTab
            form={nodeConsole.bootstrapForm}
            latestToken={nodeConsole.latestToken}
            submitting={nodeConsole.bootstrap.isPending}
            onSubmit={() => {
              const values = nodeConsole.bootstrapForm.getValues();
              nodeConsole.bootstrap.mutate(values.targetId.trim());
            }}
          />
        </section>
      </div>
    </AuthGate>
  );
}

export function NodeApprovalsPageContent() {
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const pendingNodes = nodes.filter((node) => node.status === 'pending');

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Approvals</p>
              <h3>Pending node approvals</h3>
              <p className="section-copy">Review nodes waiting for trust material and control-plane binding.</p>
            </div>
            <span className="badge">{pendingNodes.length}</span>
          </div>
          {nodeConsole.nodesQuery.isPending ? (
            <AsyncState detail="Loading" title="Loading pending approvals" />
          ) : nodeConsole.nodesQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.nodesQuery.error)}
              onAction={() => void nodeConsole.nodesQuery.refetch()}
              title="Failed to load pending approvals"
            />
          ) : pendingNodes.length === 0 ? (
            <AsyncState detail="No nodes are waiting for approval." title="Empty" />
          ) : (
            <div className="nodes-list-grid">
              {pendingNodes.map((node) => (
                <NodeCard action={
                  <button
                    className="secondary-button"
                    disabled={nodeConsole.approve.isPending}
                    onClick={() => nodeConsole.approve.mutate(node.id)}
                    type="button"
                  >
                    Approve
                  </button>
                } key={node.id} node={node} />
              ))}
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}

export function NodeRegistryPageContent() {
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [modeFilter, setModeFilter] = useState('all');
  const [editingNodeID, setEditingNodeID] = useState('');
  const [formState, setFormState] = useState({
    name: '',
    mode: 'relay',
    scopeKey: '',
    parentNodeId: '',
    publicHost: '',
    publicPort: '',
    enabled: true,
    status: 'healthy'
  });
  const normalizedQuery = query.trim().toLowerCase();
  const filteredNodes = nodes.filter((node) => {
    const matchesQuery =
      normalizedQuery.length === 0 ||
      node.name.toLowerCase().includes(normalizedQuery) ||
      node.id.toLowerCase().includes(normalizedQuery) ||
      node.scopeKey.toLowerCase().includes(normalizedQuery) ||
      node.parentNodeId.toLowerCase().includes(normalizedQuery) ||
      node.publicHost?.toLowerCase().includes(normalizedQuery);
    const matchesStatus = statusFilter === 'all' || node.status === statusFilter;
    const matchesMode = modeFilter === 'all' || node.mode === modeFilter;

    return matchesQuery && matchesStatus && matchesMode;
  });
  const availableStatuses = Array.from(new Set(nodes.map((node) => node.status))).sort();
  const availableModes = Array.from(new Set(nodes.map((node) => node.mode))).sort();
  const editingNode = nodes.find((node) => node.id === editingNodeID) || null;

  useEffect(() => {
    if (!editingNode) {
      return;
    }
    setFormState({
      name: editingNode.name,
      mode: editingNode.mode,
      scopeKey: editingNode.scopeKey || '',
      parentNodeId: editingNode.parentNodeId || '',
      publicHost: editingNode.publicHost || '',
      publicPort: editingNode.publicPort ? String(editingNode.publicPort) : '',
      enabled: editingNode.enabled,
      status: editingNode.status
    });
  }, [editingNode]);

  useEffect(() => {
    if (editingNodeID && !nodes.some((node) => node.id === editingNodeID)) {
      setEditingNodeID('');
    }
  }, [editingNodeID, nodes]);

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Registry</p>
              <h3>Node registry</h3>
              <p className="section-copy">Query registered nodes, inspect runtime state, and maintain records with read, update, and delete actions.</p>
            </div>
            <div className="inline-cluster">
              <span className="badge">{filteredNodes.length} shown</span>
              <span className="badge">{nodes.length} total</span>
            </div>
          </div>
          {nodeConsole.nodesQuery.isPending ? (
            <AsyncState detail="Loading" title="Loading node registry" />
          ) : nodeConsole.nodesQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.nodesQuery.error)}
              onAction={() => void nodeConsole.nodesQuery.refetch()}
              title="Failed to load node registry"
            />
          ) : nodes.length === 0 ? (
            <AsyncState detail="Create or connect the first node to populate the registry." title="Empty" />
          ) : (
            <div className="registry-stack">
              <div className="registry-toolbar">
                <label className="field-stack registry-filter">
                  <span>Search</span>
                  <input
                    className="field-input"
                    onChange={(event) => setQuery(event.target.value)}
                    placeholder="Search by name, id, scope, parent, or host"
                    type="search"
                    value={query}
                  />
                </label>
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Status</span>
                  <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
                    <option value="all">All statuses</option>
                    {availableStatuses.map((status) => (
                      <option key={status} value={status}>
                        {status}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="field-stack registry-filter registry-filter-short">
                  <span>Mode</span>
                  <select className="field-select" onChange={(event) => setModeFilter(event.target.value)} value={modeFilter}>
                    <option value="all">All modes</option>
                    {availableModes.map((mode) => (
                      <option key={mode} value={mode}>
                        {mode}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
              {filteredNodes.length === 0 ? (
                <AsyncState detail="Adjust the current query or filters to see matching nodes." title="No matching nodes" />
              ) : (
                <div className="table-card">
                  <table className="data-table registry-table">
                    <thead>
                      <tr>
                        <th>Name</th>
                        <th>Status</th>
                        <th>Mode</th>
                        <th>Scope</th>
                        <th>Public endpoint</th>
                        <th>Parent</th>
                        <th>ID</th>
                        <th>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredNodes.map((node) => {
                        const active = node.id === editingNodeID;

                        return (
                          <tr className={active ? 'is-active-row' : ''} key={node.id}>
                            <td>
                              <div className="registry-name-cell">
                                <strong>{node.name}</strong>
                                <span className={`badge ${node.enabled ? 'is-good-soft' : 'is-neutral'}`}>
                                  {node.enabled ? 'enabled' : 'disabled'}
                                </span>
                              </div>
                            </td>
                            <td>
                              <span className={statusBadgeClassName(node.status)}>{node.status}</span>
                            </td>
                            <td>{node.mode}</td>
                            <td>{node.scopeKey || <span className="muted-text">no-scope</span>}</td>
                            <td>{node.publicHost ? `${node.publicHost}:${node.publicPort}` : <span className="muted-text">No public endpoint</span>}</td>
                            <td>{node.parentNodeId || <span className="muted-text">root</span>}</td>
                            <td className="mono registry-id-cell">{node.id}</td>
                            <td>
                              <div className="registry-actions">
                                <button
                                  className="secondary-button"
                                  onClick={() => setEditingNodeID(active ? '' : node.id)}
                                  type="button"
                                >
                                  {active ? 'Cancel' : 'Edit'}
                                </button>
                                <button
                                  className="danger-button"
                                  disabled={nodeConsole.deleteNode.isPending}
                                  onClick={() => {
                                    if (!window.confirm(`Delete node ${node.name} (${node.id})?`)) {
                                      return;
                                    }
                                    if (editingNodeID === node.id) {
                                      setEditingNodeID('');
                                    }
                                    nodeConsole.deleteNode.mutate(node.id);
                                  }}
                                  type="button"
                                >
                                  Delete
                                </button>
                              </div>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
              {editingNode ? (
                <section className="node-editor-card">
                  <div className="panel-toolbar">
                    <div>
                      <p className="section-kicker">Update</p>
                      <h3>Edit node record</h3>
                      <p className="section-copy">Update metadata, exposure, and runtime state for the selected node.</p>
                    </div>
                    <span className="badge mono">{editingNode.id}</span>
                  </div>
                  <div className="forms-grid">
                    <label className="field-stack">
                      <span>Name</span>
                      <input className="field-input" onChange={(event) => setFormState((current) => ({...current, name: event.target.value}))} value={formState.name} />
                    </label>
                    <label className="field-stack">
                      <span>Mode</span>
                      <select className="field-select" onChange={(event) => setFormState((current) => ({...current, mode: event.target.value}))} value={formState.mode}>
                        {availableModes.map((mode) => (
                          <option key={mode} value={mode}>
                            {mode}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>Scope key</span>
                      <input className="field-input" onChange={(event) => setFormState((current) => ({...current, scopeKey: event.target.value}))} value={formState.scopeKey} />
                    </label>
                    <label className="field-stack">
                      <span>Parent node id</span>
                      <input className="field-input" onChange={(event) => setFormState((current) => ({...current, parentNodeId: event.target.value}))} value={formState.parentNodeId} />
                    </label>
                    <label className="field-stack">
                      <span>Public host</span>
                      <input className="field-input" onChange={(event) => setFormState((current) => ({...current, publicHost: event.target.value}))} value={formState.publicHost} />
                    </label>
                    <label className="field-stack">
                      <span>Public port</span>
                      <input className="field-input" inputMode="numeric" onChange={(event) => setFormState((current) => ({...current, publicPort: event.target.value}))} value={formState.publicPort} />
                    </label>
                    <label className="field-stack">
                      <span>Status</span>
                      <select className="field-select" onChange={(event) => setFormState((current) => ({...current, status: event.target.value}))} value={formState.status}>
                        {availableStatuses.map((status) => (
                          <option key={status} value={status}>
                            {status}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>Enabled</span>
                      <select
                        className="field-select"
                        onChange={(event) => setFormState((current) => ({...current, enabled: event.target.value === 'true'}))}
                        value={String(formState.enabled)}
                      >
                        <option value="true">enabled</option>
                        <option value="false">disabled</option>
                      </select>
                    </label>
                  </div>
                  <div className="submit-row">
                    <button
                      className="primary-button"
                      disabled={nodeConsole.updateNode.isPending || formState.name.trim().length === 0}
                      onClick={() =>
                        nodeConsole.updateNode.mutate(
                          {
                            nodeID: editingNode.id,
                            name: formState.name.trim(),
                            mode: formState.mode,
                            scopeKey: formState.scopeKey.trim(),
                            parentNodeId: formState.parentNodeId.trim(),
                            publicHost: formState.publicHost.trim(),
                            publicPort: formState.publicPort.trim() ? Number(formState.publicPort) : 0,
                            enabled: formState.enabled,
                            status: formState.status
                          },
                          {
                            onSuccess: () => {
                              setEditingNodeID('');
                            }
                          }
                        )
                      }
                      type="button"
                    >
                      Save changes
                    </button>
                    <button className="secondary-button" onClick={() => setEditingNodeID('')} type="button">
                      Close
                    </button>
                  </div>
                </section>
              ) : null}
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}

export function NodeTopologyPageContent() {
  const nodeConsole = useNodeConsole();
  const links = nodeConsole.linksQuery.data || [];

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Topology</p>
              <h3>Relay relationships</h3>
              <p className="section-copy">Inspect parent-child and trust links between edge and relay nodes.</p>
            </div>
            <span className="badge">{links.length}</span>
          </div>
          {nodeConsole.linksQuery.isPending ? (
            <AsyncState detail="Loading" title="Loading topology links" />
          ) : nodeConsole.linksQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.linksQuery.error)}
              onAction={() => void nodeConsole.linksQuery.refetch()}
              title="Failed to load topology links"
            />
          ) : links.length === 0 ? (
            <AsyncState detail="Links appear after parent-child registration or relay trust setup." title="Empty" />
          ) : (
            <div className="nodes-link-grid">
              {links.map((link) => (
                <NodeLinkCard key={link.id} link={link} />
              ))}
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}

function NodeCard({node, action}: {node: Node; action?: ReactNode}) {
  return (
    <article className="node-record-card">
      <div className="stack-head">
        <strong>{node.name}</strong>
        <span className={statusBadgeClassName(node.status)}>{node.status}</span>
      </div>
      <div className="nodes-ledger-meta">
        <span>{node.mode}</span>
        <span>{node.scopeKey || 'no-scope'}</span>
      </div>
      <span className="mono">{node.id}</span>
      <span className="muted-text">{node.publicHost ? `${node.publicHost}:${node.publicPort}` : 'No public endpoint'}</span>
      {node.parentNodeId ? <span className="muted-text">parent: {node.parentNodeId}</span> : null}
      {action}
    </article>
  );
}

function statusBadgeClassName(status: string) {
  if (status === 'healthy') {
    return 'badge is-good';
  }
  if (status === 'degraded') {
    return 'badge is-danger';
  }
  if (status === 'pending') {
    return 'badge is-warn';
  }
  return 'badge is-neutral';
}

function NodeLinkCard({link}: {link: NodeLink}) {
  return (
    <article className="node-record-card">
      <div className="stack-head">
        <strong>
          {link.sourceNodeId} → {link.targetNodeId}
        </strong>
        <span className="badge">{link.trustState}</span>
      </div>
      <div className="nodes-ledger-meta">
        <span>{link.linkType}</span>
      </div>
      <span className="mono">{link.id}</span>
    </article>
  );
}
