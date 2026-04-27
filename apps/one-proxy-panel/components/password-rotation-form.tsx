'use client';

import {useState} from 'react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {useAuth} from '@/components/auth-provider';
import {formatControlPlaneError} from '@/lib/presentation';

export function PasswordRotationForm() {
  const {rotatePassword} = useAuth();
  const t = useTranslations();
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError('');

    if (password.length < 8) {
      setError(t('auth.passwordRule'));
      return;
    }
    if (password !== confirmPassword) {
      setError(t('auth.passwordMismatch'));
      return;
    }

    setPending(true);
    try {
      await rotatePassword(password);
      toast.success(t('auth.passwordChanged'));
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
        <h3>{t('auth.rotateTitle')}</h3>
        <p className="section-copy">{t('auth.rotateDescription')}</p>

        <label className="field-stack">
          <span>{t('auth.newPassword')}</span>
          <input className="field-input" onChange={(event) => setPassword(event.target.value)} type="password" value={password} />
        </label>

        <label className="field-stack">
          <span>{t('auth.confirmPassword')}</span>
          <input className="field-input" onChange={(event) => setConfirmPassword(event.target.value)} type="password" value={confirmPassword} />
        </label>

        {error ? <p className="error-text">{error}</p> : null}

        <button className="primary-button" disabled={pending} type="submit">
          {pending ? t('auth.passwordChanging') : t('auth.passwordChangeSubmit')}
        </button>
      </div>
    </form>
  );
}
