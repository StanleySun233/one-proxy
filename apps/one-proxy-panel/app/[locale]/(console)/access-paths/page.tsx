'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {Edit, Trash2} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {useState} from 'react';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {NameTag} from '@/components/common/name-tag';
import {PageHero} from '@/components/page-hero';
import {createNodeAccessPath, deleteNodeAccessPath, fetchEnums, getNodeAccessPaths, getNodes, updateNodeAccessPath} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';
import type {FieldEnumEntry, Node, NodeAccessPath, NodeAccessPathPayload} from '@/lib/types';

type AccessPathFormState = {
  name: string;
  mode: string;
  protocol: string;
  serviceType: string;
  targetNodeId: string;
  entryNodeId: string;
  relayNodeIds: string;
  listenHost: string;
  listenPort: string;
  targetProtocol: string;
  targetHost: string;
  targetPort: string;
  targetSni: string;
  tlsMode: string;
  authMode: string;
  enabled: boolean;
};

const emptyForm: AccessPathFormState = {
  name: '',
  mode: 'direct',
  protocol: 'tcp',
  serviceType: 'raw_tcp',
  targetNodeId: '',
  entryNodeId: '',
  relayNodeIds: '',
  listenHost: '0.0.0.0',
  listenPort: '0',
  targetProtocol: 'tcp',
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

function nodeLabel(node: Node) {
  return `${node.name} (${node.id})`;
}

function pathFormValues(path: NodeAccessPath): AccessPathFormState {
  return {
    name: path.name,
    mode: path.mode,
    protocol: path.protocol,
    serviceType: path.serviceType,
    targetNodeId: path.targetNodeId || '',
    entryNodeId: path.entryNodeId || '',
    relayNodeIds: path.relayNodeIds.join(','),
    listenHost: path.listenHost || '',
    listenPort: String(path.listenPort || 0),
    targetProtocol: path.targetProtocol || path.protocol,
    targetHost: path.targetHost || '',
    targetPort: String(path.targetPort || ''),
    targetSni: path.targetSni || '',
    tlsMode: path.tlsMode || 'none',
    authMode: path.authMode || 'proxy_token',
    enabled: path.enabled
  };
}

function submitPayload(form: AccessPathFormState): NodeAccessPathPayload {
  return {
    name: form.name.trim(),
    mode: form.mode as NodeAccessPathPayload['mode'],
    protocol: form.protocol as NodeAccessPathPayload['protocol'],
    serviceType: form.serviceType as NodeAccessPathPayload['serviceType'],
    targetNodeId: form.targetNodeId,
    entryNodeId: form.entryNodeId,
    relayNodeIds: form.relayNodeIds.split(',').map((item) => item.trim()).filter(Boolean),
    listenHost: form.listenHost.trim(),
    listenPort: Number(form.listenPort || 0),
    targetProtocol: form.targetProtocol.trim() || form.protocol,
    targetHost: form.targetHost.trim(),
    targetPort: Number(form.targetPort || 0),
    targetSni: form.targetSni.trim(),
    tlsMode: form.tlsMode as NodeAccessPathPayload['tlsMode'],
    authMode: form.authMode as NodeAccessPathPayload['authMode'],
    options: {},
    enabled: form.enabled
  };
}

export default function AccessPathsPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const accessPathsT = useTranslations('accessPaths');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const [editingPath, setEditingPath] = useState<NodeAccessPath | null>(null);
  const [formState, setFormState] = useState<AccessPathFormState>(emptyForm);

  const pathsQuery = useQuery({
    queryKey: ['node-access-paths', accessToken],
    queryFn: () => getNodeAccessPaths(accessToken),
    enabled: !!accessToken
  });
  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });
  const enumsQuery = useQuery({
    queryKey: ['enums'],
    queryFn: () => fetchEnums()
  });

  const createMutation = useMutation({
    mutationFn: (payload: NodeAccessPathPayload) => createNodeAccessPath(accessToken, payload),
    onSuccess: () => {
      toast.success(accessPathsT('createSuccess'));
      setFormState(emptyForm);
      queryClient.invalidateQueries({queryKey: ['node-access-paths']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const updateMutation = useMutation({
    mutationFn: (payload: NodeAccessPathPayload) => updateNodeAccessPath(accessToken, editingPath!.id, {...payload, enabled: payload.enabled ?? true}),
    onSuccess: () => {
      toast.success(accessPathsT('updateSuccess'));
      setEditingPath(null);
      setFormState(emptyForm);
      queryClient.invalidateQueries({queryKey: ['node-access-paths']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const deleteMutation = useMutation({
    mutationFn: (pathID: string) => deleteNodeAccessPath(accessToken, pathID),
    onSuccess: () => {
      toast.success(accessPathsT('deleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['node-access-paths']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const setField = <K extends keyof AccessPathFormState>(field: K, value: AccessPathFormState[K]) => {
    setFormState((current) => ({...current, [field]: value}));
  };

  const handleEdit = (path: NodeAccessPath) => {
    setEditingPath(path);
    setFormState(pathFormValues(path));
  };

  const handleCancelEdit = () => {
    setEditingPath(null);
    setFormState(emptyForm);
  };

  const handleSubmit = () => {
    const payload = submitPayload(formState);
    if (!payload.name || !payload.targetHost || payload.targetPort <= 0) {
      toast.error(accessPathsT('required'));
      return;
    }
    if (editingPath) {
      updateMutation.mutate(payload);
      return;
    }
    createMutation.mutate(payload);
  };

  const paths = pathsQuery.data || [];
  const nodes = nodesQuery.data || [];
  const enums = enumsQuery.data || {};
  const nodeById = new Map(nodes.map((node) => [node.id, node]));
  const saving = createMutation.isPending || updateMutation.isPending;
  const modeOptions = enumOptions(enums.path_mode);
  const protocolOptions = enumOptions(enums.access_protocol);
  const serviceOptions = enumOptions(enums.access_service_type);
  const tlsOptions = enumOptions(enums.tls_mode);
  const authOptions = enumOptions(enums.access_auth_mode);

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow={accessPathsT('eyebrow')} title={pageT('accessPathsTitle')} />

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">{accessPathsT('formEyebrow')}</p>
              <h3>{editingPath ? accessPathsT('editTitle') : accessPathsT('createTitle')}</h3>
            </div>
            {editingPath ? (
              <button className="secondary-button" onClick={handleCancelEdit} type="button">
                {t('common.cancel')}
              </button>
            ) : null}
          </div>

          <div className="forms-grid">
            <label className="field-stack">
              <span>{t('common.name')}</span>
              <input className="field-input" onChange={(event) => setField('name', event.target.value)} value={formState.name} />
            </label>
            <label className="field-stack">
              <span>{t('common.mode')}</span>
              <select className="field-select" onChange={(event) => setField('mode', event.target.value)} value={formState.mode}>
                {modeOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <label className="field-stack">
              <span>{accessPathsT('protocol')}</span>
              <select className="field-select" onChange={(event) => setField('protocol', event.target.value)} value={formState.protocol}>
                {protocolOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <label className="field-stack">
              <span>{accessPathsT('serviceType')}</span>
              <select className="field-select" onChange={(event) => setField('serviceType', event.target.value)} value={formState.serviceType}>
                {serviceOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <label className="field-stack">
              <span>{accessPathsT('entryNode')}</span>
              <select className="field-select" onChange={(event) => setField('entryNodeId', event.target.value)} value={formState.entryNodeId}>
                <option value="">{accessPathsT('noNode')}</option>
                {nodes.map((node) => <option key={node.id} value={node.id}>{nodeLabel(node)}</option>)}
              </select>
            </label>
            <label className="field-stack">
              <span>{accessPathsT('targetNode')}</span>
              <select className="field-select" onChange={(event) => setField('targetNodeId', event.target.value)} value={formState.targetNodeId}>
                <option value="">{accessPathsT('noNode')}</option>
                {nodes.map((node) => <option key={node.id} value={node.id}>{nodeLabel(node)}</option>)}
              </select>
            </label>
            <label className="field-stack">
              <span>{accessPathsT('listenHost')}</span>
              <input className="field-input" onChange={(event) => setField('listenHost', event.target.value)} placeholder={accessPathsT('listenHostPlaceholder')} value={formState.listenHost} />
            </label>
            <label className="field-stack">
              <span>{accessPathsT('listenPort')}</span>
              <input className="field-input" inputMode="numeric" onChange={(event) => setField('listenPort', event.target.value)} value={formState.listenPort} />
            </label>
            <label className="field-stack">
              <span>{accessPathsT('targetProtocol')}</span>
              <input className="field-input" onChange={(event) => setField('targetProtocol', event.target.value)} value={formState.targetProtocol} />
            </label>
            <label className="field-stack">
              <span>{accessPathsT('targetHost')}</span>
              <input className="field-input" onChange={(event) => setField('targetHost', event.target.value)} placeholder={accessPathsT('targetHostPlaceholder')} value={formState.targetHost} />
            </label>
            <label className="field-stack">
              <span>{accessPathsT('targetPort')}</span>
              <input className="field-input" inputMode="numeric" onChange={(event) => setField('targetPort', event.target.value)} value={formState.targetPort} />
            </label>
            <label className="field-stack">
              <span>{accessPathsT('targetSni')}</span>
              <input className="field-input" onChange={(event) => setField('targetSni', event.target.value)} placeholder={accessPathsT('sniPlaceholder')} value={formState.targetSni} />
            </label>
            <label className="field-stack">
              <span>{accessPathsT('tlsMode')}</span>
              <select className="field-select" onChange={(event) => setField('tlsMode', event.target.value)} value={formState.tlsMode}>
                {tlsOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <label className="field-stack">
              <span>{accessPathsT('authMode')}</span>
              <select className="field-select" onChange={(event) => setField('authMode', event.target.value)} value={formState.authMode}>
                {authOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <label className="field-stack">
              <span>{accessPathsT('relayNodes')}</span>
              <input className="field-input" onChange={(event) => setField('relayNodeIds', event.target.value)} placeholder={accessPathsT('relayPlaceholder')} value={formState.relayNodeIds} />
            </label>
            <label className="field-stack">
              <span>{accessPathsT('enabled')}</span>
              <input checked={formState.enabled} onChange={(event) => setField('enabled', event.target.checked)} type="checkbox" />
            </label>
          </div>

          <div className="submit-row">
            <button className="primary-button" disabled={saving} onClick={handleSubmit} type="button">
              {editingPath ? accessPathsT('save') : accessPathsT('create')}
            </button>
          </div>
        </section>

        <section className="panel-card">
          <div className="panel-toolbar">
            <div>
              <p className="section-kicker">{accessPathsT('listEyebrow')}</p>
              <h3>{accessPathsT('listTitle')}</h3>
            </div>
            <span className="badge is-neutral">{paths.length}</span>
          </div>

          {pathsQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={accessPathsT('loading')} />
          ) : pathsQuery.isError ? (
            <AsyncState actionLabel={t('common.retry')} detail={formatControlPlaneError(pathsQuery.error)} onAction={() => void pathsQuery.refetch()} title={accessPathsT('failed')} />
          ) : paths.length === 0 ? (
            <AsyncState detail={accessPathsT('empty')} title={t('common.empty')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('common.name')}</th>
                    <th>{accessPathsT('protocol')}</th>
                    <th>{accessPathsT('entryNode')}</th>
                    <th>{t('common.target')}</th>
                    <th>{accessPathsT('listen')}</th>
                    <th>{t('common.status')}</th>
                    <th>{t('common.actions')}</th>
                  </tr>
                </thead>
                <tbody>
                  {paths.map((path) => (
                    <tr key={path.id}>
                      <td><NameTag kind="node">{path.name}</NameTag></td>
                      <td className="mono">{path.protocol} / {path.serviceType}</td>
                      <td>{path.entryNodeId ? nodeById.get(path.entryNodeId)?.name || path.entryNodeId : <span className="muted-text">-</span>}</td>
                      <td className="mono">{path.targetHost}:{path.targetPort}</td>
                      <td className="mono">{path.listenHost || '*'}:{path.listenPort}</td>
                      <td><span className={`badge ${path.enabled ? 'is-success' : 'is-neutral'}`}>{path.enabled ? t('common.enabled') : t('common.disabled')}</span></td>
                      <td>
                        <div className="chain-list-actions">
                          <button className="secondary-button" onClick={() => handleEdit(path)} type="button">
                            <Edit size={14} />
                            {t('common.edit')}
                          </button>
                          <button
                            className="danger-button"
                            disabled={deleteMutation.isPending}
                            onClick={() => {
                              if (window.confirm(accessPathsT('deleteConfirm'))) {
                                deleteMutation.mutate(path.id);
                              }
                            }}
                            type="button"
                          >
                            <Trash2 size={14} />
                            {t('common.delete')}
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </AuthGate>
  );
}
