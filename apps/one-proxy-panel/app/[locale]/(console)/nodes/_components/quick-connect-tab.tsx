'use client';

import {UseFormReturn} from 'react-hook-form';
import {useQuery} from '@tanstack/react-query';

import {fetchEnums} from '@/lib/control-plane-api';
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
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const modeOptions = enums?.node_mode ? Object.entries(enums.node_mode).map(([value, name]) => ({value, label: name})) : [];
  return (
    <form className="nodes-form-grid" onSubmit={form.handleSubmit(onSubmit)}>
      <div className="field-stack">
        <span>Node address</span>
        <input className="field-input" placeholder="relay.example.com:2988" {...form.register('address', {required: 'node address is required'})} />
        {form.formState.errors.address ? <p className="error-text">{form.formState.errors.address.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>Join password</span>
        <input className="field-input" type="password" placeholder="password" {...form.register('password', {required: 'join password is required'})} />
        {form.formState.errors.password ? <p className="error-text">{form.formState.errors.password.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>New join password</span>
        <input className="field-input" type="password" placeholder="required when node still uses default password" {...form.register('newPassword')} />
        <p className="field-hint">If the node started without `NODE_JOIN_PASSWORD`, enter `password` above and set the replacement password here.</p>
      </div>
      <div className="field-stack">
        <span>Name</span>
        <input className="field-input" placeholder="node-b" {...form.register('name', {required: 'name is required'})} />
        {form.formState.errors.name ? <p className="error-text">{form.formState.errors.name.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>Scope key</span>
        <input className="field-input" placeholder="target-scope" {...form.register('scopeKey', {required: 'scope key is required'})} />
        {form.formState.errors.scopeKey ? <p className="error-text">{form.formState.errors.scopeKey.message}</p> : null}
      </div>
      <div className="field-stack">
        <span>Mode</span>
        <select className="field-select" {...form.register('mode', {required: true})}>
          {modeOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
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
        <input className="field-input" placeholder="https://panel.example.com" {...form.register('controlPlaneUrl', {required: 'control plane url is required'})} />
        {form.formState.errors.controlPlaneUrl ? <p className="error-text">{form.formState.errors.controlPlaneUrl.message}</p> : null}
      </div>
      <div className="submit-row nodes-form-full">
        <button className="primary-button" disabled={submitting} type="submit">
          {submitting ? 'Submitting' : 'Connect node'}
        </button>
        <p className="field-hint">Nodes started without `NODE_JOIN_PASSWORD` now default to `password` and must rotate it during first panel connect.</p>
      </div>
    </form>
  );
}
