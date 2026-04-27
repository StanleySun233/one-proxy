'use client';

import {ReactNode, useEffect, useState} from 'react';
import {useTranslations} from 'next-intl';
import {getSetupStatus} from '@/lib/control-plane-api';
import {usePathname, useRouter} from '@/i18n/navigation';
import {useLocale} from 'next-intl';

export function SetupGuard({children}: {children: ReactNode}) {
  const t = useTranslations();
  const locale = useLocale();
  const pathname = usePathname();
  const router = useRouter();
  const [status, setStatus] = useState<'checking' | 'unconfigured' | 'configured'>('checking');

  useEffect(() => {
    let cancelled = false;

    async function check() {
      for (let attempt = 0; attempt < 30; attempt++) {
        try {
          const result = await getSetupStatus();
          if (cancelled) return;
          if (result.configured) {
            setStatus('configured');
          } else {
            setStatus('unconfigured');
          }
          return;
        } catch {
          // Backend not reachable yet, wait and retry
          if (attempt < 29) {
            await new Promise((resolve) => setTimeout(resolve, 1000));
          }
        }
      }
      // All retries exhausted, assume unconfigured
      if (!cancelled) {
        setStatus('unconfigured');
      }
    }

    check();
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    if (status === 'unconfigured' && pathname !== '/setup') {
      router.replace('/setup', {locale});
    }
  }, [status, pathname, locale, router]);

  if (status === 'checking') {
    return (
      <main className="login-screen">
        <div className="panel-card" style={{textAlign: 'center', padding: '40px'}}>
          <p>{t('setup.checkingConfig')}</p>
        </div>
      </main>
    );
  }

  return <>{children}</>;
}
