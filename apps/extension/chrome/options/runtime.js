const messages = new Map();

function text(key, substitutions) {
  if (!chrome.i18n || typeof chrome.i18n.getMessage !== 'function') {
    throw new Error('chrome_i18n_unavailable');
  }
  if (substitutions) {
    const message = chrome.i18n.getMessage(key, substitutions);
    if (!message) {
      throw new Error(`missing_i18n_message:${key}`);
    }
    return message;
  }
  if (!messages.has(key)) {
    const message = chrome.i18n.getMessage(key);
    if (!message) {
      throw new Error(`missing_i18n_message:${key}`);
    }
    messages.set(key, message);
  }
  return messages.get(key);
}

function applyLanguage(root = document) {
  document.documentElement.lang = chrome.i18n.getUILanguage().startsWith('zh') ? 'zh-CN' : 'en';
  root.querySelectorAll('[data-lang]').forEach((element) => {
    element.textContent = text(element.dataset.lang);
  });
  root.querySelectorAll('[data-lang-title]').forEach((element) => {
    element.title = text(element.dataset.langTitle);
  });
}

function normalizeThemeMode(mode) {
  return mode === 'dark' ? 'dark' : 'vivid';
}

function applyThemeMode(mode, root = document) {
  const normalized = normalizeThemeMode(mode);
  const documentElement = root.documentElement || document.documentElement;
  documentElement.dataset.theme = normalized;
  root.querySelectorAll('[data-theme-mode]').forEach((button) => {
    button.classList.toggle('is-active', button.dataset.themeMode === normalized);
  });
  return normalized;
}

function bindThemeMode(container, onChange) {
  if (!container) {
    return;
  }
  container.querySelectorAll('[data-theme-mode]').forEach((button) => {
    button.addEventListener('click', () => {
      const mode = applyThemeMode(button.dataset.themeMode);
      Promise.resolve(onChange(mode)).catch(() => {});
    });
  });
}

const controlPlaneUrl = document.getElementById('controlPlaneUrl');
const account = document.getElementById('account');
const password = document.getElementById('password');
const tenantSelect = document.getElementById('tenantSelect');
const testConnectionButton = document.getElementById('testConnectionButton');
const loginButton = document.getElementById('loginButton');
const logoutButton = document.getElementById('logoutButton');
const syncRemote = document.getElementById('syncRemote');
const sessionMeta = document.getElementById('sessionMeta');
const feedback = document.getElementById('feedback');
const groupList = document.getElementById('groupList');
const groupTitle = document.getElementById('groupTitle');
const policyMeta = document.getElementById('policyMeta');
const enabledMeta = document.getElementById('enabledMeta');
const groupCountMeta = document.getElementById('groupCountMeta');
const overrideCountMeta = document.getElementById('overrideCountMeta');
const syncTimeMeta = document.getElementById('syncTimeMeta');
const entryMeta = document.getElementById('entryMeta');
const remoteRuleSummary = document.getElementById('remoteRuleSummary');
const directHosts = document.getElementById('directHosts');
const proxyHosts = document.getElementById('proxyHosts');
const saveOverrides = document.getElementById('saveOverrides');
const proxyTokenStatus = document.getElementById('proxyTokenStatus');
const localHelperEnabled = document.getElementById('localHelperEnabled');
const localHelperScheme = document.getElementById('localHelperScheme');
const localHelperHost = document.getElementById('localHelperHost');
const localHelperPort = document.getElementById('localHelperPort');
const saveLocalHelper = document.getElementById('saveLocalHelper');
const themeMode = document.getElementById('themeMode');
const diagnosticLogs = document.getElementById('diagnosticLogs');
const refreshLogs = document.getElementById('refreshLogs');
const clearLogs = document.getElementById('clearLogs');

let bundle = null;

function sendMessage(message) {
  return chrome.runtime.sendMessage(message);
}

function setFeedback(kind, message) {
  feedback.className = `feedback is-${kind}`;
  feedback.textContent = message;
}

function formatError(message) {
  if (message === 'invalid_credentials') {
    return text('statusInvalidCredentials');
  }
  if (message === 'missing_control_plane_url') {
    return text('statusMissingControlPlaneUrl');
  }
  if (message === 'connection_failed') {
    return text('statusConnectionFailed');
  }
  if (message === 'tenant_required') {
    return text('statusTenantRequired');
  }
  return message;
}

function linesToArray(value) {
  return value
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean);
}

function arrayToLines(value) {
  return (value || []).join('\n');
}

