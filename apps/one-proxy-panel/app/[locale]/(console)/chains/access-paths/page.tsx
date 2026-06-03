'use client';

import {useQuery} from '@tanstack/react-query';
import {useTranslations} from 'next-intl';

import {AuthGate} from '@/components/auth-gate';
import {ConsolePage} from '@/components/console-template';
import {useAuth} from '@/components/auth-provider';
import {getChains, getNodes} from '@/lib/api';

import {AccessPathPanel} from '../studio/_components/access-path-panel';

export default function ChainAccessPathsPage() {
  const t = useTranslations();
  const {session, activeTenant} = useAuth();
  const accessToken = session?.accessToken || '';
  const activeTenantId = session?.activeTenantId || null;
  const canWrite = session?.account.role === 'super_admin' || activeTenant?.role === 'tenant_admin';

  const chainsQuery = useQuery({
    queryKey: ['chains', accessToken, activeTenantId],
    queryFn: () => getChains(accessToken),
    enabled: !!accessToken
  });

  const nodesQuery = useQuery({
    queryKey: ['nodes', accessToken, activeTenantId],
    queryFn: () => getNodes(accessToken),
    enabled: !!accessToken
  });

  return (
    <AuthGate>
      <ConsolePage title={t('shell.accessPaths')}>
        {canWrite ? <AccessPathPanel accessToken={accessToken} activeTenantId={activeTenantId} chains={chainsQuery.data || []} nodes={nodesQuery.data || []} /> : null}
      </ConsolePage>
    </AuthGate>
  );
}
