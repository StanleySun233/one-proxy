'use client';

import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';
import {useState} from 'react';

import {AuthGate} from '@/components/auth-gate';
import {ConsolePage} from '@/components/console-template';
import {useAuth} from '@/components/auth-provider';
import {getChains, getNodes} from '@/lib/api';

import {AccessPathPanel} from '../studio/_components/access-path-panel';

export default function ChainAccessPathsPage() {
  const t = useTranslations();
  const accessPathsT = useTranslations('accessPaths');
  const {session, activeTenant} = useAuth();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';
  const [createRequestKey, setCreateRequestKey] = useState(0);

  const chainsQuery = useQuery({
    queryKey: ['proxy-chains', accessToken, activeTenantId],
    queryFn: () => getChains(accessToken, activeTenantId),
    enabled: !!accessToken
  });

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken, activeTenantId],
    queryFn: () => getNodes(accessToken, activeTenantId),
    enabled: !!accessToken
  });

  return (
    <AuthGate>
      <ConsolePage
        actions={canWrite ? (
          <button className="primary-button" onClick={() => setCreateRequestKey((current) => current + 1)} type="button">
            {accessPathsT('create')}
          </button>
        ) : null}
        title={t('shell.accessPaths')}
      >
        {canWrite ? <AccessPathPanel accessToken={accessToken} activeTenantId={activeTenantId} chains={chainsQuery.data || []} createRequestKey={createRequestKey} nodes={nodesQuery.data || []} /> : null}
      </ConsolePage>
    </AuthGate>
  );
}
