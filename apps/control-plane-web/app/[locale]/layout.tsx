import type {Metadata} from 'next';
import {IBM_Plex_Mono, IBM_Plex_Sans} from 'next/font/google';
import {NextIntlClientProvider} from 'next-intl';
import {getMessages, setRequestLocale} from 'next-intl/server';
import {notFound} from 'next/navigation';
import {ReactNode} from 'react';

import {Providers} from '@/components/providers';
import {routing} from '@/i18n/routing';

const sans = IBM_Plex_Sans({
  subsets: ['latin'],
  variable: '--font-sans',
  weight: ['400', '500', '600', '700']
});

const mono = IBM_Plex_Mono({
  subsets: ['latin'],
  variable: '--font-mono',
  weight: ['400', '500', '600']
});

export async function generateMetadata({params: {locale}}: {params: {locale: string}}): Promise<Metadata> {
  const activeLocale = routing.locales.includes(locale as 'zh' | 'en') ? locale : routing.defaultLocale;
  const messages = (await import(`../../messages/${activeLocale}.json`)).default;

  return {
    title: messages.meta.title,
    description: messages.meta.description
  };
}

export function generateStaticParams() {
  return routing.locales.map((locale) => ({locale}));
}

export default async function LocaleLayout({
  children,
  params: {locale}
}: {
  children: ReactNode;
  params: {locale: string};
}) {
  if (!routing.locales.includes(locale as 'zh' | 'en')) {
    notFound();
  }

  setRequestLocale(locale);
  const messages = await getMessages();

  return (
    <NextIntlClientProvider locale={locale} messages={messages}>
      <Providers>
        <div className={`${sans.variable} ${mono.variable}`}>{children}</div>
      </Providers>
    </NextIntlClientProvider>
  );
}
