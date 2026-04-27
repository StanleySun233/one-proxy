'use client';

import {ReactNode, useEffect, useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';

import {AuthGate} from '@/components/auth-gate';
import {AsyncState} from '@/components/async-state';
import {FieldEnumMap, Node, NodeHealth, NodeLink, NodeTransport, UnconsumedBootstrapToken} from '@/lib/control-plane-types';
import {fetchEnums} from '@/lib/control-plane-api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {BootstrapTokenTab} from './bootstrap-token-tab';
import {ManualNodeTab} from './manual-node-tab';
import {QuickConnectTab} from './quick-connect-tab';
import {useNodeConsole} from './use-node-console';

const staleThresholdMs = 2 * 60 * 1000;

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
            nodes={nodeConsole.nodesQuery.data || []}
            onSubmit={() => {
              const values = nodeConsole.bootstrapForm.getValues();
              nodeConsole.bootstrap.mutate({targetId: values.targetId.trim(), nodeName: values.nodeName.trim()});
            }}
          />
        </section>
      </div>
    </AuthGate>
  );
}

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

export function NodeRegistryPageContent() {
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const healthRows = nodeConsole.healthQuery.data || [];
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [modeFilter, setModeFilter] = useState('all');
  const [editingNodeID, setEditingNodeID] = useState('');
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const nodeModeOptions = enums?.node_mode ? Object.entries(enums.node_mode).map(([value, item]) => ({value, label: item.name})) : [];
  const nodeStatusOptions = enums?.node_status ? Object.entries(enums.node_status).map(([value, item]) => ({value, label: item.name})) : [];
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
  const healthByNodeID = useMemo(() => new Map(healthRows.map((item) => [item.nodeId, item])), [healthRows]);
  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  const nodeRows = useMemo(
    () =>
      nodes.map((node) => {
        const health = healthByNodeID.get(node.id);
        const derivedHealth = deriveNodeHealthState(health, enums);
        return {
          ...node,
          derivedHealthStatus: derivedHealth.status,
          derivedHealthLabel: derivedHealth.label,
          heartbeatAt: health?.heartbeatAt || '',
          policyRevisionId: health?.policyRevisionId || ''
        };
      }),
    [healthByNodeID, nodes, enums]
  );
  const normalizedQuery = query.trim().toLowerCase();
  const filteredNodes = nodeRows.filter((node) => {
    const matchesQuery =
      normalizedQuery.length === 0 ||
      node.name.toLowerCase().includes(normalizedQuery) ||
      node.id.toLowerCase().includes(normalizedQuery) ||
      node.scopeKey.toLowerCase().includes(normalizedQuery) ||
      node.parentNodeId.toLowerCase().includes(normalizedQuery) ||
      node.publicHost?.toLowerCase().includes(normalizedQuery) ||
      node.derivedHealthLabel.toLowerCase().includes(normalizedQuery) ||
      node.policyRevisionId.toLowerCase().includes(normalizedQuery);
    const matchesStatus = statusFilter === 'all' || node.derivedHealthStatus === statusFilter;
    const matchesMode = modeFilter === 'all' || node.mode === modeFilter;

    return matchesQuery && matchesStatus && matchesMode;
  });
  const summary = useMemo(() => {
    return {
      healthy: nodeRows.filter((node) => node.derivedHealthStatus === 'healthy').length,
      degraded: nodeRows.filter((node) => node.derivedHealthStatus === 'degraded').length,
      stale: nodeRows.filter((node) => node.derivedHealthStatus === 'stale').length,
      unreported: nodeRows.filter((node) => node.derivedHealthStatus === 'unreported').length
    };
  }, [nodeRows]);
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
        <section className="metrics-grid">
          <article className="metric-card panel-card">
            <span className="metric-label">Healthy nodes</span>
            <strong>{summary.healthy}</strong>
            <span className="metric-foot">Nodes with recent heartbeat and no degraded signal.</span>
          </article>
          <article className="metric-card panel-card soft-card">
            <span className="metric-label">Degraded nodes</span>
            <strong>{summary.degraded}</strong>
            <span className="metric-foot">Nodes reporting non-healthy listener or certificate state.</span>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">Stale nodes</span>
            <strong>{summary.stale}</strong>
            <span className="metric-foot">Nodes whose heartbeat freshness window has already expired.</span>
          </article>
          <article className="metric-card panel-card">
            <span className="metric-label">Unreported nodes</span>
            <strong>{summary.unreported}</strong>
            <span className="metric-foot">Registered nodes without any heartbeat record yet.</span>
          </article>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Registry</p>
              <h3>Node registry</h3>
              <p className="section-copy">Query registered nodes, inspect derived health and policy attachment, and maintain records with read, update, and delete actions.</p>
            </div>
            <div className="inline-cluster">
              <span className="badge">{filteredNodes.length} shown</span>
              <span className="badge">{nodeRows.length} total</span>
            </div>
          </div>
          {nodeConsole.nodesQuery.isPending || nodeConsole.healthQuery.isPending ? (
            <AsyncState detail="Loading" title="Loading node registry" />
          ) : nodeConsole.nodesQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.nodesQuery.error)}
              onAction={() => void nodeConsole.nodesQuery.refetch()}
              title="Failed to load node registry"
            />
          ) : nodeConsole.healthQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.healthQuery.error)}
              onAction={() => void nodeConsole.healthQuery.refetch()}
              title="Failed to load node health"
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
                  <span>Health</span>
                  <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
                    <option value="all">All health states</option>
                    <option value="healthy">healthy</option>
                    <option value="degraded">degraded</option>
                    <option value="stale">stale</option>
                    <option value="unreported">unreported</option>
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
                        <th>Health</th>
                        <th>Mode</th>
                        <th>Scope</th>
                        <th>Heartbeat</th>
                        <th>Policy</th>
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
                              <div className="registry-name-cell">
                                <span className={healthBadgeClassName(node.derivedHealthStatus, enums)}>{node.derivedHealthLabel}</span>
                                <span className={statusBadgeClassName(node.status, enums)}>{node.status}</span>
                              </div>
                            </td>
                            <td>{node.mode}</td>
                            <td>{node.scopeKey || <span className="muted-text">no-scope</span>}</td>
                            <td className="mono">{node.heartbeatAt ? formatISODateTime(node.heartbeatAt) : <span className="muted-text">never</span>}</td>
                            <td>{node.policyRevisionId || <span className="muted-text">unassigned</span>}</td>
                            <td>{node.publicHost ? `${node.publicHost}:${node.publicPort}` : <span className="muted-text">No public endpoint</span>}</td>
                            <td>{describeNodeName(node.parentNodeId, nodesByID) || <span className="muted-text">root</span>}</td>
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
                        {nodeModeOptions.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {opt.label}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>Scope key</span>
                      <input className="field-input" onChange={(event) => setFormState((current) => ({...current, scopeKey: event.target.value}))} value={formState.scopeKey} />
                    </label>
                    <label className="field-stack">
                      <span>Parent node</span>
                      <select className="field-select" onChange={(event) => setFormState((current) => ({...current, parentNodeId: event.target.value}))} value={formState.parentNodeId}>
                        <option value="">None (root node)</option>
                        {nodes.filter((n) => n.id !== editingNode!.id).map((n) => (
                          <option key={n.id} value={n.id}>
                            {n.name} ({n.mode})
                          </option>
                        ))}
                      </select>
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
                        {nodeStatusOptions.map((opt) => (
                          <option key={opt.value} value={opt.value}>{opt.label}</option>
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
  const nodes = nodeConsole.nodesQuery.data || [];
  const links = nodeConsole.linksQuery.data || [];
  const transports = nodeConsole.transportsQuery.data || [];
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
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
            <span className="metric-label">Public transports</span>
            <strong>{transportSummary.publicEndpoints}</strong>
            <span className="metric-foot">Directly reachable node entrypoints synthesized from public endpoint records.</span>
          </article>
          <article className="metric-card panel-card soft-card">
            <span className="metric-label">Reverse tunnels up</span>
            <strong>{transportSummary.reverseConnected}</strong>
            <span className="metric-foot">Parent-child `reverse_ws_parent` sessions currently reporting connected.</span>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">Reverse tunnels blocked</span>
            <strong>{transportSummary.reverseBlocked}</strong>
            <span className="metric-foot">Configured reverse tunnels that are not currently connected.</span>
          </article>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">Topology</p>
              <h3>Relay relationships</h3>
              <p className="section-copy">Inspect parent-child links together with live transport state to verify a → b → c tunnel overlays.</p>
            </div>
            <span className="badge">{links.length}</span>
          </div>
          <CreateNodeLinkForm
            accessToken={nodeConsole.accessToken}
            nodes={nodes}
            pending={nodeConsole.createNodeLink.isPending}
            onSubmit={(payload) => nodeConsole.createNodeLink.mutate(payload)}
            defaultLinkType={LINK_TYPE_RELAY}
            defaultTrustState={TRUST_STATE_TRUSTED}
          />
          {nodeConsole.linksQuery.isPending || nodeConsole.nodesQuery.isPending || nodeConsole.transportsQuery.isPending ? (
            <AsyncState detail="Loading" title="Loading topology links" />
          ) : nodeConsole.nodesQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.nodesQuery.error)}
              onAction={() => void nodeConsole.nodesQuery.refetch()}
              title="Failed to load node registry"
            />
          ) : nodeConsole.linksQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.linksQuery.error)}
              onAction={() => void nodeConsole.linksQuery.refetch()}
              title="Failed to load topology links"
            />
          ) : nodeConsole.transportsQuery.error ? (
            <AsyncState
              actionLabel="Retry"
              detail={formatControlPlaneError(nodeConsole.transportsQuery.error)}
              onAction={() => void nodeConsole.transportsQuery.refetch()}
              title="Failed to load transport registry"
            />
          ) : links.length === 0 ? (
            <AsyncState detail="Links appear after parent-child registration or relay trust setup." title="Empty" />
          ) : (
            <div className="topology-stack">
              <div className="nodes-link-grid">
                {links.map((link) => (
                  <NodeLinkCard key={link.id} link={link} nodesByID={nodesByID} transports={transports} reverseWsType={REVERSE_WS_PARENT} />
                ))}
              </div>
              <div className="table-card">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Node</th>
                      <th>Type</th>
                      <th>Direction</th>
                      <th>Status</th>
                      <th>Address</th>
                      <th>Parent</th>
                      <th>Heartbeat</th>
                    </tr>
                  </thead>
                  <tbody>
                    {transports.length === 0 ? (
                      <tr>
                        <td className="muted-text" colSpan={7}>
                          No runtime transports have been reported yet.
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
                          <td>{describeNodeName(transport.parentNodeId, nodesByID) || <span className="muted-text">root</span>}</td>
                          <td className="mono">{transport.lastHeartbeatAt ? formatISODateTime(transport.lastHeartbeatAt) : <span className="muted-text">never</span>}</td>
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

function NodeCard({node, action, detail}: {node: Node; action?: ReactNode; detail?: ReactNode}) {
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
      {detail}
      <span className="mono">{node.id}</span>
      <span className="muted-text">{node.publicHost ? `${node.publicHost}:${node.publicPort}` : 'No public endpoint'}</span>
      {node.parentNodeId ? <span className="muted-text">parent: {node.parentNodeId}</span> : null}
      {action}
    </article>
  );
}

function describeNodeName(nodeID: string, nodesByID: Map<string, Node>) {
  if (!nodeID) {
    return '';
  }
  return nodesByID.get(nodeID)?.name || nodeID;
}

function deriveNodeHealthState(item?: NodeHealth, enums?: FieldEnumMap) {
  if (!item) {
    return {status: 'unreported', label: 'unreported'};
  }
  const heartbeatTime = Date.parse(item.heartbeatAt);
  const isStale = Number.isFinite(heartbeatTime) ? Date.now() - heartbeatTime > staleThresholdMs : true;
  const listenerValues = Object.values(item.listenerStatus || {});
  const certValues = Object.values(item.certStatus || {});
  const allValues = [...listenerValues, ...certValues];
  const isGoodValue = (value: string) =>
    enums?.listener_status?.[value]?.meta?.className === 'is-good' ||
    enums?.cert_status?.[value]?.meta?.className === 'is-good';
  const hasDegradedSignal = enums
    ? allValues.some(v => !isGoodValue(v))
    : allValues.some((value) => value !== 'up' && value !== 'healthy' && value !== 'renewed');
  if (isStale) {
    return {status: 'stale', label: 'stale'};
  }
  if (hasDegradedSignal) {
    return {status: 'degraded', label: 'degraded'};
  }
  return {status: 'healthy', label: 'healthy'};
}

function healthBadgeClassName(status: string, enums?: FieldEnumMap) {
  if (!enums) {
    if (status === 'healthy') return 'badge is-good';
    if (status === 'stale') return 'badge is-warn';
    if (status === 'unreported') return 'badge is-neutral';
    return 'badge is-danger';
  }
  const cls = enums.node_status?.[status]?.meta?.className;
  if (cls) return `badge ${cls}`;
  if (status === 'stale') return 'badge is-warn';
  if (status === 'unreported') return 'badge is-neutral';
  return 'badge is-danger';
}

function statusBadgeClassName(status: string, enums?: FieldEnumMap) {
  if (!enums) {
    if (status === 'healthy') return 'badge is-good';
    if (status === 'degraded') return 'badge is-danger';
    if (status === 'pending') return 'badge is-warn';
    return 'badge is-neutral';
  }
  const cls = enums.node_status?.[status]?.meta?.className;
  return `badge ${cls || 'is-neutral'}`;
}

function transportBadgeClassName(status: string, enums?: FieldEnumMap) {
  if (!enums) {
    if (status === 'connected' || status === 'ready') return 'badge is-good';
    if (status === 'degraded' || status === 'failed') return 'badge is-danger';
    if (status === 'pending') return 'badge is-warn';
    return 'badge is-neutral';
  }
  const cls = enums.transport_status?.[status]?.meta?.className;
  return `badge ${cls || 'is-neutral'}`;
}

function NodeLinkCard({
  link,
  nodesByID,
  transports,
  reverseWsType
}: {
  link: NodeLink;
  nodesByID: Map<string, Node>;
  transports: NodeTransport[];
  reverseWsType: string;
}) {
  const childTunnel = transports.find(
    (transport) =>
      transport.nodeId === link.targetNodeId &&
      transport.parentNodeId === link.sourceNodeId &&
      transport.transportType === reverseWsType
  );

  return (
    <article className="node-record-card">
      <div className="stack-head">
        <strong>
          {describeNodeName(link.sourceNodeId, nodesByID)} → {describeNodeName(link.targetNodeId, nodesByID)}
        </strong>
        <span className="badge">{link.trustState}</span>
      </div>
      <div className="nodes-ledger-meta">
        <span>{link.linkType}</span>
      </div>
      {childTunnel ? (
        <div className="node-approval-meta">
          <span className={transportBadgeClassName(childTunnel.status)}>{childTunnel.status}</span>
          <span className="mono">{childTunnel.transportType}</span>
          <span className="muted-text">{childTunnel.lastHeartbeatAt ? formatISODateTime(childTunnel.lastHeartbeatAt) : 'heartbeat: never'}</span>
        </div>
      ) : (
        <span className="badge is-neutral">no active child tunnel</span>
      )}
      <span className="muted-text">source: {link.sourceNodeId}</span>
      <span className="muted-text">target: {link.targetNodeId}</span>
      <span className="mono">{link.id}</span>
    </article>
  );
}
function CreateNodeLinkForm({
  accessToken,
  nodes,
  pending,
  onSubmit,
  defaultLinkType,
  defaultTrustState
}: {
  accessToken: string;
  nodes: Node[];
  pending: boolean;
  onSubmit: (payload: {sourceNodeId: string; targetNodeId: string; linkType: string; trustState: string}) => void;
  defaultLinkType: string;
  defaultTrustState: string;
}) {
  const [sourceNodeId, setSourceNodeId] = useState('');
  const [targetNodeId, setTargetNodeId] = useState('');

  return (
    <div className="forms-grid" style={{marginBottom: 16}}>
      <label className="field-stack">
        <span>Source</span>
        <select className="field-select" onChange={(e) => setSourceNodeId(e.target.value)} value={sourceNodeId}>
          <option value="">Select source</option>
          {nodes.map((n) => (
            <option key={n.id} value={n.id}>{n.id} - {n.name} ({n.mode})</option>
          ))}
        </select>
      </label>
      <label className="field-stack">
        <span>Target</span>
        <select className="field-select" onChange={(e) => setTargetNodeId(e.target.value)} value={targetNodeId}>
          <option value="">Select target</option>
          {nodes.map((n) => (
            <option key={n.id} value={n.id}>{n.id} - {n.name} ({n.mode})</option>
          ))}
        </select>
      </label>
      <div className="field-stack" style={{alignSelf: 'flex-end'}}>
        <button
          className="secondary-button"
          disabled={pending || !sourceNodeId || !targetNodeId}
          onClick={() => onSubmit({sourceNodeId, targetNodeId, linkType: defaultLinkType, trustState: defaultTrustState})}
          type="button"
        >
          {pending ? 'Creating...' : 'Add Link'}
        </button>
      </div>
    </div>
  );
}

