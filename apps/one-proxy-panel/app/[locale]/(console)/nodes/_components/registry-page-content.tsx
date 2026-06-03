'use client';

import {useEffect, useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AuthGate} from '@/components/auth-gate';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleList, ConsolePage} from '@/components/console-template';
import {fetchEnums} from '@/lib/api';

import {useNodeConsole} from './use-node-console';
import {deriveNodeHealthState} from './node-utils';
import {RegistryNodeEditor} from './registry-node-editor';
import {RegistryNodeTable} from './registry-node-table';
import {RegistryNodeFormState, RegistryNodeRow} from './types';

export function NodeRegistryPageContent() {
  const t = useTranslations();
  const nodesT = useTranslations('nodesConsole');
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const scopes = nodeConsole.scopesQuery.data || [];
  const healthRows = nodeConsole.healthQuery.data || [];
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [modeFilter, setModeFilter] = useState('all');
  const [editingNodeID, setEditingNodeID] = useState('');
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const nodeModeKeys = Object.keys(enums?.node_mode || {});
  const nodeStatusKeys = Object.keys(enums?.node_status || {});
  const DEFAULT_MODE = nodeModeKeys.find(k => k === 'relay') || 'relay';
  const DEFAULT_STATUS = nodeStatusKeys.find(k => k === 'healthy') || 'healthy';
  const [formState, setFormState] = useState<RegistryNodeFormState>({
    name: '',
    mode: DEFAULT_MODE,
    scopeKey: '',
    parentNodeId: '',
    publicHost: '',
    publicPort: '',
    enabled: true,
    status: DEFAULT_STATUS
  });
  const healthByNodeID = useMemo(() => new Map(healthRows.map((item) => [item.nodeId, item])), [healthRows]);
  const nodesByID = useMemo(() => new Map(nodes.map((node) => [node.id, node])), [nodes]);
  const nodeRows: RegistryNodeRow[] = useMemo(
    () =>
      nodes.map((node) => {
        const health = healthByNodeID.get(node.id);
        const derivedHealth = deriveNodeHealthState(health, enums);
        return {
          ...node,
          derivedHealthStatus: derivedHealth.status,
          derivedHealthLabel: derivedHealth.label,
          heartbeatAt: health?.heartbeatAt || '',
          policyRevisionId: health?.policyRevisionId || ''
        };
      }),
    [healthByNodeID, nodes, enums]
  );
  const normalizedQuery = query.trim().toLowerCase();
  const filteredNodes = nodeRows.filter((node) => {
    const matchesQuery =
      normalizedQuery.length === 0 ||
      node.name.toLowerCase().includes(normalizedQuery) ||
      node.id.toLowerCase().includes(normalizedQuery) ||
      node.scopeKey.toLowerCase().includes(normalizedQuery) ||
      node.parentNodeId.toLowerCase().includes(normalizedQuery) ||
      node.publicHost?.toLowerCase().includes(normalizedQuery) ||
      node.derivedHealthLabel.toLowerCase().includes(normalizedQuery) ||
      node.policyRevisionId.toLowerCase().includes(normalizedQuery);
    const matchesStatus = statusFilter === 'all' || node.derivedHealthStatus === statusFilter;
    const matchesMode = modeFilter === 'all' || node.mode === modeFilter;

    return matchesQuery && matchesStatus && matchesMode;
  });
  const availableModes = Array.from(new Set(nodes.map((node) => node.mode))).sort();
  const editingNode = nodes.find((node) => node.id === editingNodeID) || null;

  useEffect(() => {
    if (!editingNode) {
      return;
    }
    setFormState({
      name: editingNode.name,
      mode: editingNode.mode,
      scopeKey: editingNode.scopeKey || '',
      parentNodeId: editingNode.parentNodeId || '',
      publicHost: editingNode.publicHost || '',
      publicPort: editingNode.publicPort ? String(editingNode.publicPort) : '',
      enabled: editingNode.enabled,
      status: editingNode.status
    });
  }, [editingNode]);

  useEffect(() => {
    if (editingNodeID && !nodes.some((node) => node.id === editingNodeID)) {
      setEditingNodeID('');
    }
  }, [editingNodeID, nodes]);

  const deleteNode = (node: RegistryNodeRow) => {
    if (!window.confirm(nodesT('deleteNodeConfirm', {name: node.name, id: node.id}))) {
      return;
    }
    if (editingNodeID === node.id) {
      setEditingNodeID('');
    }
    nodeConsole.deleteNode.mutate(node.id);
  };

  const saveNode = () => {
    if (!editingNode) {
      return;
    }
    nodeConsole.updateNode.mutate(
      {
        nodeID: editingNode.id,
        name: formState.name.trim(),
        mode: formState.mode,
        scopeKey: formState.scopeKey.trim(),
        parentNodeId: formState.parentNodeId.trim(),
        publicHost: formState.publicHost.trim(),
        publicPort: formState.publicPort.trim() ? Number(formState.publicPort) : 0,
        enabled: formState.enabled,
        status: formState.status
      },
      {
        onSuccess: () => {
          setEditingNodeID('');
        }
      }
    );
  };

  return (
    <AuthGate>
      <ConsolePage eyebrow={nodesT('registry')} title={nodesT('registryTitle')}>
        <ConsoleFilterBar title={t('common.filter')}>
          <label className="field-stack">
            <span>{t('common.search')}</span>
            <input
              className="field-input"
              onChange={(event) => setQuery(event.target.value)}
              placeholder={nodesT('registrySearchPlaceholder')}
              type="search"
              value={query}
            />
          </label>
          <label className="field-stack">
            <span>{t('common.status')}</span>
            <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
              <option value="all">{nodesT('allHealthStates')}</option>
              <option value="healthy">{nodesT('healthyNodes')}</option>
              <option value="degraded">{nodesT('degradedNodes')}</option>
              <option value="stale">{nodesT('staleNodes')}</option>
              <option value="unreported">{nodesT('unreportedNodes')}</option>
            </select>
          </label>
          <label className="field-stack">
            <span>{t('common.mode')}</span>
            <select className="field-select" onChange={(event) => setModeFilter(event.target.value)} value={modeFilter}>
              <option value="all">{nodesT('allModes')}</option>
              {availableModes.map((mode) => (
                <option key={mode} value={mode}>
                  {mode}
                </option>
              ))}
            </select>
          </label>
        </ConsoleFilterBar>

        <ConsoleList count={filteredNodes.length} title={nodesT('registryTitle')}>
          <RegistryNodeTable
            deletePending={nodeConsole.deleteNode.isPending}
            editingNodeID={editingNodeID}
            enums={enums}
            filteredNodes={filteredNodes}
            healthError={nodeConsole.healthQuery.error}
            healthPending={nodeConsole.healthQuery.isPending}
            nodeRows={nodeRows}
            nodes={nodes}
            nodesByID={nodesByID}
            nodesError={nodeConsole.nodesQuery.error}
            nodesPending={nodeConsole.nodesQuery.isPending}
            nodesT={nodesT}
            onDelete={deleteNode}
            onRetryHealth={() => void nodeConsole.healthQuery.refetch()}
            onRetryNodes={() => void nodeConsole.nodesQuery.refetch()}
            onToggleEdit={(nodeID) => setEditingNodeID(editingNodeID === nodeID ? '' : nodeID)}
            t={t}
          />
        </ConsoleList>

        <ConsoleCrudModal
          onClose={() => setEditingNodeID('')}
          open={Boolean(editingNode)}
          subtitle={editingNode?.id}
          title={nodesT('editNode')}
        >
          {editingNode ? (
          <RegistryNodeEditor
            editingNode={editingNode}
            enums={enums}
            formState={formState}
            nodes={nodes}
            nodesT={nodesT}
            onClose={() => setEditingNodeID('')}
            onFormChange={setFormState}
            onSave={saveNode}
            scopes={scopes}
            t={t}
            updatePending={nodeConsole.updateNode.isPending}
          />
          ) : null}
        </ConsoleCrudModal>
      </ConsolePage>
    </AuthGate>
  );
}
