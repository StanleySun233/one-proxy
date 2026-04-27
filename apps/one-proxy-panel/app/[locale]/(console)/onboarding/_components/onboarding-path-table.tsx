'use client';

import type { Node, NodeAccessPath } from '@/lib/types';
import { formatControlPlaneError, joinList } from '@/lib/presentation';
import { AsyncState } from '@/components/async-state';
import { describeNodeLabel, describePathTarget, describePathTaskSummary } from '../_hooks/use-onboarding';
import { OnboardingPathEditor } from './onboarding-path-editor';

type PathQueryState = {
  value: string;
  set: (v: string) => void;
  modeFilter: string;
  setModeFilter: (v: string) => void;
  enabledFilter: string;
  setEnabledFilter: (v: string) => void;
  editingPathID: string;
  setEditingPathID: (v: string) => void;
  pathEditorState: any;
  setPathEditorState: (v: any) => void;
};

type Props = {
  t: (key: string) => string;
  pathsQuery: { isPending: boolean; isError: boolean; error: Error | null; refetch: () => void; data?: NodeAccessPath[] };
  paths: NodeAccessPath[];
  totalPaths: number;
  nodesByID: Map<string, Node>;
  taskCountByPathID: Map<string, number>;
  taskSummaryByPathID: Map<string, { pending: number; failed: number; connected: number }>;
  editingPath: NodeAccessPath | null;
  pathState: PathQueryState;
  deletePathMutation: { isPending: boolean; mutate: (id: string) => void };
  pathModeOptions: string[];
  updatePathMutation: { isPending: boolean; mutate: (p: any) => void };
  nodes: { id: string; name: string }[];
};

export function OnboardingPathTable({
  t, pathsQuery, paths, totalPaths, nodesByID, taskCountByPathID, taskSummaryByPathID,
  editingPath, pathState, deletePathMutation, pathModeOptions, updatePathMutation, nodes,
}: Props) {
  if (pathsQuery.isPending) {
    return (
      <section className="panel-card">
        <AsyncState detail={t('common.loading')} title="Loading access paths" />
      </section>
    );
  }

  if (pathsQuery.isError) {
    return (
      <section className="panel-card">
        <AsyncState
          actionLabel={t('common.retry')}
          detail={formatControlPlaneError(pathsQuery.error)}
          onAction={() => void pathsQuery.refetch()}
          title="Failed to load access paths"
        />
      </section>
    );
  }

  if (paths.length === 0) {
    return (
      <section className="panel-card">
        <AsyncState detail="Create the first path before dispatching relay or upstream onboarding tasks." title={t('common.empty')} />
      </section>
    );
  }

  return (
    <section className="panel-card">
      <div className="panel-toolbar">
        <div>
          <p className="section-kicker">Path Registry</p>
          <h3>Access paths</h3>
          <p className="section-copy">Query definitions, inspect relay hops, and maintain records with update and delete actions.</p>
        </div>
        <div className="inline-cluster">
          <span className="badge">{paths.length} shown</span>
          <span className="badge">{totalPaths} total</span>
        </div>
      </div>
      <div className="registry-stack">
        <div className="registry-toolbar">
          <label className="field-stack registry-filter">
            <span>Search</span>
            <input
              className="field-input"
              onChange={(event) => pathState.set(event.target.value)}
              placeholder="Search by name, id, host, node, or relay hop"
              type="search"
              value={pathState.value}
            />
          </label>
          <label className="field-stack registry-filter registry-filter-short">
            <span>Mode</span>
            <select className="field-select" onChange={(event) => pathState.setModeFilter(event.target.value)} value={pathState.modeFilter}>
              <option value="all">All modes</option>
              {pathModeOptions.map((mode) => (
                <option key={mode} value={mode}>{mode}</option>
              ))}
            </select>
          </label>
          <label className="field-stack registry-filter registry-filter-short">
            <span>Enabled</span>
            <select className="field-select" onChange={(event) => pathState.setEnabledFilter(event.target.value)} value={pathState.enabledFilter}>
              <option value="all">All</option>
              <option value="enabled">enabled</option>
              <option value="disabled">disabled</option>
            </select>
          </label>
        </div>
        {paths.length === 0 ? (
          <AsyncState detail="Adjust the current query or filters to see matching paths." title="No matching access paths" />
        ) : (
          <div className="table-card">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Mode</th>
                  <th>Target</th>
                  <th>Entry</th>
                  <th>Relay chain</th>
                  <th>Tasks</th>
                  <th>Status</th>
                  <th>ID</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {paths.map((path) => {
                  const isActive = path.id === pathState.editingPathID;
                  return (
                    <tr className={isActive ? 'is-active-row' : ''} key={path.id}>
                      <td>{path.name}</td>
                      <td>{path.mode}</td>
                      <td>{describePathTarget(path, nodesByID)}</td>
                      <td>{describeNodeLabel(path.entryNodeId, nodesByID)}</td>
                      <td>{path.relayNodeIds.length > 0 ? path.relayNodeIds.join(' -> ') : <span className="muted-text">direct</span>}</td>
                      <td>
                        <div className="registry-name-cell">
                          <strong>{taskCountByPathID.get(path.id) || 0}</strong>
                          <span className="muted-text">{describePathTaskSummary(taskSummaryByPathID.get(path.id))}</span>
                        </div>
                      </td>
                      <td>
                        <span className={`badge ${path.enabled ? 'is-good-soft' : 'is-neutral'}`}>{path.enabled ? 'enabled' : 'disabled'}</span>
                      </td>
                      <td className="mono registry-id-cell">{path.id}</td>
                      <td>
                        <div className="registry-actions">
                          <button
                            className="secondary-button"
                            onClick={() => {
                              if (isActive) {
                                pathState.setEditingPathID('');
                                pathState.setPathEditorState(null);
                                return;
                              }
                              pathState.setEditingPathID(path.id);
                              pathState.setPathEditorState({
                                name: path.name,
                                mode: path.mode,
                                targetNodeId: path.targetNodeId,
                                entryNodeId: path.entryNodeId,
                                relayNodeIds: joinList(path.relayNodeIds),
                                targetHost: path.targetHost,
                                targetPort: path.targetPort > 0 ? String(path.targetPort) : '',
                                enabled: path.enabled
                              });
                            }}
                            type="button"
                          >
                            {isActive ? 'Cancel' : 'Edit'}
                          </button>
                          <button
                            className="danger-button"
                            disabled={deletePathMutation.isPending}
                            onClick={() => {
                              if (!window.confirm(`Delete access path ${path.name} (${path.id})?`)) {
                                return;
                              }
                              deletePathMutation.mutate(path.id);
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
        {editingPath && pathState.pathEditorState ? (
          <OnboardingPathEditor
            editingPath={editingPath}
            pathEditorState={pathState.pathEditorState}
            setPathEditorState={pathState.setPathEditorState}
            pathModeOptions={pathModeOptions}
            nodes={nodes}
            updatePathMutation={updatePathMutation}
            onClose={() => { pathState.setEditingPathID(''); pathState.setPathEditorState(null); }}
          />
        ) : null}
      </div>
    </section>
  );
}
