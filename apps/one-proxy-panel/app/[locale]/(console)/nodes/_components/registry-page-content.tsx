'use client';

import {useEffect, useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {fetchEnums} from '@/lib/control-plane-api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {useNodeConsole} from './use-node-console';
import {describeNodeName, deriveNodeHealthState, healthBadgeClassName, statusBadgeClassName} from './node-utils';

export function NodeRegistryPageContent() {
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const healthRows = nodeConsole.healthQuery.data || [];
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [modeFilter, setModeFilter] = useState('all');
  const [editingNodeID, setEditingNodeID] = useState('');
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const nodeModeKeys = Object.keys(enums?.node_mode || {});
  const nodeStatusKeys = Object.keys(enums?.node_status || {});
  const DEFAULT_MODE: string = nodeModeKeys.find(k => k === 'relay') || 'relay';
  const DEFAULT_STATUS: string = nodeStatusKeys.find(k => k === 'healthy') || 'healthy';
  const nodeModeOptions = enums?.node_mode ? Object.entries(enums.node_mode).map(([value, item]) => ({value, label: item.name})) : [];
  const nodeStatusOptions = enums?.node_status ? Object.entries(enums.node_status).map(([value, item]) => ({value, label: item.name})) : [];
  const [formState, setFormState] = useState({
    name: '',
    mode: DEFAULT_MODE,
    scopeKey: '',
    parentNodeId: '',
    publicHost: '',
    publicPort: '',
    enabled: true,
    status: DEFAULT_STATUS
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
