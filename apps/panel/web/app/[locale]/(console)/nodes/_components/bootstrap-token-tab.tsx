'use client';

import {useEffect, useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import {UseFormReturn} from 'react-hook-form';
import {toast} from 'sonner';

import {getNodeReleaseTags} from '@/lib/api';
import {BootstrapToken, Node, NodeParentURLProbeResult, Scope} from '@/lib/types';
import {BootstrapFormValues} from './types';

const nodeImageRepo = 'ghcr.io/stanleysun233/oneproxy-node';
const buildNodeImage = process.env.NEXT_PUBLIC_ONEPROXY_NODE_IMAGE || '';
const releaseTagPattern = /^v\d+\.\d+\.\d+$/;
const buildNodeImageCandidate = buildNodeImage.startsWith(`${nodeImageRepo}:`) ? buildNodeImage.slice(nodeImageRepo.length + 1) : '';
const buildNodeImageTag = releaseTagPattern.test(buildNodeImageCandidate) ? buildNodeImageCandidate : '';

function shellQuote(value: string) {
  return `'${value.replace(/'/g, `'\\''`)}'`;
}

function highlightedBash(line: string) {
  const parts = line.split(/(docker|rm|run|true|ghcr\.io\/stanleysun233\/oneproxy-node:[^\s]+|--[a-z-]+|-e|-p|-v|\\|'[^']*'|\S+:[^\s]+)/g).filter(Boolean);

  return parts.map((part, index) => {
    let className = 'bash-plain';
    if (/^(docker|rm|run|true)$/.test(part)) {
      className = 'bash-command';
    } else if (/^(--[a-z-]+|-e|-p|-v)$/.test(part)) {
      className = 'bash-flag';
    } else if (/^'/.test(part)) {
      className = 'bash-string';
    } else if (part.includes(':') || part.includes('/')) {
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
  parentProbePending,
  parentProbeResult,
  nodes,
  scopes,
  onProbeParentURL,
  onSubmit
}: {
  form: UseFormReturn<BootstrapFormValues>;
  submitting: boolean;
  latestToken: BootstrapToken | null;
  parentProbePending: boolean;
  parentProbeResult: NodeParentURLProbeResult | null;
  nodes: Node[];
  scopes: Scope[];
  onProbeParentURL: (url: string) => void;
  onSubmit: (data: BootstrapFormValues) => void;
}) {
  const t = useTranslations();
  const [copied, setCopied] = useState('');
  const [selectedNodeImageTag, setSelectedNodeImageTag] = useState('');
  const controlPlaneURL = useMemo(() => {
    if (typeof window === 'undefined') {
      return '';
    }
    return window.location.origin;
  }, []);
  const releaseTagsQuery = useQuery({
    queryKey: ['node-release-tags'],
    queryFn: getNodeReleaseTags,
    staleTime: 300000
  });
  const selectedNodeMode = form.watch('nodeMode');
  const selectedScopeKey = form.watch('scopeKey');
  const selectedParentNodeId = form.watch('parentNodeId');
  const parentReachableUrl = form.watch('parentReachableUrl');
  const publicPort = form.watch('publicPort');
  const selectedParentNode = nodes.find((node) => node.id === selectedParentNodeId);
  const parentCandidates = useMemo(() => {
    const enabled = nodes.filter((node) => node.enabled);
    const ranked = [
      ...enabled.filter((node) => node.mode === 'edge' && node.scopeKey === selectedScopeKey),
      ...enabled.filter((node) => node.mode === 'edge' && node.scopeKey !== selectedScopeKey),
      ...enabled.filter((node) => node.mode !== 'edge' && node.scopeKey === selectedScopeKey),
      ...enabled.filter((node) => node.mode !== 'edge' && node.scopeKey !== selectedScopeKey)
    ];
    const seen = new Set<string>();
    return ranked.filter((node) => {
      if (seen.has(node.id)) {
        return false;
      }
      seen.add(node.id);
      return true;
    });
  }, [nodes, selectedScopeKey]);
  const autoParentReachableUrl = selectedParentNode?.publicHost
    ? `http://${selectedParentNode.publicHost}:${selectedParentNode.publicPort || 2988}`
    : '';
  const releaseTags = useMemo(
    () => releaseTagsQuery.data?.tags?.length ? releaseTagsQuery.data.tags : (buildNodeImageTag ? [buildNodeImageTag] : []),
    [releaseTagsQuery.data?.tags]
  );
  const selectedNodeImageRepo = releaseTagsQuery.data?.imageRepo || nodeImageRepo;
  const selectedNodeImage = selectedNodeImageTag ? `${selectedNodeImageRepo}:${selectedNodeImageTag}` : buildNodeImage;

  useEffect(() => {
    if (selectedNodeMode === 'relay') {
      if (parentCandidates.length > 0 && !parentCandidates.some((node) => node.id === selectedParentNodeId)) {
        form.setValue('parentNodeId', parentCandidates[0].id, {shouldDirty: true, shouldValidate: true});
      }
      return;
    }
    if (selectedParentNodeId) {
      form.setValue('parentNodeId', '', {shouldDirty: true, shouldValidate: true});
    }
  }, [form, parentCandidates, selectedNodeMode, selectedParentNodeId]);

  useEffect(() => {
    form.setValue('parentReachableUrl', autoParentReachableUrl, {shouldDirty: false, shouldValidate: true});
  }, [autoParentReachableUrl, form]);

  useEffect(() => {
    if (releaseTags.length === 0) {
      return;
    }
    const latestTag = releaseTagsQuery.data?.latestTag || releaseTags[0];
    if (!selectedNodeImageTag || !releaseTags.includes(selectedNodeImageTag)) {
      setSelectedNodeImageTag(latestTag);
    }
  }, [releaseTags, releaseTagsQuery.data?.latestTag, selectedNodeImageTag]);

  const dockerCommand = useMemo(() => {
    if (!latestToken) {
      return '';
    }
    const parentUrl = parentReachableUrl.trim();
    const bootstrapURL = selectedNodeMode === 'relay' ? parentUrl : controlPlaneURL;
    const hostPublicPort = publicPort.trim() || '2988';
    return [
      'docker rm -f one-proxy-node >/dev/null 2>&1 || true',
      'if docker volume inspect one-proxy-node-runtime >/dev/null 2>&1; then docker volume rm -f one-proxy-node-runtime; fi',
      'docker run -d --name one-proxy-node --restart unless-stopped \\',
      `  -p ${hostPublicPort}:2988 \\`,
      '  -p 2989:2989 \\',
      '  -v one-proxy-node-runtime:/app/runtime \\',
      `  -e NODE_MODE=${shellQuote(selectedNodeMode)} \\`,
      `  -e NODE_PARENT_URL=${shellQuote(bootstrapURL)} \\`,
      `  -e NODE_BOOTSTRAP_TOKEN=${shellQuote(latestToken.token)} \\`,
      `  ${selectedNodeImage}`
    ].join('\n');
  }, [controlPlaneURL, latestToken, parentReachableUrl, publicPort, selectedNodeImage, selectedNodeMode]);
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
            <option key={scope.id} value={scope.id}>{scope.name}</option>
          ))}
        </select>
      </div>
      <div className="field-stack">
        <span>{t('nodes.bootstrap.parentNodeId')}</span>
        <select className="field-input" disabled={selectedNodeMode !== 'relay'} {...form.register('parentNodeId')}>
          <option value="">{t('nodes.bootstrap.noParent')}</option>
          {parentCandidates.map((node) => (
            <option key={node.id} value={node.id}>{node.name}</option>
          ))}
        </select>
        <p className="field-hint">{t('nodes.bootstrap.parentNodeAutoHint')}</p>
      </div>
      {needsParentReachableUrl ? (
        <div className="field-stack nodes-form-full">
          <span>{t('nodes.bootstrap.parentReachableUrl')}</span>
          <div style={{display: 'flex', gap: 8, alignItems: 'center'}}>
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
            <button
              className="secondary-button"
              disabled={parentProbePending || !parentReachableUrl.trim()}
              onClick={() => onProbeParentURL(parentReachableUrl.trim())}
              type="button"
            >
              {parentProbePending ? t('nodes.bootstrap.testingParentUrl') : t('nodes.bootstrap.testParentUrl')}
            </button>
          </div>
          {parentProbeResult ? (
            <span className={`conn-status is-${parentProbeResult.reachable ? 'success' : 'failed'}`}>
              <span className="conn-status-dot" />
              {parentProbeResult.reachable ? t('nodes.bootstrap.parentUrlReachable') : t('nodes.bootstrap.parentUrlUnreachable')}
            </span>
          ) : null}
          {form.formState.errors.parentReachableUrl ? (
            <p className="field-error">{form.formState.errors.parentReachableUrl.message}</p>
          ) : null}
        </div>
      ) : null}
      <input type="hidden" {...form.register('publicHost')} />
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
        {selectedNodeMode === 'edge' ? (
          <p className="field-hint">{t('nodes.bootstrap.publicEndpointAutoHint')}</p>
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
            <label className="field-stack">
              <span>{t('nodes.bootstrap.nodeImageTag')}</span>
              <select
                className="field-select"
                disabled={releaseTagsQuery.isPending || releaseTags.length === 0}
                onChange={(event) => setSelectedNodeImageTag(event.target.value)}
                value={selectedNodeImageTag}
              >
                {releaseTags.map((tag) => (
                  <option key={tag} value={tag}>{tag}</option>
                ))}
              </select>
              <span className="field-hint">
                {releaseTagsQuery.isPending
                  ? t('nodes.bootstrap.loadingNodeImageTags')
                  : releaseTagsQuery.isError
                    ? t('nodes.bootstrap.nodeImageTagsFallback')
                    : t('nodes.bootstrap.nodeImageTagsHint')}
              </span>
            </label>
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
