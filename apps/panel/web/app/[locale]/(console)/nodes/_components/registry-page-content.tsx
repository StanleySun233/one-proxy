'use client';

import {useEffect, useMemo, useState} from 'react';
import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AuthGate} from '@/components/auth-gate';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {ResourceGrantModal} from '@/components/resource-grant-modal';
import {fetchEnums} from '@/lib/api';
import type {NodeDeleteImpact} from '@/lib/types';

import {useNodeConsole} from './use-node-console';
import {deriveNodeHealthState} from './node-utils';
import {RegistryNodeEditor} from './registry-node-editor';
import {RegistryNodeTable} from './registry-node-table';
import {RegistryNodeFormState, RegistryNodeRow} from './types';

const deleteImpactRows: Array<[keyof NodeDeleteImpact['delete'], string]> = [
  ['node', 'deleteImpactNode'],
  ['chains', 'deleteImpactChains'],
  ['chainHops', 'deleteImpactChainHops'],
  ['routeRules', 'deleteImpactRouteRules'],
  ['accessPaths', 'deleteImpactAccessPaths'],
  ['onboardingTasks', 'deleteImpactOnboardingTasks'],
  ['chainProbeResults', 'deleteImpactChainProbeResults'],
  ['runtimeTransports', 'deleteImpactRuntimeTransports'],
  ['nodeLinks', 'deleteImpactNodeLinks'],
  ['policyAssignments', 'deleteImpactPolicyAssignments'],
  ['healthSnapshots', 'deleteImpactHealthSnapshots'],
  ['slaMinutes', 'deleteImpactSLAMinutes'],
  ['apiTokens', 'deleteImpactAPITokens'],
  ['trustMaterials', 'deleteImpactTrustMaterials'],
  ['bootstrapTokens', 'deleteImpactBootstrapTokens'],
  ['tenantBindings', 'deleteImpactTenantBindings']
];

const updateImpactRows: Array<[keyof NodeDeleteImpact['update'], string]> = [
  ['childNodesDetached', 'deleteImpactChildNodesDetached']
];

