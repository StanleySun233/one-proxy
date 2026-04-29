'use client';

import {useCallback} from 'react';

import {AuthGate} from '@/components/auth-gate';

import {BootstrapTokenTab} from './bootstrap-token-tab';
import {BootstrapFormValues} from './types';
import {useNodeConsole} from './use-node-console';

export function NodeBootstrapPageContent() {
  const nodeConsole = useNodeConsole();

  const handleBootstrap = useCallback(
    (data: BootstrapFormValues) => {
      nodeConsole.bootstrap.mutate({targetId: data.targetId.trim(), nodeName: data.nodeName.trim()});
    },
    [nodeConsole.bootstrap]
  );

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card nodes-single-panel">
          <div>
            <p className="section-kicker">Bootstrap</p>
            <h3>Bootstrap token</h3>
            <p className="section-copy">Issue a one-time token for remote self-enroll flows.</p>
          </div>
          <BootstrapTokenTab
            form={nodeConsole.bootstrapForm}
            latestToken={nodeConsole.latestToken}
            submitting={nodeConsole.bootstrap.isPending}
            nodes={nodeConsole.nodesQuery.data || []}
            onSubmit={handleBootstrap}
          />
        </section>
      </div>
    </AuthGate>
  );
}
