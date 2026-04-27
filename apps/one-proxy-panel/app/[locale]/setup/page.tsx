'use client';

import {useEffect, useState} from 'react';
import {useLocale, useTranslations} from 'next-intl';

import {SetupForm} from '@/components/setup-form';
import {getSetupStatus} from '@/lib/api';
import {useRouter} from '@/i18n/navigation';

type PageStatus = 'loading' | 'unconfigured' | 'configured';

export default function SetupPage() {
  const locale = useLocale();
  const router = useRouter();
  const t = useTranslations();
  const [status, setStatus] = useState<PageStatus>('loading');

  useEffect(() => {
    let cancelled = false;

    async function check() {
      for (let attempt = 0; attempt < 2; attempt++) {
        try {
          const result = await getSetupStatus();
          if (cancelled) return;
          if (result.configured) {
            setStatus('configured');
            router.replace('/login', {locale});
          } else {
            setStatus('unconfigured');
          }
          return;
        } catch {
          if (attempt === 0) {
            await new Promise((resolve) => setTimeout(resolve, 2000));
          }
        }
      }
      if (!cancelled) {
        setStatus('unconfigured');
      }
    }

    check();
    return () => { cancelled = true; };
  }, [locale, router]);

  return (
    <main className="login-screen">
      {status === 'loading' ? (
        <div className="panel-card" style={{textAlign: 'center', padding: '40px'}}>
          <p>{t('setup.checkingConfig')}</p>
        </div>
      ) : null}
      {status === 'unconfigured' ? <SetupForm /> : null}
    </main>
  );
}
