'use client';

import type { UseFormReturn } from 'react-hook-form';

import type { OnboardingTaskFormValues } from '../_hooks/use-onboarding';

type Props = {
  t: (key: string) => string;
  taskForm: UseFormReturn<OnboardingTaskFormValues>;
  taskMode: string;
  pathModeOptions: string[];
  paths: { id: string; name: string }[];
  nodes: { id: string; name: string }[];
  createTaskMutation: { mutate: (p: any) => void; isPending: boolean };
};

export function OnboardingTaskForm({ t, taskForm, taskMode, pathModeOptions, paths, nodes, createTaskMutation }: Props) {
  return (
    <article className="panel-card soft-card">
      <div className="panel-toolbar">
        <div>
          <p className="section-kicker">Create Task</p>
          <h3>Onboarding task dispatch</h3>
          <p className="section-copy">Generate a concrete execution task, then track and manually advance status until the node-side flow is fully wired.</p>
        </div>
      </div>
      <form
        className="sub-grid"
        onSubmit={(e) => { taskForm.handleSubmit((values) => {
          createTaskMutation.mutate({
            mode: values.mode,
            pathId: values.pathId.trim(),
            targetNodeId: values.targetNodeId.trim(),
            targetHost: values.targetHost.trim(),
            targetPort: values.targetPort ? Number(values.targetPort) : 0
          });
        })(e); }}
      >
        <label className="field-stack">
          <span>Mode</span>
          <select className="field-select" {...taskForm.register('mode', {required: true})}>
            {pathModeOptions.map((mode) => (
              <option key={mode} value={mode}>{mode}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>Path</span>
          <select
            aria-invalid={taskForm.formState.errors.pathId ? 'true' : 'false'}
            className="field-select"
            {...taskForm.register('pathId', {
              validate: (value) => (taskMode === 'direct' || value.trim() !== '' ? true : 'path is required for upstream_pull or relay_chain')
            })}
          >
            <option value="">Select path</option>
            {paths.map((path) => (
              <option key={path.id} value={path.id}>{path.name}</option>
            ))}
          </select>
          {taskForm.formState.errors.pathId ? <p className="error-text">{taskForm.formState.errors.pathId.message}</p> : null}
        </label>
        <label className="field-stack">
          <span>Target node</span>
          <select className="field-select" {...taskForm.register('targetNodeId')}>
            <option value="">Optional target node</option>
            {nodes.map((node) => (
              <option key={node.id} value={node.id}>{node.name}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>Target host</span>
          <input
            aria-invalid={taskForm.formState.errors.targetHost ? 'true' : 'false'}
            className="field-input"
            placeholder="db.internal.example.com"
            {...taskForm.register('targetHost', {
              validate: (value) => (taskMode !== 'direct' || value.trim() !== '' ? true : 'target host is required for direct mode')
            })}
          />
          {taskForm.formState.errors.targetHost ? <p className="error-text">{taskForm.formState.errors.targetHost.message}</p> : null}
        </label>
        <label className="field-stack">
          <span>Target port</span>
          <input
            aria-invalid={taskForm.formState.errors.targetPort ? 'true' : 'false'}
            className="field-input"
            placeholder="3306"
            type="number"
            {...taskForm.register('targetPort', {
              validate: (value) => (taskMode !== 'direct' || Number(value) > 0 ? true : 'target port must be greater than 0 for direct mode')
            })}
          />
          {taskForm.formState.errors.targetPort ? <p className="error-text">{taskForm.formState.errors.targetPort.message}</p> : null}
        </label>
        <div className="submit-row">
          <button className="primary-button" disabled={createTaskMutation.isPending} type="submit">
            {createTaskMutation.isPending ? t('common.submitting') : 'Create task'}
          </button>
        </div>
      </form>
    </article>
  );
}
