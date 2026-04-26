'use client';

import {ReactNode, useEffect} from 'react';
import {useLocale} from 'next-intl';

import {useAuth} from '@/components/auth-provider';
import {usePathname, useRouter} from '@/i18n/navigation';

export function AuthGate({children}: {children: ReactNode}) {
  const locale = useLocale();
  const pathname = usePathname();
  const router = useRouter();
  const {ready, session} = useAuth();

  useEffect(() => {
    if (ready && !session && pathname !== '/login') {
      router.replace('/login', {locale});
    }
    if (ready && session?.mustRotatePassword && pathname !== '/login') {
      router.replace('/login', {locale});
    }
  }, [locale, pathname, ready, router, session]);

  if (!ready) {
    return <div className="panel-card">Loading session…</div>;
  }

  if (!session || session.mustRotatePassword) {
    return <div className="panel-card">Redirecting to login…</div>;
  }

  return <>{children}</>;
}
