'use client';

import {useCallback} from 'react';
import {useTranslations} from 'next-intl';

import {AuthGate} from '@/components/auth-gate';

import {ManualNodeTab} from './manual-node-tab';
import {NodeFormValues} from './types';
import {useNodeConsole} from './use-node-console';

export function NodeManualPageContent() {
  const nodesT = useTranslations('nodesConsole');
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
            <p className="section-kicker">{nodesT('nodeRecord')}</p>
            <h3>{nodesT('manualRecord')}</h3>
          </div>
          <ManualNodeTab
            form={nodeConsole.nodeForm}
            scopes={nodeConsole.scopesQuery.data || []}
            submitting={nodeConsole.createNode.isPending}
            onSubmit={handleCreateNode}
          />
        </section>
      </div>
    </AuthGate>
  );
}
