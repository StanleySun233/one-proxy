import { applyLanguage, text } from '../shared/locale.js';
import { applyThemeMode, bindThemeMode } from '../shared/theme.js';

const routingModes = ['smart', 'all', 'internal', 'external'];
const schemeModes = ['PROXY', 'HTTPS', 'SOCKS5'];

const groupList = document.getElementById('groupList');
const profileList = document.getElementById('profileList');
const editorTitle = document.getElementById('editorTitle');
const groupName = document.getElementById('groupName');
const groupMode = document.getElementById('groupMode');
const bypassHosts = document.getElementById('bypassHosts');
const proxyHosts = document.getElementById('proxyHosts');
const internalDomains = document.getElementById('internalDomains');
const internalCidrs = document.getElementById('internalCidrs');
const quickBypass = document.getElementById('quickBypass');
const quickProxy = document.getElementById('quickProxy');
const quickInternal = document.getElementById('quickInternal');
const addGroup = document.getElementById('addGroup');
const removeGroup = document.getElementById('removeGroup');
const addProfile = document.getElementById('addProfile');
const saveAll = document.getElementById('saveAll');
const themeMode = document.getElementById('themeMode');

let draft = null;
let selectedGroupId = null;

function uid(prefix) {
  return `${prefix}-${crypto.randomUUID()}`;
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

function currentGroup() {
  return draft.groups.find((group) => group.id === selectedGroupId) || draft.groups[0];
}

function currentProfiles(group) {
  return draft.profiles.filter((profile) => group.profileIds.includes(profile.id));
}

function routingModeLabel(mode) {
  return text(`routingMode_${mode}`);
}

async function loadState() {
  applyLanguage();
  const bundle = await chrome.runtime.sendMessage({ type: 'get-state' });
  draft = structuredClone(bundle.state);
  selectedGroupId = draft.activeGroupId;
  render();
}

function renderGroupList() {
  groupList.innerHTML = '';
  for (const group of draft.groups) {
    const profileCount = currentProfiles(group).length;
    const button = document.createElement('button');
    button.type = 'button';
    button.className = `group-row${group.id === selectedGroupId ? ' is-active' : ''}`;
    button.innerHTML = `
      <div class="group-name">
        <strong>${group.name}</strong>
        <span>${routingModeLabel(group.routingMode)}</span>
      </div>
      <div class="group-meta">${text('nodeCount', [String(profileCount)])}</div>
    `;
    button.addEventListener('click', () => {
      selectedGroupId = group.id;
      render();
    });
    groupList.appendChild(button);
  }
}

function renderGroupEditor() {
  const group = currentGroup();
  if (!group) {
    return;
  }
  editorTitle.textContent = group.name;
  groupName.value = group.name;
  groupMode.innerHTML = routingModes.map((mode) => `<option value="${mode}" ${group.routingMode === mode ? 'selected' : ''}>${routingModeLabel(mode)}</option>`).join('');
  bypassHosts.value = arrayToLines(group.bypassHosts);
  proxyHosts.value = arrayToLines(group.proxyHosts);
  internalDomains.value = arrayToLines(group.internalDomains);
  internalCidrs.value = arrayToLines(group.internalCidrs);
}

function renderProfiles() {
  const group = currentGroup();
  profileList.innerHTML = '';
  for (const profile of currentProfiles(group)) {
    const selected = profile.id === group.activeProfileId;
    const item = document.createElement('div');
    item.className = 'profile-item';
    item.innerHTML = `
      <div class="profile-head">
        <strong>${profile.name}</strong>
        <label><input type="radio" name="activeProfile" value="${profile.id}" ${selected ? 'checked' : ''}/> ${text('active')}</label>
      </div>
      <div class="profile-meta">${profile.scheme} ${profile.host}:${profile.port}</div>
      <div class="profile-grid">
        <label>
          <span>${text('name')}</span>
          <input data-field="name" value="${profile.name}" />
        </label>
        <label>
          <span>${text('scheme')}</span>
          <select data-field="scheme">${schemeModes.map((scheme) => `<option value="${scheme}" ${profile.scheme === scheme ? 'selected' : ''}>${scheme}</option>`).join('')}</select>
        </label>
        <label class="wide">
          <span>${text('host')}</span>
          <input data-field="host" value="${profile.host}" />
        </label>
        <label>
          <span>${text('port')}</span>
          <input data-field="port" type="number" min="1" max="65535" value="${profile.port}" />
        </label>
      </div>
      <div class="profile-tools">
        <button type="button" class="ghost" data-action="duplicate">${text('duplicate')}</button>
        <button type="button" class="danger" data-action="delete">${text('delete')}</button>
      </div>
    `;

    item.querySelectorAll('[data-field]').forEach((input) => {
      input.addEventListener('input', (event) => {
        const field = event.target.dataset.field;
        profile[field] = field === 'port' ? Number(event.target.value) : event.target.value;
      });
    });

    item.querySelector('input[type="radio"]').addEventListener('change', () => {
      group.activeProfileId = profile.id;
    });

    item.querySelector('[data-action="duplicate"]').addEventListener('click', () => {
      const next = {
        ...structuredClone(profile),
        id: uid('profile'),
        name: text('copyName', [profile.name])
      };
      draft.profiles.push(next);
      group.profileIds.push(next.id);
      renderProfiles();
      renderGroupList();
    });

    item.querySelector('[data-action="delete"]').addEventListener('click', () => {
      group.profileIds = group.profileIds.filter((id) => id !== profile.id);
      draft.profiles = draft.profiles.filter((itemProfile) => itemProfile.id !== profile.id);
      if (group.activeProfileId === profile.id) {
        group.activeProfileId = group.profileIds[0] || null;
      }
      render();
    });

    profileList.appendChild(item);
  }
}

function renderQuickRules() {
  quickBypass.value = arrayToLines(draft.quickRules.bypassHosts);
  quickProxy.value = arrayToLines(draft.quickRules.proxyHosts);
  quickInternal.value = arrayToLines(draft.quickRules.internalDomains);
}

function render() {
  applyThemeMode(draft.themeMode);
  renderGroupList();
  renderGroupEditor();
  renderProfiles();
  renderQuickRules();
}

function syncEditorsIntoDraft() {
  const group = currentGroup();
  group.name = groupName.value.trim() || text('untitledGroup');
  group.routingMode = groupMode.value;
  group.bypassHosts = linesToArray(bypassHosts.value);
  group.proxyHosts = linesToArray(proxyHosts.value);
  group.internalDomains = linesToArray(internalDomains.value);
  group.internalCidrs = linesToArray(internalCidrs.value);
  draft.quickRules.bypassHosts = linesToArray(quickBypass.value);
  draft.quickRules.proxyHosts = linesToArray(quickProxy.value);
  draft.quickRules.internalDomains = linesToArray(quickInternal.value);
}

addGroup.addEventListener('click', () => {
  const profileId = uid('profile');
  const groupId = uid('group');
  draft.profiles.push({
    id: profileId,
    name: text('newNode'),
    scheme: 'PROXY',
    host: '127.0.0.1',
    port: 2388
  });
  draft.groups.push({
    id: groupId,
    name: text('newGroup'),
    routingMode: 'smart',
    profileIds: [profileId],
    activeProfileId: profileId,
    bypassHosts: [],
    proxyHosts: [],
    internalDomains: [],
    internalCidrs: []
  });
  selectedGroupId = groupId;
  render();
});

removeGroup.addEventListener('click', () => {
  if (draft.groups.length === 1) {
    return;
  }
  const group = currentGroup();
  draft.profiles = draft.profiles.filter((profile) => !group.profileIds.includes(profile.id));
  draft.groups = draft.groups.filter((item) => item.id !== group.id);
  selectedGroupId = draft.groups[0].id;
  if (draft.activeGroupId === group.id) {
    draft.activeGroupId = selectedGroupId;
  }
  render();
});

addProfile.addEventListener('click', () => {
  const group = currentGroup();
  const profileId = uid('profile');
  draft.profiles.push({
    id: profileId,
    name: text('newNode'),
    scheme: 'PROXY',
    host: '127.0.0.1',
    port: 2388
  });
  group.profileIds.push(profileId);
  if (!group.activeProfileId) {
    group.activeProfileId = profileId;
  }
  render();
});

bindThemeMode(themeMode, async (mode) => {
  if (draft.themeMode === mode) {
    return;
  }
  draft.themeMode = mode;
  applyThemeMode(draft.themeMode);
  const saved = await chrome.runtime.sendMessage({ type: 'set-state', payload: draft });
  draft = structuredClone(saved);
  selectedGroupId = draft.activeGroupId;
  render();
});

saveAll.addEventListener('click', async () => {
  syncEditorsIntoDraft();
  draft.activeGroupId = selectedGroupId;
  const saved = await chrome.runtime.sendMessage({ type: 'set-state', payload: draft });
  draft = structuredClone(saved);
  selectedGroupId = draft.activeGroupId;
  render();
});

[groupName, groupMode, bypassHosts, proxyHosts, internalDomains, internalCidrs, quickBypass, quickProxy, quickInternal].forEach((element) => {
  element.addEventListener('input', syncEditorsIntoDraft);
  element.addEventListener('change', syncEditorsIntoDraft);
});

loadState();
