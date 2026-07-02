// Large-scale preview modal, opened from an item's left icon (or the context
// menu). Uses the same scrim + blur chrome as the settings/vault overlays but
// sizes itself to the content: text previews grow with the text (scrolling
// when long), image previews keep the image's aspect ratio at the largest
// size that fits the window.

import type { ClipItem } from "../types";

export interface PreviewChrome {
    scrim: HTMLElement;
    mainContent: HTMLElement;
}

export interface PreviewModalDeps {
    /** Full-resolution image data URL for image items (lazy-loaded). */
    imageFor(id: number): Promise<string>;
    copy(item: ClipItem): void;
    download(item: ClipItem): void;
    chrome: PreviewChrome;
}

export interface PreviewModal {
    open(item: ClipItem): Promise<void>;
    close(): void;
    isOpen(): boolean;
}

const SVG_X = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`;
const SVG_COPY = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`;
const SVG_DOWNLOAD = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`;
const SVG_EYE = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>`;

export function createPreviewModal(root: HTMLElement, deps: PreviewModalDeps): PreviewModal {
    const { scrim, mainContent } = deps.chrome;
    let current: ClipItem | null = null;

    const isOpen = () => root.classList.contains("open");

    const close = () => {
        current = null;
        mainContent.classList.remove("blurred");
        scrim.classList.remove("open");
        root.classList.remove("open");
        root.setAttribute("aria-hidden", "true");
        scrim.onclick = null;
        document.removeEventListener("keydown", escHandler, true);
        // Drop the (possibly large) image data URL from the DOM.
        root.innerHTML = "";
    };

    const escHandler = (e: KeyboardEvent) => {
        if (e.key === "Escape" && isOpen()) {
            e.stopPropagation();
            e.preventDefault();
            close();
        }
    };

    const open = async (item: ClipItem) => {
        current = item;
        const isImage = item.contentType === "image";
        render(item, isImage ? item.imageThumb ?? "" : "");
        mainContent.classList.add("blurred");
        scrim.classList.add("open");
        root.classList.add("open");
        root.setAttribute("aria-hidden", "false");
        scrim.onclick = close;
        document.addEventListener("keydown", escHandler, true);

        if (isImage) {
            // Swap the small thumbnail for the full-resolution image once it
            // arrives (also covers large images stored with no thumbnail).
            try {
                const full = await deps.imageFor(item.id);
                if (current?.id !== item.id || !full) return;
                const box = root.querySelector<HTMLElement>(".preview-image");
                if (box) box.innerHTML = `<img alt="Clipboard image preview" src="${full}"/>`;
            } catch {
                /* keep showing the thumbnail (if any) */
            }
        }
    };

    const render = (item: ClipItem, thumb: string) => {
        const isImage = item.contentType === "image";
        const meta = isImage
            ? `${item.imageW || "?"}×${item.imageH || "?"} · ${formatDate(item.lastUsedAt)}`
            : `${item.textContent?.length ?? 0} chars · ${formatDate(item.lastUsedAt)}`;
        const body = isImage
            ? `<div class="preview-image">${
                  thumb
                      ? `<img alt="Clipboard image preview" src="${thumb}"/>`
                      : `<div class="preview-image-loading">Loading image…</div>`
              }</div>`
            : `<pre class="preview-text">${escapeHtml(item.textContent ?? item.preview ?? "")}</pre>`;
        root.classList.toggle("kind-image", isImage);
        root.classList.toggle("kind-text", !isImage);
        root.innerHTML = `
      <div class="preview-head">
        <span class="preview-head-icon">${SVG_EYE}</span>
        <strong>Preview</strong>
        <span class="preview-meta">${meta}</span>
        <div class="preview-head-actions">
          <button class="iconbtn" data-act="copy" title="Copy to clipboard">${SVG_COPY}</button>
          <button class="iconbtn" data-act="download" title="Download">${SVG_DOWNLOAD}</button>
          <button class="iconbtn" data-act="close" title="Close (Esc)">${SVG_X}</button>
        </div>
      </div>
      <div class="preview-body">${body}</div>
    `;
        root.querySelector<HTMLButtonElement>('[data-act="close"]')!.addEventListener("click", close);
        root.querySelector<HTMLButtonElement>('[data-act="copy"]')!.addEventListener("click", () => {
            if (current) deps.copy(current);
        });
        root.querySelector<HTMLButtonElement>('[data-act="download"]')!.addEventListener("click", () => {
            if (current) deps.download(current);
        });
    };

    return { open, close, isOpen };
}

function escapeHtml(s: string): string {
    return s
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

function formatDate(unixSec: number): string {
    if (!unixSec) return "";
    return new Date(unixSec * 1000).toLocaleString();
}
