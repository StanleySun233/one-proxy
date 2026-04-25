const messages = new Map();

export function text(key, substitutions) {
  if (substitutions) {
    return chrome.i18n.getMessage(key, substitutions) || key;
  }
  if (!messages.has(key)) {
    messages.set(key, chrome.i18n.getMessage(key) || key);
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
