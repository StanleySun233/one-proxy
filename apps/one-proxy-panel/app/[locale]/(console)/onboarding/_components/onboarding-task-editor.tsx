'use client';

import type { NodeOnboardingTask } from '@/lib/types';

type EditorState = {
  status: string;
  statusMessage: string;
};

type Props = {
  editingTask: NodeOnboardingTask;
  taskEditorState: EditorState;
  setTaskEditorState: (state: EditorState | null) => void;
  taskStatusOptions: string[];
  updateTaskStatusMutation: { isPending: boolean; mutate: (p: any) => void };
  onClose: () => void;
};

export function OnboardingTaskEditor({
  editingTask, taskEditorState, setTaskEditorState, taskStatusOptions,
  updateTaskStatusMutation, onClose,
}: Props) {
  return (
    <section className="node-editor-card">
      <div className="panel-toolbar">
        <div>
          <p className="section-kicker">Update</p>
          <h3>Advance task status</h3>
          <p className="section-copy">Use explicit status and operator notes so onboarding progress remains queryable before automated relay execution is complete.</p>
        </div>
        <span className="badge mono">{editingTask.id}</span>
      </div>
      <div className="forms-grid">
        <label className="field-stack">
          <span>Status</span>
          <select className="field-select" onChange={(event) => setTaskEditorState(taskEditorState ? {...taskEditorState, status: event.target.value} : null)} value={taskEditorState.status}>
            {taskStatusOptions.map((status) => (
              <option key={status} value={status}>{status}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>Status message</span>
          <input className="field-input" onChange={(event) => setTaskEditorState(taskEditorState ? {...taskEditorState, statusMessage: event.target.value} : null)} value={taskEditorState.statusMessage} />
        </label>
      </div>
      <div className="submit-row">
        <button
          className="primary-button"
          disabled={updateTaskStatusMutation.isPending || taskEditorState.status.trim().length === 0}
          onClick={() =>
            updateTaskStatusMutation.mutate({
              taskID: editingTask.id,
              status: taskEditorState.status,
              statusMessage: taskEditorState.statusMessage.trim()
            })
          }
          type="button"
        >
          Save task status
        </button>
        <button className="secondary-button" onClick={onClose} type="button">
          Close
        </button>
      </div>
    </section>
  );
}
