// Lightweight right-click context menu used by list rows. It auto-closes
// when the user clicks anywhere else, or presses Escape.

import type { ClipItem } from '../types';

export interface ContextMenuActions {
  onPasteCopy(item: ClipItem): void;
  onPinToggle(item: ClipItem): void;
  onDelete(item: ClipItem): void;
}

export interface ContextMenu {
  open(at: { x: number; y: number }, item: ClipItem): void;
  close(): void;
}

const SVG_PIN = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="17" x2="12" y2="22"/><path d="M5 17h14v-1.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6h1a1 1 0 0 0 0-2H8a1 1 0 0 0 0 2h1v4.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24Z"/></svg>`;
const SVG_COPY = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`;
const SVG_DELETE = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/></svg>`;

export function createContextMenu(root: HTMLElement, actions: ContextMenuActions): ContextMenu {
  let currentItem: ClipItem | null = null;

  const close = () => {
    currentItem = null;
    root.classList.add('hidden');
  };

  const open = (at: { x: number; y: number }, item: ClipItem) => {
    currentItem = item;
    root.innerHTML = `
      <div class="mi" data-act="paste">${SVG_COPY}<span>Copy to clipboard</span></div>
      <div class="mi" data-act="pin">${SVG_PIN}<span>${item.pinned ? 'Unpin' : 'Pin to top'}</span></div>
      <div class="sep"></div>
      <div class="mi danger" data-act="delete">${SVG_DELETE}<span>Delete</span></div>
    `;
    root.classList.remove('hidden');

    // Position with clamp to viewport so the menu never overflows.
    const vw = window.innerWidth;
    const vh = window.innerHeight;
    const w = root.offsetWidth;
    const h = root.offsetHeight;
    const x = Math.min(at.x, vw - w - 6);
    const y = Math.min(at.y, vh - h - 6);
    root.style.left = `${Math.max(6, x)}px`;
    root.style.top = `${Math.max(6, y)}px`;
  };

  root.addEventListener('click', (e) => {
    const target = (e.target as HTMLElement).closest<HTMLElement>('.mi');
    if (!target || !currentItem) return;
    const item = currentItem;
    close();
    switch (target.dataset.act) {
      case 'paste':
        actions.onPasteCopy(item);
        break;
      case 'pin':
        actions.onPinToggle(item);
        break;
      case 'delete':
        actions.onDelete(item);
        break;
    }
  });

  // Close on outside click / Escape.
  document.addEventListener('mousedown', (e) => {
    if (root.classList.contains('hidden')) return;
    if (!root.contains(e.target as Node)) close();
  });
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape' && !root.classList.contains('hidden')) close();
  });

  return { open, close };
}
