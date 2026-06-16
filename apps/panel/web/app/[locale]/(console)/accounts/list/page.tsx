'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useMemo, useState} from 'react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AsyncState} from '@/components/async-state';
import {AuthGate} from '@/components/auth-gate';
import {NameTag} from '@/components/common/name-tag';
import {useAuth} from '@/components/auth-provider';
import {ConsoleCrudModal, ConsoleFilterBar, ConsoleFilterItem, ConsoleList, ConsolePage} from '@/components/console-template';
import {DeleteConfirmationModal, DeleteImpactSection} from '@/components/delete-confirmation-modal';
import {createAccount, deleteAccount, fetchEnums, getAccounts} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';

import EditAccountDialog from '../_components/edit-account-dialog';

type AccountFormValues = {
  account: string;
  password: string;
  role: string;
};

export default function AccountListPage() {
  const t = useTranslations();
  const accountsT = useTranslations('accounts');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const isSuperAdmin = session?.account.role === 'super_admin';

  const [editAccount, setEditAccount] = useState<{id: string; account: string; role: string; status: string} | null>(null);
  const [nameFilter, setNameFilter] = useState('');
  const [roleFilter, setRoleFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [createOpen, setCreateOpen] = useState(false);
  const [deletingAccount, setDeletingAccount] = useState<{id: string; account: string; role: string; status: string} | null>(null);
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const accountRoleKeys = Object.keys(enums?.account_role || {});
  const DEFAULT_ROLE = accountRoleKeys.find(k => k === 'user') || 'user';
  const accountRoleOptions = enums?.account_role ? Object.entries(enums.account_role).map(([value, item]) => ({value, label: item.name})) : [];
  const createForm = useForm<AccountFormValues>({
    defaultValues: {
      account: '',
      password: '',
      role: DEFAULT_ROLE
    }
  });

  const accountsQuery = useQuery({
    queryKey: ['accounts', accessToken],
    queryFn: () => getAccounts(accessToken),
    enabled: !!accessToken
  });

  const deleteAccountMutation = useMutation({
    mutationFn: (accountID: string) => deleteAccount(accessToken, accountID),
    onSuccess: () => {
      toast.success(accountsT('deleteSuccess'));
      queryClient.invalidateQueries({queryKey: ['accounts']});
      setDeletingAccount(null);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });
  const createAccountMutation = useMutation({
    mutationFn: (payload: AccountFormValues) => createAccount(accessToken, payload),
    onSuccess: () => {
      toast.success(accountsT('createSuccess'));
      queryClient.invalidateQueries({queryKey: ['accounts']});
      createForm.reset({account: '', password: '', role: DEFAULT_ROLE});
      setCreateOpen(false);
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  const accounts = accountsQuery.data || [];
  const filteredAccounts = useMemo(() => accounts.filter((account) =>
    (!nameFilter.trim() || account.account.toLowerCase().includes(nameFilter.trim().toLowerCase())) &&
    (!roleFilter.trim() || account.role === roleFilter) &&
    (!statusFilter.trim() || account.status === statusFilter)
  ), [accounts, nameFilter, roleFilter, statusFilter]);
  const roleOptions = Array.from(new Set(accounts.map((account) => account.role))).sort();
  const statusOptions = Array.from(new Set(accounts.map((account) => account.status))).sort();
  const accountDeleteSections: DeleteImpactSection[] = deletingAccount ? [
    {id: 'account', label: accountsT('deleteImpactAccount'), items: [{id: deletingAccount.id, name: deletingAccount.account, detail: deletingAccount.role}]}
  ] : [];

  return (
    <AuthGate>
      <ConsolePage
        actions={isSuperAdmin ? <button className="primary-button" onClick={() => setCreateOpen(true)} type="button">{accountsT('createTitle')}</button> : null}
        title={accountsT('listTitle')}
      >
        <ConsoleFilterBar title={t('common.filter')}>
          <ConsoleFilterItem label={t('common.name')} match={t('common.contains')}>
            <input className="field-input" onChange={(event) => setNameFilter(event.target.value)} placeholder={t('common.name')} value={nameFilter} />
          </ConsoleFilterItem>
          <ConsoleFilterItem label={accountsT('fieldRole')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setRoleFilter(event.target.value)} value={roleFilter}>
              <option value="">{t('common.all')}</option>
              {roleOptions.map((role) => <option key={role} value={role}>{role}</option>)}
            </select>
          </ConsoleFilterItem>
          <ConsoleFilterItem label={accountsT('fieldStatus')} match={t('common.equals')}>
            <select className="field-select" onChange={(event) => setStatusFilter(event.target.value)} value={statusFilter}>
              <option value="">{t('common.all')}</option>
              {statusOptions.map((status) => <option key={status} value={status}>{status}</option>)}
            </select>
          </ConsoleFilterItem>
        </ConsoleFilterBar>

        <ConsoleList count={filteredAccounts.length} title={accountsT('listTitle')}>
          {accountsQuery.isPending ? (
            <AsyncState detail={t('common.loading')} title={accountsT('loadingAccounts')} />
          ) : accountsQuery.isError ? (
            <AsyncState
              actionLabel={t('common.retry')}
              detail={formatControlPlaneError(accountsQuery.error)}
              onAction={() => void accountsQuery.refetch()}
              title={accountsT('failedToLoadAccounts')}
            />
          ) : accounts.length === 0 ? (
            <AsyncState detail={accountsT('emptyAccountsList')} title={t('common.empty')} />
          ) : filteredAccounts.length === 0 ? (
            <AsyncState detail={t('common.noMatching')} title={t('common.empty')} />
          ) : (
            <div className="table-card">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>{t('common.name')}</th>
                    <th>{accountsT('fieldRole')}</th>
                    <th>{accountsT('fieldStatus')}</th>
                    {isSuperAdmin ? <th>{t('common.actions')}</th> : null}
                  </tr>
                </thead>
                <tbody>
                  {filteredAccounts.map((account) => (
                    <tr key={account.id}>
                      <td><NameTag kind="account">{account.account}</NameTag></td>
                      <td><span className="badge is-neutral">{account.role}</span></td>
                      <td>{account.status}</td>
                      {isSuperAdmin ? (
                        <td>
                          <div className="chain-list-actions">
                            <button className="secondary-button" onClick={() => setEditAccount(account)} type="button">
                              {t('common.edit')}
                            </button>
                            {account.account !== 'admin' ? (
                              <button
                                className="danger-button"
                                disabled={deleteAccountMutation.isPending}
                                onClick={() => setDeletingAccount(account)}
                                type="button"
                              >
                                {t('common.delete')}
                              </button>
                            ) : null}
                          </div>
                        </td>
                      ) : null}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </ConsoleList>

        {isSuperAdmin ? (
          <ConsoleCrudModal
            footer={(
              <>
                <button className="secondary-button" onClick={() => setCreateOpen(false)} type="button">{t('common.cancel')}</button>
                <button className="primary-button" disabled={createAccountMutation.isPending} onClick={() => void createForm.handleSubmit((values) => {
                  createAccountMutation.mutate({
                    account: values.account.trim(),
                    password: values.password,
                    role: values.role.trim()
                  });
                })()} type="button">
                  {createAccountMutation.isPending ? t('common.submitting') : accountsT('createTitle')}
                </button>
              </>
            )}
            onClose={() => setCreateOpen(false)}
            open={createOpen}
            title={accountsT('createTitle')}
          >
            <div className="sub-grid">
              <div className="field-stack">
                <span>{accountsT('fieldAccount')}</span>
                <input
                  aria-invalid={createForm.formState.errors.account ? 'true' : 'false'}
                  className="field-input"
                  placeholder={accountsT('placeholderAccount')}
                  {...createForm.register('account', {required: accountsT('accountRequired')})}
                />
                {createForm.formState.errors.account ? <p className="error-text">{createForm.formState.errors.account.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>{accountsT('fieldPassword')}</span>
                <input
                  aria-invalid={createForm.formState.errors.password ? 'true' : 'false'}
                  className="field-input"
                  type="password"
                  {...createForm.register('password', {
                    required: accountsT('passwordRequired'),
                    minLength: {value: 8, message: accountsT('passwordMinLength')}
                  })}
                />
                {createForm.formState.errors.password ? <p className="error-text">{createForm.formState.errors.password.message}</p> : null}
              </div>
              <div className="field-stack">
                <span>{accountsT('fieldRole')}</span>
                <select className="field-select" {...createForm.register('role', {required: accountsT('roleRequired')})}>
                  {accountRoleOptions.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
                {createForm.formState.errors.role ? <p className="error-text">{createForm.formState.errors.role.message}</p> : null}
              </div>
            </div>
          </ConsoleCrudModal>
        ) : null}

        {isSuperAdmin && editAccount ? (
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

        <DeleteConfirmationModal
          onClose={() => setDeletingAccount(null)}
          onConfirm={() => {
            if (deletingAccount) {
              deleteAccountMutation.mutate(deletingAccount.id);
            }
          }}
          open={Boolean(deletingAccount)}
          pending={deleteAccountMutation.isPending}
          sections={accountDeleteSections}
          targetName={deletingAccount?.account || ''}
          title={accountsT('deleteAccountTitle')}
        />
      </ConsolePage>
    </AuthGate>
  );
}
