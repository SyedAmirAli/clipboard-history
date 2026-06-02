// Entry point. Wires together titlebar → search → chips → list → context
// menu → settings, and subscribes to backend events so the UI updates
// whenever a new clipboard entry arrives.

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
  chips: $<HTMLElement>('#chips'),
  ctxMenu: $<HTMLElement>('#ctx-menu'),
  settings: $<HTMLElement>('#settings-modal'),
  scrim: $<HTMLElement>('#settings-scrim'),
  mainContent: $<HTMLElement>('#main-content'),
  toast: $<HTMLElement>('#toast'),
  openSettings: $<HTMLButtonElement>('#open-settings'),
  winClose: $<HTMLButtonElement>('#win-close'),
  winMin: $<HTMLButtonElement>('#win-min'),
  winMax: $<HTMLButtonElement>('#win-max'),
  statusCount: $<HTMLElement>('#status-count'),
};

type Filter = 'all' | 'text' | 'images' | 'pinned';

let currentSettings: AppSettings | null = null;
let allItems: ClipItem[] = [];
let activeFilter: Filter = 'all';

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
  chrome: { scrim: els.scrim, mainContent: els.mainContent },
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
  onDelete: async (item) => {
    await api.deleteItem(item.id);
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

// Filter chips
els.chips.querySelectorAll<HTMLElement>('.chip').forEach((chip) => {
  chip.addEventListener('click', () => {
    els.chips.querySelectorAll('.chip').forEach((c) => c.classList.remove('active'));
    chip.classList.add('active');
    activeFilter = (chip.dataset.filter as Filter) ?? 'all';
    applyFilter();
  });
});

// Window controls (macOS traffic lights)
els.winClose.addEventListener('click', () => api.hidePopup().catch(() => undefined));
els.winMin.addEventListener('click', () => api.minimizeWindow());
els.winMax.addEventListener('click', () => api.toggleMaximizeWindow());

document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    if (settings.isOpen()) return; // overlay handles its own close
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
  if (currentSettings?.windowFrame) return; // windowed mode lives in the taskbar
  if (currentSettings?.hideOnBlur === false) return;
  if (settings.isOpen()) return;
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

// ---------- Rendering ----------

async function refresh() {
  try {
    allItems = await api.listItems(search.value(), currentSettings?.maxItems ?? 200);
    applyFilter();
  } catch (err) {
    console.error('list items failed', err);
  }
}

/** Apply the active chip filter on top of the backend search result. */
function applyFilter() {
  const visible = allItems.filter((it) => {
    switch (activeFilter) {
      case 'text':
        return it.contentType === 'text';
      case 'images':
        return it.contentType === 'image';
      case 'pinned':
        return it.pinned;
      default:
        return true;
    }
  });
  const isFiltered = activeFilter !== 'all' || search.value().trim() !== '';
  list.setItems(visible, { filtered: isFiltered });
  setCount(visible);
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

// ---------- Boot ----------

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
