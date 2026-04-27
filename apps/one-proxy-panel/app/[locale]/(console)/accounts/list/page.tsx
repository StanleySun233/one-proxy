'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useState} from 'react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {deleteAccount, getAccounts} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';

import EditAccountDialog from '../_components/edit-account-dialog';

export default function AccountListPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';

  const [editAccount, setEditAccount] = useState<{id: string; account: string; role: string; status: string} | null>(null);

  const accountsQuery = useQuery({
    queryKey: ['accounts', accessToken],
    queryFn: () => getAccounts(accessToken),
    enabled: !!accessToken
  });

  const deleteAccountMutation = useMutation({
    mutationFn: (accountID: string) => deleteAccount(accessToken, accountID),
    onSuccess: () => {
      toast.success('account deleted');
      queryClient.invalidateQueries({queryKey: ['accounts']});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const accounts = accountsQuery.data || [];

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Accounts" title={pageT('accountsTitle')} description={pageT('accountsDesc')} />
        <article className="panel-card soft-card">
          <h3>Accounts</h3>
          {accountsQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title="Loading accounts" />
          ) : accountsQuery.isError ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(accountsQuery.error)}
              onAction={() => void accountsQuery.refetch()}
              title="Failed to load accounts"
            />
          ) : accounts.length === 0 ? (
            <AsyncState detail="Create additional operators when you need delegated access." title={t('common.empty')} />
          ) : (
            <div className="stack-list">
              {accounts.map((account) => (
                <div className="stack-item" key={account.id}>
                  <div className="stack-head">
                    <strong>{account.account}</strong>
                    <span className="badge">{account.role}</span>
                  </div>
                  <span className="muted-text">{account.status}</span>
                  <span className="mono">{account.id}</span>
                  <div className="stack-actions">
                    <button className="secondary-button" onClick={() => setEditAccount(account)} type="button">
                      Edit
                    </button>
                    {account.account !== 'admin' ? (
                      <button
                        className="secondary-button"
                        disabled={deleteAccountMutation.isPending}
                        onClick={() => {
                          if (window.confirm(`Delete account ${account.account}?`)) {
                            deleteAccountMutation.mutate(account.id);
                          }
                        }}
                        type="button"
                      >
                        Delete
                      </button>
                    ) : null}
                  </div>
                </div>
              ))}
            </div>
          )}
        </article>
      </div>

      {editAccount ? (
        <EditAccountDialog
          accessToken={accessToken}
          account={editAccount}
          onClose={() => setEditAccount(null)}
          onSaved={() => {
            queryClient.invalidateQueries({queryKey: ['accounts']});
          }}
          open={!!editAccount}
        />
      ) : null}
    </AuthGate>
  );
}
