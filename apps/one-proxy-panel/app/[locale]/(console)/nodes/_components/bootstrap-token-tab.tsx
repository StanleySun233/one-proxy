'use client';

import {useEffect, useMemo, useState} from 'react';
import {useTranslations} from 'next-intl';
import {UseFormReturn} from 'react-hook-form';
import {toast} from 'sonner';

import {BootstrapToken, Node, Scope} from '@/lib/types';
import {BootstrapFormValues} from './types';

function shellQuote(value: string) {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}

function highlightedBash(line: string) {
  const parts = line.split(/(docker|rm|run|true|ghcr\.io\/stanleysun233\/one-proxy-node:latest|--[a-z-]+|-e|-p|-v|\\|'[^']*'|\S+:[^\s]+)/g).filter(Boolean);

  return parts.map((part, index) => {
    let className = 'bash-plain';
    if (/^(docker|rm|run|true)$/.test(part)) {
      className = 'bash-command';
    } else if (/^(--[a-z-]+|-e|-p|-v)$/.test(part)) {
      className = 'bash-flag';
    } else if (/^'/.test(part)) {
      className = 'bash-string';
    } else if (part.includes(':') || part.startsWith('ghcr.io/')) {
      className = 'bash-target';
    } else if (part === '\\') {
      className = 'bash-continuation';
    }

    return <span className={className} key={`${part}-${index}`}>{part}</span>;
  });
}

export function BootstrapTokenTab({
  form,
  submitting,
  latestToken,
  nodes,
  scopes,
  onSubmit
}: {
  form: UseFormReturn<BootstrapFormValues>;
  submitting: boolean;
  latestToken: BootstrapToken | null;
  nodes: Node[];
  scopes: Scope[];
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
  const selectedNodeMode = form.watch('nodeMode');
  const selectedParentNodeId = form.watch('parentNodeId');
  const parentReachableUrl = form.watch('parentReachableUrl');
  const publicPort = form.watch('publicPort');
  const selectedParentNode = nodes.find((node) => node.id === selectedParentNodeId);
  const autoParentReachableUrl = selectedParentNode?.publicHost
    ? `http://${selectedParentNode.publicHost}:${selectedParentNode.publicPort || 2988}`
    : '';

  useEffect(() => {
    form.setValue('parentReachableUrl', autoParentReachableUrl, {shouldDirty: false, shouldValidate: true});
  }, [autoParentReachableUrl, form]);

  const dockerCommand = useMemo(() => {
    if (!latestToken) {
      return '';
    }
    const parentUrl = parentReachableUrl.trim();
    const parentID = selectedParentNodeId.trim();
    const parentEndpointLines = parentUrl
      ? [
        `  -e NODE_PARENT_URL=${shellQuote(parentUrl)} \\`,
        `  -e NODE_PARENT_TUNNEL_URL=${shellQuote(parentUrl)} \\`
      ]
      : [];
    const parentLines = parentID ? [`  -e NODE_PARENT_ID=${shellQuote(parentID)} \\`] : [];
    const hostPublicPort = publicPort.trim() || '2988';
    return [
      'docker rm -f one-proxy-node >/dev/null 2>&1 || true',
      'docker volume rm -f one-proxy-node-runtime >/dev/null 2>&1 || true',
      'docker run -d --name one-proxy-node --restart unless-stopped \\',
      `  -p ${hostPublicPort}:2988 \\`,
      '  -p 2989:2989 \\',
      '  -v one-proxy-node-runtime:/app/runtime \\',
      `  -e CONTROL_PLANE_URL=${shellQuote(controlPlaneURL)} \\`,
      ...parentEndpointLines,
      ...parentLines,
      `  -e NODE_BOOTSTRAP_TOKEN=${shellQuote(latestToken.token)} \\`,
      "  -e TZ='Asia/Shanghai' \\",
      '  ghcr.io/stanleysun233/one-proxy-node:latest'
    ].join('\n');
  }, [controlPlaneURL, latestToken, parentReachableUrl, publicPort, selectedParentNodeId]);
  const dockerCommandLines = useMemo(() => dockerCommand.split('\n'), [dockerCommand]);
  const needsParentReachableUrl = selectedNodeMode === 'relay' && selectedParentNodeId.trim() !== '';

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
          placeholder={t('nodes.bootstrap.nodeNamePlaceholder')}
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
        <span>{t('nodes.bootstrap.nodeMode')} <span className="muted-text">({t('common.required')})</span></span>
        <div className="node-mode-picker">
          {(['edge', 'relay'] as const).map((mode) => (
            <label className={`node-mode-option ${selectedNodeMode === mode ? 'is-selected' : ''}`} key={mode}>
              <input
                type="radio"
                value={mode}
                {...form.register('nodeMode', {required: t('nodes.bootstrap.nodeModeRequired')})}
              />
              <span className="node-mode-title">{t(`nodes.bootstrap.nodeModeOptions.${mode}.label`)}</span>
              <span className="node-mode-copy">{t(`nodes.bootstrap.nodeModeOptions.${mode}.hint`)}</span>
            </label>
          ))}
        </div>
        {form.formState.errors.nodeMode ? (
          <p className="field-error">{form.formState.errors.nodeMode.message}</p>
        ) : null}
      </div>
      <div className="field-stack">
        <span>{t('nodes.bootstrap.scopeKey')} <span className="muted-text">({t('common.required')})</span></span>
        <select className="field-select" {...form.register('scopeKey', {required: t('nodes.bootstrap.scopeKeyRequired')})}>
          <option value="">{t('nodes.bootstrap.scopeKeyPlaceholder')}</option>
          {scopes.map((scope) => (
            <option key={scope.id} value={scope.id}>{scope.name} ({scope.id})</option>
          ))}
        </select>
      </div>
      <div className="field-stack">
        <span>{t('nodes.bootstrap.parentNodeId')}</span>
        <select className="field-input" {...form.register('parentNodeId')}>
          <option value="">{t('nodes.bootstrap.noParent')}</option>
          {nodes.map((node) => (
            <option key={node.id} value={node.id}>{node.id} - {node.name}</option>
          ))}
        </select>
      </div>
      {needsParentReachableUrl ? (
        <div className="field-stack nodes-form-full">
          <span>{t('nodes.bootstrap.parentReachableUrl')}</span>
          <input
            className="field-input"
            placeholder={t('nodes.bootstrap.parentReachableUrlPlaceholder')}
            {...form.register('parentReachableUrl', {
              validate: (value) => {
                if (!needsParentReachableUrl) return true;
                return value.trim() !== '' ? true : t('nodes.bootstrap.parentReachableUrlRequired');
              }
            })}
          />
          {form.formState.errors.parentReachableUrl ? (
            <p className="field-error">{form.formState.errors.parentReachableUrl.message}</p>
          ) : null}
        </div>
      ) : null}
      <div className="field-stack">
        <span>{t('nodes.bootstrap.publicHost')}</span>
        <input className="field-input" placeholder={t('nodes.bootstrap.publicHostPlaceholder')} {...form.register('publicHost')} />
      </div>
      <div className="field-stack">
        <span>{t('nodes.bootstrap.publicPort')}</span>
        <input
          className="field-input"
          inputMode="numeric"
          placeholder={t('nodes.bootstrap.publicPortPlaceholder')}
          {...form.register('publicPort', {
            validate: (value) => {
              if (!value) return true;
              const port = Number(value);
              return Number.isInteger(port) && port > 0 ? true : t('nodes.bootstrap.publicPortInvalid');
            }
          })}
        />
        {form.formState.errors.publicPort ? (
          <p className="field-error">{form.formState.errors.publicPort.message}</p>
        ) : null}
      </div>
      <div className="field-stack nodes-form-full">
        <span>{t('nodes.bootstrap.targetNodeId')}</span>
        <input className="field-input" placeholder={t('nodes.bootstrap.targetNodeIdHint')} {...form.register('targetId')} />
      </div>
      <div className="submit-row nodes-form-full">
        <button className="primary-button" disabled={submitting} onClick={() => void form.handleSubmit(onSubmit)()} type="button">
          {submitting ? t('nodes.bootstrap.submitting') : t('nodes.bootstrap.generateToken')}
        </button>
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
            <div className="command-block" role="region" aria-label={t('nodes.bootstrap.dockerOneLiner')}>
              <div className="command-gutter" aria-hidden="true">
                {dockerCommandLines.map((_, index) => (
                  <span key={index}>{index + 1}</span>
                ))}
              </div>
              <code className="mono command-code">
                {dockerCommandLines.map((line, index) => (
                  <span className="command-line" key={`${line}-${index}`}>
                    {highlightedBash(line)}
                  </span>
                ))}
              </code>
            </div>
          </div>
        </div>
      ) : (
        null
      )}
    </div>
  );
}
