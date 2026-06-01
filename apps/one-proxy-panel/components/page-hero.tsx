import {ReactNode} from 'react';

export function PageHero({
  eyebrow,
  title,
  aside
}: {
  eyebrow: string;
  title: string;
  aside?: ReactNode;
}) {
  return (
    <section className="hero-panel">
      <div className="hero-copy">
        <p className="section-kicker">{eyebrow}</p>
        <h2>{title}</h2>
      </div>
      {aside ? <div className="hero-aside">{aside}</div> : null}
    </section>
  );
}
