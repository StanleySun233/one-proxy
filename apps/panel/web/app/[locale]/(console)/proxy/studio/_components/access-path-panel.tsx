'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {Edit, Share2, Trash2} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {useEffect, useMemo, useState} from 'react';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList} from '@/components/console-template';
import {DeleteConfirmationModal, DeleteImpactSection} from '@/components/delete-confirmation-modal';
import {NameTag} from '@/components/common/name-tag';
import {ResourceGrantModal} from '@/components/resource-grant-modal';
import {createNodeAccessPath, deleteNodeAccessPath, fetchEnums, getNodeAccessPaths, getNodeAccessPathDeleteImpact, updateNodeAccessPath} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';
import type {Chain, FieldEnumEntry, Node, NodeAccessPath, NodeAccessPathDeleteImpact, NodeAccessPathPayload} from '@/lib/types';

type AccessPathFormState = {
  chainId: string;
  name: string;
  protocol: string;
  listenHost: string;
  listenPort: string;
  targetHost: string;
  targetPort: string;
  targetSni: string;
  tlsMode: string;
  authMode: string;
  enabled: boolean;
};

type AccessPathHealth = {
  status: 'available' | 'degraded' | 'unavailable' | 'unknown';
  reason: string;
  checkedAt: string;
};

type NodeAccessPathWithHealth = NodeAccessPath & {
  health?: AccessPathHealth;
};

const emptyForm: AccessPathFormState = {
  chainId: '',
  name: '',
  protocol: 'tcp',
  listenHost: '0.0.0.0',
  listenPort: '',
  targetHost: '',
  targetPort: '',
  targetSni: '',
  tlsMode: 'none',
  authMode: 'proxy_token',
  enabled: true
};

function enumOptions(values?: Record<string, FieldEnumEntry>) {
  return values ? Object.entries(values).map(([value, item]) => ({value, label: item.name})) : [];
}

function serviceTypeForProtocol(protocol: string) {
  const mapping: Record<string, string> = {
    http: 'http',
    tcp: 'raw_tcp',
    tls: 'tls_passthrough',
    ssh: 'ssh',
    rdp: 'rdp',
    socks5: 'socks5',
    ss5: 'ss5',
    udp: 'raw_udp'
  };
  return mapping[protocol] || 'raw_tcp';
}

function targetProtocolFor(protocol: string) {
  return protocol === 'tls' ? 'tcp' : protocol;
}

function pathFormValues(path: NodeAccessPath): AccessPathFormState {
  return {
    chainId: path.chainId,
    name: path.name,
    protocol: path.protocol,
    listenHost: path.listenHost || '0.0.0.0',
    listenPort: String(path.listenPort || ''),
    targetHost: path.targetHost || '',
    targetPort: String(path.targetPort || ''),
    targetSni: path.targetSni || '',
    tlsMode: path.tlsMode || 'none',
    authMode: path.authMode || 'proxy_token',
    enabled: path.enabled
  };
}

function portNumber(value: string) {
  return Number(value.trim());
}

function isValidPort(value: string) {
  const port = portNumber(value);
  return Number.isInteger(port) && port >= 1 && port <= 65535;
}

function chainNodeIds(chain?: Chain) {
  return chain?.hops || [];
}

function accessPathHealth(path: NodeAccessPath) {
  return (path as NodeAccessPathWithHealth).health;
}

function accessPathHealthBadgeClassName(status?: string) {
  if (status === 'available') {
    return 'badge is-good';
  }
  if (status === 'degraded') {
    return 'badge is-warn';
  }
  if (status === 'unavailable') {
    return 'badge is-danger';
  }
  return 'badge is-neutral';
}

