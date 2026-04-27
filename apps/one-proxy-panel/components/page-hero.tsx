import {ReactNode} from 'react';

export function PageHero({
  eyebrow,
  title,
  description,
  aside
}: {
  eyebrow: string;
  title: string;
  description: string;
  aside?: ReactNode;
}) {
  return (
    <section className="hero-panel">
      <div className="hero-copy">
        <p className="section-kicker">{eyebrow}</p>
        <h2>{title}</h2>
        <p className="section-copy">{description}</p>
      </div>
      {aside ? <div className="hero-aside">{aside}</div> : null}
    </section>
  );
}
