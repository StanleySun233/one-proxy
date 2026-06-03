'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';
import {useMemo, useState} from 'react';
import {Edit, Trash2} from 'lucide-react';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {NameTag} from '@/components/common/name-tag';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {useAuth} from '@/components/auth-provider';
import {createChain, getChains, getNodes, getScopes, previewChain, probeChain, updateChain} from '@/lib/api';
import {Chain, ChainPreviewResult, ChainProbeResult, CompiledChainConfig} from '@/lib/types';
import {formatControlPlaneError} from '@/lib/presentation';

import {ChainEditor} from './_components/chain-editor';
import {CompilationPreviewModal} from './_components/compilation-preview-modal';

function probeReasonLabel(reason: string, chainsT: ReturnType<typeof useTranslations<'chains'>>) {
  if (!reason) {
    return '';
  }
  if (['missing_entry_transport', 'missing_parent_transport', 'unknown_or_disabled_node', 'probe_dispatch_failed', 'chain_transport_ready', 'chain_blocked'].includes(reason)) {
    return chainsT(`probeReasons.${reason}`);
  }
  return reason;
}

export default function ChainsPage() {
  const t = useTranslations();
  const chainsT = useTranslations('chains');
  const {session, activeTenant} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';

  const [editorOpen, setEditorOpen] = useState(false);
  const [editingChain, setEditingChain] = useState<Chain | null>(null);
  const [chainName, setChainName] = useState('');
  const [destinationScope, setDestinationScope] = useState('');
  const [hops, setHops] = useState<string[]>([]);
  const [probeResults, setProbeResults] = useState<Record<string, ChainProbeResult>>({});
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewConfig, setPreviewConfig] = useState<CompiledChainConfig | null>(null);
  const [search, setSearch] = useState('');

  const chainsQuery = useQuery({
    queryKey: ['chains', accessToken, activeTenantId],
    queryFn: () => getChains(accessToken),
    enabled: !!accessToken
  });

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken, activeTenantId],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });

  const scopesQuery = useQuery({
    queryKey: ['chains-scopes', accessToken, activeTenantId],
    queryFn: () => getScopes(accessToken),
    enabled: !!accessToken
  });

  const createChainMutation = useMutation({
    mutationFn: (payload: {name: string; destinationScope: string; hops: string[]}) => createChain(accessToken, payload),
    onSuccess: () => {
      toast.success(chainsT('createSuccess'));
      queryClient.invalidateQueries({queryKey: ['chains']});
      handleCloseEditor();
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const updateChainMutation = useMutation({
    mutationFn: (payload: {chainID: string; name: string; destinationScope: string; hops: string[]; enabled: boolean}) =>
      updateChain(accessToken, payload.chainID, {
        name: payload.name,
        destinationScope: payload.destinationScope,
        hops: payload.hops,
        enabled: payload.enabled
      }),
    onSuccess: () => {
      toast.success(chainsT('updateSuccess'));
      queryClient.invalidateQueries({queryKey: ['chains']});
      handleCloseEditor();
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const probeChainMutation = useMutation({
    mutationFn: (chainID: string) => probeChain(accessToken, chainID),
    onSuccess: (result) => {
      toast.success(result.status === 'connected' ? chainsT('probeReady') : chainsT('probeBlocked'));
      setProbeResults((current) => ({...current, [result.chainId]: result}));
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const previewMutation = useMutation({
    mutationFn: (payload: {name: string; destinationScope: string; hops: string[]}) => previewChain(accessToken, payload),
    onSuccess: (result: ChainPreviewResult) => {
      setPreviewConfig(result.compiledConfig);
      setPreviewOpen(true);
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

  const chains = chainsQuery.data || [];
  const nodes = nodesQuery.data || [];
  const scopes = scopesQuery.data || [];
  const filteredChains = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    if (!keyword) {
      return chains;
    }
    return chains.filter((chain) =>
      [chain.id, chain.name, chain.destinationScope, chain.hops.join(' ')]
        .some((value) => String(value || '').toLowerCase().includes(keyword))
    );
  }, [chains, search]);

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
          <ConsoleFilterItem label={`${t('common.name')} / ${chainsT('destinationScope')} / ${chainsT('hops')}`} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setSearch(event.target.value)} placeholder={t('common.searchPlaceholder')} value={search} />
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
                    <th>{t('common.id')}</th>
                    <th>{t('common.name')}</th>
                    <th>{chainsT('hops')}</th>
                    <th>{chainsT('destinationScope')}</th>
                    <th>{t('common.status')}</th>
                    <th>{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredChains.map((chain) => (
                    <tr key={chain.id}>
                      <td className="mono">{chain.id}</td>
                      <td>
                        <NameTag kind="chain">{chain.name}</NameTag>
                      </td>
                      <td className="mono">{chain.hops.join(' → ')}</td>
                      <td><NameTag kind="scope">{chain.destinationScope}</NameTag></td>
                      <td>
                        <span className={`badge ${chain.enabled ? 'is-good' : 'is-warn'}`}>{chain.enabled ? t('common.enabled') : t('common.disabled')}</span>
                      </td>
                      <td>
                        <div className="chain-list-actions">
                          {canWrite ? (
                            <button className="secondary-button" onClick={() => handleOpenEditor(chain)} type="button">
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
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </ConsoleList>

        {Object.keys(probeResults).length > 0 && (
          <ConsoleList count={Object.keys(probeResults).length} title={chainsT('probeResults')}>
            <div className="stack-list">
              {Object.entries(probeResults).map(([chainId, result]) => (
                <div className="stack-item" key={chainId}>
                  <strong>{result.status === 'connected' ? chainsT('transportReady') : chainsT('transportBlocked')}</strong>
                  <span className="field-hint">{probeReasonLabel(result.blockingReason || result.message, chainsT)}</span>
                  {result.resolvedHops.length > 0 && (
                    <span className="mono">{result.resolvedHops.map((hop) => `${hop.nodeName}:${hop.transportType}`).join(' → ')}</span>
                  )}
                  {result.blockingNodeId && <span className="muted-text">{chainsT('blockingNode')}: {result.blockingNodeId}</span>}
                </div>
              ))}
            </div>
          </ConsoleList>
        )}

        <ConsoleCrudModal
          onClose={handleCloseEditor}
          open={editorOpen}
          subtitle={chainsT('editorDesc')}
          title={editingChain ? chainsT('editChain') : chainsT('createChain')}
        >
            <ChainEditor
              accessToken={accessToken}
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
      </ConsolePage>
    </AuthGate>
  );
}
