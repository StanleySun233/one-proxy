'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';
import {useMemo, useState} from 'react';
import {Edit, Share2, Trash2} from 'lucide-react';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {NameTag} from '@/components/common/name-tag';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {DeleteConfirmationModal, DeleteImpactSection} from '@/components/delete-confirmation-modal';
import {ResourceGrantModal} from '@/components/resource-grant-modal';
import {useAuth} from '@/components/auth-provider';
import {createChain, deleteChain, getChainDeleteImpact, getChains, getNodes, getScopes, previewChain, probeChain, updateChain} from '@/lib/api';
import {Chain, ChainDeleteImpact, ChainPreviewResult, ChainProbeResult, CompiledChainConfig} from '@/lib/types';
import {formatControlPlaneError} from '@/lib/presentation';

import {ChainEditor} from './_components/chain-editor';
import {CompilationPreviewModal} from './_components/compilation-preview-modal';

const knownProbeReasons = [
  'missing_entry_transport',
  'missing_parent_transport',
  'unknown_or_disabled_node',
  'probe_dispatch_failed',
  'chain_transport_ready',
  'chain_blocked',
  'chain_probe_failed',
  'relay_unreachable',
  'target_unreachable',
  'target_unhealthy',
  'invalid_target'
];

const silentProbeReasons = new Set(['chain_reachable', 'target_reachable']);

function probeReasonLabel(reason: string, chainsT: ReturnType<typeof useTranslations<'proxyChains'>>) {
  if (!reason) {
    return '';
  }
  if (silentProbeReasons.has(reason)) {
    return '';
  }
  if (knownProbeReasons.includes(reason)) {
    return chainsT(`probeReasons.${reason}`);
  }
  return reason;
}

function NodeTagPath({labels}: {labels: string[]}) {
  if (labels.length === 0) {
    return <span className="muted-text">-</span>;
  }
  return (
    <span className="tag-path">
      {labels.map((label, index) => (
        <span className="tag-path-step" key={`${label}-${index}`}>
          {index > 0 ? <span className="tag-path-arrow">→</span> : null}
          <NameTag kind="node">{label}</NameTag>
        </span>
      ))}
    </span>
  );
}

function ProbeResultToast({
  title,
  reason,
  hops,
  blockingNodeLabel,
  blocked
}: {
  title: string;
  reason: string;
  hops: ChainProbeResult['resolvedHops'];
  blockingNodeLabel: string;
  blocked: boolean;
}) {
  return (
    <div className={`probe-toast${blocked ? ' is-blocked' : ''}`}>
      <div className="probe-toast-head">
        <strong>{title}</strong>
        {reason ? <span>{reason}</span> : null}
      </div>
      {hops.length > 0 ? <NodeTagPath labels={hops.map((hop) => `${hop.nodeName}:${hop.transportType}`)} /> : null}
      {blockingNodeLabel ? <span className="muted-text">{blockingNodeLabel}</span> : null}
      <span className="probe-toast-progress" />
    </div>
  );
}

