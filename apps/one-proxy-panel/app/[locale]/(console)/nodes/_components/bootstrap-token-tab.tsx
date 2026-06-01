'use client';

import {useMemo, useState} from 'react';
import {useTranslations} from 'next-intl';
import {UseFormReturn} from 'react-hook-form';
import {toast} from 'sonner';

import {BootstrapToken, Node} from '@/lib/types';
import {BootstrapFormValues} from './types';

function shellQuote(value: string) {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}

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
  onSubmit: (data: BootstrapFormValues) => void;
}) {
  const t = useTranslations();
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
    return `docker rm -f one-proxy-node >/dev/null 2>&1 || true && docker volume rm -f one-proxy-node-runtime >/dev/null 2>&1 || true && docker run -d --name one-proxy-node --restart unless-stopped -p 2988:2988 -p 2989:2989 -v one-proxy-node-runtime:/app/runtime -e CONTROL_PLANE_URL=${shellQuote(controlPlaneURL)} -e NODE_BOOTSTRAP_TOKEN=${shellQuote(latestToken.token)} -e NODE_NAME=${shellQuote(latestToken.nodeName)} -e NODE_SCOPE_KEY='scope-key' -e NODE_MODE='relay' -e NODE_JOIN_PASSWORD='password' -e TZ='Asia/Shanghai' ghcr.io/stanleysun233/one-proxy-node:latest`;
  }, [controlPlaneURL, latestToken]);

  async function copy(value: string, key: string) {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(value);
      } else {
        const textarea = document.createElement('textarea');
        textarea.value = value;
        textarea.setAttribute('readonly', '');
        textarea.style.position = 'fixed';
        textarea.style.left = '-9999px';
        document.body.appendChild(textarea);
        try {
          textarea.select();
          if (!document.execCommand('copy')) {
            throw new Error('copy_failed');
          }
        } finally {
          document.body.removeChild(textarea);
        }
      }
      setCopied(key);
      toast.success(t('common.copied'));
    } catch {
      toast.error(t('common.copyFailed'));
    }
  }

  return (
    <div className="nodes-form-grid">
      <div className="field-stack nodes-form-full">
        <span>{t('nodes.bootstrap.nodeName')} <span className="muted-text">({t('common.required')})</span></span>
        <input
          className="field-input"
          placeholder="e.g. hk-gateway"
          {...form.register('nodeName', {
            required: t('nodes.bootstrap.nodeNameRequired'),
            validate: (value) => {
              const candidate = String(value || '').trim().toLowerCase();
              if (!candidate) return true;
              const exists = nodes.some((n) => String(n.name || '').toLowerCase() === candidate);
              return exists ? t('nodes.bootstrap.nodeNameDuplicate') : true;
            }
          })}
        />
        {form.formState.errors.nodeName ? (
          <p className="field-error">{form.formState.errors.nodeName.message}</p>
        ) : null}
      </div>
      <div className="field-stack nodes-form-full">
        <span>{t('nodes.bootstrap.targetNodeId')}</span>
        <input className="field-input" placeholder={t('nodes.bootstrap.targetNodeIdHint')} {...form.register('targetId')} />
        <p className="field-hint">{t('nodes.bootstrap.targetNodeIdHint')}</p>
      </div>
      <div className="submit-row nodes-form-full">
        <button className="primary-button" disabled={submitting} onClick={() => void form.handleSubmit(onSubmit)()} type="button">
          {submitting ? t('nodes.bootstrap.submitting') : t('nodes.bootstrap.generateToken')}
        </button>
        <p className="field-hint">{t('nodes.bootstrap.bootstrapHint')}</p>
      </div>
      {latestToken ? (
        <div className="bootstrap-result-stack nodes-form-full">
          <div className="token-box">
            <div className="stack-head">
              <strong>{t('nodes.bootstrap.bootstrapToken')}</strong>
              <button className="secondary-button" onClick={() => void copy(latestToken.token, 'token')} type="button">
                {copied === 'token' ? t('nodes.bootstrap.copied') : t('nodes.bootstrap.copyToken')}
              </button>
            </div>
            <span className="mono">{latestToken.token}</span>
            <span className="field-hint">{t('nodes.bootstrap.tokenShownOnce')}</span>
          </div>
          <div className="token-box">
            <div className="stack-head">
              <strong>{t('nodes.bootstrap.dockerOneLiner')}</strong>
              <button className="secondary-button" onClick={() => void copy(dockerCommand, 'docker')} type="button">
                {copied === 'docker' ? t('nodes.bootstrap.copied') : t('nodes.bootstrap.copyCommand')}
              </button>
            </div>
            <code className="mono command-block">{dockerCommand}</code>
            <span className="field-hint">{t('nodes.bootstrap.dockerScopeHint')}</span>
          </div>
        </div>
      ) : (
        <p className="field-hint nodes-form-full">{t('nodes.bootstrap.generateOnDemand')}</p>
      )}
    </div>
  );
}
