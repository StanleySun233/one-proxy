'use client';

import {DndContext, closestCenter, DragEndEvent, PointerSensor, useSensor, useSensors} from '@dnd-kit/core';
import {SortableContext, verticalListSortingStrategy, arrayMove} from '@dnd-kit/sortable';
import {useSortable} from '@dnd-kit/sortable';
import {CSS} from '@dnd-kit/utilities';
import {GripVertical, Plus, X} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {useCallback, useEffect, useRef, useState} from 'react';

import {NameTag} from '@/components/common/name-tag';
import {validateChain} from '@/lib/api';
import {ChainValidationResult, Node, Scope} from '@/lib/types';

type HopItem = {
  id: string;
  nodeId: string;
  nodeName: string;
  nodeMode: string;
};

type ChainEditorProps = {
  accessToken: string;
  chainName: string;
  destinationScope: string;
  hops: string[];
  nodes: Node[];
  scopes: Scope[];
  onNameChange: (name: string) => void;
  onScopeChange: (scope: string) => void;
  onHopsChange: (hops: string[]) => void;
  onSave: () => void;
  onCancel: () => void;
  onPreview: () => void;
  saving: boolean;
  previewing: boolean;
};

export function ChainEditor({
  accessToken,
  chainName,
  destinationScope,
  hops,
  nodes,
  scopes,
  onNameChange,
  onScopeChange,
  onHopsChange,
  onSave,
  onCancel,
  onPreview,
  saving,
  previewing
}: ChainEditorProps) {
  const t = useTranslations();
  const chainsT = useTranslations('chains');
  const [hopItems, setHopItems] = useState<HopItem[]>([]);
  const [selectedNodeId, setSelectedNodeId] = useState<string>('');
  const [validationResult, setValidationResult] = useState<ChainValidationResult | null>(null);
  const [validationPending, setValidationPending] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8
      }
    })
  );

  useEffect(() => {
    const items = hops.map((nodeId, index) => {
      const node = nodes.find((n) => n.id === nodeId);
      return {
        id: `hop-${index}`,
        nodeId,
        nodeName: node?.name || `${t('common.name')} ${nodeId}`,
        nodeMode: node?.mode || 'unknown'
      };
    });
    setHopItems(items);
  }, [hops, nodes]);

  const runValidation = useCallback(async (name: string, scope: string, hopList: string[]) => {
    if (!name.trim() || !scope.trim()) {
      setValidationResult(null);
      return;
    }
    setValidationPending(true);
    try {
      const result = await validateChain(accessToken, {name, destinationScope: scope, hops: hopList});
      setValidationResult(result);
    } catch {
      setValidationResult(null);
    } finally {
      setValidationPending(false);
    }
  }, [accessToken]);

  useEffect(() => {
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    debounceRef.current = setTimeout(() => {
      runValidation(chainName, destinationScope, hops);
    }, 500);
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [chainName, destinationScope, hops, runValidation]);

  const handleDragEnd = (event: DragEndEvent) => {
    const {active, over} = event;
    if (!over || active.id === over.id) {
      return;
    }

    const oldIndex = hopItems.findIndex((item) => item.id === active.id);
    const newIndex = hopItems.findIndex((item) => item.id === over.id);

    const newItems = arrayMove(hopItems, oldIndex, newIndex);
    const newHops = newItems.map((item) => item.nodeId);
    onHopsChange(newHops);
  };

  const handleAddHop = () => {
    if (!selectedNodeId) {
      return;
    }
    if (hops.includes(selectedNodeId)) {
      return;
    }
    onHopsChange([...hops, selectedNodeId]);
    setSelectedNodeId('');
  };

  const handleRemoveHop = (index: number) => {
    const newHops = hops.filter((_, i) => i !== index);
    onHopsChange(newHops);
  };

  const availableNodes = nodes.filter((node) => !hops.includes(node.id));
  return (
    <div className="chain-editor">
      <div className="panel-toolbar">
        <div>
          <p className="section-kicker">{chainsT('chainEditor')}</p>
          <div className="inline-cluster" style={{gap: 8}}>
            <h3>{chainName || chainsT('newChain')}</h3>
            {validationPending && <span className="badge is-neutral">{t('common.validating')}</span>}
            {!validationPending && validationResult && (
              <span className={`badge ${validationResult.valid ? 'is-good' : 'is-danger'}`}>
                {validationResult.valid ? t('common.valid') : t('common.invalid')}
              </span>
            )}
          </div>
        </div>
      </div>

      <div className="forms-grid">
        <label className="field-stack">
          <span>{chainsT('chainName')}</span>
          <input className="field-input" onChange={(e) => onNameChange(e.target.value)} placeholder={chainsT('chainNamePlaceholder')} value={chainName} />
        </label>

        <label className="field-stack">
          <span>{chainsT('destinationScope')}</span>
          <select className="field-select" onChange={(e) => onScopeChange(e.target.value)} value={destinationScope}>
            <option value="">{chainsT('destinationScopePlaceholder')}</option>
            {scopes.map((scope) => (
              <option key={scope.id} value={scope.id}>{scope.name} ({scope.id})</option>
            ))}
          </select>
        </label>
      </div>

      <div className="hop-editor-section">
        <div className="section-header">
          <h4>{chainsT('hops')}</h4>
          <span className="badge">{hopItems.length}</span>
        </div>

        <DndContext collisionDetection={closestCenter} onDragEnd={handleDragEnd} sensors={sensors}>
          <SortableContext items={hopItems.map((item) => item.id)} strategy={verticalListSortingStrategy}>
            <div className="hop-list">
              {hopItems.map((item, index) => (
                <SortableHopCard index={index} item={item} key={item.id} onRemove={() => handleRemoveHop(index)} />
              ))}
            </div>
          </SortableContext>
        </DndContext>

        {hopItems.length === 0 && (
          <div className="empty-hops">
            <span className="muted-text">{chainsT('noHops')}</span>
          </div>
        )}

        <div className="add-hop-section">
          <label className="field-stack">
            <span>{chainsT('addHop')}</span>
            <div className="inline-cluster">
              <select className="field-select" onChange={(e) => setSelectedNodeId(e.target.value)} value={selectedNodeId}>
                <option value="">{chainsT('selectNode')}</option>
                {availableNodes.map((node) => (
                  <option key={node.id} value={node.id}>
                    {node.id} - {node.name} ({node.mode})
                  </option>
                ))}
              </select>
              <button className="secondary-button" disabled={!selectedNodeId} onClick={handleAddHop} type="button">
                <Plus size={16} />
                {t('common.create')}
              </button>
            </div>
          </label>
        </div>
      </div>

      {validationResult && (validationResult.errors.length > 0 || validationResult.warnings.length > 0) && (
        <div className="probe-results-section">
          {validationResult.errors.map((msg, i) => (
            <div className="token-box" key={`err-${i}`} style={{borderColor: 'var(--danger)'}}>
              <span className="field-hint" style={{color: 'var(--danger)'}}>{msg}</span>
            </div>
          ))}
          {validationResult.warnings.map((msg, i) => (
            <div className="token-box" key={`warn-${i}`} style={{borderColor: 'var(--accent)'}}>
              <span className="field-hint" style={{color: 'var(--accent)'}}>{msg}</span>
            </div>
          ))}
        </div>
      )}

      <div className="submit-row">
        <button className="primary-button" disabled={saving || !chainName || !destinationScope || hopItems.length === 0} onClick={onSave} type="button">
          {saving ? t('common.saving') : chainsT('saveChain')}
        </button>
        <button className="secondary-button" disabled={previewing || !chainName || !destinationScope} onClick={onPreview} type="button">
          {previewing ? t('common.compiling') : chainsT('preview')}
        </button>
        <button className="secondary-button" onClick={onCancel} type="button">
          {t('common.cancel')}
        </button>
      </div>
    </div>
  );
}

function SortableHopCard({item, index, onRemove}: {item: HopItem; index: number; onRemove: () => void}) {
  const {attributes, listeners, setNodeRef, transform, transition, isDragging} = useSortable({id: item.id});

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1
  };

  return (
    <div className="hop-card" ref={setNodeRef} style={style}>
      <div className="hop-card-drag" {...attributes} {...listeners}>
        <GripVertical size={16} />
      </div>
      <div className="hop-card-content">
        <div className="hop-card-header">
          <span className="hop-index">{index + 1}</span>
          <NameTag kind="node">{item.nodeName}</NameTag>
          <span className="badge is-neutral">{item.nodeMode}</span>
        </div>
        <span className="muted-text mono">ID: {item.nodeId}</span>
      </div>
      <button className="hop-card-remove" onClick={onRemove} type="button">
        <X size={16} />
      </button>
    </div>
  );
}
