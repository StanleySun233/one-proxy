import {getRequestConfig} from 'next-intl/server';

import {routing} from '@/i18n/routing';

export default getRequestConfig(async ({requestLocale}) => {
  const locale = await requestLocale;
  const activeLocale = routing.locales.includes(locale as 'zh' | 'en') ? locale : routing.defaultLocale;

  return {
    locale: activeLocale,
    messages: (await import(`../messages/${activeLocale}.json`)).default
  };
});
