'use client';

type RegistrySummaryCardsProps = {
  summary: {
    healthy: number;
    degraded: number;
    stale: number;
    unreported: number;
  };
  nodesT: (key: string) => string;
};

export function RegistrySummaryCards({summary, nodesT}: RegistrySummaryCardsProps) {
  return (
    <section className="metrics-grid">
      <article className="metric-card panel-card">
        <span className="metric-label">{nodesT('healthyNodes')}</span>
        <strong>{summary.healthy}</strong>
      </article>
      <article className="metric-card panel-card soft-card">
        <span className="metric-label">{nodesT('degradedNodes')}</span>
        <strong>{summary.degraded}</strong>
      </article>
      <article className="metric-card panel-card warm-card">
        <span className="metric-label">{nodesT('staleNodes')}</span>
        <strong>{summary.stale}</strong>
      </article>
      <article className="metric-card panel-card">
        <span className="metric-label">{nodesT('unreportedNodes')}</span>
        <strong>{summary.unreported}</strong>
      </article>
    </section>
  );
}
