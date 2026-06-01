'use client';

import {useEffect, useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {NameTag} from '@/components/common/name-tag';
import {fetchEnums} from '@/lib/api';
import {formatControlPlaneError, formatISODateTime} from '@/lib/presentation';

import {useNodeConsole} from './use-node-console';
import {describeNodeName, deriveNodeHealthState, healthBadgeClassName, statusBadgeClassName} from './node-utils';

export function NodeRegistryPageContent() {
  const t = useTranslations();
  const nodesT = useTranslations('nodesConsole');
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const scopes = nodeConsole.scopesQuery.data || [];
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
            <span className="metric-label">{nodesT('healthyNodes')}</span>
            <strong>{summary.healthy}</strong>
          </article>
          <article className="metric-card panel-card soft-card">
            <span className="metric-label">{nodesT('degradedNodes')}</span>
            <strong>{summary.degraded}</strong>
          </article>
          <article className="metric-card panel-card warm-card">
            <span className="metric-label">{nodesT('staleNodes')}</span>
            <strong>{summary.stale}</strong>
          </article>
          <article className="metric-card panel-card">
            <span className="metric-label">{nodesT('unreportedNodes')}</span>
            <strong>{summary.unreported}</strong>
          </article>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">{nodesT('registry')}</p>
              <h3>{nodesT('registryTitle')}</h3>
            </div>
            <div className="inline-cluster">
              <span className="badge">{filteredNodes.length} {t('common.shown')}</span>
              <span className="badge">{nodeRows.length} {t('common.total')}</span>
            </div>
          </div>
          {nodeConsole.nodesQuery.isPending || nodeConsole.healthQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={nodesT('loadingRegistry')} />
          ) : nodeConsole.nodesQuery.error ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(nodeConsole.nodesQuery.error)}
              onAction={() => void nodeConsole.nodesQuery.refetch()}
              title={nodesT('failedRegistry')}
            />
          ) : nodeConsole.healthQuery.error ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(nodeConsole.healthQuery.error)}
              onAction={() => void nodeConsole.healthQuery.refetch()}
              title={nodesT('failedHealth')}
            />
          ) : nodes.length === 0 ? (
            <AsyncState detail={nodesT('emptyRegistry')} title={t('common.empty')} />
          ) : (
            <div className="registry-stack">
              <div className="registry-toolbar">
                <label className="field-stack registry-filter">
                  <span>{t('common.search')}</span>
                  <input
                    className="field-input"
                    onChange={(event) => setQuery(event.target.value)}
                    placeholder={nodesT('registrySearchPlaceholder')}
                    type="search"
                    value={query}
                  />
                </label>
                <label className="field-stack registry-filter registry-filter-short">
                  <span>{t('common.status')}</span>
                  <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
                    <option value="all">{nodesT('allHealthStates')}</option>
                    <option value="healthy">{nodesT('healthyNodes')}</option>
                    <option value="degraded">{nodesT('degradedNodes')}</option>
                    <option value="stale">{nodesT('staleNodes')}</option>
                    <option value="unreported">{nodesT('unreportedNodes')}</option>
                  </select>
                </label>
                <label className="field-stack registry-filter registry-filter-short">
                  <span>{t('common.mode')}</span>
                  <select className="field-select" onChange={(event) => setModeFilter(event.target.value)} value={modeFilter}>
                    <option value="all">{nodesT('allModes')}</option>
                    {availableModes.map((mode) => (
                      <option key={mode} value={mode}>
                        {mode}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
              {filteredNodes.length === 0 ? (
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
                        <th>{t('common.id')}</th>
                        <th>{t('common.actions')}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {filteredNodes.map((node) => {
                        const active = node.id === editingNodeID;

                        return (
                          <tr className={active ? 'is-active-row' : ''} key={node.id}>
                            <td>
                              <div className="registry-name-cell">
                                <NameTag kind="node">{node.name}</NameTag>
                                <span className={`badge ${node.enabled ? 'is-good-soft' : 'is-neutral'}`}>
                                  {node.enabled ? t('common.enabled') : t('common.disabled')}
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
                            <td>{node.scopeKey ? <NameTag kind="scope">{node.scopeKey}</NameTag> : <span className="muted-text">{t('common.noScope')}</span>}</td>
                            <td className="mono">{node.heartbeatAt ? formatISODateTime(node.heartbeatAt) : <span className="muted-text">{t('common.never')}</span>}</td>
                            <td>{node.policyRevisionId || <span className="muted-text">{t('common.unassigned')}</span>}</td>
                            <td>{node.publicHost ? `${node.publicHost}:${node.publicPort}` : <span className="muted-text">{nodesT('noPublicEndpoint')}</span>}</td>
                            <td>{describeNodeName(node.parentNodeId, nodesByID) || <span className="muted-text">{t('common.root')}</span>}</td>
                            <td className="mono registry-id-cell">{node.id}</td>
                            <td>
                              <div className="registry-actions">
                                <button
                                  className="secondary-button"
                                  onClick={() => setEditingNodeID(active ? '' : node.id)}
                                  type="button"
                                >
                                  {active ? t('common.cancel') : t('common.edit')}
                                </button>
                                <button
                                  className="danger-button"
                                  disabled={nodeConsole.deleteNode.isPending}
                                  onClick={() => {
                                    if (!window.confirm(nodesT('deleteNodeConfirm', {name: node.name, id: node.id}))) {
                                      return;
                                    }
                                    if (editingNodeID === node.id) {
                                      setEditingNodeID('');
                                    }
                                    nodeConsole.deleteNode.mutate(node.id);
                                  }}
                                  type="button"
                                >
                                  {t('common.delete')}
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
                      <p className="section-kicker">{t('common.update')}</p>
                      <h3>{nodesT('editNode')}</h3>
                    </div>
                    <span className="badge mono">{editingNode.id}</span>
                  </div>
                  <div className="forms-grid">
                    <label className="field-stack">
                      <span>{t('common.name')}</span>
                      <input className="field-input" onChange={(event) => setFormState((current) => ({...current, name: event.target.value}))} value={formState.name} />
                    </label>
                    <label className="field-stack">
                      <span>{t('common.mode')}</span>
                      <select className="field-select" onChange={(event) => setFormState((current) => ({...current, mode: event.target.value}))} value={formState.mode}>
                        {nodeModeOptions.map((opt) => (
                          <option key={opt.value} value={opt.value}>
                            {opt.label}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>{nodesT('scopeKey')}</span>
                      <select className="field-select" onChange={(event) => setFormState((current) => ({...current, scopeKey: event.target.value}))} value={formState.scopeKey}>
                        <option value="">{t('common.noScope')}</option>
                        {scopes.map((scope) => (
                          <option key={scope.id} value={scope.id}>{scope.name} ({scope.id})</option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>{t('common.parent')}</span>
                      <select className="field-select" onChange={(event) => setFormState((current) => ({...current, parentNodeId: event.target.value}))} value={formState.parentNodeId}>
                        <option value="">{t('common.root')}</option>
                        {nodes.filter((n) => n.id !== editingNode!.id).map((n) => (
                          <option key={n.id} value={n.id}>
                            {n.name} ({n.mode})
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>{nodesT('publicHost')}</span>
                      <input className="field-input" onChange={(event) => setFormState((current) => ({...current, publicHost: event.target.value}))} value={formState.publicHost} />
                    </label>
                    <label className="field-stack">
                      <span>{nodesT('publicPort')}</span>
                      <input className="field-input" inputMode="numeric" onChange={(event) => setFormState((current) => ({...current, publicPort: event.target.value}))} value={formState.publicPort} />
                    </label>
                    <label className="field-stack">
                      <span>{t('common.status')}</span>
                      <select className="field-select" onChange={(event) => setFormState((current) => ({...current, status: event.target.value}))} value={formState.status}>
                        {nodeStatusOptions.map((opt) => (
                          <option key={opt.value} value={opt.value}>{opt.label}</option>
                        ))}
                      </select>
                    </label>
                    <label className="field-stack">
                      <span>{t('common.enabled')}</span>
                      <select
                        className="field-select"
                        onChange={(event) => setFormState((current) => ({...current, enabled: event.target.value === 'true'}))}
                        value={String(formState.enabled)}
                      >
                        <option value="true">{t('common.enabled')}</option>
                        <option value="false">{t('common.disabled')}</option>
                      </select>
                    </label>
                  </div>
                  <div className="submit-row">
                    <button
                      className="primary-button"
                      disabled={nodeConsole.updateNode.isPending || formState.name.trim().length === 0 || formState.scopeKey.trim().length === 0}
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
                      {nodesT('saveChanges')}
                    </button>
                    <button className="secondary-button" onClick={() => setEditingNodeID('')} type="button">
                      {t('common.close')}
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
