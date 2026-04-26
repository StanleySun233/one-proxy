'use client';

import {UseFormReturn} from 'react-hook-form';

import {BootstrapFormValues} from './types';

export function BootstrapTokenTab({
  form,
  submitting,
  latestToken,
  onSubmit
}: {
  form: UseFormReturn<BootstrapFormValues>;
  submitting: boolean;
  latestToken: string;
  onSubmit: () => void;
}) {
  return (
    <form className="nodes-form-grid" onSubmit={form.handleSubmit(onSubmit)}>
      <div className="field-stack nodes-form-full">
        <span>Target node id</span>
        <input className="field-input" placeholder="optional existing node id" {...form.register('targetId')} />
        <p className="field-hint">Leave blank for a brand new node, or bind this token to a manually created record.</p>
      </div>
      <div className="submit-row nodes-form-full">
        <button className="primary-button" disabled={submitting} type="submit">
          {submitting ? 'Submitting' : 'Generate bootstrap token'}
        </button>
        <p className="field-hint">Remote node self-enroll flow for machines not directly reachable from the panel.</p>
      </div>
      {latestToken ? (
        <div className="token-box nodes-form-full">
          <strong>Bootstrap token</strong>
          <span className="mono">{latestToken}</span>
          <span className="field-hint">Run on the target machine after setting `CONTROL_PLANE_URL`, `NODE_BOOTSTRAP_TOKEN`, `NODE_NAME`, and `NODE_SCOPE_KEY`.</span>
        </div>
      ) : (
        <p className="field-hint nodes-form-full">Generate on demand. Token content is only kept in current page state.</p>
      )}
    </form>
  );
}
