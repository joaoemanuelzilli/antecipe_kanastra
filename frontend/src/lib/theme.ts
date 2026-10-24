import { useSyncExternalStore } from 'react';

export type Theme = 'light' | 'dark';
const STORAGE_KEY = 'antecipe:theme';
const CHANGE_EVENT = 'antecipe:theme-change';

function readStoredTheme(): Theme {
  try { return localStorage.getItem(STORAGE_KEY) === 'dark' ? 'dark' : 'light'; }
  catch { return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'; }
}

export function initializeTheme() {
  document.documentElement.dataset.theme = readStoredTheme();
}

function readTheme(): Theme {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light';
}

function subscribe(callback: () => void) {
  window.addEventListener(CHANGE_EVENT, callback);
  window.addEventListener('storage', callback);
  return () => {
    window.removeEventListener(CHANGE_EVENT, callback);
    window.removeEventListener('storage', callback);
  };
}

export function useTheme() {
  const theme = useSyncExternalStore(subscribe, readTheme);
  function setTheme(next: Theme) {
    document.documentElement.dataset.theme = next;
    try { localStorage.setItem(STORAGE_KEY, next); } catch { /* Keep the theme usable when storage is blocked. */ }
    window.dispatchEvent(new Event(CHANGE_EVENT));
  }
  return { theme, setTheme };
}

window.addEventListener('storage', initializeTheme);
