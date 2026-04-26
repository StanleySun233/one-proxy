'use client';

import {useEffect} from 'react';
import {useLocale} from 'next-intl';

import {useAuth} from '@/components/auth-provider';
import {LoginForm} from '@/components/login-form';
import {PasswordRotationForm} from '@/components/password-rotation-form';
import {useRouter} from '@/i18n/navigation';

export default function LoginPage() {
  const locale = useLocale();
  const router = useRouter();
  const {ready, session} = useAuth();

  useEffect(() => {
    if (ready && session && !session.mustRotatePassword) {
      router.replace('/', {locale});
    }
  }, [locale, ready, router, session]);

  return (
    <main className="login-screen">
      {session?.mustRotatePassword ? <PasswordRotationForm /> : <LoginForm />}
    </main>
  );
}
