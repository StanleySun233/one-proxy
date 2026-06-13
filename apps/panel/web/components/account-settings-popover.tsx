'use client';

import {FormEvent, useCallback, useEffect, useId, useRef, useState} from 'react';
import {LogOut, Settings, UserRound} from 'lucide-react';
import {useTranslations} from 'next-intl';
import {toast} from 'sonner';

import {useAuth} from '@/components/auth-provider';
import {formatControlPlaneError} from '@/lib/presentation';

type AccountSettingsPopoverProps = {
  accountInitial: string;
  accountRoleLabel: string;
};

export function AccountSettingsPopover({accountInitial, accountRoleLabel}: AccountSettingsPopoverProps) {
  const t = useTranslations();
  const {session, rotatePassword, logout} = useAuth();
  const [open, setOpen] = useState(false);
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [pending, setPending] = useState(false);
  const [logoutPending, setLogoutPending] = useState(false);
  const [error, setError] = useState('');
  const dialogId = useId();
  const rootRef = useRef<HTMLDivElement>(null);

  const closePopover = useCallback(() => {
    setOpen(false);
    setError('');
    setPassword('');
    setConfirmPassword('');
  }, []);

  useEffect(() => {
    if (!open) {
      return;
    }

    const handlePointerDown = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        closePopover();
      }
    };
    document.addEventListener('pointerdown', handlePointerDown);
    return () => document.removeEventListener('pointerdown', handlePointerDown);
  }, [closePopover, open]);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
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
      closePopover();
      toast.success(t('auth.passwordChanged'));
    } catch (err) {
      const message = formatControlPlaneError(err);
      setError(message);
      toast.error(message);
    } finally {
      setPending(false);
    }
  }

  async function onLogout() {
    setLogoutPending(true);
    closePopover();
    await logout();
  }

  if (!session) {
    return null;
  }

  return (
    <div className="account-settings" ref={rootRef}>
      <button
        aria-expanded={open}
        aria-haspopup="dialog"
        aria-controls={open ? dialogId : undefined}
        className="console-user-card"
        onClick={() => {
          if (open) {
            closePopover();
          } else {
            setOpen(true);
          }
        }}
        type="button"
      >
        <div className="console-user-avatar">{accountInitial}</div>
        <div className="console-user-copy">
          <strong>{session.account.account || t('shell.name')}</strong>
          <span>{accountRoleLabel}</span>
        </div>
        <Settings size={15} />
      </button>

      {open ? (
        <div aria-modal="false" className="account-settings-panel" id={dialogId} role="dialog">
          <div className="account-settings-head">
            <div className="console-user-avatar is-large">{accountInitial}</div>
            <div>
              <p className="section-kicker">{t('auth.profileTitle')}</p>
              <h3>{session.account.account || t('shell.name')}</h3>
              <span>{accountRoleLabel}</span>
            </div>
          </div>

          <form className="account-settings-form" onSubmit={onSubmit}>
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
              <UserRound size={15} />
              {pending ? t('auth.passwordChanging') : t('auth.passwordChangeSubmit')}
            </button>
          </form>

          <button className="secondary-button account-settings-logout" disabled={logoutPending} onClick={() => void onLogout()} type="button">
            <LogOut size={15} />
            {t('auth.logout')}
          </button>
        </div>
      ) : null}
    </div>
  );
}
