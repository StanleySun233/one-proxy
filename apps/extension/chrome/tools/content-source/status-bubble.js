(function () {
  const ROOT_ID = 'one-proxy-status-root';
  const POSITION_KEY = 'oneProxyStatusBubblePosition';
  const FADE_DELAY_MS = 5000;
  const REFRESH_INTERVAL_MS = 30000;
  const state = {
    root: null,
    icon: null,
    panel: null,
    payload: null,
    opened: false,
    fadeTimer: 0,
    refreshTimer: 0,
    dragging: false,
    dragOffset: { x: 0, y: 0 }
  };

  function label(key) {
    const labels = state.payload && state.payload.labels;
    const value = labels && labels[key];
    if (!value) {
      throw new Error(`missing_status_bubble_label:${key}`);
    }
    return value;
  }

  function formatMb(bytes) {
    return `${(Number(bytes || 0) / 1024 / 1024).toFixed(2)} MB`;
  }

  function formatLatency(ms) {
    const value = Number(ms || 0);
    return value > 0 ? `${Math.round(value)} ms` : '-';
  }

  function pad2(value) {
    return String(value).padStart(2, '0');
  }

  function formatDateTime(value) {
    if (!value) {
      return '-';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '-';
    }
    return `${date.getFullYear()}-${pad2(date.getMonth() + 1)}-${pad2(date.getDate())} ${pad2(date.getHours())}:${pad2(date.getMinutes())}:${pad2(date.getSeconds())}`;
  }

  function text(value, fallback) {
    const clean = String(value || '').trim();
    return clean || fallback || '-';
  }

  function clear(node) {
    while (node.firstChild) {
      node.removeChild(node.firstChild);
    }
  }

  function el(tag, className, content) {
    const node = document.createElement(tag);
    if (className) {
      node.className = className;
    }
    if (content !== undefined) {
      node.textContent = content;
    }
    return node;
  }

  function svgIcon(kind) {
    const icons = {
      user: '<path d="M12 12a4 4 0 1 0-4-4 4 4 0 0 0 4 4Zm0 2c-4 0-7 2.2-7 5v1h14v-1c0-2.8-3-5-7-5Z"/>',
      node: '<path d="M5 4h14v6H5Zm0 10h14v6H5Zm2-8v2h2V6Zm0 10v2h2v-2Z"/>',
      web: '<path d="M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18Zm6.7 8h-3.1a14.4 14.4 0 0 0-1.1-5 7 7 0 0 1 4.2 5ZM12 5c.7 1 1.3 3 1.5 6h-3C10.7 8 11.3 6 12 5Zm0 14c-.7-1-1.3-3-1.5-6h3c-.2 3-.8 5-1.5 6ZM5.3 13h3.1a14.4 14.4 0 0 0 1.1 5 7 7 0 0 1-4.2-5Zm3.1-2H5.3a7 7 0 0 1 4.2-5 14.4 14.4 0 0 0-1.1 5Zm6.1 7a14.4 14.4 0 0 0 1.1-5h3.1a7 7 0 0 1-4.2 5Z"/>'
    };
    const wrap = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    wrap.setAttribute('viewBox', '0 0 24 24');
    wrap.setAttribute('aria-hidden', 'true');
    wrap.innerHTML = icons[kind] || icons.node;
    return wrap;
  }

  function row(label, value) {
    const item = el('div', 'opsb-row');
    item.append(el('span', 'opsb-label', label));
    item.append(el('span', 'opsb-value', text(value)));
    return item;
  }

  function stat(label, value) {
    const item = el('div', 'opsb-stat');
    item.append(el('span', 'opsb-stat-value', value));
    item.append(el('span', 'opsb-stat-label', label));
    return item;
  }

  function routeText(payload) {
    const route = payload.route || {};
    if (!route.matchType && !route.matchValue) {
      return text(route.source, label('statusBubbleUnknown'));
    }
    return `${text(route.source)} · ${text(route.matchType)} ${text(route.matchValue)}`;
  }

  function nodeTiming(payload, nodeId) {
    const timing = (payload.nodeTimings || []).find(function (item) {
      return item && item.nodeId === nodeId;
    });
    return timing ? Number(timing.processAvgMs || 0) : null;
  }

  function linkTiming(payload, fromNodeId, toNodeId) {
    const timing = (payload.linkTimings || []).find(function (item) {
      return item && item.fromNodeId === fromNodeId && item.toNodeId === toNodeId;
    });
    return timing ? Number(timing.roundTripMs || 0) : 0;
  }

  function edgeLatency(payload, previous, node) {
    return linkTiming(payload, previous.id, node.id);
  }

  function transportLabel(value) {
    switch (String(value || '').toLowerCase()) {
      case 'direct_quic':
        return label('statusBubbleDirectQUIC');
      case 'relay':
      case 'relay_ws_parent':
        return label('statusBubbleRelay');
      case 'client':
        return label('statusBubbleUserMachine');
      case 'target':
        return label('statusBubbleWebsite');
      default:
        return text(value, label('statusBubbleUnknown'));
    }
  }

  function pathConnector(payload, previous, node, className) {
    const connector = el('div', className || 'opsb-path-link');
    const transport = node.transport || previous.transport || '';
    connector.append(el('span', 'opsb-hop-latency', `${label('statusBubbleRoundTrip')} ${formatLatency(edgeLatency(payload, previous, node))}`));
    connector.append(el('span', 'opsb-hop-transport', transportLabel(transport)));
    connector.append(el('span', 'opsb-hop-segment'));
    return connector;
  }

  function pathNode(node) {
    const item = el('div', 'opsb-hop');
    const icon = el('span', `opsb-hop-icon opsb-hop-${node.kind}`);
    icon.append(svgIcon(node.kind));
    item.append(icon);
    item.append(el('span', 'opsb-hop-name', text(node.name, node.id)));
    if (node.transport && node.kind !== 'web') {
      item.append(el('span', 'opsb-hop-mode', transportLabel(node.transport)));
    }
    if (node.kind === 'node') {
      item.append(el('span', 'opsb-hop-process', node.processMs === null ? '-' : `${Math.round(node.processMs)} ms`));
    }
    return item;
  }

  function renderTopology(payload) {
    const rail = el('div', 'opsb-topology');
    const path = payload.path || {};
    if (!Array.isArray(path.nodes) || path.nodes.length < 2) {
      rail.append(el('div', 'opsb-path-fallback', label('statusBubbleUnknown')));
      return rail;
    }
    const nodes = path.nodes.map((node) => ({
      id: node.id,
      name: node.name || (node.kind === 'user' ? label('statusBubbleUserMachine') : node.kind === 'web' ? label('statusBubbleWebsite') : node.id),
      kind: node.kind || 'node',
      transport: node.transport || path.transport || path.mode || '',
      processMs: node.kind === 'node' ? nodeTiming(payload, node.id) : null
    }));
    if (path.fallbackReason) {
      rail.append(el('div', 'opsb-path-fallback', `${label('statusBubbleFallback')}: ${path.fallbackReason}`));
    }
    for (let start = 0; start < nodes.length; start += 4) {
      const rowIndex = start / 4;
      const chunk = nodes.slice(start, start + 4);
      const visualNodes = rowIndex % 2 === 0 ? chunk : chunk.slice().reverse();
      const rowNode = el('div', 'opsb-path-row');
      visualNodes.forEach((node, visualIndex) => {
        if (visualIndex > 0) {
          const previous = rowIndex % 2 === 0 ? visualNodes[visualIndex - 1] : visualNodes[visualIndex];
          const next = rowIndex % 2 === 0 ? visualNodes[visualIndex] : visualNodes[visualIndex - 1];
          rowNode.append(pathConnector(payload, previous, next));
        }
        rowNode.append(pathNode(node));
      });
      rail.append(rowNode);
      if (start + 4 < nodes.length) {
        const turn = pathConnector(payload, nodes[start + 3], nodes[start + 4], rowIndex % 2 === 0 ? 'opsb-path-turn opsb-path-turn-right' : 'opsb-path-turn opsb-path-turn-left');
        rail.append(turn);
      }
    }
    return rail;
  }

  function renderActions() {
    const actions = el('div', 'opsb-actions');
    const refresh = el('button', 'opsb-button', label('statusBubbleRefresh'));
    refresh.type = 'button';
    refresh.addEventListener('click', function () {
      refreshStatus(true);
    });
    const copy = el('button', 'opsb-button', label('statusBubbleCopy'));
    copy.type = 'button';
    copy.addEventListener('click', function () {
      copyDiagnostics(copy);
    });
    actions.append(refresh);
    actions.append(copy);
    return actions;
  }

  function diagnosticsPayload() {
    return JSON.stringify(state.payload || {}, null, 2);
  }

  function copyDiagnostics(button) {
    const value = diagnosticsPayload();
    const done = function () {
      button.textContent = label('statusBubbleCopied');
      window.setTimeout(function () {
        button.textContent = label('statusBubbleCopy');
      }, 1200);
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(value).then(done);
      return;
    }
    throw new Error('clipboard_unavailable');
  }

  function renderPanel() {
    clear(state.panel);
    const payload = state.payload || {};
    const page = payload.page || {};
    const io = payload.io || {};
    const cache = payload.cache || {};
    const header = el('div', 'opsb-panel-head');
    header.append(el('div', 'opsb-title', label('statusBubbleTitle')));
    const status = el('div', `opsb-status opsb-${payload.color || 'gray'}`, text(payload.status, 'unknown'));
    header.append(status);
    state.panel.append(header);
    state.panel.append(renderActions());

    const stats = el('div', 'opsb-stats');
    stats.append(stat(label('statusBubbleUpload'), formatMb(io.uploadBytes)));
    stats.append(stat(label('statusBubbleDownload'), formatMb(io.downloadBytes)));
    stats.append(stat(label('statusBubbleLatency'), formatLatency(payload.latencyMs)));
    stats.append(stat(label('statusBubbleRequests'), String(page.requestCount || 0)));
    state.panel.append(stats);

    const section = el('div', 'opsb-section');
    section.append(row(label('account'), payload.account));
    section.append(row(label('tenant'), payload.tenant && payload.tenant.name));
    section.append(row(label('activeGroup'), payload.group && payload.group.name));
    section.append(row(label('statusBubbleRoute'), routeText(payload)));
    section.append(row(label('policyRevision'), payload.policyRevision));
    section.append(row(label('syncedAt'), formatDateTime(payload.configFetchedAt)));
    section.append(row(label('statusBubbleOpenedAt'), formatDateTime(page.openedAt)));
    section.append(row(label('statusBubbleRequestMixShort'), `${page.proxiedRequestCount || 0} / ${page.directRequestCount || 0} / ${page.failureCount || 0}`));
    section.append(row(label('statusBubbleCorrelation'), io.correlated ? label('statusBubbleCorrelated') : label('statusBubbleNotCorrelated')));
    if (cache.status) {
      section.append(row(label('statusBubbleCache'), cache.status));
      section.append(row(label('statusBubbleCacheStoredAt'), formatDateTime(cache.storedAt)));
      section.append(row(label('statusBubbleCacheResponses'), String(cache.responseCount || 0)));
    }
    section.append(row(label('statusBubbleLastError'), payload.lastError && (payload.lastError.code || payload.lastError.message) ? `${payload.lastError.code || ''} ${payload.lastError.message || ''}` : '-'));
    state.panel.append(section);

    const topoTitle = el('div', 'opsb-subtitle', label('statusBubbleTopology'));
    state.panel.append(topoTitle);
    try {
      state.panel.append(renderTopology(payload));
    } catch (_error) {
      state.panel.append(el('div', 'opsb-path-fallback', label('statusBubbleUnknown')));
    }
  }

  function setOpen(opened) {
    state.opened = opened;
    state.root.classList.toggle('opsb-open', opened);
    state.root.classList.remove('opsb-fading');
    if (opened) {
      renderPanel();
    }
  }

  function scheduleFade() {
    window.clearTimeout(state.fadeTimer);
    state.fadeTimer = window.setTimeout(function () {
      state.root.classList.add('opsb-fading');
      window.setTimeout(function () {
        setOpen(false);
      }, 250);
    }, FADE_DELAY_MS);
  }

  function savePosition(left, top) {
    localStorage.setItem(POSITION_KEY, JSON.stringify({ left, top }));
  }

  function applySavedPosition() {
    const saved = JSON.parse(localStorage.getItem(POSITION_KEY) || 'null');
    if (!saved) {
      return;
    }
    state.root.style.left = `${Math.max(8, Number(saved.left) || 16)}px`;
    state.root.style.top = `${Math.max(8, Number(saved.top) || 0)}px`;
    state.root.style.bottom = 'auto';
  }

  function enableDrag() {
    state.icon.addEventListener('pointerdown', function (event) {
      state.dragging = true;
      const rect = state.root.getBoundingClientRect();
      state.dragOffset = { x: event.clientX - rect.left, y: event.clientY - rect.top };
      state.icon.setPointerCapture(event.pointerId);
    });
    state.icon.addEventListener('pointermove', function (event) {
      if (!state.dragging) {
        return;
      }
      const left = Math.min(window.innerWidth - 52, Math.max(8, event.clientX - state.dragOffset.x));
      const top = Math.min(window.innerHeight - 52, Math.max(8, event.clientY - state.dragOffset.y));
      state.root.style.left = `${left}px`;
      state.root.style.top = `${top}px`;
      state.root.style.bottom = 'auto';
    });
    state.icon.addEventListener('pointerup', function () {
      if (!state.dragging) {
        return;
      }
      state.dragging = false;
      const rect = state.root.getBoundingClientRect();
      savePosition(rect.left, rect.top);
    });
  }

  function ensureRoot() {
    if (state.root) {
      return;
    }
    const existing = document.getElementById(ROOT_ID);
    if (existing) {
      existing.remove();
    }
    state.root = el('div', 'opsb-root opsb-gray');
    state.root.id = ROOT_ID;
    state.icon = el('button', 'opsb-icon');
    state.icon.type = 'button';
    state.icon.append(svgIcon('node'));
    state.panel = el('div', 'opsb-panel');
    state.root.append(state.icon);
    state.root.append(state.panel);
    document.documentElement.append(state.root);
    applySavedPosition();
    enableDrag();
    state.icon.addEventListener('click', function () {
      if (!state.dragging) {
        setOpen(!state.opened);
      }
    });
    state.panel.addEventListener('mouseenter', function () {
      window.clearTimeout(state.fadeTimer);
      state.root.classList.remove('opsb-fading');
    });
    state.panel.addEventListener('mouseleave', scheduleFade);
  }

  function removeRoot() {
    if (state.root) {
      state.root.remove();
    }
    state.root = null;
    window.clearInterval(state.refreshTimer);
  }

  function applyPayload(payload) {
    state.payload = payload;
    if (!payload || payload.display === false) {
      removeRoot();
      return;
    }
    ensureRoot();
    state.icon.title = label('statusBubbleTitle');
    state.root.classList.remove('opsb-red', 'opsb-yellow', 'opsb-green', 'opsb-gray');
    state.root.classList.add(`opsb-${payload.color || 'gray'}`);
    if (state.opened) {
      renderPanel();
    }
  }

  function refreshStatus(force) {
    chrome.runtime.sendMessage({
      type: 'status-bubble-page-status',
      url: window.location.href,
      refresh: Boolean(force)
    }, function (response) {
      if (chrome.runtime.lastError) {
        throw new Error(chrome.runtime.lastError.message);
      }
      applyPayload(response);
    });
  }

  if (!/^https?:$/.test(window.location.protocol)) {
    return;
  }
  refreshStatus(false);
  state.refreshTimer = window.setInterval(function () {
    refreshStatus(false);
  }, REFRESH_INTERVAL_MS);
})();
