const nodeCards = [
  { name: 'edge-a', scope: 'public-edge', status: 'Healthy', hops: 'a -> b -> c' },
  { name: 'relay-b', scope: 'b-lan', status: 'Healthy', hops: 'b -> c' },
  { name: 'relay-c', scope: 'c-k8s', status: 'Degraded', hops: 'c' },
  { name: 'relay-d', scope: 'd-office', status: 'Healthy', hops: 'd' }
];

const routeRules = [
  { match: '*.corp.internal', action: 'chain edge-a-b-c', scope: 'c-k8s' },
  { match: '10.30.0.0/16', action: 'chain edge-a-b', scope: 'b-lan' },
  { match: 'grafana.office.local', action: 'chain edge-a-d', scope: 'd-office' }
];

const certRows = [
  { owner: 'edge-a', type: 'public', expires: '18 days', state: 'Renew soon' },
  { owner: 'relay-b', type: 'internal', expires: '41 days', state: 'Healthy' },
  { owner: 'relay-c', type: 'internal', expires: '9 days', state: 'Rotate' }
];

export default function Page() {
  return (
    <main className="page-shell">
      <aside className="side-rail">
        <div className="brand-card">
          <p className="eyebrow">One Proxy</p>
          <h1>Control Plane</h1>
          <p className="muted">Warm operational surfaces, same palette family as the browser client.</p>
        </div>
        <nav className="nav-card">
          <a className="nav-link is-active" href="#overview">Overview</a>
          <a className="nav-link" href="#nodes">Nodes</a>
          <a className="nav-link" href="#chains">Chains</a>
          <a className="nav-link" href="#rules">Route Rules</a>
          <a className="nav-link" href="#certs">Certificates</a>
          <a className="nav-link" href="#accounts">Accounts</a>
        </nav>
      </aside>

      <section className="content-stack">
        <header className="hero-card" id="overview">
          <div>
            <p className="eyebrow">Multi-node orchestration</p>
            <h2>Single entry, explicit chain ownership, zero ambiguous LAN routing.</h2>
          </div>
          <div className="hero-stats">
            <div className="metric-card">
              <span className="metric-label">Healthy Nodes</span>
              <strong>3</strong>
            </div>
            <div className="metric-card">
              <span className="metric-label">Degraded Nodes</span>
              <strong>1</strong>
            </div>
            <div className="metric-card">
              <span className="metric-label">Policy Revision</span>
              <strong>rev-0007</strong>
            </div>
          </div>
        </header>

        <section className="grid grid-wide" id="nodes">
          <div className="panel-card">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Nodes</p>
                <h3>Reachability by scope</h3>
              </div>
              <span className="pill">4 nodes</span>
            </div>
            <div className="node-grid">
              {nodeCards.map((node) => (
                <article className="node-card" key={node.name}>
                  <div className="node-line">
                    <strong>{node.name}</strong>
                    <span className={`state-pill ${node.status === 'Healthy' ? 'is-good' : 'is-warn'}`}>{node.status}</span>
                  </div>
                  <p className="mono">{node.scope}</p>
                  <p className="muted">{node.hops}</p>
                </article>
              ))}
            </div>
          </div>

          <div className="panel-card" id="chains">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Chains</p>
                <h3>Compiled path ownership</h3>
              </div>
            </div>
            <div className="chain-stack">
              <div className="chain-row">
                <span className="chain-name">corp-k8s</span>
                <span className="mono">edge-a -> relay-b -> relay-c</span>
                <span className="pill">scope c-k8s</span>
              </div>
              <div className="chain-row">
                <span className="chain-name">office-tools</span>
                <span className="mono">edge-a -> relay-d</span>
                <span className="pill">scope d-office</span>
              </div>
            </div>
          </div>
        </section>

        <section className="grid" id="rules">
          <div className="panel-card">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Route rules</p>
                <h3>Whitelist-driven forwarding</h3>
              </div>
            </div>
            <div className="rule-stack">
              {routeRules.map((rule) => (
                <div className="rule-row" key={rule.match}>
                  <div>
                    <strong>{rule.match}</strong>
                    <p className="muted">{rule.action}</p>
                  </div>
                  <span className="pill">{rule.scope}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="panel-card" id="certs">
            <div className="panel-head">
              <div>
                <p className="eyebrow">Certificates</p>
                <h3>Renewal-only operational status</h3>
              </div>
            </div>
            <div className="cert-stack">
              {certRows.map((cert) => (
                <div className="cert-row" key={`${cert.owner}-${cert.type}`}>
                  <div>
                    <strong>{cert.owner}</strong>
                    <p className="muted">{cert.type}</p>
                  </div>
                  <div className="cert-side">
                    <span className="mono">{cert.expires}</span>
                    <span className={`state-pill ${cert.state === 'Healthy' ? 'is-good' : 'is-warn'}`}>{cert.state}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>
      </section>
    </main>
  );
}
