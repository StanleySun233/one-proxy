import { applyLanguage, text } from '../shared/locale.js';
import { applyThemeMode, bindThemeMode } from '../shared/theme.js';

const routingModes = [
  { value: 'smart', labelKey: 'routingSmart' },
  { value: 'all', labelKey: 'routingAll' },
  { value: 'internal', labelKey: 'routingInternalShort' },
  { value: 'external', labelKey: 'routingExternalShort' }
];

const masterToggle = document.getElementById('masterToggle');
const internalToggle = document.getElementById('internalToggle');
const masterValue = document.getElementById('masterValue');
const internalValue = document.getElementById('internalValue');
const activeGroupName = document.getElementById('activeGroupName');
const activeProfileName = document.getElementById('activeProfileName');
const groupSelect = document.getElementById('groupSelect');
const profileSelect = document.getElementById('profileSelect');
const routingMode = document.getElementById('routingMode');
const currentSite = document.getElementById('currentSite');
const addBypass = document.getElementById('addBypass');
const addProxy = document.getElementById('addProxy');
const addInternal = document.getElementById('addInternal');
const openOptions = document.getElementById('openOptions');
const themeMode = document.getElementById('themeMode');

let viewState = null;

function sendMessage(message) {
  return chrome.runtime.sendMessage(message);
}

function setToggleState(element, enabled) {
  element.classList.toggle('is-on', enabled);
  element.classList.toggle('is-off', !enabled);
}

function getActiveGroup(state) {
  return state.groups.find((group) => group.id === state.activeGroupId) || state.groups[0];
}

function getProfileMap(state) {
  return new Map(state.profiles.map((profile) => [profile.id, profile]));
}

function renderRouting(group) {
  routingMode.innerHTML = '';
  for (const mode of routingModes) {
    const button = document.createElement('button');
    button.type = 'button';
    button.textContent = text(mode.labelKey);
    button.classList.toggle('is-active', group.routingMode === mode.value);
    button.addEventListener('click', async () => {
      const next = structuredClone(viewState.state);
      const target = next.groups.find((item) => item.id === group.id);
      target.routingMode = mode.value;
      await sendMessage({ type: 'set-state', payload: next });
    });
    routingMode.appendChild(button);
  }
}

function render(stateBundle) {
  viewState = stateBundle;
  const { state, activeGroup, activeProfile, currentTab } = stateBundle;
  activeGroupName.textContent = (activeGroup && activeGroup.name) || text('noGroup');
  activeProfileName.textContent = activeProfile ? `${activeProfile.name} · ${activeProfile.host}:${activeProfile.port}` : text('noNode');
  masterValue.textContent = state.enabled ? text('on') : text('off');
  internalValue.textContent = state.internalProxyEnabled ? text('proxy') : text('direct');
  setToggleState(masterToggle, state.enabled);
  setToggleState(internalToggle, state.internalProxyEnabled);
  currentSite.textContent = (currentTab && currentTab.host) || text('noSite');
  applyThemeMode(state.themeMode);

  groupSelect.innerHTML = '';
  for (const group of state.groups) {
    const option = document.createElement('option');
    option.value = group.id;
    option.textContent = group.name;
    option.selected = group.id === state.activeGroupId;
    groupSelect.appendChild(option);
  }

  const profileMap = getProfileMap(state);
  profileSelect.innerHTML = '';
  for (const profileId of ((activeGroup && activeGroup.profileIds) || [])) {
    const profile = profileMap.get(profileId);
    if (!profile) {
      continue;
    }
    const option = document.createElement('option');
    option.value = profile.id;
    option.textContent = `${profile.name} · ${profile.host}:${profile.port}`;
    option.selected = profile.id === activeGroup.activeProfileId;
    profileSelect.appendChild(option);
  }

  renderRouting(activeGroup);
}

masterToggle.addEventListener('click', async () => {
  const next = structuredClone(viewState.state);
  next.enabled = !next.enabled;
  await sendMessage({ type: 'set-state', payload: next });
});

internalToggle.addEventListener('click', async () => {
  const next = structuredClone(viewState.state);
  next.internalProxyEnabled = !next.internalProxyEnabled;
  await sendMessage({ type: 'set-state', payload: next });
});

groupSelect.addEventListener('change', async (event) => {
  const next = structuredClone(viewState.state);
  next.activeGroupId = event.target.value;
  await sendMessage({ type: 'set-state', payload: next });
});

profileSelect.addEventListener('change', async (event) => {
  const next = structuredClone(viewState.state);
  const group = getActiveGroup(next);
  group.activeProfileId = event.target.value;
  await sendMessage({ type: 'set-state', payload: next });
});

addBypass.addEventListener('click', async () => {
  const result = await sendMessage({ type: 'add-current-host-to-bypass' });
  render(result);
});

addProxy.addEventListener('click', async () => {
  const result = await sendMessage({ type: 'add-current-host-to-proxy' });
  render(result);
});

addInternal.addEventListener('click', async () => {
  const result = await sendMessage({ type: 'add-current-host-to-internal' });
  render(result);
});

bindThemeMode(themeMode, async (mode) => {
  if (!viewState || viewState.state.themeMode === mode) {
    return;
  }
  const next = structuredClone(viewState.state);
  next.themeMode = mode;
  await sendMessage({ type: 'set-state', payload: next });
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