function accessPathHealthTitle(health?: AccessPathHealth) {
  return [health?.reason, health?.checkedAt].filter(Boolean).join(' | ') || health?.status || 'unknown';
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

function submitPayload(form: AccessPathFormState, chains: Chain[]): NodeAccessPathPayload {
  const chain = chains.find((item) => item.id === form.chainId);
  const hops = chainNodeIds(chain);
  return {
    chainId: form.chainId,
    name: form.name.trim(),
    mode: 'relay_chain',
    protocol: form.protocol as NodeAccessPathPayload['protocol'],
    serviceType: serviceTypeForProtocol(form.protocol) as NodeAccessPathPayload['serviceType'],
    targetNodeId: hops[hops.length - 1] || '',
    entryNodeId: hops[0] || '',
    relayNodeIds: hops,
    listenHost: form.listenHost.trim(),
    listenPort: portNumber(form.listenPort),
    targetProtocol: targetProtocolFor(form.protocol),
    targetHost: form.targetHost.trim(),
    targetPort: portNumber(form.targetPort),
    targetSni: form.targetSni.trim(),
    tlsMode: form.tlsMode as NodeAccessPathPayload['tlsMode'],
    authMode: form.authMode as NodeAccessPathPayload['authMode'],
    options: {},
    enabled: form.enabled
  };
}

export function AccessPathPanel({
  accessToken,
  activeTenantId,
  canWrite,
  chains,
  createRequestKey = 0,
  globalSuperAdmin,
  nodes
}: {
  accessToken: string;
  activeTenantId: string | null;
  canWrite: boolean;
  chains: Chain[];
  createRequestKey?: number;
  globalSuperAdmin: boolean;
  nodes: Node[];
}) {
  const t = useTranslations();
  const accessPathsT = useTranslations('accessPaths');
  const queryClient = useQueryClient();
  const [editingPath, setEditingPath] = useState<NodeAccessPath | null>(null);
  const [grantPath, setGrantPath] = useState<NodeAccessPath | null>(null);
  const [deletingPath, setDeletingPath] = useState<NodeAccessPath | null>(null);
  const [deleteImpact, setDeleteImpact] = useState<NodeAccessPathDeleteImpact | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [nameFilter, setNameFilter] = useState('');
  const [chainFilter, setChainFilter] = useState('');
  const [protocolFilter, setProtocolFilter] = useState('');
  const [targetFilter, setTargetFilter] = useState('');
  const [listenFilter, setListenFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [formState, setFormState] = useState<AccessPathFormState>(emptyForm);

  const pathsQuery = useQuery({
    queryKey: ['proxy-access-paths', accessToken, activeTenantId],
    queryFn: () => getNodeAccessPaths(accessToken, activeTenantId),
    enabled: !!accessToken
  });
  const enumsQuery = useQuery({
    queryKey: ['enums'],
    queryFn: () => fetchEnums()
  });

  const nodeById = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  const chainById = useMemo(() => new Map(chains.map((chain) => [chain.id, chain])), [chains]);
  const nodeNameFor = (nodeId: string) => nodeById.get(nodeId)?.name || t('common.unknown');

  const createMutation = useMutation({
    mutationFn: (payload: NodeAccessPathPayload) => createNodeAccessPath(accessToken, activeTenantId, payload),
    onSuccess: () => {
      toast.success(accessPathsT('createSuccess'));
      setFormState(emptyForm);
      setCreateOpen(false);
      queryClient.invalidateQueries({queryKey: ['proxy-access-paths']});
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });

  const updateMutation = useMutation({
    mutationFn: (payload: NodeAccessPathPayload) => updateNodeAccessPath(accessToken, activeTenantId, editingPath!.id, {...payload, enabled: payload.enabled ?? true}),
    onSuccess: () => {
      toast.success(accessPathsT('updateSuccess'));
      setEditingPath(null);
      setFormState(emptyForm);
      queryClient.invalidateQueries({queryKey: ['proxy-access-paths']});
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });

  const deleteMutation = useMutation({
    mutationFn: (pathID: string) => deleteNodeAccessPath(accessToken, activeTenantId, pathID),
    onSuccess: () => {
      toast.success(accessPathsT('deleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['proxy-access-paths']});
      setDeletingPath(null);
      setDeleteImpact(null);
    },
    onError: (error) => toast.error(formatControlPlaneError(error))
  });

  const deleteImpactMutation = useMutation({
    mutationFn: (pathID: string) => getNodeAccessPathDeleteImpact(accessToken, activeTenantId, pathID),
    onSuccess: (result) => setDeleteImpact(result),
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
      setDeletingPath(null);
    }
  });

  const setField = <K extends keyof AccessPathFormState>(field: K, value: AccessPathFormState[K]) => {
    setFormState((current) => ({...current, [field]: value}));
  };

  const handleEdit = (path: NodeAccessPath) => {
    setEditingPath(path);
    setCreateOpen(false);
    setFormState(pathFormValues(path));
  };

  const handleCancelEdit = () => {
    setEditingPath(null);
    setCreateOpen(false);
    setFormState(emptyForm);
  };

  const handleOpenCreate = () => {
    setEditingPath(null);
    setFormState(emptyForm);
    setCreateOpen(true);
  };

  const openDeletePath = (path: NodeAccessPath) => {
    setDeletingPath(path);
    setDeleteImpact(null);
    deleteImpactMutation.mutate(path.id);
  };

  useEffect(() => {
    if (createRequestKey > 0) {
      handleOpenCreate();
    }
  }, [createRequestKey]);

  const handleSubmit = () => {
    const chain = chainById.get(formState.chainId);
    if (!chain || chain.hops.length === 0 || !formState.name.trim() || !formState.listenHost.trim() || !formState.targetHost.trim()) {
      toast.error(accessPathsT('required'));
      return;
    }
    if (!isValidPort(formState.listenPort)) {
      toast.error(`${accessPathsT('listenPort')}: ${t('common.invalid')}`);
      return;
    }
    if (!isValidPort(formState.targetPort)) {
      toast.error(`${accessPathsT('targetPort')}: ${t('common.invalid')}`);
      return;
    }
    const payload = submitPayload(formState, chains);
    if (editingPath) {
      updateMutation.mutate(payload);
      return;
    }
    createMutation.mutate(payload);
  };

  const paths = pathsQuery.data || [];
  const enums = enumsQuery.data || {};
  const protocolOptions = enumOptions(enums.access_protocol);
  const tlsOptions = enumOptions(enums.tls_mode);
  const authOptions = enumOptions(enums.access_auth_mode);
  const selectedChain = chainById.get(formState.chainId);
  const saving = createMutation.isPending || updateMutation.isPending;
  const filteredPaths = useMemo(() => {
    return paths.filter((path) =>
      (!nameFilter.trim() || path.name.toLowerCase().includes(nameFilter.trim().toLowerCase())) &&
      (!chainFilter || path.chainId === chainFilter) &&
      (!protocolFilter || path.protocol === protocolFilter) &&
      (!targetFilter.trim() || `${path.targetHost}:${path.targetPort}`.toLowerCase().includes(targetFilter.trim().toLowerCase())) &&
      (!listenFilter.trim() || `${path.listenHost || '*'}:${path.listenPort}`.toLowerCase().includes(listenFilter.trim().toLowerCase())) &&
      (!statusFilter || (statusFilter === 'enabled' ? path.enabled : !path.enabled))
    );
  }, [chainFilter, listenFilter, nameFilter, paths, protocolFilter, statusFilter, targetFilter]);
  const modalOpen = createOpen || Boolean(editingPath);
  const pathDeleteSections: DeleteImpactSection[] = deleteImpact ? [
    {id: 'accessPath', label: accessPathsT('deleteImpactAccessPath'), items: deleteImpact.delete.accessPath},
    {id: 'onboardingTasks', label: accessPathsT('deleteImpactOnboardingTasks'), items: deleteImpact.delete.onboardingTasks},
    {id: 'tenantBindings', label: accessPathsT('deleteImpactTenantBindings'), items: deleteImpact.delete.tenantBindings}
  ] : deletingPath ? [
    {id: 'accessPath', label: accessPathsT('deleteImpactAccessPath'), count: 1}
  ] : [];

  return (
    <>
      <ConsoleFilterBar title={t('common.filter')}>
        <ConsoleFilterItem label={t('common.name')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => setNameFilter(event.target.value)} placeholder={t('common.name')} value={nameFilter} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={accessPathsT('chain')} match={t('common.equals')}>
          <select className="field-select" onChange={(event) => setChainFilter(event.target.value)} value={chainFilter}>
            <option value="">{t('common.all')}</option>
            {chains.map((chain) => <option key={chain.id} value={chain.id}>{chain.name}</option>)}
          </select>
        </ConsoleFilterItem>
        <ConsoleFilterItem label={accessPathsT('protocol')} match={t('common.equals')}>
          <select className="field-select" onChange={(event) => setProtocolFilter(event.target.value)} value={protocolFilter}>
            <option value="">{t('common.all')}</option>
            {protocolOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
          </select>
        </ConsoleFilterItem>
        <ConsoleFilterItem label={t('common.target')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => setTargetFilter(event.target.value)} placeholder={accessPathsT('targetHost')} value={targetFilter} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={accessPathsT('listen')} match={t('common.contains')}>
          <input className="field-input" onChange={(event) => setListenFilter(event.target.value)} placeholder={accessPathsT('listenHost')} value={listenFilter} />
        </ConsoleFilterItem>
        <ConsoleFilterItem label={t('common.status')} match={t('common.equals')}>
          <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
            <option value="">{t('common.all')}</option>
            <option value="enabled">{t('common.enabled')}</option>
            <option value="disabled">{t('common.disabled')}</option>
          </select>
        </ConsoleFilterItem>
      </ConsoleFilterBar>

      <ConsoleList count={filteredPaths.length} title={accessPathsT('listTitle')}>
        {pathsQuery.isPending ? (
          <AsyncState detail={t('common.loading')} title={accessPathsT('loading')} />
        ) : pathsQuery.isError ? (
          <AsyncState actionLabel={t('common.retry')} detail={formatControlPlaneError(pathsQuery.error)} onAction={() => void pathsQuery.refetch()} title={accessPathsT('failed')} />
        ) : paths.length === 0 ? (
          <AsyncState detail={accessPathsT('empty')} title={t('common.empty')} />
        ) : filteredPaths.length === 0 ? (
          <AsyncState detail={t('common.noMatching')} title={t('common.empty')} />
        ) : (
          <div className="table-card">
            <table className="data-table">
              <thead>
                <tr>
                  <th>{t('common.name')}</th>
                  <th>{accessPathsT('chain')}</th>
                  <th>{accessPathsT('protocol')}</th>
                  <th>{t('common.target')}</th>
                  <th>{accessPathsT('listen')}</th>
                  <th>{t('common.status')}</th>
                  {canWrite ? <th>{t('common.actions')}</th> : null}
                </tr>
              </thead>
              <tbody>
                {filteredPaths.map((path) => {
                  const canManage = globalSuperAdmin || path.permission === 'manage';
                  const health = accessPathHealth(path);
                  return (
                    <tr key={path.id}>
                      <td><NameTag kind="node">{path.name}</NameTag></td>
                      <td><NameTag kind="chain">{chainById.get(path.chainId)?.name || t('common.unknown')}</NameTag></td>
                      <td className="mono">{path.protocol} / {path.serviceType}</td>
                      <td className="mono">{path.targetHost}:{path.targetPort}</td>
                      <td className="mono">{path.listenHost || '*'}:{path.listenPort}</td>
                      <td>
                        <div className="inline-cluster">
                          <span className={`badge ${path.enabled ? 'is-good' : 'is-neutral'}`}>{path.enabled ? t('common.enabled') : t('common.disabled')}</span>
                          <span className={accessPathHealthBadgeClassName(health?.status)} title={accessPathHealthTitle(health)}>
                            {health?.status || t('common.unknown')}
                          </span>
                        </div>
                      </td>
                      {canWrite ? (
                        <td>
                          <div className="chain-list-actions">
                            {canManage ? (
                              <button className="secondary-button" onClick={() => setGrantPath(path)} type="button">
                                <Share2 size={14} />
                                {t('common.grant')}
                              </button>
                            ) : null}
                            <button className="secondary-button" disabled={!canManage} onClick={() => handleEdit(path)} type="button">
                              <Edit size={14} />
                              {t('common.edit')}
                            </button>
                            <button
                              className="danger-button"
                              disabled={deleteMutation.isPending || deleteImpactMutation.isPending || !canManage}
                              onClick={() => openDeletePath(path)}
                              type="button"
                            >
                              <Trash2 size={14} />
                              {t('common.delete')}
                            </button>
                          </div>
                        </td>
                      ) : null}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </ConsoleList>

      <ConsoleCrudModal
        footer={(
          <>
            {selectedChain ? (
              <span className="tag-path">
                <span className="muted-text">{accessPathsT('chainPath')}:</span>
                <NodeTagPath labels={selectedChain.hops.map(nodeNameFor)} />
              </span>
            ) : (
              <span className="muted-text">{accessPathsT('selectChain')}</span>
            )}
            <button className="secondary-button" onClick={handleCancelEdit} type="button">
              {t('common.cancel')}
            </button>
            <button className="primary-button" disabled={saving} onClick={handleSubmit} type="button">
              {editingPath ? accessPathsT('save') : accessPathsT('create')}
            </button>
          </>
        )}
        onClose={handleCancelEdit}
        open={modalOpen}
        subtitle={accessPathsT('formEyebrow')}
        title={editingPath ? accessPathsT('editTitle') : accessPathsT('createTitle')}
      >
        <div className="forms-grid">
        <label className="field-stack">
          <span>{accessPathsT('chain')}</span>
          <select className="field-select" onChange={(event) => setField('chainId', event.target.value)} value={formState.chainId}>
            <option value="">{accessPathsT('selectChain')}</option>
            {chains.map((chain) => <option key={chain.id} value={chain.id}>{chain.name}</option>)}
          </select>
        </label>
        <label className="field-stack">
          <span>{t('common.name')}</span>
          <input className="field-input" onChange={(event) => setField('name', event.target.value)} value={formState.name} />
        </label>
        <label className="field-stack">
          <span>{accessPathsT('protocol')}</span>
          <select className="field-select" onChange={(event) => setField('protocol', event.target.value)} value={formState.protocol}>
            {protocolOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
          </select>
        </label>
        <label className="field-stack">
          <span>{accessPathsT('listenHost')}</span>
          <input className="field-input" onChange={(event) => setField('listenHost', event.target.value)} placeholder={accessPathsT('listenHostPlaceholder')} value={formState.listenHost} />
        </label>
        <label className="field-stack">
          <span>{accessPathsT('listenPort')}</span>
          <input className="field-input" inputMode="numeric" max={65535} min={1} onChange={(event) => setField('listenPort', event.target.value)} type="number" value={formState.listenPort} />
        </label>
        <label className="field-stack">
          <span>{accessPathsT('targetHost')}</span>
          <input className="field-input" onChange={(event) => setField('targetHost', event.target.value)} placeholder={accessPathsT('targetHostPlaceholder')} value={formState.targetHost} />
        </label>
        <label className="field-stack">
          <span>{accessPathsT('targetPort')}</span>
          <input className="field-input" inputMode="numeric" max={65535} min={1} onChange={(event) => setField('targetPort', event.target.value)} type="number" value={formState.targetPort} />
        </label>
        <label className="field-stack">
          <span>{accessPathsT('tlsMode')}</span>
          <select className="field-select" onChange={(event) => setField('tlsMode', event.target.value)} value={formState.tlsMode}>
            {tlsOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
          </select>
        </label>
        <label className="field-stack">
          <span>{accessPathsT('targetSni')}</span>
          <input className="field-input" onChange={(event) => setField('targetSni', event.target.value)} placeholder={accessPathsT('sniPlaceholder')} value={formState.targetSni} />
        </label>
        <label className="field-stack">
          <span>{accessPathsT('authMode')}</span>
          <select className="field-select" onChange={(event) => setField('authMode', event.target.value)} value={formState.authMode}>
            {authOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
          </select>
        </label>
        <label className="field-stack">
          <span>{accessPathsT('enabled')}</span>
          <button
            className={`toggle-button${formState.enabled ? ' is-active' : ''}`}
            onClick={() => setField('enabled', !formState.enabled)}
            type="button"
          >
            <span>{formState.enabled ? accessPathsT('enabled') : t('common.disabled')}</span>
            <span className="toggle-button-track"><span /></span>
          </button>
        </label>
      </div>
      </ConsoleCrudModal>

      {grantPath ? (
        <ResourceGrantModal
          onChanged={() => queryClient.invalidateQueries({queryKey: ['proxy-access-paths']})}
          onClose={() => setGrantPath(null)}
          open={Boolean(grantPath)}
          resourceId={grantPath.id}
          resourceName={grantPath.name}
          resourceType="access_path"
        />
      ) : null}

      <DeleteConfirmationModal
        onClose={() => {
          setDeletingPath(null);
          setDeleteImpact(null);
        }}
        onConfirm={() => {
          if (deletingPath) {
            deleteMutation.mutate(deletingPath.id);
          }
        }}
        open={Boolean(deletingPath)}
        pending={deleteMutation.isPending || deleteImpactMutation.isPending}
        sections={pathDeleteSections}
        targetName={deletingPath?.name || ''}
        title={accessPathsT('deleteConfirmTitle')}
      />
    </>
  );
}
