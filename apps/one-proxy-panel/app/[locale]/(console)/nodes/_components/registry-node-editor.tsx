'use client';

import {FieldEnumMap, Node, Scope} from '@/lib/types';

import {RegistryNodeFormState} from './types';

type RegistryNodeEditorProps = {
  editingNode: Node;
  nodes: Node[];
  scopes: Scope[];
  enums: FieldEnumMap | undefined;
  formState: RegistryNodeFormState;
  updatePending: boolean;
  t: (key: string) => string;
  nodesT: (key: string) => string;
  onClose: () => void;
  onFormChange: (next: RegistryNodeFormState) => void;
  onSave: () => void;
};

export function RegistryNodeEditor({
  editingNode,
  nodes,
  scopes,
  enums,
  formState,
  updatePending,
  t,
  nodesT,
  onClose,
  onFormChange,
  onSave
}: RegistryNodeEditorProps) {
  const nodeModeOptions = enums?.node_mode ? Object.entries(enums.node_mode).map(([value, item]) => ({value, label: item.name})) : [];
  const nodeStatusOptions = enums?.node_status ? Object.entries(enums.node_status).map(([value, item]) => ({value, label: item.name})) : [];

  return (
    <div>
      <div className="forms-grid">
        <label className="field-stack">
          <span>{t('common.name')}</span>
          <input className="field-input" onChange={(event) => onFormChange({...formState, name: event.target.value})} value={formState.name} />
        </label>
        <label className="field-stack">
          <span>{t('common.mode')}</span>
          <select className="field-select" onChange={(event) => onFormChange({...formState, mode: event.target.value})} value={formState.mode}>
            {nodeModeOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>{nodesT('scopeKey')}</span>
          <select className="field-select" onChange={(event) => onFormChange({...formState, scopeKey: event.target.value})} value={formState.scopeKey}>
            <option value="">{t('common.noScope')}</option>
            {scopes.map((scope) => (
              <option key={scope.id} value={scope.id}>{scope.name} ({scope.id})</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>{t('common.parent')}</span>
          <select className="field-select" onChange={(event) => onFormChange({...formState, parentNodeId: event.target.value})} value={formState.parentNodeId}>
            <option value="">{t('common.root')}</option>
            {nodes.filter((node) => node.id !== editingNode.id).map((node) => (
              <option key={node.id} value={node.id}>
                {node.name} ({node.mode})
              </option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>{nodesT('publicHost')}</span>
          <input className="field-input" onChange={(event) => onFormChange({...formState, publicHost: event.target.value})} value={formState.publicHost} />
        </label>
        <label className="field-stack">
          <span>{nodesT('publicPort')}</span>
          <input className="field-input" inputMode="numeric" onChange={(event) => onFormChange({...formState, publicPort: event.target.value})} value={formState.publicPort} />
        </label>
        <label className="field-stack">
          <span>{t('common.status')}</span>
          <select className="field-select" onChange={(event) => onFormChange({...formState, status: event.target.value})} value={formState.status}>
            {nodeStatusOptions.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        </label>
        <label className="field-stack">
          <span>{t('common.enabled')}</span>
          <select
            className="field-select"
            onChange={(event) => onFormChange({...formState, enabled: event.target.value === 'true'})}
            value={String(formState.enabled)}
          >
            <option value="true">{t('common.enabled')}</option>
            <option value="false">{t('common.disabled')}</option>
          </select>
        </label>
      </div>
      <div className="submit-row">
        <button
          className="primary-button"
          disabled={updatePending || formState.name.trim().length === 0 || formState.scopeKey.trim().length === 0}
          onClick={onSave}
          type="button"
        >
          {nodesT('saveChanges')}
        </button>
        <button className="secondary-button" onClick={onClose} type="button">
          {t('common.close')}
        </button>
      </div>
    </div>
  );
}
