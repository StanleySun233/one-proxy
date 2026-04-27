import { applyLanguage, text } from '../shared/locale.js';
import { applyThemeMode, bindThemeMode } from '../shared/theme.js';

const controlPlaneUrl = document.getElementById('controlPlaneUrl');
const account = document.getElementById('account');
const password = document.getElementById('password');
const testConnectionButton = document.getElementById('testConnectionButton');
const loginButton = document.getElementById('loginButton');
const logoutButton = document.getElementById('logoutButton');
const syncRemote = document.getElementById('syncRemote');
const sessionMeta = document.getElementById('sessionMeta');
const feedback = document.getElementById('feedback');
const groupList = document.getElementById('groupList');
const groupTitle = document.getElementById('groupTitle');
const policyMeta = document.getElementById('policyMeta');
const entryMeta = document.getElementById('entryMeta');
const remoteRuleSummary = document.getElementById('remoteRuleSummary');
const directHosts = document.getElementById('directHosts');
const proxyHosts = document.getElementById('proxyHosts');
const saveOverrides = document.getElementById('saveOverrides');
const themeMode = document.getElementById('themeMode');

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

function renderGroupList() {
  groupList.innerHTML = '';
  const groups = bundle.remote.groups || [];
  for (const group of groups) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `group-row${bundle.activeGroup && bundle.activeGroup.id === group.id ? ' is-active' : ''}`;
    button.innerHTML = `
      <div class="group-name">
        <strong>${group.name}</strong>
        <span>${group.entryNodeName}</span>
      </div>
      <div class="group-meta">${text('ruleSummary', [String((group.proxyHosts || []).length + (group.proxyCidrs || []).length), String((group.directHosts || []).length + (group.directCidrs || []).length)])}</div>
    `;
    button.addEventListener('click', async () => {
      const result = await sendMessage({ type: 'select-group', groupId: group.id });
      if (result && result.error) {
        setFeedback('error', result.error);
        return;
      }
      render(result);
    });
    groupList.appendChild(button);
  }
}

function renderGroupDetail() {
  const group = bundle.activeGroup;
  groupTitle.textContent = group ? group.name : text('groupDetail');
  entryMeta.textContent = group ? `${group.proxyScheme} ${group.proxyHost}:${group.proxyPort}` : '-';
  remoteRuleSummary.textContent = group ? text('ruleSummaryLong', [String((group.proxyHosts || []).length), String((group.proxyCidrs || []).length), String((group.directHosts || []).length), String((group.directCidrs || []).length)]) : text('noRules');
  directHosts.value = arrayToLines(bundle.state.localOverrides.directHosts);
  proxyHosts.value = arrayToLines(bundle.state.localOverrides.proxyHosts);
}

function renderSession() {
  const { state, session, remote } = bundle;
  applyThemeMode(state.themeMode);
  controlPlaneUrl.value = state.controlPlaneUrl || '';
  account.value = session.account || '';
  policyMeta.textContent = remote.policyRevision ? `${text('policyShort')} ${remote.policyRevision}` : text('notSynced');
  sessionMeta.textContent = session.accessToken
    ? text('sessionSummary', [session.account || '-', session.expiresAt ? new Date(session.expiresAt).toLocaleString() : '-'])
    : text('statusLoginRequired');
  logoutButton.disabled = !session.accessToken;
  syncRemote.disabled = !session.accessToken;
}

function render(nextBundle) {
  bundle = nextBundle;
  renderSession();
  renderGroupList();
  renderGroupDetail();
}

loginButton.addEventListener('click', async () => {
  const result = await sendMessage({
    type: 'login',
    controlPlaneUrl: controlPlaneUrl.value.trim(),
    account: account.value.trim(),
    password: password.value
  });
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  password.value = '';
  render(result);
  setFeedback('ok', text('statusLoggedIn'));
});

testConnectionButton.addEventListener('click', async () => {
  const result = await sendMessage({
    type: 'test-connection',
    controlPlaneUrl: controlPlaneUrl.value.trim()
  });
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  setFeedback('ok', text('statusConnectionOk'));
});

logoutButton.addEventListener('click', async () => {
  const result = await sendMessage({ type: 'logout' });
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  render(result);
  setFeedback('idle', text('statusLoggedOut'));
});

syncRemote.addEventListener('click', async () => {
  const result = await sendMessage({ type: 'sync-remote-config' });
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  render(result);
  setFeedback('ok', text('statusSynced'));
});

saveOverrides.addEventListener('click', async () => {
  const result = await sendMessage({
    type: 'set-local-overrides',
    directHosts: linesToArray(directHosts.value),
    proxyHosts: linesToArray(proxyHosts.value)
  });
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  render(result);
  setFeedback('ok', text('statusOverridesSaved'));
});

bindThemeMode(themeMode, async (mode) => {
  const result = await sendMessage({ type: 'set-theme-mode', themeMode: mode });
  if (result && result.error) {
    setFeedback('error', formatError(result.error));
    return;
  }
  render(result);
});

chrome.runtime.onMessage.addListener((message) => {
  if (message && message.type === 'state-updated') {
    render(message.payload);
  }
});

async function init() {
  applyLanguage();
  render(await sendMessage({ type: 'get-state' }));
  setFeedback('idle', text('statusIdle'));
}

init();