export function NodeRegistryPageContent() {
  const t = useTranslations();
  const nodesT = useTranslations('nodesConsole');
  const nodeConsole = useNodeConsole();
  const nodes = nodeConsole.nodesQuery.data || [];
  const scopes = nodeConsole.scopesQuery.data || [];
  const healthRows = nodeConsole.healthQuery.data || [];
  const [nameFilter, setNameFilter] = useState('');
  const [scopeFilter, setScopeFilter] = useState('');
  const [parentFilter, setParentFilter] = useState('');
  const [publicEndpointFilter, setPublicEndpointFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('all');
  const [modeFilter, setModeFilter] = useState('all');
  const [editingNodeID, setEditingNodeID] = useState('');
  const [grantNodeID, setGrantNodeID] = useState('');
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
  const scopeNameByID = useMemo(() => new Map(scopes.map((scope) => [scope.id, scope.name])), [scopes]);
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
  const filteredNodes = nodeRows.filter((node) => {
    const matchesName = !nameFilter.trim() || node.name.toLowerCase().includes(nameFilter.trim().toLowerCase());
    const scopeName = scopeNameByID.get(node.scopeKey) || '';
    const parentName = nodesByID.get(node.parentNodeId)?.name || '';
    const matchesScope = !scopeFilter.trim() || scopeName.toLowerCase().includes(scopeFilter.trim().toLowerCase());
    const matchesParent = !parentFilter.trim() || parentName.toLowerCase().includes(parentFilter.trim().toLowerCase());
    const matchesPublicEndpoint = !publicEndpointFilter.trim() || `${node.publicHost || ''}:${node.publicPort || ''}`.toLowerCase().includes(publicEndpointFilter.trim().toLowerCase());
    const matchesStatus = statusFilter === 'all' || node.derivedHealthStatus === statusFilter;
    const matchesMode = modeFilter === 'all' || node.mode === modeFilter;

    return matchesName && matchesScope && matchesParent && matchesPublicEndpoint && matchesStatus && matchesMode;
  });
  const availableModes = Array.from(new Set(nodes.map((node) => node.mode))).sort();
  const editingNode = nodes.find((node) => node.id === editingNodeID) || null;
  const grantNode = nodes.find((node) => node.id === grantNodeID) || null;

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

  const deleteNode = async (node: RegistryNodeRow) => {
    let impact: NodeDeleteImpact;
    try {
      impact = await nodeConsole.nodeDeleteImpact.mutateAsync(node.id);
    } catch {
      return;
    }
    const deletedLines = deleteImpactRows
      .map(([key, labelKey]) => ({count: impact.delete[key], label: nodesT(labelKey)}))
      .filter((item) => item.count > 0)
      .map((item) => `- ${item.label}: ${item.count}`);
    const updatedLines = updateImpactRows
      .map(([key, labelKey]) => ({count: impact.update[key], label: nodesT(labelKey)}))
      .filter((item) => item.count > 0)
      .map((item) => `- ${item.label}: ${item.count}`);
    const confirmLines = [nodesT('deleteNodeConfirm', {name: node.name})];
    if (deletedLines.length > 0) {
      confirmLines.push('', nodesT('deleteImpactDeleted'), ...deletedLines);
    } else {
      confirmLines.push('', nodesT('deleteImpactEmpty'));
    }
    if (updatedLines.length > 0) {
      confirmLines.push('', nodesT('deleteImpactUpdated'), ...updatedLines);
    }
    confirmLines.push('', nodesT('deleteImpactContinue'));
    if (!window.confirm(confirmLines.join('\n'))) {
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
      <ConsolePage title={t('shell.nodeRegistry')}>
        <ConsoleFilterBar title={t('common.filter')}>
          <ConsoleFilterItem label={t('common.name')} match={t('common.contains')}>
            <input
              className="field-input"
              onChange={(event) => setNameFilter(event.target.value)}
              placeholder={t('common.name')}
              type="search"
              value={nameFilter}
            />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.scope')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setScopeFilter(event.target.value)} placeholder={t('common.scope')} value={scopeFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.parent')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setParentFilter(event.target.value)} placeholder={t('common.parent')} value={parentFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={nodesT('publicEndpoint')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setPublicEndpointFilter(event.target.value)} placeholder={nodesT('publicEndpoint')} value={publicEndpointFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.status')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
              <option value="all">{nodesT('allHealthStates')}</option>
              <option value="healthy">{nodesT('healthyNodes')}</option>
              <option value="degraded">{nodesT('degradedNodes')}</option>
              <option value="stale">{nodesT('staleNodes')}</option>
              <option value="unreported">{nodesT('unreportedNodes')}</option>
            </select>
          </ConsoleFilterItem>
          <ConsoleFilterItem label={t('common.mode')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setModeFilter(event.target.value)} value={modeFilter}>
              <option value="all">{nodesT('allModes')}</option>
              {availableModes.map((mode) => (
                <option key={mode} value={mode}>
                  {mode}
                </option>
              ))}
            </select>
          </ConsoleFilterItem>
        </ConsoleFilterBar>

        <ConsoleList count={filteredNodes.length} title={nodesT('registryTitle')}>
          <RegistryNodeTable
            canWrite={nodeConsole.canWrite}
            deletePending={nodeConsole.deleteNode.isPending || nodeConsole.nodeDeleteImpact.isPending}
            editingNodeID={editingNodeID}
            enums={enums}
            filteredNodes={filteredNodes}
            globalSuperAdmin={nodeConsole.globalSuperAdmin}
            healthError={nodeConsole.healthQuery.error}
            healthPending={nodeConsole.healthQuery.isPending}
            nodeRows={nodeRows}
            nodes={nodes}
            nodesByID={nodesByID}
            nodesError={nodeConsole.nodesQuery.error}
            nodesPending={nodeConsole.nodesQuery.isPending}
            nodesT={nodesT}
            onDelete={deleteNode}
            onGrant={(nodeID) => setGrantNodeID(nodeID)}
            onRetryHealth={() => void nodeConsole.healthQuery.refetch()}
            onRetryNodes={() => void nodeConsole.nodesQuery.refetch()}
            onToggleEdit={(nodeID) => setEditingNodeID(editingNodeID === nodeID ? '' : nodeID)}
            scopes={scopes}
            t={t}
          />
        </ConsoleList>

        <ConsoleCrudModal
          onClose={() => setEditingNodeID('')}
          open={Boolean(editingNode)}
          subtitle={editingNode?.name}
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

        {grantNode ? (
          <ResourceGrantModal
            onChanged={() => void nodeConsole.nodesQuery.refetch()}
            onClose={() => setGrantNodeID('')}
            open={Boolean(grantNode)}
            resourceId={grantNode.id}
            resourceName={grantNode.name}
            resourceType="node"
          />
        ) : null}
      </ConsolePage>
    </AuthGate>
  );
}
