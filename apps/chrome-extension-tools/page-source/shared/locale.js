const messages = new Map();

export function text(key, substitutions) {
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

export function applyLanguage(root = document) {
  document.documentElement.lang = chrome.i18n.getUILanguage().startsWith('zh') ? 'zh-CN' : 'en';
  root.querySelectorAll('[data-lang]').forEach((element) => {
    element.textContent = text(element.dataset.lang);
  });
  root.querySelectorAll('[data-lang-title]').forEach((element) => {
    element.title = text(element.dataset.langTitle);
  });
}
