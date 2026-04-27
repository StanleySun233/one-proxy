'use client';

import {AuthGate} from '@/components/auth-gate';

import {BootstrapTokenTab} from './bootstrap-token-tab';
import {useNodeConsole} from './use-node-console';

export function NodeBootstrapPageContent() {
  const nodeConsole = useNodeConsole();

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
            onSubmit={() => {
              const values = nodeConsole.bootstrapForm.getValues();
              nodeConsole.bootstrap.mutate({targetId: values.targetId.trim(), nodeName: values.nodeName.trim()});
            }}
          />
        </section>
      </div>
    </AuthGate>
  );
}
