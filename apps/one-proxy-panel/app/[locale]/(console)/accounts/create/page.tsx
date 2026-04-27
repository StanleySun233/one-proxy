'use client';

import {useMutation, useQueryClient} from '@tanstack/react-query';
import {useForm} from 'react-hook-form';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {AuthGate} from '@/components/auth-gate';
import {useAuth} from '@/components/auth-provider';
import {PageHero} from '@/components/page-hero';
import {createAccount} from '@/lib/control-plane-api';
import {formatControlPlaneError} from '@/lib/presentation';

type AccountFormValues = {
  account: string;
  password: string;
  role: string;
};

export default function CreateAccountPage() {
  const t = useTranslations();
  const pageT = useTranslations('pages');
  const {session} = useAuth();
  const queryClient = useQueryClient();
  const accessToken = session?.accessToken || '';
  const form = useForm<AccountFormValues>({
    defaultValues: {
      account: '',
      password: '',
      role: 'operator'
    }
  });

  const createAccountMutation = useMutation({
    mutationFn: (payload: AccountFormValues) => createAccount(accessToken, payload),
    onSuccess: () => {
      toast.success('account created');
      queryClient.invalidateQueries({queryKey: ['accounts']});
      form.reset({account: '', password: '', role: 'operator'});
    },
    onError: (error) => {
      toast.error(formatControlPlaneError(error));
    }
  });

  return (
    <AuthGate>
      <div className="page-stack">
        <PageHero eyebrow="Accounts" title={pageT('accountsTitle')} description={pageT('accountsDesc')} />
        <article className="panel-card">
          <h3>Create account</h3>
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
              <span>Account</span>
              <input
                aria-invalid={form.formState.errors.account ? 'true' : 'false'}
                className="field-input"
                placeholder="operator-a"
                {...form.register('account', {required: 'account is required'})}
              />
              {form.formState.errors.account ? <p className="error-text">{form.formState.errors.account.message}</p> : null}
            </div>
            <div className="field-stack">
              <span>Password</span>
              <input
                aria-invalid={form.formState.errors.password ? 'true' : 'false'}
                className="field-input"
                type="password"
                {...form.register('password', {
                  required: 'password is required',
                  minLength: {value: 8, message: 'password must be at least 8 characters'}
                })}
              />
              {form.formState.errors.password ? <p className="error-text">{form.formState.errors.password.message}</p> : null}
            </div>
            <div className="field-stack">
              <span>Role</span>
              <input
                aria-invalid={form.formState.errors.role ? 'true' : 'false'}
                className="field-input"
                placeholder="operator"
                {...form.register('role', {required: 'role is required'})}
              />
              {form.formState.errors.role ? <p className="error-text">{form.formState.errors.role.message}</p> : null}
            </div>
            <div className="submit-row">
              <button className="primary-button" disabled={createAccountMutation.isPending} type="submit">
                {createAccountMutation.isPending ? t('common.submitting') : 'Create account'}
              </button>
            </div>
          </form>
        </article>
      </div>
    </AuthGate>
  );
}
