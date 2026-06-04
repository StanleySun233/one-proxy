const routes = [
  { path: "/overview", label: "Overview" },
  { path: "/health", label: "Health" },
  { path: "/audit", label: "Audit" },
  { path: "/policy", label: "Policy" },
  { path: "/diagnostics", label: "Diagnostics" }
];

const statusTone = {
  healthy: "good",
  degraded: "warn",
  unhealthy: "bad",
  info: "info",
  warning: "warn",
  error: "bad"
};

const app = document.querySelector("#app");
let session = { authenticated: false, user: null };

async function api(path, options = {}) {
  const response = await fetch(path, {
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    ...options
  });
  const body = await response.json();
  if (!response.ok || !body.ok) {
    const message = body.error?.message || `Request failed with ${response.status}`;
    throw new Error(message);
  }
  return body.data;
}

function navigate(path) {
  history.pushState(null, "", path);
  render();
}

function escapeHtml(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "\"": "&quot;",
    "'": "&#39;"
  }[char]));
}

function formatDate(value) {
  if (!value) return "Never";
  return new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(new Date(value));
}

function duration(seconds) {
  if (!Number.isFinite(seconds)) return "Unknown";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return [days && `${days}d`, hours && `${hours}h`, `${minutes}m`].filter(Boolean).join(" ");
}

function shell(activePath, body) {
  const user = session.user || {};
  app.innerHTML = `
    <div class="shell">
      <aside class="rail">
        <div class="brand">
          <div class="mark">OP</div>
          <div>
            <strong>OneProxy Node</strong>
            <span>Local console</span>
          </div>
        </div>
        <nav class="nav">
          ${routes.map((route) => `
            <button class="nav-item ${route.path === activePath ? "active" : ""}" data-route="${route.path}">
              ${escapeHtml(route.label)}
            </button>
          `).join("")}
        </nav>
      </aside>
      <main class="workspace">
        <header class="topbar">
          <div>
            <p class="eyebrow">Read-only node operations</p>
            <h1>${escapeHtml(routes.find((route) => route.path === activePath)?.label || "Overview")}</h1>
          </div>
          <div class="account">
            <span>${escapeHtml(user.name || user.email || "Operator")}</span>
            <button class="ghost" data-action="logout">Log out</button>
          </div>
        </header>
        <section class="content">${body}</section>
      </main>
    </div>
  `;
  bindShell();
}

function bindShell() {
  app.querySelectorAll("[data-route]").forEach((button) => {
    button.addEventListener("click", () => navigate(button.dataset.route));
  });
  app.querySelector("[data-action='logout']")?.addEventListener("click", async () => {
    await api("/api/local/logout", { method: "POST" });
    session = { authenticated: false, user: null };
    navigate("/login");
  });
}

function stateBlock(title, detail) {
  return `
    <div class="state-block">
      <strong>${escapeHtml(title)}</strong>
      <span>${escapeHtml(detail)}</span>
    </div>
  `;
}

function metric(label, value, detail = "") {
  return `
    <article class="metric">
      <span>${escapeHtml(label)}</span>
      <strong>${escapeHtml(value)}</strong>
      ${detail ? `<small>${escapeHtml(detail)}</small>` : ""}
    </article>
  `;
}

function kv(label, value) {
  return `
    <div class="kv">
      <span>${escapeHtml(label)}</span>
      <strong>${escapeHtml(value)}</strong>
    </div>
  `;
}

async function render() {
  const path = location.pathname === "/" ? "/overview" : location.pathname;
  if (!session.authenticated) {
    try {
      const data = await api("/api/local/session");
      session = data;
    } catch {
      session = { authenticated: false, user: null };
    }
  }

  if (!session.authenticated || path === "/login") {
    renderLogin();
    return;
  }

  if (!routes.some((route) => route.path === path)) {
    navigate("/overview");
    return;
  }

  const renderers = {
    "/overview": renderOverview,
    "/health": renderHealth,
    "/audit": renderAudit,
    "/policy": renderPolicy,
    "/diagnostics": renderDiagnostics
  };
  shell(path, stateBlock("Loading", "Fetching current node data."));
  try {
    shell(path, await renderers[path]());
  } catch (error) {
    shell(path, stateBlock("Unavailable", error.message));
  }
}

function renderLogin() {
  app.innerHTML = `
    <main class="login-screen">
      <form class="login-panel" id="login-form">
        <div class="brand login-brand">
          <div class="mark">OP</div>
          <div>
            <strong>OneProxy Node</strong>
            <span>Local console</span>
          </div>
        </div>
        <label>
          <span>Email or username</span>
          <input name="username" type="text" autocomplete="username" required />
        </label>
        <label>
          <span>Password</span>
          <input name="password" type="password" autocomplete="current-password" required />
        </label>
        <button class="primary" type="submit">Log in</button>
        <p class="form-error" id="login-error"></p>
      </form>
    </main>
  `;
  app.querySelector("#login-form").addEventListener("submit", submitLogin);
}

