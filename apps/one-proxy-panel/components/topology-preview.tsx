'use client';

import 'reactflow/dist/style.css';

import {Background, Controls, MarkerType, MiniMap, ReactFlow} from 'reactflow';

import {Node as ControlNode, NodeAccessPath} from '@/lib/types';

export function TopologyPreview({nodes, paths}: {nodes?: ControlNode[]; paths?: NodeAccessPath[]}) {
  const visibleNodes = nodes && nodes.length > 0 ? nodes.slice(0, 6) : [];
  const flowNodes = [
    {id: 'panel', position: {x: 30, y: 120}, data: {label: 'panel'}, type: 'input'},
    ...visibleNodes.map((node, index) => ({
      id: node.id,
      position: {x: 270 + (index % 3) * 220, y: 30 + Math.floor(index / 3) * 170},
      data: {label: `${node.name} · ${node.status}`}
    }))
  ];

  const flowEdges = [
    ...visibleNodes
      .filter((node) => node.parentNodeId)
      .map((node) => ({
        id: `parent-${node.parentNodeId}-${node.id}`,
        source: node.parentNodeId,
        target: node.id,
        markerEnd: {type: MarkerType.ArrowClosed}
      })),
    ...(paths || [])
      .filter((path) => path.entryNodeId)
      .map((path) => ({
        id: `path-${path.id}`,
        source: 'panel',
        target: path.entryNodeId,
        markerEnd: {type: MarkerType.ArrowClosed},
        label: path.mode
      }))
  ];

  return (
    <div className="flow-card">
      <ReactFlow edges={flowEdges} fitView nodes={flowNodes} proOptions={{hideAttribution: true}}>
        <MiniMap pannable zoomable />
        <Controls showInteractive={false} />
        <Background gap={18} size={1.2} />
      </ReactFlow>
    </div>
  );
}
