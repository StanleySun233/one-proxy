'use client';

import {useState} from 'react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {useAuth} from '@/components/auth-provider';
import {formatControlPlaneError} from '@/lib/presentation';

export function LoginForm() {
  const {login} = useAuth();
  const t = useTranslations();
  const [account, setAccount] = useState('');
  const [password, setPassword] = useState('');
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError('');

    try {
      await login(account, password);
    } catch (err) {
      const message = formatControlPlaneError(err);
      setError(message);
      toast.error(message);
    } finally {
      setPending(false);
    }
  }

  return (
    <form className="login-form" onSubmit={onSubmit}>
      <div className="panel-card">
        <p className="section-kicker">{t('shell.product')}</p>
        <h3>{t('auth.title')}</h3>
        <p className="section-copy">{t('auth.description')}</p>

        <label className="field-stack">
          <span>{t('auth.account')}</span>
          <input className="field-input" onChange={(event) => setAccount(event.target.value)} value={account} />
        </label>

        <label className="field-stack">
          <span>{t('auth.password')}</span>
          <input className="field-input" onChange={(event) => setPassword(event.target.value)} type="password" value={password} />
        </label>

        {error ? <p className="error-text">{error}</p> : null}

        <button className="primary-button" disabled={pending} type="submit">
          {pending ? t('auth.loading') : t('auth.submit')}
        </button>
      </div>
    </form>
  );
}
