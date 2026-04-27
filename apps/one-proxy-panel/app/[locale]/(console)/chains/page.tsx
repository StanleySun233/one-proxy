'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';
import {useState} from 'react';
import {Edit, Trash2} from 'lucide-react';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {createChain, getChains, getNodes, previewChain, probeChain} from '@/lib/control-plane-api';
import {Chain, ChainPreviewResult, ChainProbeResult, CompiledChainConfig} from '@/lib/control-plane-types';
import {formatControlPlaneError} from '@/lib/presentation';

import {ChainEditor} from './_components/chain-editor';
import {CompilationPreviewModal} from './_components/compilation-preview-modal';

export default function ChainsPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';

  const [editorOpen, setEditorOpen] = useState(false);
  const [editingChain, setEditingChain] = useState<Chain | null>(null);
  const [chainName, setChainName] = useState('');
  const [destinationScope, setDestinationScope] = useState('');
  const [hops, setHops] = useState<number[]>([]);
  const [probeResults, setProbeResults] = useState<Record<string, ChainProbeResult>>({});
  const [previewOpen, setPreviewOpen] = useState(false);
  const [previewConfig, setPreviewConfig] = useState<CompiledChainConfig | null>(null);

  const chainsQuery = useQuery({
    queryKey: ['chains', accessToken],
    queryFn: () => getChains(accessToken),
    enabled: !!accessToken
  });

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });

  const createChainMutation = useMutation({
    mutationFn: (payload: {name: string; destinationScope: string; hops: number[]}) => createChain(accessToken, payload),
    onSuccess: () => {
      toast.success('chain created');
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
      toast.success(result.status === 'connected' ? 'chain probe ready' : 'chain probe blocked');
      setProbeResults((current) => ({...current, [result.chainId]: result}));
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const previewMutation = useMutation({
    mutationFn: (payload: {name: string; destinationScope: string; hops: number[]}) => previewChain(accessToken, payload),
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
      setHops(chain.hops.map((h) => (typeof h === 'string' ? parseInt(h, 10) : h)));
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

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Chains" title={pageT('chainsTitle')} description={pageT('chainsDesc')} />

        {editorOpen ? (
          <section className="panel-card">
            <ChainEditor
              chainName={chainName}
              destinationScope={destinationScope}
              hops={hops}
              nodes={nodes}
              onCancel={handleCloseEditor}
              onHopsChange={setHops}
              onNameChange={setChainName}
              onPreview={handlePreview}
              onSave={handleSaveChain}
              onScopeChange={setDestinationScope}
              previewing={previewMutation.isPending}
              saving={createChainMutation.isPending}
            />
          </section>
        ) : (
          <section className="panel-card">
            <div className="panel-toolbar">
              <div>
                <p className="section-kicker">Management</p>
                <h3>Chain List</h3>
                <p className="section-copy">Manage relay chains and configure hop sequences</p>
              </div>
              <button className="primary-button" onClick={() => handleOpenEditor()} type="button">
                Create Chain
              </button>
            </div>

            {chainsQuery.isPending ? (
              <AsyncState detail={t('common.loading')} title="Loading chains" />
            ) : chainsQuery.isError ? (
              <AsyncState
                actionLabel={t('common.retry')}
                detail={formatControlPlaneError(chainsQuery.error)}
                onAction={() => void chainsQuery.refetch()}
                title="Failed to load chains"
              />
            ) : chains.length === 0 ? (
              <AsyncState detail="Create relay chains before binding route rules to them." title={t('common.empty')} />
            ) : (
              <div className="table-card">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>Name</th>
                      <th>Hops</th>
                      <th>Destination Scope</th>
                      <th>Status</th>
                      <th>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {chains.map((chain) => (
                      <tr key={chain.id}>
                        <td className="mono">{chain.id}</td>
                        <td>
                          <strong>{chain.name}</strong>
                        </td>
                        <td className="mono">{chain.hops.join(' → ')}</td>
                        <td>{chain.destinationScope}</td>
                        <td>
                          <span className={`badge ${chain.enabled ? 'is-good' : 'is-warn'}`}>{chain.enabled ? 'enabled' : 'disabled'}</span>
                        </td>
                        <td>
                          <div className="chain-list-actions">
                            <button className="secondary-button" onClick={() => handleOpenEditor(chain)} type="button">
                              <Edit size={14} />
                              Edit
                            </button>
                            <button
                              className="secondary-button"
                              disabled={probeChainMutation.isPending}
                              onClick={() => probeChainMutation.mutate(chain.id)}
                              type="button"
                            >
                              Probe
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {Object.keys(probeResults).length > 0 && (
              <div className="probe-results-section">
                <h4>Probe Results</h4>
                {Object.entries(probeResults).map(([chainId, result]) => (
                  <div className="token-box" key={chainId}>
                    <strong>{result.status === 'connected' ? 'Transport ready' : 'Transport blocked'}</strong>
                    <span className="field-hint">{result.blockingReason || result.message}</span>
                    {result.resolvedHops.length > 0 && (
                      <span className="mono">{result.resolvedHops.map((hop) => `${hop.nodeName}:${hop.transportType}`).join(' → ')}</span>
                    )}
                    {result.blockingNodeId && <span className="muted-text">blocking node: {result.blockingNodeId}</span>}
                  </div>
                ))}
              </div>
            )}
          </section>
        )}

        {previewOpen && previewConfig && (
          <CompilationPreviewModal config={previewConfig} onClose={() => setPreviewOpen(false)} />
        )}
      </div>
    </AuthGate>
  );
}
