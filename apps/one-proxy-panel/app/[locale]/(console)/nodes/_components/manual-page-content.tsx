'use client';

import {useCallback} from 'react';

import {AuthGate} from '@/components/auth-gate';

import {ManualNodeTab} from './manual-node-tab';
import {NodeFormValues} from './types';
import {useNodeConsole} from './use-node-console';

export function NodeManualPageContent() {
  const nodeConsole = useNodeConsole();

  const handleCreateNode = useCallback(
    (values: NodeFormValues) => {
      nodeConsole.createNode.mutate({
        name: values.name.trim(),
        mode: values.mode,
        scopeKey: values.scopeKey.trim(),
        parentNodeId: values.parentNodeId.trim(),
        publicHost: values.publicHost.trim(),
        publicPort: values.publicPort ? Number(values.publicPort) : 0
      });
    },
    [nodeConsole.createNode]
  );

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card nodes-single-panel">
          <div>
            <p className="section-kicker">Node record</p>
            <h3>Manual record</h3>
            <p className="section-copy">Create node metadata first, then bind the real runtime later.</p>
          </div>
          <ManualNodeTab
            form={nodeConsole.nodeForm}
            submitting={nodeConsole.createNode.isPending}
            onSubmit={handleCreateNode}
          />
        </section>
      </div>
    </AuthGate>
  );
}
