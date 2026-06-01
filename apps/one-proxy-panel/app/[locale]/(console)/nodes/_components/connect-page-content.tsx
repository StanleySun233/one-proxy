'use client';

import {useCallback} from 'react';
import {useTranslations} from 'next-intl';

import {AuthGate} from '@/components/auth-gate';

import {QuickConnectTab} from './quick-connect-tab';
import {QuickConnectFormValues} from './types';
import {useNodeConsole} from './use-node-console';

export function NodeConnectPageContent() {
  const nodesT = useTranslations('nodesConsole');
  const nodeConsole = useNodeConsole();

  const handleQuickConnect = useCallback(
    (values: QuickConnectFormValues) => {
      nodeConsole.quickConnect.mutate({
        address: values.address.trim(),
        password: values.password,
        newPassword: values.newPassword,
        name: values.name.trim(),
        mode: values.mode,
        scopeKey: values.scopeKey.trim(),
        parentNodeId: values.parentNodeId.trim(),
        publicHost: values.publicHost.trim(),
        publicPort: values.publicPort ? Number(values.publicPort) : 0,
        controlPlaneUrl: values.controlPlaneUrl.trim()
      });
    },
    [nodeConsole.quickConnect]
  );

  return (
    <AuthGate>
      <div className="page-stack">
        <section className="panel-card nodes-single-panel">
          <div>
            <p className="section-kicker">{nodesT('nodeEntry')}</p>
            <h3>{nodesT('quickConnect')}</h3>
          </div>
          <QuickConnectTab
            form={nodeConsole.quickConnectForm}
            submitting={nodeConsole.quickConnect.isPending}
            onSubmit={handleQuickConnect}
          />
        </section>
      </div>
    </AuthGate>
  );
}