export default function ChainsPage() {
  const t = useTranslations();
  const chainsT = useTranslations('proxyChains');
  const {session, activeTenant} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';
  const globalSuperAdmin = session?.account.role === 'super_admin';

  const [editorOpen, setEditorOpen] = useState(false);
  const [editingChain, setEditingChain] = useState<Chain | null>(null);
  const [grantChain, setGrantChain] = useState<Chain | null>(null);
  const [chainName, setChainName] = useState('');
  const [destinationScope, setDestinationScope] = useState('');
  const [hops, setHops] = useState<string[]>([]);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewConfig, setPreviewConfig] = useState<CompiledChainConfig | null>(null);
  const [deletingChain, setDeletingChain] = useState<Chain | null>(null);
  const [deleteImpact, setDeleteImpact] = useState<ChainDeleteImpact | null>(null);
  const [nameFilter, setNameFilter] = useState('');
  const [destinationScopeFilter, setDestinationScopeFilter] = useState('');
  const [hopsFilter, setHopsFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');

  const chainsQuery = useQuery({
    queryKey: ['proxy-chains', accessToken, activeTenantId],
    queryFn: () => getChains(accessToken, activeTenantId),
    enabled: !!accessToken
  });

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken, activeTenantId],
    queryFn: () => getNodes(accessToken, activeTenantId),
    enabled: !!accessToken
  });

  const scopesQuery = useQuery({
    queryKey: ['proxy-scopes', accessToken, activeTenantId],
    queryFn: () => getScopes(accessToken, activeTenantId),
    enabled: !!accessToken
  });

  const createChainMutation = useMutation({
    mutationFn: (payload: {name: string; destinationScope: string; hops: string[]}) => createChain(accessToken, activeTenantId, payload),
    onSuccess: () => {
      toast.success(chainsT('createSuccess'));
      queryClient.invalidateQueries({queryKey: ['proxy-chains']});
      handleCloseEditor();
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const updateChainMutation = useMutation({
    mutationFn: (payload: {chainID: string; name: string; destinationScope: string; hops: string[]; enabled: boolean}) =>
      updateChain(accessToken, activeTenantId, payload.chainID, {
        name: payload.name,
        destinationScope: payload.destinationScope,
        hops: payload.hops,
        enabled: payload.enabled
      }),
    onSuccess: () => {
      toast.success(chainsT('updateSuccess'));
      queryClient.invalidateQueries({queryKey: ['proxy-chains']});
      handleCloseEditor();
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const probeChainMutation = useMutation({
    mutationFn: (chainID: string) => probeChain(accessToken, activeTenantId, chainID),
    onSuccess: (result) => {
      const blocked = result.status !== 'connected';
      const reason = blocked ? probeReasonLabel(result.blockingReason || result.message, chainsT) : '';
      const blockingNodeLabel = result.blockingNodeId ? `${chainsT('blockingNode')}: ${nodeLabelFor(result.blockingNodeId)}` : '';
      toast.custom(() => (
        <ProbeResultToast
          blocked={blocked}
          blockingNodeLabel={blockingNodeLabel}
          hops={result.resolvedHops}
          reason={reason}
          title={blocked ? chainsT('transportBlocked') : chainsT('transportReady')}
        />
      ), {duration: 5000, unstyled: true});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const previewMutation = useMutation({
    mutationFn: (payload: {name: string; destinationScope: string; hops: string[]}) => previewChain(accessToken, activeTenantId, payload),
    onSuccess: (result: ChainPreviewResult) => {
      setPreviewConfig(result.compiledConfig);
      setPreviewOpen(true);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const deleteImpactMutation = useMutation({
    mutationFn: (chainID: string) => getChainDeleteImpact(accessToken, activeTenantId, chainID),
    onSuccess: (result) => setDeleteImpact(result),
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
      setDeletingChain(null);
    }
  });

  const deleteChainMutation = useMutation({
    mutationFn: (chainID: string) => deleteChain(accessToken, activeTenantId, chainID),
    onSuccess: () => {
      toast.success(chainsT('deleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['proxy-chains']});
      queryClient.invalidateQueries({queryKey: ['route-rules']});
      queryClient.invalidateQueries({queryKey: ['proxy-access-paths']});
      setDeletingChain(null);
      setDeleteImpact(null);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const handleOpenEditor = (chain?: Chain) => {
    if (chain) {
      setEditingChain(chain);
      setChainName(chain.name);
      setDestinationScope(chain.destinationScope);
      setHops(chain.hops);
    } else {
      setEditingChain(null);
      setChainName('');
      setDestinationScope('');
      setHops([]);
    }
    setEditorOpen(true);
  };

  const handleCloseEditor = () => {
    setEditorOpen(false);
    setEditingChain(null);
    setChainName('');
    setDestinationScope('');
    setHops([]);
  };

  const handleSaveChain = () => {
    if (editingChain) {
      updateChainMutation.mutate({
        chainID: editingChain.id,
        name: chainName,
        destinationScope,
        hops,
        enabled: editingChain.enabled
      });
      return;
    }
    createChainMutation.mutate({
      name: chainName,
      destinationScope,
      hops
    });
  };

  const handlePreview = () => {
    previewMutation.mutate({
      name: chainName,
      destinationScope,
      hops
    });
  };

  const openDeleteChain = (chain: Chain) => {
    setDeletingChain(chain);
    setDeleteImpact(null);
    deleteImpactMutation.mutate(chain.id);
  };

  const chains = chainsQuery.data || [];
  const nodes = nodesQuery.data || [];
  const scopes = scopesQuery.data || [];
  const nodeNameById = useMemo(() => new Map(nodes.map((node) => [node.id, node.name])), [nodes]);
  const nodeLabelFor = (nodeId: string) => nodeNameById.get(nodeId) || t('common.unknown');
  const scopeNameById = useMemo(() => new Map(scopes.map((scope) => [scope.id, scope.name])), [scopes]);
  const scopeLabelFor = (scopeId: string) => scopeNameById.get(scopeId) || t('common.unknown');
  const filteredChains = useMemo(() => {
    return chains.filter((chain) =>
      (!nameFilter.trim() || chain.name.toLowerCase().includes(nameFilter.trim().toLowerCase())) &&
      (!destinationScopeFilter.trim() || scopeLabelFor(chain.destinationScope).toLowerCase().includes(destinationScopeFilter.trim().toLowerCase())) &&
      (!hopsFilter.trim() || chain.hops.map(nodeLabelFor).join(' ').toLowerCase().includes(hopsFilter.trim().toLowerCase())) &&
      (!statusFilter || (statusFilter === 'enabled' ? chain.enabled : !chain.enabled))
    );
  }, [chains, destinationScopeFilter, hopsFilter, nameFilter, nodeNameById, scopeNameById, statusFilter]);
  const chainDeleteSections: DeleteImpactSection[] = deleteImpact ? [
    {id: 'chain', label: chainsT('deleteImpactChain'), items: deleteImpact.delete.chain},
    {id: 'chainHops', label: chainsT('deleteImpactChainHops'), items: deleteImpact.delete.chainHops},
    {id: 'routeRules', label: chainsT('deleteImpactRouteRules'), items: deleteImpact.delete.routeRules},
    {id: 'accessPaths', label: chainsT('deleteImpactAccessPaths'), items: deleteImpact.delete.accessPaths},
    {id: 'onboardingTasks', label: chainsT('deleteImpactOnboardingTasks'), items: deleteImpact.delete.onboardingTasks},
    {id: 'chainProbeResults', label: chainsT('deleteImpactChainProbeResults'), items: deleteImpact.delete.chainProbeResults},
    {id: 'tenantBindings', label: chainsT('deleteImpactTenantBindings'), items: deleteImpact.delete.tenantBindings}
  ] : deletingChain ? [
    {id: 'chain', label: chainsT('deleteImpactChain'), count: 1}
  ] : [];

  return (
    <AuthGate>
      <ConsolePage
        actions={canWrite ? (
          <button className="primary-button" onClick={() => handleOpenEditor()} type="button">
            {chainsT('createChain')}
          </button>
        ) : null}
        title={t('shell.chainStudio')}
      >
        <ConsoleFilterBar title={t('common.filter')}>
          <ConsoleFilterItem label={t('common.name')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setNameFilter(event.target.value)} placeholder={t('common.name')} value={nameFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={chainsT('hops')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setHopsFilter(event.target.value)} placeholder={chainsT('hops')} value={hopsFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={chainsT('destinationScope')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setDestinationScopeFilter(event.target.value)} placeholder={chainsT('destinationScope')} value={destinationScopeFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.status')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
              <option value="">{t('common.all')}</option>
              <option value="enabled">{t('common.enabled')}</option>
              <option value="disabled">{t('common.disabled')}</option>
            </select>
          </ConsoleFilterItem>
        </ConsoleFilterBar>

        <ConsoleList count={filteredChains.length} title={chainsT('listTitle')}>
          {chainsQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={chainsT('loadingChains')} />
          ) : chainsQuery.isError ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(chainsQuery.error)}
              onAction={() => void chainsQuery.refetch()}
              title={chainsT('failedChains')}
            />
          ) : chains.length === 0 ? (
            <AsyncState detail={chainsT('emptyChains')} title={t('common.empty')} />
          ) : filteredChains.length === 0 ? (
            <AsyncState detail={t('common.noMatching')} title={t('common.empty')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('common.name')}</th>
                    <th>{chainsT('hops')}</th>
                    <th>{chainsT('destinationScope')}</th>
                    <th>{t('common.status')}</th>
                    <th>{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredChains.map((chain) => {
                    const canManage = globalSuperAdmin || chain.permission === 'manage';
                    return (
                      <tr key={chain.id}>
                        <td>
                          <NameTag kind="chain">{chain.name}</NameTag>
                        </td>
                        <td><NodeTagPath labels={chain.hops.map(nodeLabelFor)} /></td>
                        <td><NameTag kind="scope">{scopeLabelFor(chain.destinationScope)}</NameTag></td>
                        <td>
                          <span className={`badge ${chain.enabled ? 'is-good' : 'is-warn'}`}>{chain.enabled ? t('common.enabled') : t('common.disabled')}</span>
                        </td>
                        <td>
                          <div className="chain-list-actions">
                            {canWrite && canManage ? (
                              <button className="secondary-button" onClick={() => setGrantChain(chain)} type="button">
                                <Share2 size={14} />
                                {t('common.grant')}
                              </button>
                            ) : null}
                            {canWrite ? (
                              <button className="secondary-button" disabled={!canManage} onClick={() => handleOpenEditor(chain)} type="button">
                                <Edit size={14} />
                                {t('common.edit')}
                              </button>
                            ) : null}
                            <button
                              className="secondary-button"
                              disabled={probeChainMutation.isPending}
                              onClick={() => probeChainMutation.mutate(chain.id)}
                              type="button"
                            >
                              {chainsT('probe')}
                            </button>
                            {canWrite ? (
                              <button
                                className="danger-button"
                                disabled={!canManage || deleteChainMutation.isPending || deleteImpactMutation.isPending}
                                onClick={() => openDeleteChain(chain)}
                                type="button"
                              >
                                <Trash2 size={14} />
                                {t('common.delete')}
                              </button>
                            ) : null}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </ConsoleList>

        <ConsoleCrudModal
          onClose={handleCloseEditor}
          open={editorOpen}
          subtitle={chainsT('editorDesc')}
          title={editingChain ? chainsT('editChain') : chainsT('createChain')}
        >
            <ChainEditor
              accessToken={accessToken}
              activeTenantId={activeTenantId}
              chainName={chainName}
              destinationScope={destinationScope}
              hops={hops}
              nodes={nodes}
              scopes={scopes}
              onCancel={handleCloseEditor}
              onHopsChange={setHops}
              onNameChange={setChainName}
              onPreview={handlePreview}
              onSave={handleSaveChain}
              onScopeChange={setDestinationScope}
              previewing={previewMutation.isPending}
              saving={createChainMutation.isPending || updateChainMutation.isPending}
            />
        </ConsoleCrudModal>

        {previewOpen && previewConfig && (
          <CompilationPreviewModal config={previewConfig} onClose={() => setPreviewOpen(false)} />
        )}

        {grantChain ? (
          <ResourceGrantModal
            onChanged={() => queryClient.invalidateQueries({queryKey: ['proxy-chains']})}
            onClose={() => setGrantChain(null)}
            open={Boolean(grantChain)}
            resourceId={grantChain.id}
            resourceName={grantChain.name}
            resourceType="chain"
          />
        ) : null}

        <DeleteConfirmationModal
          onClose={() => {
            setDeletingChain(null);
            setDeleteImpact(null);
          }}
          onConfirm={() => {
            if (deletingChain) {
              deleteChainMutation.mutate(deletingChain.id);
            }
          }}
          open={Boolean(deletingChain)}
          pending={deleteChainMutation.isPending || deleteImpactMutation.isPending}
          sections={chainDeleteSections}
          targetName={deletingChain?.name || ''}
          title={chainsT('deleteConfirmTitle')}
        />
      </ConsolePage>
    </AuthGate>
  );
}
