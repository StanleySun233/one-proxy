'use client';

import type { NodeAccessPath } from '@/lib/types';
import { splitList } from '@/lib/presentation';

type EditorState = {
  name: string;
  mode: string;
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string;
  targetHost: string;
  targetPort: string;
  enabled: boolean;
};

type Props = {
  editingPath: NodeAccessPath;
  pathEditorState: EditorState;
  setPathEditorState: (state: EditorState | null) => void;
  pathModeOptions: string[];
  nodes: { id: string; name: string }[];
  updatePathMutation: { isPending: boolean; mutate: (p: any) => void };
  onClose: () => void;
};

export function OnboardingPathEditor({
  editingPath, pathEditorState, setPathEditorState, pathModeOptions, nodes,
  updatePathMutation, onClose,
}: Props) {
  return (
    <section className="node-editor-card">
      <div className="panel-toolbar">
        <div>
          <p className="section-kicker">Update</p>
          <h3>Edit access path</h3>
          <p className="section-copy">Tune path routing, relay order, and enablement without recreating the whole record.</p>
        </div>
        <span className="badge mono">{editingPath.id}</span>
      </div>
      <div className="forms-grid">
        <label className="field-stack">
          <span>Name</span>
          <input className="field-input" onChange={(event) => setPathEditorState(pathEditorState ? {...pathEditorState, name: event.target.value} : null)} value={pathEditorState.name} />
        </label>
        <label className="field-stack">
          <span>Mode</span>
          <select className="field-select" onChange={(event) => setPathEditorState(pathEditorState ? {...pathEditorState, mode: event.target.value} : null)} value={pathEditorState.mode}>
            {pathModeOptions.map((mode) => (
              <option key={mode} value={mode}>{mode}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>Target node</span>
          <select className="field-select" onChange={(event) => setPathEditorState(pathEditorState ? {...pathEditorState, targetNodeId: event.target.value} : null)} value={pathEditorState.targetNodeId}>
            <option value="">Optional target node</option>
            {nodes.map((node) => (
              <option key={node.id} value={node.id}>{node.name}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>Entry node</span>
          <select className="field-select" onChange={(event) => setPathEditorState(pathEditorState ? {...pathEditorState, entryNodeId: event.target.value} : null)} value={pathEditorState.entryNodeId}>
            <option value="">Optional entry node</option>
            {nodes.map((node) => (
              <option key={node.id} value={node.id}>{node.name}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>Relay node ids</span>
          <input className="field-input" onChange={(event) => setPathEditorState(pathEditorState ? {...pathEditorState, relayNodeIds: event.target.value} : null)} value={pathEditorState.relayNodeIds} />
        </label>
        <label className="field-stack">
          <span>Target host</span>
          <input className="field-input" onChange={(event) => setPathEditorState(pathEditorState ? {...pathEditorState, targetHost: event.target.value} : null)} value={pathEditorState.targetHost} />
        </label>
        <label className="field-stack">
          <span>Target port</span>
          <input className="field-input" inputMode="numeric" onChange={(event) => setPathEditorState(pathEditorState ? {...pathEditorState, targetPort: event.target.value} : null)} value={pathEditorState.targetPort} />
        </label>
        <label className="field-stack">
          <span>Enabled</span>
          <select className="field-select" onChange={(event) => setPathEditorState(pathEditorState ? {...pathEditorState, enabled: event.target.value === 'true'} : null)} value={String(pathEditorState.enabled)}>
            <option value="true">enabled</option>
            <option value="false">disabled</option>
          </select>
        </label>
      </div>
      <div className="submit-row">
        <button
          className="primary-button"
          disabled={updatePathMutation.isPending || pathEditorState.name.trim().length === 0}
          onClick={() =>
            updatePathMutation.mutate({
              pathID: editingPath.id,
              name: pathEditorState.name.trim(),
              mode: pathEditorState.mode,
              targetNodeId: pathEditorState.targetNodeId.trim(),
              entryNodeId: pathEditorState.entryNodeId.trim(),
              relayNodeIds: splitList(pathEditorState.relayNodeIds),
              targetHost: pathEditorState.targetHost.trim(),
              targetPort: pathEditorState.targetPort.trim() ? Number(pathEditorState.targetPort) : 0,
              enabled: pathEditorState.enabled
            })
          }
          type="button"
        >
          Save changes
        </button>
        <button className="secondary-button" onClick={onClose} type="button">
          Close
        </button>
      </div>
    </section>
  );
}
