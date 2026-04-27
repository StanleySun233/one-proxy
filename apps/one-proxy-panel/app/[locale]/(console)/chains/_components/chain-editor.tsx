'use client';

import {DndContext, closestCenter, DragEndEvent, PointerSensor, useSensor, useSensors} from '@dnd-kit/core';
import {SortableContext, verticalListSortingStrategy, arrayMove} from '@dnd-kit/sortable';
import {useSortable} from '@dnd-kit/sortable';
import {CSS} from '@dnd-kit/utilities';
import {GripVertical, Plus, X} from 'lucide-react';
import {useCallback, useEffect, useRef, useState} from 'react';

import {validateChain} from '@/lib/control-plane-api';
import {ChainValidationResult, Node} from '@/lib/control-plane-types';

type HopItem = {
  id: string;
  nodeId: number;
  nodeName: string;
  nodeMode: string;
};

type ChainEditorProps = {
  accessToken: string;
  chainName: string;
  destinationScope: string;
  hops: number[];
  nodes: Node[];
  onNameChange: (name: string) => void;
  onScopeChange: (scope: string) => void;
  onHopsChange: (hops: number[]) => void;
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
  onNameChange,
  onScopeChange,
  onHopsChange,
  onSave,
  onCancel,
  onPreview,
  saving,
  previewing
}: ChainEditorProps) {
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
      const node = nodes.find((n) => Number(n.id) === nodeId);
      return {
        id: `hop-${index}`,
        nodeId,
        nodeName: node?.name || `Node ${nodeId}`,
        nodeMode: node?.mode || 'unknown'
      };
    });
    setHopItems(items);
  }, [hops, nodes]);

  const runValidation = useCallback(async (name: string, scope: string, hopList: number[]) => {
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
    const nodeId = Number(selectedNodeId);
    if (hops.includes(nodeId)) {
      return;
    }
    onHopsChange([...hops, nodeId]);
    setSelectedNodeId('');
  };

  const handleRemoveHop = (index: number) => {
    const newHops = hops.filter((_, i) => i !== index);
    onHopsChange(newHops);
  };

  const availableNodes = nodes.filter((node) => !hops.includes(Number(node.id)));
  const availableScopes = Array.from(new Set(nodes.map((node) => node.scopeKey).filter(Boolean)));

  return (
    <div className="chain-editor">
      <div className="panel-toolbar">
        <div>
          <p className="section-kicker">Chain Editor</p>
          <div className="inline-cluster" style={{gap: 8}}>
            <h3>{chainName || 'New Chain'}</h3>
            {validationPending && <span className="badge is-neutral">validating...</span>}
            {!validationPending && validationResult && (
              <span className={`badge ${validationResult.valid ? 'is-good' : 'is-danger'}`}>
                {validationResult.valid ? 'valid' : 'invalid'}
              </span>
            )}
          </div>
          <p className="section-copy">Configure chain hops and destination scope</p>
        </div>
      </div>

      <div className="forms-grid">
        <label className="field-stack">
          <span>Chain Name</span>
          <input className="field-input" onChange={(e) => onNameChange(e.target.value)} placeholder="prod-k8s-path" value={chainName} />
        </label>

        <label className="field-stack">
          <span>Destination Scope</span>
          <input
            className="field-input"
            list="scope-suggestions"
            onChange={(e) => onScopeChange(e.target.value)}
            placeholder="k8s-prod"
            value={destinationScope}
          />
          <datalist id="scope-suggestions">
            {availableScopes.map((scope) => (
              <option key={scope} value={scope} />
            ))}
          </datalist>
        </label>
      </div>

      <div className="hop-editor-section">
        <div className="section-header">
          <h4>Hops</h4>
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
            <span className="muted-text">No hops added yet. Add nodes to create a chain.</span>
          </div>
        )}

        <div className="add-hop-section">
          <label className="field-stack">
            <span>Add Hop</span>
            <div className="inline-cluster">
              <select className="field-select" onChange={(e) => setSelectedNodeId(e.target.value)} value={selectedNodeId}>
                <option value="">Select a node</option>
                {availableNodes.map((node) => (
                  <option key={node.id} value={node.id}>
                    {node.id} - {node.name} ({node.mode})
                  </option>
                ))}
              </select>
              <button className="secondary-button" disabled={!selectedNodeId} onClick={handleAddHop} type="button">
                <Plus size={16} />
                Add
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
          {saving ? 'Saving...' : 'Save Chain'}
        </button>
        <button className="secondary-button" disabled={previewing || !chainName || !destinationScope} onClick={onPreview} type="button">
          {previewing ? 'Compiling...' : 'Preview'}
        </button>
        <button className="secondary-button" onClick={onCancel} type="button">
          Cancel
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
          <strong>{item.nodeName}</strong>
          <span className="badge is-neutral">{item.nodeMode}</span>
        </div>
        <span className="muted-text mono">Node ID: {item.nodeId}</span>
      </div>
      <button className="hop-card-remove" onClick={onRemove} type="button">
        <X size={16} />
      </button>
    </div>
  );
}
