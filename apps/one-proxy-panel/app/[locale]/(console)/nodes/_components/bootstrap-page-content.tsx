'use client';

import {useCallback} from 'react';
import {useTranslations} from 'next-intl';

import {AuthGate} from '@/components/auth-gate';

import {BootstrapTokenTab} from './bootstrap-token-tab';
import {BootstrapFormValues} from './types';
import {useNodeConsole} from './use-node-console';

export function NodeBootstrapPageContent() {
  const nodesT = useTranslations('nodesConsole');
  const nodeConsole = useNodeConsole();

  const handleBootstrap = useCallback(
    (data: BootstrapFormValues) => {
      nodeConsole.bootstrap.mutate({
        targetId: data.targetId.trim(),
        nodeName: data.nodeName.trim(),
        nodeMode: data.nodeMode,
        scopeKey: data.scopeKey.trim(),
        parentNodeId: data.parentNodeId.trim(),
        parentReachableUrl: data.parentReachableUrl.trim(),
        publicHost: data.publicHost.trim(),
        publicPort: Number(data.publicPort) || 0
      });
    },
    [nodeConsole.bootstrap]
  );

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card nodes-single-panel">
          <div>
            <p className="section-kicker">{nodesT('bootstrap')}</p>
            <h3>{nodesT('bootstrapToken')}</h3>
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
