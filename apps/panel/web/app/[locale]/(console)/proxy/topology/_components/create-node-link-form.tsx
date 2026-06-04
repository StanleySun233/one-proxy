'use client';

import {useEffect, useState} from 'react';
import {useTranslations} from 'next-intl';

import {Node, NodeLink} from '@/lib/types';

export function CreateNodeLinkForm({
  nodes,
  pending,
  onSubmit,
  editingLink,
  onCancelEdit,
  defaultLinkType,
  defaultTrustState
}: {
  nodes: Node[];
  pending: boolean;
  onSubmit: (payload: {sourceNodeId: string; targetNodeId: string; linkType: string; trustState: string}) => void;
  editingLink?: NodeLink | null;
  onCancelEdit?: () => void;
  defaultLinkType: string;
  defaultTrustState: string;
}) {
  const t = useTranslations();
  const nodesT = useTranslations('nodesConsole');
  const [sourceNodeId, setSourceNodeId] = useState('');
  const [targetNodeId, setTargetNodeId] = useState('');
  const isEditing = !!editingLink;

  useEffect(() => {
    setSourceNodeId(editingLink?.sourceNodeId || '');
    setTargetNodeId(editingLink?.targetNodeId || '');
  }, [editingLink]);

  return (
    <div className="forms-grid" style={{marginBottom: 16}}>
      <label className="field-stack">
        <span>{nodesT('source')}</span>
        <select className="field-select" onChange={(e) => setSourceNodeId(e.target.value)} value={sourceNodeId}>
          <option value="">{nodesT('selectSource')}</option>
          {nodes.map((n) => (
            <option key={n.id} value={n.id}>{n.name} ({n.mode})</option>
          ))}
        </select>
      </label>
      <label className="field-stack">
        <span>{nodesT('target')}</span>
        <select className="field-select" onChange={(e) => setTargetNodeId(e.target.value)} value={targetNodeId}>
          <option value="">{nodesT('selectTarget')}</option>
          {nodes.map((n) => (
            <option key={n.id} value={n.id}>{n.name} ({n.mode})</option>
          ))}
        </select>
      </label>
      <div className="field-stack" style={{alignSelf: 'flex-end'}}>
        <button
          className="secondary-button"
          disabled={pending || !sourceNodeId || !targetNodeId}
          onClick={() => onSubmit({
            sourceNodeId,
            targetNodeId,
            linkType: editingLink?.linkType || defaultLinkType,
            trustState: editingLink?.trustState || defaultTrustState
          })}
          type="button"
        >
          {pending ? t('common.creating') : isEditing ? nodesT('saveLink') : nodesT('addLink')}
        </button>
        {isEditing && onCancelEdit ? (
          <button className="ghost-button" disabled={pending} onClick={onCancelEdit} type="button">
            {t('common.cancel')}
          </button>
        ) : null}
      </div>
    </div>
  );
}
