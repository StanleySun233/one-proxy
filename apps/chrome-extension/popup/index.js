import { applyLanguage, text } from '../shared/locale.js';
import { applyThemeMode, bindThemeMode } from '../shared/theme.js';

const masterToggle = document.getElementById('masterToggle');
const masterValue = document.getElementById('masterValue');
const accountName = document.getElementById('accountName');
const syncMeta = document.getElementById('syncMeta');
const statusMessage = document.getElementById('statusMessage');
const groupSelect = document.getElementById('groupSelect');
const currentSite = document.getElementById('currentSite');
const entryNodeName = document.getElementById('entryNodeName');
const entryNodeAddress = document.getElementById('entryNodeAddress');
const policyRevision = document.getElementById('policyRevision');
const ruleCounts = document.getElementById('ruleCounts');
const addDirect = document.getElementById('addDirect');
const addProxy = document.getElementById('addProxy');
const openOptions = document.getElementById('openOptions');
const syncButton = document.getElementById('syncButton');
const themeMode = document.getElementById('themeMode');

let viewState = null;

function sendMessage(message) {
  return chrome.runtime.sendMessage(message);
}

function setToggleState(element, enabled) {
  element.classList.toggle('is-on', enabled);
  element.classList.toggle('is-off', !enabled);
}

function setStatus(kind, message) {
  statusMessage.className = `status-strip is-${kind}`;
  statusMessage.textContent = message;
}

function countLabel(group) {
  if (!group) {
    return text('noRules');
  }
  return text('ruleSummary', [String((group.proxyHosts || []).length + (group.proxyCidrs || []).length), String((group.directHosts || []).length + (group.directCidrs || []).length)]);
}

function render(bundle) {
  viewState = bundle;
  const { state, session, remote, activeGroup, currentTab } = bundle;
  applyThemeMode(state.themeMode);
  accountName.textContent = session.account || text('notSignedIn');
  syncMeta.textContent = remote.fetchedAt ? `${text('syncedAt')} ${new Date(remote.fetchedAt).toLocaleTimeString()}` : text('notSynced');
  currentSite.textContent = (currentTab && currentTab.host) || text('noSite');
  masterValue.textContent = state.enabled ? text('on') : text('off');
  setToggleState(masterToggle, state.enabled);
  policyRevision.textContent = remote.policyRevision || '-';
  ruleCounts.textContent = countLabel(activeGroup);
  entryNodeName.textContent = activeGroup ? activeGroup.entryNodeName : text('noGroup');
  entryNodeAddress.textContent = activeGroup ? `${activeGroup.proxyScheme} ${activeGroup.proxyHost}:${activeGroup.proxyPort}` : '-';

  groupSelect.innerHTML = '';
  for (const group of remote.groups || []) {
    const option = document.createElement('option');
    option.value = group.id;
    option.textContent = group.name;
    option.selected = activeGroup && group.id === activeGroup.id;
    groupSelect.appendChild(option);
  }
  groupSelect.disabled = !(remote.groups || []).length;
  masterToggle.disabled = !activeGroup;
  addDirect.disabled = !(currentTab && currentTab.host);
  addProxy.disabled = !(currentTab && currentTab.host);

  if (!session.accessToken) {
    setStatus('warning', text('statusLoginRequired'));
  } else if (!activeGroup) {
    setStatus('warning', text('statusNoGroups'));
  } else if (!state.enabled) {
    setStatus('idle', text('statusReadyOff'));
  } else {
    setStatus('ok', text('statusReadyOn'));
  }
}

masterToggle.addEventListener('click', async () => {
  if (!viewState || !viewState.activeGroup) {
    return;
  }
  const result = await sendMessage({ type: 'set-enabled', enabled: !viewState.state.enabled });
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
});

groupSelect.addEventListener('change', async (event) => {
  const result = await sendMessage({ type: 'select-group', groupId: event.target.value });
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
});

addDirect.addEventListener('click', async () => {
  const result = await sendMessage({ type: 'add-current-host-to-direct' });
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
});

addProxy.addEventListener('click', async () => {
  const result = await sendMessage({ type: 'add-current-host-to-proxy' });
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
});

syncButton.addEventListener('click', async () => {
  const result = await sendMessage({ type: 'sync-remote-config' });
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
  setStatus('ok', text('statusSynced'));
});

bindThemeMode(themeMode, async (mode) => {
  const result = await sendMessage({ type: 'set-theme-mode', themeMode: mode });
  if (result && result.error) {
    setStatus('error', result.error);
    return;
  }
  render(result);
});

openOptions.addEventListener('click', () => chrome.runtime.openOptionsPage());

chrome.runtime.onMessage.addListener((message) => {
  if (message && message.type === 'state-updated') {
    render(message.payload);
  }
});

async function init() {
  applyLanguage();
  render(await sendMessage({ type: 'get-state' }));
}

init();
