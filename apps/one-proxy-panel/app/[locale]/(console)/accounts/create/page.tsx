'use client';

import {useMutation, useQuery, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {createAccount, fetchEnums} from '@/lib/api';
import {formatControlPlaneError} from '@/lib/presentation';

type AccountFormValues = {
  account: string;
  password: string;
  role: string;
};

export default function CreateAccountPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const accountsT = useTranslations('accounts');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const {data: enums} = useQuery({queryKey: ['enums'], queryFn: () => fetchEnums()});
  const accountRoleKeys = Object.keys(enums?.account_role || {});
  const DEFAULT_ROLE = accountRoleKeys.find(k => k === 'operator') || 'operator';
  const accountRoleOptions = enums?.account_role ? Object.entries(enums.account_role).map(([value, item]) => ({value, label: item.name})) : [];
  const form = useForm<AccountFormValues>({
    defaultValues: {
      account: '',
      password: '',
      role: DEFAULT_ROLE
    }
  });

  const createAccountMutation = useMutation({
    mutationFn: (payload: AccountFormValues) => createAccount(accessToken, payload),
    onSuccess: () => {
      toast.success(accountsT('createSuccess'));
      queryClient.invalidateQueries({queryKey: ['accounts']});
      form.reset({account: '', password: '', role: DEFAULT_ROLE});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow={accountsT('listTitle')} title={pageT('accountsTitle')} description={pageT('accountsDesc')} />
        <article className="panel-card">
          <h3>{accountsT('createTitle')}</h3>
          <form
            className="sub-grid"
            onSubmit={form.handleSubmit((values) => {
              createAccountMutation.mutate({
                account: values.account.trim(),
                password: values.password,
                role: values.role.trim()
              });
            })}
          >
            <div className="field-stack">
              <span>{accountsT('fieldAccount')}</span>
              <input
                aria-invalid={form.formState.errors.account ? 'true' : 'false'}
                className="field-input"
                placeholder={accountsT('placeholderAccount')}
                {...form.register('account', {required: accountsT('accountRequired')})}
              />
              {form.formState.errors.account ? <p className="error-text">{form.formState.errors.account.message}</p> : null}
            </div>
            <div className="field-stack">
              <span>{accountsT('fieldPassword')}</span>
              <input
                aria-invalid={form.formState.errors.password ? 'true' : 'false'}
                className="field-input"
                type="password"
                {...form.register('password', {
                  required: accountsT('passwordRequired'),
                  minLength: {value: 8, message: accountsT('passwordMinLength')}
                })}
              />
              {form.formState.errors.password ? <p className="error-text">{form.formState.errors.password.message}</p> : null}
            </div>
            <div className="field-stack">
              <span>{accountsT('fieldRole')}</span>
              <select
                className="field-select"
                {...form.register('role', {required: accountsT('roleRequired')})}
              >
                {accountRoleOptions.map((opt) => (
                  <option key={opt.value} value={opt.value}>{opt.label}</option>
                ))}
              </select>
              {form.formState.errors.role ? <p className="error-text">{form.formState.errors.role.message}</p> : null}
            </div>
            <div className="submit-row">
              <button className="primary-button" disabled={createAccountMutation.isPending} type="submit">
                {createAccountMutation.isPending ? t('common.submitting') : accountsT('createTitle')}
              </button>
            </div>
          </form>
        </article>
      </div>
    </AuthGate>
  );
}
