'use client';

import {useMemo, useState} from 'react';
import {UseFormReturn} from 'react-hook-form';
import {toast} from 'sonner';

import {BootstrapToken, Node} from '@/lib/control-plane-types';
import {BootstrapFormValues} from './types';

export function BootstrapTokenTab({
  form,
  submitting,
  latestToken,
  nodes,
  onSubmit
}: {
  form: UseFormReturn<BootstrapFormValues>;
  submitting: boolean;
  latestToken: BootstrapToken | null;
  nodes: Node[];
  onSubmit: () => void;
}) {
  const [copied, setCopied] = useState('');
  const controlPlaneURL = useMemo(() => {
    if (typeof window === 'undefined') {
      return '';
    }
    return window.location.origin;
  }, []);
  const dockerCommand = useMemo(() => {
    if (!latestToken) {
      return '';
    }
    return `docker rm -f one-proxy-node >/dev/null 2>&1 || true && docker volume rm -f one-proxy-node-runtime >/dev/null 2>&1 || true && docker run -d --name one-proxy-node --restart unless-stopped -p 2988:2988 -p 2989:2989 -v one-proxy-node-runtime:/app/runtime -e CONTROL_PLANE_URL='${controlPlaneURL}' -e NODE_BOOTSTRAP_TOKEN='${latestToken.token}' -e NODE_SCOPE_KEY='scope-key' -e NODE_MODE='relay' -e NODE_JOIN_PASSWORD='password' -e TZ='Asia/Shanghai' ghcr.io/stanleysun233/one-proxy-node:latest`;
  }, [controlPlaneURL, latestToken]);

  async function copy(value: string, key: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(key);
      toast.success('copied');
    } catch {
      toast.error('copy_failed');
    }
  }

  const watchedNodeName = form.watch('nodeName');

  return (
    <form className="nodes-form-grid" onSubmit={form.handleSubmit(onSubmit)}>
      <div className="field-stack nodes-form-full">
        <span>Node name <span className="muted-text">(required)</span></span>
        <input
          className="field-input"
          placeholder="e.g. hk-gateway"
          {...form.register('nodeName', {
            required: 'Node name is required',
            validate: (value) => {
              if (!value) return true;
              const exists = nodes.some((n) => n.name.toLowerCase() === value.trim().toLowerCase());
              return exists ? 'A node with this name already exists' : true;
            }
          })}
        />
        {form.formState.errors.nodeName ? (
          <p className="field-error">{form.formState.errors.nodeName.message}</p>
        ) : null}
      </div>
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
        <div className="bootstrap-result-stack nodes-form-full">
          <div className="token-box">
            <div className="stack-head">
              <strong>Bootstrap token</strong>
              <button className="secondary-button" onClick={() => void copy(latestToken.token, 'token')} type="button">
                {copied === 'token' ? 'Copied' : 'Copy token'}
              </button>
            </div>
            <span className="mono">{latestToken.token}</span>
            <span className="field-hint">Token is shown once in this page state. Generate a new one if the machine was not enrolled.</span>
          </div>
          <div className="token-box">
            <div className="stack-head">
              <strong>Docker one-liner</strong>
              <button className="secondary-button" onClick={() => void copy(dockerCommand, 'docker')} type="button">
                {copied === 'docker' ? 'Copied' : 'Copy command'}
              </button>
            </div>
            <code className="mono command-block">{dockerCommand}</code>
            <span className="field-hint">Replace `NODE_SCOPE_KEY` before running on the target machine. The control-plane domain is taken from the current panel URL.</span>
          </div>
        </div>
      ) : (
        <p className="field-hint nodes-form-full">Generate on demand. Token content is only kept in current page state.</p>
      )}
    </form>
  );
}
