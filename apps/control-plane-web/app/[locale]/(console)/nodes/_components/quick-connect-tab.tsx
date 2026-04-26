'use client';

import {UseFormReturn} from 'react-hook-form';

import {QuickConnectFormValues} from './types';

export function QuickConnectTab({
  form,
  submitting,
  onSubmit
}: {
  form: UseFormReturn<QuickConnectFormValues>;
  submitting: boolean;
  onSubmit: () => void;
}) {
  return (
    <form className="nodes-form-grid" onSubmit={form.handleSubmit(onSubmit)}>
      <div className="field-stack">
        <span>Node address</span>
        <input className="field-input" placeholder="127.0.0.1:2888" {...form.register('address', {required: 'node address is required'})} />
        {form.formState.errors.address ? <p className="error-text">{form.formState.errors.address.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>Join password</span>
        <input className="field-input" type="password" placeholder="NODE_JOIN_PASSWORD" {...form.register('password', {required: 'join password is required'})} />
        {form.formState.errors.password ? <p className="error-text">{form.formState.errors.password.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>Name</span>
        <input className="field-input" placeholder="node-b" {...form.register('name', {required: 'name is required'})} />
        {form.formState.errors.name ? <p className="error-text">{form.formState.errors.name.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>Scope key</span>
        <input className="field-input" placeholder="scope-b" {...form.register('scopeKey', {required: 'scope key is required'})} />
        {form.formState.errors.scopeKey ? <p className="error-text">{form.formState.errors.scopeKey.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>Mode</span>
        <select className="field-select" {...form.register('mode', {required: true})}>
          <option value="relay">relay</option>
          <option value="edge">edge</option>
        </select>
      </div>
      <div className="field-stack">
        <span>Parent node id</span>
        <input className="field-input" placeholder="optional upstream node id" {...form.register('parentNodeId')} />
      </div>
      <div className="field-stack">
        <span>Public host</span>
        <input className="field-input" placeholder="optional, fallback to address host" {...form.register('publicHost')} />
      </div>
      <div className="field-stack">
        <span>Public port</span>
        <input className="field-input" placeholder="optional, fallback to address port" type="number" {...form.register('publicPort')} />
      </div>
      <div className="field-stack nodes-form-full">
        <span>Control plane URL</span>
        <input className="field-input" placeholder="http://127.0.0.1:2887" {...form.register('controlPlaneUrl', {required: 'control plane url is required'})} />
        {form.formState.errors.controlPlaneUrl ? <p className="error-text">{form.formState.errors.controlPlaneUrl.message}</p> : null}
      </div>
      <div className="submit-row nodes-form-full">
        <button className="primary-button" disabled={submitting} type="submit">
          {submitting ? 'Submitting' : 'Connect node'}
        </button>
        <p className="field-hint">Fast path for nodes already running with a join password.</p>
      </div>
    </form>
  );
}
