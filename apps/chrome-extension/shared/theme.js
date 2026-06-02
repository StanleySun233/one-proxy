export function normalizeThemeMode(mode) {
  return mode === 'dark' ? 'dark' : 'vivid';
}

export function applyThemeMode(mode, root = document) {
  const normalized = normalizeThemeMode(mode);
  const documentElement = root.documentElement || document.documentElement;
  documentElement.dataset.theme = normalized;
  root.querySelectorAll('[data-theme-mode]').forEach((button) => {
    button.classList.toggle('is-active', button.dataset.themeMode === normalized);
  });
  return normalized;
}

export function bindThemeMode(container, onChange) {
  if (!container) {
    return;
  }
  container.querySelectorAll('[data-theme-mode]').forEach((button) => {
    button.addEventListener('click', async () => {
      const mode = applyThemeMode(button.dataset.themeMode);
      await onChange(mode);
    });
  });
}
