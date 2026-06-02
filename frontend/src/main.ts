// Entry point. Wires together search → list → context menu → settings,
// and subscribes to backend events so the UI updates whenever a new
// clipboard entry arrives.

import './styles/main.css';
import './styles/components.css';

import { api, onEvent } from './api';
import { createSearchBar } from './components/searchBar';
import { createItemList } from './components/itemList';
import { createContextMenu } from './components/itemContextMenu';
import { createSettingsModal } from './components/settingsModal';
import type { AppSettings, ClipItem } from './types';

const $ = <T extends HTMLElement>(sel: string) => document.querySelector<T>(sel)!;

const els = {
  searchInput: $<HTMLInputElement>('#search'),
  list: $<HTMLElement>('#list'),
  ctxMenu: $<HTMLElement>('#ctx-menu'),
  settings: $<HTMLElement>('#settings-modal'),
  toast: $<HTMLElement>('#toast'),
  openSettings: $<HTMLButtonElement>('#open-settings'),
  statusCount: $<HTMLElement>('#status-count'),
};

let currentSettings: AppSettings | null = null;

const search = createSearchBar(els.searchInput);
const settings = createSettingsModal(els.settings, {
  load: () => api.getSettings(),
  save: async (s) => {
    const saved = await api.updateSettings(s);
    currentSettings = saved;
    settings.applyTheme(saved.theme);
    return saved;
  },
  clearAll: async (keep) => {
    await api.clearAll(keep);
    await refresh();
  },
  systemInfo: () => api.systemInfo(),
});

const list = createItemList(els.list, {
  onPaste: async (item) => {
    try {
      await api.pasteItem(item.id);
      flash('Copied to clipboard');
    } catch (err) {
      flash('Paste failed: ' + String(err));
    }
  },
  onPinToggle: async (item) => {
    await api.pinItem(item.id, !item.pinned);
    await refresh();
  },
  onContextMenu: (at, item) => ctxMenu.open(at, item),
});

const ctxMenu = createContextMenu(els.ctxMenu, {
  onPasteCopy: async (item) => {
    try {
      await api.pasteItem(item.id);
      flash('Copied to clipboard');
    } catch (err) {
      flash('Paste failed: ' + String(err));
    }
  },
  onPinToggle: async (item) => {
    await api.pinItem(item.id, !item.pinned);
    await refresh();
  },
  onDelete: async (item) => {
    await api.deleteItem(item.id);
    await refresh();
  },
});

// ---------- Wiring ----------

search.onChange(async () => {
  await refresh();
});
search.onKeyToList((e) => list.handleKey(e));

els.list.addEventListener('keydown', (e) => list.handleKey(e));

els.openSettings.addEventListener('click', () => settings.open());

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    if (!els.settings.classList.contains('hidden')) return; // modal handles its own close
    if (!els.ctxMenu.classList.contains('hidden')) return; // ctx menu closes itself
    api.hidePopup();
  } else if ((e.ctrlKey || e.metaKey) && e.key === ',') {
    e.preventDefault();
    settings.open();
  } else if (e.key === '/' && document.activeElement !== els.searchInput) {
    e.preventDefault();
    search.focus();
  }
});

window.addEventListener('blur', () => {
  if (currentSettings?.hideOnBlur === false) return;
  if (!els.settings.classList.contains('hidden')) return;
  if (!els.ctxMenu.classList.contains('hidden')) return;
  // Slight delay so click-to-paste flow finishes first.
  setTimeout(() => api.hidePopup().catch(() => undefined), 150);
});

// React to backend pushes.
onEvent('clipboard:new-item', async () => {
  await refresh();
});
onEvent('clipboard:cleared', async () => {
  await refresh();
});

// ---------- Boot ----------

async function refresh() {
  try {
    const items = await api.listItems(search.value(), currentSettings?.maxItems ?? 200);
    list.setItems(items);
    setCount(items);
  } catch (err) {
    console.error('list items failed', err);
  }
}

function setCount(items: ClipItem[]) {
  const total = items.length;
  const pinned = items.filter((i) => i.pinned).length;
  els.statusCount.textContent =
    total === 0
      ? 'No items'
      : pinned > 0
      ? `${total} items · ${pinned} pinned`
      : `${total} items`;
}

let toastTimer: number | undefined;
function flash(msg: string) {
  els.toast.textContent = msg;
  els.toast.classList.remove('hidden');
  requestAnimationFrame(() => els.toast.classList.add('show'));
  if (toastTimer) window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => {
    els.toast.classList.remove('show');
    setTimeout(() => els.toast.classList.add('hidden'), 200);
  }, 1500);
}

async function boot() {
  try {
    currentSettings = await api.getSettings();
    settings.applyTheme(currentSettings.theme);
  } catch (err) {
    console.warn('initial settings load failed (running outside Wails?)', err);
  }
  await refresh();
  search.focus();
}

boot();