function formatLogDetails(details) {
  if (!details || Object.keys(details).length === 0) {
    return '';
  }
  return JSON.stringify(details);
}

function renderLogs(logs) {
  diagnosticLogs.innerHTML = '';
  const items = Array.isArray(logs) ? [...logs].reverse() : [];
  if (items.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'log-empty';
    empty.textContent = text('noLogs');
    diagnosticLogs.appendChild(empty);
    return;
  }
  for (const item of items) {
    const row = document.createElement('div');
    row.className = `log-row is-${item.level || 'info'}`;
    const meta = document.createElement('div');
    meta.className = 'log-meta';
    const time = document.createElement('span');
    time.textContent = item.at ? new Date(item.at).toLocaleString() : '-';
    const level = document.createElement('strong');
    level.textContent = item.level || 'info';
    const event = document.createElement('code');
    event.textContent = item.event || '-';
    const details = document.createElement('pre');
    details.textContent = formatLogDetails(item.details);
    meta.append(time, level, event);
    row.append(meta, details);
    diagnosticLogs.appendChild(row);
  }
}

function refreshDiagnosticLogs() {
  return sendMessage({ type: 'get-diagnostic-logs' }).then(renderLogs);
}

function renderGroupList() {
  groupList.innerHTML = '';
  const accessPaths = bundle.accessPaths || bundle.remote.accessPaths || [];
  if (accessPaths.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'empty-state';
    empty.textContent = text('statusNoAccessPaths');
    groupList.appendChild(empty);
    return;
  }
  for (const accessPath of accessPaths) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `group-row${accessPath.effectiveEnabled ? ' is-active' : ' is-disabled'}`;
    button.disabled = !accessPath.enabled;
    const name = document.createElement('div');
    name.className = 'group-name';
    const title = document.createElement('strong');
    title.textContent = accessPath.name;
    const node = document.createElement('span');
    node.textContent = accessPath.enabled ? (accessPath.effectiveEnabled ? text('on') : text('off')) : text('disabled');
    const meta = document.createElement('div');
    meta.className = 'group-meta';
    meta.textContent = `${String(accessPath.protocol || '').toUpperCase()} ${accessPath.listenHost}:${accessPath.listenPort}`;
    name.append(title, node);
    button.append(name, meta);
    button.addEventListener('click', () => {
      sendMessage({ type: 'set-access-path-enabled', accessPathId: accessPath.id, enabled: !accessPath.effectiveEnabled }).then((result) => {
      if (result && result.error) {
        setFeedback('error', result.error);
        return;
      }
      render(result);
      });
    });
    groupList.appendChild(button);
  }
}

function renderGroupDetail() {
  const routes = bundle.remote.routes || [];
  const accessPaths = bundle.accessPaths || bundle.remote.accessPaths || [];
  const enabledPaths = accessPaths.filter((accessPath) => accessPath.effectiveEnabled);
  const chainRoutes = routes.filter((route) => route.actionType === 'chain');
  const directRoutes = routes.filter((route) => route.actionType === 'direct');
  groupTitle.textContent = text('accessPathDetail');
  entryMeta.textContent = `${enabledPaths.length}/${accessPaths.length} ${text('on')}`;
  remoteRuleSummary.textContent = routes.length > 0 ? text('ruleSummary', [String(chainRoutes.length), String(directRoutes.length)]) : text('noRules');
  directHosts.value = arrayToLines(bundle.state.localOverrides.directHosts);
  proxyHosts.value = arrayToLines(bundle.state.localOverrides.proxyHosts);
}