async function submitLogin(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const error = app.querySelector("#login-error");
  const button = form.querySelector("button");
  const payload = Object.fromEntries(new FormData(form));
  error.textContent = "";
  button.disabled = true;
  button.textContent = "Logging in";
  try {
    const data = await api("/api/local/login", {
      method: "POST",
      body: JSON.stringify(payload)
    });
    session = { authenticated: true, ...data };
    navigate("/overview");
  } catch (caught) {
    error.textContent = caught.message;
  } finally {
    button.disabled = false;
    button.textContent = "Log in";
  }
}

async function renderOverview() {
  const status = await api("/api/local/status");
  return `
    <div class="grid metrics">
      ${metric("Node", status.node.name, status.node.id)}
      ${metric("Version", status.node.version, status.node.role)}
      ${metric("Uptime", duration(status.node.uptimeSeconds), `Started ${formatDate(status.node.startedAt)}`)}
      ${metric("Policy", status.runtime.policyRevision, `${status.runtime.activeProxySessions} active sessions`)}
    </div>
    <div class="grid two">
      <article class="panel">
        <h2>Control plane</h2>
        ${kv("URL", status.controlPlane.url)}
        ${kv("Binding", status.controlPlane.bound ? "Bound" : "Not bound")}
        ${kv("Last sync", formatDate(status.controlPlane.lastSyncAt))}
      </article>
      <article class="panel">
        <h2>Listeners</h2>
        ${Object.entries(status.listeners).map(([name, value]) => kv(name.toUpperCase(), value || "Disabled")).join("")}
      </article>
    </div>
  `;
}

async function renderHealth() {
  const health = await api("/api/local/health");
  return `
    <article class="panel hero-status">
      <span class="pill ${statusTone[health.status] || "info"}">${escapeHtml(health.status)}</span>
      <div>
        <h2>Checked ${escapeHtml(formatDate(health.checkedAt))}</h2>
        <p>${health.checks.length} runtime checks reported by this node.</p>
      </div>
    </article>
    <div class="list">
      ${health.checks.map((check) => `
        <article class="row">
          <span class="pill ${statusTone[check.status] || "info"}">${escapeHtml(check.status)}</span>
          <div>
            <strong>${escapeHtml(check.name)}</strong>
            <span>${escapeHtml(check.message)} · ${escapeHtml(formatDate(check.lastCheckedAt))}</span>
          </div>
        </article>
      `).join("")}
    </div>
  `;
}

async function renderAudit() {
  const audit = await api("/api/local/audit?limit=50");
  return `
    <div class="table">
      <div class="thead">
        <span>Time</span><span>Type</span><span>Severity</span><span>Message</span>
      </div>
      ${audit.items.map((item) => `
        <div class="tr">
          <span>${escapeHtml(formatDate(item.timestamp))}</span>
          <span>${escapeHtml(item.type)}</span>
          <span><b class="pill ${statusTone[item.severity] || "info"}">${escapeHtml(item.severity)}</b></span>
          <span>${escapeHtml(item.message)}</span>
        </div>
      `).join("")}
    </div>
    ${audit.nextCursor ? `<p class="muted">Next cursor: ${escapeHtml(audit.nextCursor)}</p>` : ""}
  `;
}

async function renderPolicy() {
  const policy = await api("/api/local/policy");
  return `
    <div class="grid metrics">
      ${metric("Revision", policy.revision, policy.source)}
      ${metric("Loaded", formatDate(policy.loadedAt))}
      ${metric("Nodes", policy.nodes.length)}
      ${metric("Routes", policy.routes.length)}
    </div>
    <div class="grid two">
      <article class="panel">
        <h2>Nodes</h2>
        ${policy.nodes.map((node) => kv(node.name, `${node.role} · ${node.id}`)).join("")}
      </article>
      <article class="panel">
        <h2>Routes</h2>
        ${policy.routes.map((route) => kv(route.name, `${route.enabled ? "Enabled" : "Disabled"} · ${route.action} · ${route.match.protocol}://${route.match.host}`)).join("")}
      </article>
    </div>
  `;
}

async function renderDiagnostics() {
  const diagnostics = await api("/api/local/diagnostics");
  return `
    <div class="grid two">
      <article class="panel">
        <h2>Environment</h2>
        ${kv("Generated", formatDate(diagnostics.generatedAt))}
        ${Object.entries(diagnostics.environment).map(([key, value]) => kv(key, value)).join("")}
      </article>
      <article class="panel">
        <h2>Network</h2>
        ${kv("Local addresses", diagnostics.network.localAddresses.join(", ") || "None")}
        ${kv("NAT type", diagnostics.network.natType)}
        ${kv("Control plane", diagnostics.controlPlane.reachable ? "Reachable" : "Unreachable")}
        ${diagnostics.controlPlane.lastError ? kv("Last error", diagnostics.controlPlane.lastError) : ""}
      </article>
    </div>
    <article class="panel">
      <h2>Recent errors</h2>
      <div class="list compact">
        ${diagnostics.recentErrors.length ? diagnostics.recentErrors.map((item) => `
          <div class="row">
            <span>${escapeHtml(formatDate(item.timestamp))}</span>
            <div>
              <strong>${escapeHtml(item.component)}</strong>
              <span>${escapeHtml(item.message)}</span>
            </div>
          </div>
        `).join("") : "<p class=\"muted\">No recent errors.</p>"}
      </div>
    </article>
  `;
}

window.addEventListener("popstate", render);
render();
