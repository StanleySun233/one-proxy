'use client';

import {UseFormReturn} from 'react-hook-form';

import {NodeFormValues} from './types';

export function ManualNodeTab({
  form,
  submitting,
  onSubmit
}: {
  form: UseFormReturn<NodeFormValues>;
  submitting: boolean;
  onSubmit: () => void;
}) {
  return (
    <form className="nodes-form-grid" onSubmit={form.handleSubmit(onSubmit)}>
      <div className="field-stack">
        <span>Name</span>
        <input className="field-input" placeholder="relay-a" {...form.register('name', {required: 'name is required'})} />
        {form.formState.errors.name ? <p className="error-text">{form.formState.errors.name.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>Mode</span>
        <select className="field-select" {...form.register('mode', {required: true})}>
          <option value="relay">relay</option>
          <option value="edge">edge</option>
        </select>
      </div>
      <div className="field-stack">
        <span>Scope key</span>
        <input className="field-input" placeholder="cn-hz-a" {...form.register('scopeKey', {required: 'scope key is required'})} />
        {form.formState.errors.scopeKey ? <p className="error-text">{form.formState.errors.scopeKey.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>Parent node id</span>
        <input className="field-input" placeholder="optional upstream node id" {...form.register('parentNodeId')} />
      </div>
      <div className="field-stack">
        <span>Public host</span>
        <input className="field-input" placeholder="127.0.0.1" {...form.register('publicHost')} />
      </div>
      <div className="field-stack">
        <span>Public port</span>
        <input
          className="field-input"
          placeholder="2888"
          type="number"
          {...form.register('publicPort', {
            validate: (value) => {
              if (!value) {
                return true;
              }
              return Number(value) > 0 || 'public port must be greater than 0';
            }
          })}
        />
        {form.formState.errors.publicPort ? <p className="error-text">{form.formState.errors.publicPort.message}</p> : null}
      </div>
      <div className="submit-row nodes-form-full">
        <button className="primary-button" disabled={submitting} type="submit">
          {submitting ? 'Submitting' : 'Create node record'}
        </button>
        <p className="field-hint">Use this when you want IDs and topology records ready before the node comes online.</p>
      </div>
    </form>
  );
}
