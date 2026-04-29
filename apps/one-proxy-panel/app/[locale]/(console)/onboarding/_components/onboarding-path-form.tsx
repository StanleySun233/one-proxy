'use client';

import type { UseFormReturn } from 'react-hook-form';

import type { AccessPathFormValues } from '../_hooks/use-onboarding';
import { splitList } from '@/lib/presentation';

type Props = {
  t: (key: string) => string;
  pathForm: UseFormReturn<AccessPathFormValues>;
  pathMode: string;
  pathModeOptions: string[];
  nodes: { id: string; name: string }[];
  createPathMutation: { mutate: (p: any) => void; isPending: boolean };
};

export function OnboardingPathForm({ t, pathForm, pathMode, pathModeOptions, nodes, createPathMutation }: Props) {
  return (
    <article className="panel-card">
      <div className="panel-toolbar">
        <div>
          <p className="section-kicker">Create Path</p>
          <h3>Access path definition</h3>
          <p className="section-copy">Register the entry, relay sequence, and final target so later onboarding tasks can reuse a stable path record.</p>
        </div>
      </div>
      <form
        className="sub-grid"
        onSubmit={(e) => { pathForm.handleSubmit((values) => {
          createPathMutation.mutate({
            name: values.name.trim(),
            mode: values.mode,
            targetNodeId: values.targetNodeId.trim(),
            entryNodeId: values.entryNodeId.trim(),
            relayNodeIds: splitList(values.relayNodeIds),
            targetHost: values.targetHost.trim(),
            targetPort: values.targetPort ? Number(values.targetPort) : 0
          });
        })(e); }}
      >
        <label className="field-stack">
          <span>Name</span>
          <input
            aria-invalid={pathForm.formState.errors.name ? 'true' : 'false'}
            className="field-input"
            placeholder="gateway-to-db"
            {...pathForm.register('name', {required: 'name is required'})}
          />
          {pathForm.formState.errors.name ? <p className="error-text">{pathForm.formState.errors.name.message}</p> : null}
        </label>
        <label className="field-stack">
          <span>Mode</span>
          <select className="field-select" {...pathForm.register('mode', {required: true})}>
            {pathModeOptions.map((mode) => (
              <option key={mode} value={mode}>{mode}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>Target node</span>
          <select className="field-select" {...pathForm.register('targetNodeId')}>
            <option value="">Optional target node</option>
            {nodes.map((node) => (
              <option key={node.id} value={node.id}>{node.name}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>Entry node</span>
          <select className="field-select" {...pathForm.register('entryNodeId')}>
            <option value="">Optional entry node</option>
            {nodes.map((node) => (
              <option key={node.id} value={node.id}>{node.name}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>Relay node ids</span>
          <input className="field-input" placeholder="relay-a, relay-b" {...pathForm.register('relayNodeIds')} />
        </label>
        <label className="field-stack">
          <span>Target host</span>
          <input
            aria-invalid={pathForm.formState.errors.targetHost ? 'true' : 'false'}
            className="field-input"
            placeholder="db.internal.example.com"
            {...pathForm.register('targetHost', {
              validate: (value) =>
                pathMode === 'upstream_pull' || value.trim() !== '' ? true : 'target host is required for direct or relay_chain'
            })}
          />
          {pathForm.formState.errors.targetHost ? <p className="error-text">{pathForm.formState.errors.targetHost.message}</p> : null}
        </label>
        <label className="field-stack">
          <span>Target port</span>
          <input
            aria-invalid={pathForm.formState.errors.targetPort ? 'true' : 'false'}
            className="field-input"
            placeholder="3306"
            type="number"
            {...pathForm.register('targetPort', {
              validate: (value) =>
                pathMode === 'upstream_pull' || Number(value) > 0 ? true : 'target port must be greater than 0 for direct or relay_chain'
            })}
          />
          {pathForm.formState.errors.targetPort ? <p className="error-text">{pathForm.formState.errors.targetPort.message}</p> : null}
        </label>
        <div className="submit-row">
          <button className="primary-button" disabled={createPathMutation.isPending} type="submit">
            {createPathMutation.isPending ? t('common.submitting') : 'Create path'}
          </button>
        </div>
      </form>
    </article>
  );
}
