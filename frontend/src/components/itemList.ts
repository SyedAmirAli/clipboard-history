// Renders the clipboard history list with a "Pinned" group at the top
// followed by "Recent". Supports full keyboard navigation (↑/↓ to move,
// Enter to paste, Esc handled by the page-level handler in main.ts).
//
// Rendering is plain innerHTML — for an MVP cap of 200 items, the cost
// is well under one frame and avoids a virtual-DOM dependency. If the
// cap is raised significantly, swap in a simple windowing scheme here.

import type { ClipItem } from "../types";

const SVG_TEXT = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="6" x2="20" y2="6"/><line x1="4" y1="12" x2="20" y2="12"/><line x1="4" y1="18" x2="14" y2="18"/></svg>`;
const SVG_PIN_FILLED = `<svg viewBox="0 0 24 24" fill="currentColor" stroke="none"><path d="M12 2.5c-.4 0-.78.16-1.06.44L8.06 5.88a1.5 1.5 0 0 0 0 2.12L9 8.94v3.32l-3.06 3.06a1.5 1.5 0 0 0 0 2.12L7.06 18.6a1.5 1.5 0 0 0 2.12 0L11 16.74V22h2v-5.26l1.82 1.82a1.5 1.5 0 0 0 2.12 0l1.12-1.12a1.5 1.5 0 0 0 0-2.12L15 12.26V8.94l.94-.94a1.5 1.5 0 0 0 0-2.12l-2.88-2.94c-.28-.28-.66-.44-1.06-.44z"/></svg>`;
const SVG_PIN_OUTLINE = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="17" x2="12" y2="22"/><path d="M5 17h14v-1.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V6h1a1 1 0 0 0 0-2H8a1 1 0 0 0 0 2h1v4.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24Z"/></svg>`;
const SVG_COPY = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`;
const SVG_DELETE = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/></svg>`;
const SVG_EMPTY = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="8" y="3" width="8" height="4" rx="1" ry="1"/><path d="M8 5H6a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2"/></svg>`;
const SVG_LOCK = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/></svg>`;
const SVG_DOWNLOAD = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`;

export interface ItemListActions {
    onPaste(item: ClipItem): void;
    onCopy(item: ClipItem): void;
    onPinToggle(item: ClipItem): void;
    onMoveToVault(item: ClipItem): void;
    onDownload(item: ClipItem): void;
    onPreview(item: ClipItem): void;
    onDelete(item: ClipItem): void;
    onContextMenu(at: { x: number; y: number }, item: ClipItem): void;
}

export interface ItemList {
    setItems(items: ClipItem[], opts?: { filtered?: boolean }): void;
    handleKey(e: KeyboardEvent): void;
    count(): number;
    pasteFocused(): void;
}

export function createItemList(root: HTMLElement, actions: ItemListActions): ItemList {
    let items: ClipItem[] = [];
    let focusIdx = -1;
    let filtered = false;

    const render = () => {
        if (items.length === 0) {
            root.innerHTML = filtered
                ? `
        <div class="empty-state">
          ${SVG_EMPTY.replace("<svg", '<svg class="empty-icon"')}
          <h3>No results</h3>
          <p>Try a different search or filter.</p>
        </div>
      `
                : `
        <div class="empty-state">
          ${SVG_EMPTY.replace("<svg", '<svg class="empty-icon"')}
          <h3>Your clipboard is empty</h3>
          <p>Copy some text or an image — it will appear here automatically.</p>
        </div>
      `;
            return;
        }
        const pinned = items.filter((i) => i.pinned);
        const recent = items.filter((i) => !i.pinned);

        let html = "";
        if (pinned.length > 0) {
            html += `<div class="section-header">Pinned</div>`;
            pinned.forEach((it) => (html += renderItem(it, items.indexOf(it))));
        }
        if (recent.length > 0) {
            html += `<div class="section-header">Recent</div>`;
            recent.forEach((it) => (html += renderItem(it, items.indexOf(it))));
        }
        root.innerHTML = html;
        applyFocus();
    };

    const renderItem = (it: ClipItem, idx: number): string => {
        const isImage = it.contentType === "image";
        const iconHtml = isImage
            ? it.imageThumb
                ? `<img alt="" src="${it.imageThumb}"/>`
                : `<div class="item-thumb-fallback"></div>`
            : SVG_TEXT;
        // Both states get a clickable pin button so the user can pin AND
        // unpin. Pinned shows a filled amber pin (title "Unpin").
        const pinHtml = it.pinned
            ? `<button class="ab pin-btn pinned" title="Unpin">${SVG_PIN_FILLED}</button>`
            : `<button class="ab pin-btn" title="Pin">${SVG_PIN_OUTLINE}</button>`;
        const preview = escapeHtml(it.preview || "");
        const time = formatRelative(it.lastUsedAt);
        const meta = isImage ? `${it.imageW || "?"}×${it.imageH || "?"} · ${time}` : time;
        return `
      <div class="item ${isImage ? "kind-image" : "kind-text"} ${it.pinned ? "pinned" : ""}"
           data-idx="${idx}" role="button" tabindex="-1">
        <button class="item-icon preview-btn" title="Preview" aria-label="Preview">${iconHtml}</button>
        <div class="item-main">
          <div class="item-preview">${preview}</div>
          <div class="item-meta">${meta}</div>
        </div>
        <div class="item-right">
          <div class="item-actions">
            ${pinHtml}
            <button class="ab copy-btn" title="Copy">${SVG_COPY}</button>
            <button class="ab vault-btn" title="Move to Vault">${SVG_LOCK}</button>
            <button class="ab dl-btn" title="Download">${SVG_DOWNLOAD}</button>
            <button class="ab danger del-btn" title="Delete">${SVG_DELETE}</button>
          </div>
        </div>
      </div>
    `;
    };

    const applyFocus = () => {
        root.querySelectorAll<HTMLElement>(".item.focus").forEach((el) => el.classList.remove("focus"));
        if (focusIdx < 0 || focusIdx >= items.length) return;
        const el = root.querySelector<HTMLElement>(`.item[data-idx="${focusIdx}"]`);
        if (!el) return;
        el.classList.add("focus");
        el.scrollIntoView({ block: "nearest" });
    };

    const focusAt = (i: number) => {
        if (items.length === 0) {
            focusIdx = -1;
            return;
        }
        focusIdx = Math.max(0, Math.min(items.length - 1, i));
        applyFocus();
    };

    // Click handling. Action buttons run their own action and NEVER fall
    // through to paste. Crucially, any click inside the actions zone
    // (.item-right) that doesn't land precisely on a button is swallowed —
    // otherwise an imprecise click on the gap/padding would paste-and-close
    // the window. Only clicks on the row body (icon/preview) paste.
    root.addEventListener("click", (e) => {
        const target = e.target as HTMLElement;
        const itemEl = target.closest<HTMLElement>(".item");
        if (!itemEl) return;
        const idx = Number(itemEl.dataset.idx);
        const item = items[idx];
        if (!item) return;

        if (target.closest(".del-btn")) {
            actions.onDelete(item);
            return;
        }
        if (target.closest(".pin-btn")) {
            actions.onPinToggle(item);
            return;
        }
        if (target.closest(".copy-btn")) {
            actions.onCopy(item);
            return;
        }
        if (target.closest(".vault-btn")) {
            actions.onMoveToVault(item);
            return;
        }
        if (target.closest(".dl-btn")) {
            actions.onDownload(item);
            return;
        }
        // Left icon = preview button: opens the large preview modal instead
        // of pasting.
        if (target.closest(".preview-btn")) {
            actions.onPreview(item);
            return;
        }
        // Click was somewhere in the action/marker zone but not on a button:
        // ignore it so we never accidentally paste + close.
        if (target.closest(".item-right")) return;

        // Row body click → paste.
        actions.onPaste(item);
    });

    root.addEventListener("contextmenu", (e) => {
        const itemEl = (e.target as HTMLElement).closest<HTMLElement>(".item");
        if (!itemEl) return;
        e.preventDefault();
        const idx = Number(itemEl.dataset.idx);
        const item = items[idx];
        if (!item) return;
        focusAt(idx);
        actions.onContextMenu({ x: e.clientX, y: e.clientY }, item);
    });

    root.addEventListener("mousemove", (e) => {
        const itemEl = (e.target as HTMLElement).closest<HTMLElement>(".item");
        if (!itemEl) return;
        const idx = Number(itemEl.dataset.idx);
        if (!Number.isNaN(idx)) focusAt(idx);
    });

    return {
        setItems(next, opts) {
            items = next;
            filtered = opts?.filtered ?? false;
            if (focusIdx >= items.length) focusIdx = items.length - 1;
            if (focusIdx < 0 && items.length > 0) focusIdx = 0;
            render();
        },
        handleKey(e) {
            if (items.length === 0) return;
            switch (e.key) {
                case "ArrowDown":
                    e.preventDefault();
                    focusAt(focusIdx + 1);
                    break;
                case "ArrowUp":
                    e.preventDefault();
                    focusAt(focusIdx - 1);
                    break;
                case "PageDown":
                    e.preventDefault();
                    focusAt(focusIdx + 5);
                    break;
                case "PageUp":
                    e.preventDefault();
                    focusAt(focusIdx - 5);
                    break;
                case "Home":
                    e.preventDefault();
                    focusAt(0);
                    break;
                case "End":
                    e.preventDefault();
                    focusAt(items.length - 1);
                    break;
                case "Enter":
                    e.preventDefault();
                    if (items[focusIdx]) actions.onPaste(items[focusIdx]);
                    break;
            }
        },
        count() {
            return items.length;
        },
        pasteFocused() {
            if (items[focusIdx]) actions.onPaste(items[focusIdx]);
        },
    };
}

// ---------- helpers ----------

function escapeHtml(s: string): string {
    return s
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

function formatRelative(unixSec: number): string {
    if (!unixSec) return "";
    const diff = Math.max(0, Math.floor(Date.now() / 1000 - unixSec));
    if (diff < 5) return "just now";
    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    if (diff < 604800) return `${Math.floor(diff / 86400)}d ago`;
    const d = new Date(unixSec * 1000);
    return d.toLocaleDateString();
}
