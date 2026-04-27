'use client';

type Props = {
  enabledPaths: number;
  relayPaths: number;
  pendingTasks: number;
  failedTasks: number;
};

export function OnboardingMetrics({ enabledPaths, relayPaths, pendingTasks, failedTasks }: Props) {
  return (
    <section className="metrics-grid">
      <article className="metric-card panel-card">
        <span className="metric-label">Enabled paths</span>
        <strong>{enabledPaths}</strong>
        <span className="metric-foot">Reusable path definitions currently available for dispatch.</span>
      </article>
      <article className="metric-card panel-card soft-card">
        <span className="metric-label">Relay-chain paths</span>
        <strong>{relayPaths}</strong>
        <span className="metric-foot">Multi-hop definitions that need the clearest task visibility.</span>
      </article>
      <article className="metric-card panel-card warm-card">
        <span className="metric-label">Pending tasks</span>
        <strong>{pendingTasks}</strong>
        <span className="metric-foot">Onboarding work still waiting for node-side completion or operator follow-through.</span>
      </article>
      <article className="metric-card panel-card">
        <span className="metric-label">Failed tasks</span>
        <strong>{failedTasks}</strong>
        <span className="metric-foot">Tasks that need path or target remediation before retrying.</span>
      </article>
    </section>
  );
}
