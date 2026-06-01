import {useTranslations} from 'next-intl';

import {PageHero} from '@/components/page-hero';

export function PlaceholderPage({titleKey, descriptionKey}: {titleKey: string; descriptionKey: string}) {
  const t = useTranslations('pages');
  const shell = useTranslations('shell');
  const common = useTranslations('common');
  void descriptionKey;

  return (
    <div className="page-stack">
      <PageHero eyebrow={shell('product')} title={t(titleKey)} />
      <section className="two-column-grid">
        <article className="panel-card soft-card">
          <p className="section-kicker">{shell('name')}</p>
          <h3>{t(titleKey)}</h3>
        </article>
        <article className="panel-card warm-card">
          <p className="section-kicker">{t('healthTitle')}</p>
          <h3>{common('serverDrivenTitle')}</h3>
        </article>
      </section>
    </div>
  );
}
