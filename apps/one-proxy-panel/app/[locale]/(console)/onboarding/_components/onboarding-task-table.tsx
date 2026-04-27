'use client';

import type { Node, NodeAccessPath, NodeOnboardingTask, FieldEnumMap } from '@/lib/types';
import { formatControlPlaneError, formatISODateTime } from '@/lib/presentation';
import { AsyncState } from '@/components/async-state';
import { describeTaskTarget, describeTaskPath, taskBadgeClassName } from '../_hooks/use-onboarding';
import { OnboardingTaskEditor } from './onboarding-task-editor';

type TaskQueryState = {
  value: string;
  set: (v: string) => void;
  modeFilter: string;
  setModeFilter: (v: string) => void;
  statusFilter: string;
  setStatusFilter: (v: string) => void;
  editingTaskID: string;
  setEditingTaskID: (v: string) => void;
  taskEditorState: any;
  setTaskEditorState: (v: any) => void;
};

type Props = {
  t: (key: string) => string;
  tasksQuery: { isPending: boolean; isError: boolean; error: Error; refetch: () => void; data?: NodeOnboardingTask[] };
  tasks: NodeOnboardingTask[];
  totalTasks: number;
  nodesByID: Map<string, Node>;
  pathsByID: Map<string, NodeAccessPath>;
  enums: FieldEnumMap | undefined;
  availableTaskStatuses: string[];
  pathModeOptions: string[];
  editingTask: NodeOnboardingTask | null;
  taskState: TaskQueryState;
  taskStatusOptions: string[];
  updateTaskStatusMutation: { isPending: boolean; mutate: (p: any) => void };
};

export function OnboardingTaskTable({
  t, tasksQuery, tasks, totalTasks, nodesByID, pathsByID, enums, availableTaskStatuses,
  pathModeOptions, editingTask, taskState, taskStatusOptions, updateTaskStatusMutation,
}: Props) {
  if (tasksQuery.isPending) {
    return (
      <section className="panel-card">
        <AsyncState detail={t('common.loading')} title="Loading onboarding tasks" />
      </section>
    );
  }

  if (tasksQuery.isError) {
    return (
      <section className="panel-card">
        <AsyncState
          actionLabel={t('common.retry')}
          detail={formatControlPlaneError(tasksQuery.error)}
          onAction={() => void tasksQuery.refetch()}
          title="Failed to load onboarding tasks"
        />
      </section>
    );
  }

  if (tasks.length === 0) {
    return (
      <section className="panel-card">
        <AsyncState detail="Tasks will appear here once an operator triggers direct, relay_chain or upstream_pull onboarding." title={t('common.empty')} />
      </section>
    );
  }

  return (
    <section className="panel-card">
      <div className="panel-toolbar">
        <div>
          <p className="section-kicker">Task Registry</p>
          <h3>Onboarding tasks</h3>
          <p className="section-copy">Track execution state, inspect task intent, and manually advance status while node-side automation is still evolving.</p>
        </div>
        <div className="inline-cluster">
          <span className="badge">{tasks.length} shown</span>
          <span className="badge">{totalTasks} total</span>
        </div>
      </div>
      <div className="registry-stack">
        <div className="registry-toolbar">
          <label className="field-stack registry-filter">
            <span>Search</span>
            <input
              className="field-input"
              onChange={(event) => taskState.set(event.target.value)}
              placeholder="Search by id, target, path, mode, or status message"
              type="search"
              value={taskState.value}
            />
          </label>
          <label className="field-stack registry-filter registry-filter-short">
            <span>Status</span>
            <select className="field-select" onChange={(event) => taskState.setStatusFilter(event.target.value)} value={taskState.statusFilter}>
              <option value="all">All statuses</option>
              {availableTaskStatuses.map((status) => (
                <option key={status} value={status}>{status}</option>
              ))}
            </select>
          </label>
          <label className="field-stack registry-filter registry-filter-short">
            <span>Mode</span>
            <select className="field-select" onChange={(event) => taskState.setModeFilter(event.target.value)} value={taskState.modeFilter}>
              <option value="all">All modes</option>
              {pathModeOptions.map((mode) => (
                <option key={mode} value={mode}>{mode}</option>
              ))}
            </select>
          </label>
        </div>
        {tasks.length === 0 ? (
          <AsyncState detail="Adjust the current query or filters to see matching onboarding tasks." title="No matching tasks" />
        ) : (
          <div className="table-card">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Target</th>
                  <th>Status</th>
                  <th>Mode</th>
                  <th>Path</th>
                  <th>Requested at</th>
                  <th>Requested by</th>
                  <th>Updated</th>
                  <th>ID</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {tasks.map((task) => {
                  const isActive = task.id === taskState.editingTaskID;
                  return (
                    <tr className={isActive ? 'is-active-row' : ''} key={task.id}>
                      <td>
                        <div className="registry-name-cell">
                          <strong>{describeTaskTarget(task, nodesByID)}</strong>
                          <span className="muted-text">{task.statusMessage || 'no-status-message'}</span>
                        </div>
                      </td>
                      <td>
                        <span className={taskBadgeClassName(task.status, enums)}>{task.status}</span>
                      </td>
                      <td>{task.mode}</td>
                      <td>{describeTaskPath(task.pathId, pathsByID)}</td>
                      <td className="mono">{formatISODateTime(task.createdAt)}</td>
                      <td>{task.requestedByAccountId || <span className="muted-text">system</span>}</td>
                      <td className="mono">{formatISODateTime(task.updatedAt || task.createdAt)}</td>
                      <td className="mono registry-id-cell">{task.id}</td>
                      <td>
                        <div className="registry-actions">
                          <button
                            className="secondary-button"
                            onClick={() => {
                              if (isActive) {
                                taskState.setEditingTaskID('');
                                taskState.setTaskEditorState(null);
                                return;
                              }
                              taskState.setEditingTaskID(task.id);
                              taskState.setTaskEditorState({
                                status: task.status,
                                statusMessage: task.statusMessage
                              });
                            }}
                            type="button"
                          >
                            {isActive ? 'Cancel' : 'Update'}
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
        {editingTask && taskState.taskEditorState ? (
          <OnboardingTaskEditor
            editingTask={editingTask}
            taskEditorState={taskState.taskEditorState}
            setTaskEditorState={taskState.setTaskEditorState}
            taskStatusOptions={taskStatusOptions}
            updateTaskStatusMutation={updateTaskStatusMutation}
            onClose={() => { taskState.setEditingTaskID(''); taskState.setTaskEditorState(null); }}
          />
        ) : null}
      </div>
    </section>
  );
}
