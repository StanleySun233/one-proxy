'use client';

import {useState, useEffect, useRef} from 'react';
import {useTranslations} from 'next-intl';
import {useLocale} from 'next-intl';
import {useRouter} from '@/i18n/navigation';
import {toast} from 'sonner';
import {testSetupConnection, generateSetupKey, submitSetupInit, getSetupStatus} from '@/lib/control-plane-api';
import {formatControlPlaneError} from '@/lib/presentation';

export function SetupForm() {
  const t = useTranslations();
  const locale = useLocale();
  const router = useRouter();
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [host, setHost] = useState('127.0.0.1');
  const [port, setPort] = useState(3306);
  const [user, setUser] = useState('root');
  const [password, setPassword] = useState('');
  const [database, setDatabase] = useState('one_proxy');
  const [needInitialize, setNeedInitialize] = useState(true);
  const [jwtKey, setJwtKey] = useState('');
  const [testPending, setTestPending] = useState(false);
  const [keyPending, setKeyPending] = useState(false);
  const [initPending, setInitPending] = useState(false);
  const [phase, setPhase] = useState<'form' | 'transitioning'>('form');

  useEffect(() => {
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  async function handleTestConnection() {
    setTestPending(true);
    try {
      const result = await testSetupConnection({host, port, user, password, database});
      if (result.success) {
        toast.success(t('setup.connectionSuccess'));
      } else {
        toast.error(result.message || t('setup.connectionFailed'));
      }
    } catch (err) {
      toast.error(formatControlPlaneError(err));
    } finally {
      setTestPending(false);
    }
  }

  async function handleGenerateKey() {
    setKeyPending(true);
    try {
      const result = await generateSetupKey();
      setJwtKey(result.key);
    } catch (err) {
      toast.error(formatControlPlaneError(err));
    } finally {
      setKeyPending(false);
    }
  }

  async function handleInit() {
    setInitPending(true);
    try {
      await submitSetupInit({
        host,
        port,
        user,
        password,
        database,
        jwtSigningKey: jwtKey,
        needInitialize,
      });

      setPhase('transitioning');

      let retries = 0;
      const maxRetries = 30;
      intervalRef.current = setInterval(async () => {
        retries++;
        try {
          const status = await getSetupStatus();
          if (status.configured) {
            if (intervalRef.current) clearInterval(intervalRef.current);
            router.replace('/login', {locale});
          } else if (retries >= maxRetries) {
            if (intervalRef.current) clearInterval(intervalRef.current);
            toast.error(t('setup.initFailed'));
            setPhase('form');
            setInitPending(false);
          }
        } catch {
          if (retries >= maxRetries) {
            if (intervalRef.current) clearInterval(intervalRef.current);
            toast.error(t('setup.initFailed'));
            setPhase('form');
            setInitPending(false);
          }
        }
      }, 1000);
    } catch (err) {
      toast.error(formatControlPlaneError(err));
      setInitPending(false);
    }
  }

  if (phase === 'transitioning') {
    return (
      <main className="login-screen">
        <div className="panel-card" style={{textAlign: 'center', padding: '40px'}}>
          <p>{t('setup.initSuccess')}</p>
        </div>
      </main>
    );
  }

  return (
    <form className="login-form" onSubmit={(e) => { e.preventDefault(); handleInit(); }}>
      <div className="panel-card">
        <p className="section-kicker">{t('shell.product')}</p>
        <h3>{t('setup.title')}</h3>
        <p className="section-copy">{t('setup.description')}</p>

        <label className="field-stack">
          <span>{t('setup.mysqlHost')}</span>
          <input className="field-input" value={host} onChange={(e) => setHost(e.target.value)} />
        </label>

        <label className="field-stack">
          <span>{t('setup.mysqlPort')}</span>
          <input className="field-input" type="number" value={port} onChange={(e) => setPort(Number(e.target.value))} />
        </label>

        <label className="field-stack">
          <span>{t('setup.mysqlUser')}</span>
          <input className="field-input" value={user} onChange={(e) => setUser(e.target.value)} />
        </label>

        <label className="field-stack">
          <span>{t('setup.mysqlPassword')}</span>
          <input className="field-input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        </label>

        <label className="field-stack">
          <span>{t('setup.mysqlDatabase')}</span>
          <input className="field-input" value={database} onChange={(e) => setDatabase(e.target.value)} />
        </label>

        <span>{t('setup.needInitialize')}</span>
        <div className="capsule-toggle-group">
          <button
            type="button"
            className={`capsule-toggle-option${needInitialize ? ' is-active' : ''}`}
            onClick={() => setNeedInitialize(true)}
          >
            {t('setup.needInitialize')}
          </button>
          <button
            type="button"
            className={`capsule-toggle-option${!needInitialize ? ' is-active' : ''}`}
            onClick={() => setNeedInitialize(false)}
          >
            {t('setup.skipInitialize')}
          </button>
        </div>

        <div style={{display: 'flex', gap: '8px'}}>
          <button type="button" className="secondary-button" disabled={testPending} onClick={handleTestConnection}>
            {testPending ? t('setup.testing') : t('setup.testConnection')}
          </button>
          <button type="button" className="secondary-button" disabled={keyPending} onClick={handleGenerateKey}>
            {keyPending ? t('setup.generating') : t('setup.generateJwt')}
          </button>
        </div>

        <label className="field-stack">
          <span>{t('setup.jwtKey')}</span>
          <input className="field-input" readOnly={!!jwtKey} value={jwtKey} onChange={(e) => setJwtKey(e.target.value)} />
        </label>

        <button className="primary-button" disabled={initPending} type="submit">
          {initPending ? t('setup.initializing') : t('setup.initSubmit')}
        </button>
      </div>
    </form>
  );
}