function renderSession() {
  const { state, session, remote } = bundle;
  applyThemeMode(state.themeMode);
  controlPlaneUrl.value = state.controlPlaneUrl || '';
  account.value = session.account || '';
  tenantSelect.innerHTML = '';
  const emptyTenant = document.createElement('option');
  emptyTenant.value = '';
  emptyTenant.textContent = text('tenantSelectPlaceholder');
  tenantSelect.appendChild(emptyTenant);
  for (const membership of session.tenantMemberships || []) {
    const option = document.createElement('option');
    option.value = membership.tenantId;
    option.textContent = membership.tenantName || membership.tenantId;
    tenantSelect.appendChild(option);
  }
  tenantSelect.value = session.activeTenantId || '';
  tenantSelect.disabled = !session.authenticated || (session.tenantMemberships || []).length === 0;
  policyMeta.textContent = remote.policyRevision ? `${text('policyShort')} ${remote.policyRevision}` : text('notSynced');
  enabledMeta.textContent = state.enabled ? text('on') : text('off');
  groupCountMeta.textContent = String((remote.accessPaths || []).length);
  overrideCountMeta.textContent = String((state.localOverrides.directHosts || []).length + (state.localOverrides.proxyHosts || []).length);
  syncTimeMeta.textContent = remote.fetchedAt ? new Date(remote.fetchedAt).toLocaleString() : text('notSynced');
  proxyTokenStatus.textContent = session.proxyTokenAvailable
    ? text('proxyTokenSummary', [session.proxyTokenExpiresAt ? new Date(session.proxyTokenExpiresAt).toLocaleString() : '-'])
    : text('proxyTokenMissing');
  localHelperEnabled.checked = Boolean(state.localHelper && state.localHelper.enabled);
  localHelperScheme.value = state.localHelper && state.localHelper.scheme === 'PROXY' ? 'PROXY' : 'SOCKS5';
  localHelperHost.value = (state.localHelper && state.localHelper.host) || '127.0.0.1';
  localHelperPort.value = String((state.localHelper && state.localHelper.port) || 1080);
  sessionMeta.textContent = session.authenticated
    ? text('sessionSummary', [session.account || '-', session.expiresAt ? new Date(session.expiresAt).toLocaleString() : '-'])
    : text('statusLoginRequired');
  logoutButton.disabled = !session.authenticated;
  syncRemote.disabled = !session.authenticated || !session.activeTenantId;
}

function render(nextBundle) {
  bundle = nextBundle;
  renderSession();
  renderGroupList();
  renderGroupDetail();
  refreshDiagnosticLogs();
}

loginButton.addEventListener('click', () => {
  sendMessage({
    type: 'login',
    controlPlaneUrl: controlPlaneUrl.value.trim(),
    account: account.value.trim(),
    password: password.value
  }).then((result) => {
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  password.value = '';
  render(result);
  setFeedback('ok', result.session.activeTenantId ? text('statusLoggedIn') : text('statusTenantRequired'));
  });
});

tenantSelect.addEventListener('change', () => {
  sendMessage({
    type: 'select-tenant',
    tenantId: tenantSelect.value
  }).then((result) => {
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  render(result);
  setFeedback('ok', text('statusSynced'));
  });
});

testConnectionButton.addEventListener('click', () => {
  sendMessage({
    type: 'test-connection',
    controlPlaneUrl: controlPlaneUrl.value.trim()
  }).then((result) => {
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  setFeedback('ok', text('statusConnectionOk'));
  });
});

logoutButton.addEventListener('click', () => {
  sendMessage({ type: 'logout' }).then((result) => {
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  render(result);
  setFeedback('idle', text('statusLoggedOut'));
  });
});

syncRemote.addEventListener('click', () => {
  sendMessage({ type: 'sync-remote-config' }).then((result) => {
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  render(result);
  setFeedback('ok', text('statusSynced'));
  });
});

saveOverrides.addEventListener('click', () => {
  sendMessage({
    type: 'set-local-overrides',
    directHosts: linesToArray(directHosts.value),
    proxyHosts: linesToArray(proxyHosts.value)
  }).then((result) => {
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  render(result);
  setFeedback('ok', text('statusOverridesSaved'));
  });
});

saveLocalHelper.addEventListener('click', () => {
  sendMessage({
    type: 'set-local-helper',
    enabled: localHelperEnabled.checked,
    scheme: localHelperScheme.value,
    host: localHelperHost.value.trim(),
    port: Number(localHelperPort.value)
  }).then((result) => {
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  render(result);
  setFeedback('ok', text('statusLocalHelperSaved'));
  });
});

refreshLogs.addEventListener('click', () => {
  refreshDiagnosticLogs();
});

clearLogs.addEventListener('click', () => {
  sendMessage({ type: 'clear-diagnostic-logs' }).then(renderLogs);
});

bindThemeMode(themeMode, (mode) => sendMessage({ type: 'set-theme-mode', themeMode: mode }).then((result) => {
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    if (bundle && bundle.state) {
      applyThemeMode(bundle.state.themeMode);
    }
    return;
  }
  render(result);
}));

chrome.runtime.onMessage.addListener((message) => {
  if (message && message.type === 'state-updated') {
    render(message.payload);
  }
});

function init() {
  applyLanguage();
  sendMessage({ type: 'record-diagnostic-event', event: 'options_opened' })
    .then(renderLogs)
    .then(() => sendMessage({ type: 'get-state' }))
    .then((result) => {
      render(result);
      setFeedback('idle', text('statusIdle'));
    });
}

init();
