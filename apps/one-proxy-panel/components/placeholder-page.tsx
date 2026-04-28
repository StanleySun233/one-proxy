import {useTranslations} from 'next-intl';

import {PageHero} from '@/components/page-hero';

export function PlaceholderPage({titleKey, descriptionKey}: {titleKey: string; descriptionKey: string}) {
  const t = useTranslations('pages');
  const shell = useTranslations('shell');
  const common = useTranslations('common');

  return (
    <div className="page-stack">
      <PageHero eyebrow={shell('product')} title={t(titleKey)} description={t(descriptionKey)} />
      <section className="two-column-grid">
        <article className="panel-card soft-card">
          <p className="section-kicker">{shell('name')}</p>
          <h3>{t(titleKey)}</h3>
          <p className="section-copy">{t(descriptionKey)}</p>
        </article>
        <article className="panel-card warm-card">
          <p className="section-kicker">{t('healthTitle')}</p>
          <h3>{common('serverDrivenTitle')}</h3>
          <p className="section-copy">{common('serverDrivenDesc')}</p>
        </article>
      </section>
    </div>
  );
}
