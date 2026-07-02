// Entry point. Wires together titlebar → search → chips → list → context
// menu → settings, and subscribes to backend events so the UI updates
// whenever a new clipboard entry arrives.

import "./styles/main.css";
import "./styles/components.css";

import { api, onEvent } from "./api";
import { createSearchBar } from "./components/searchBar";
import { createItemList } from "./components/itemList";
import { createContextMenu } from "./components/itemContextMenu";
import { createSettingsModal } from "./components/settingsModal";
import { createConfirm } from "./components/confirmModal";
import { createPrompt } from "./components/promptModal";
import { createVaultPanel } from "./components/vaultPanel";
import { createPreviewModal } from "./components/previewModal";
import type { AppSettings, ClipItem } from "./types";

const $ = <T extends HTMLElement>(sel: string) => document.querySelector<T>(sel)!;

const els = {
    app: $<HTMLElement>("#app"),
    searchInput: $<HTMLInputElement>("#search"),
    list: $<HTMLElement>("#list"),
    vaultModal: $<HTMLElement>("#vault-modal"),
    previewModal: $<HTMLElement>("#preview-modal"),
    chips: $<HTMLElement>("#chips"),
    ctxMenu: $<HTMLElement>("#ctx-menu"),
    settings: $<HTMLElement>("#settings-modal"),
    scrim: $<HTMLElement>("#settings-scrim"),
    mainContent: $<HTMLElement>("#main-content"),
    toast: $<HTMLElement>("#toast"),
    openSettings: $<HTMLButtonElement>("#open-settings"),
    openVault: $<HTMLButtonElement>("#open-vault"),
    refreshList: $<HTMLButtonElement>("#refresh-list"),
    exportZip: $<HTMLButtonElement>("#export-zip"),
    winClose: $<HTMLButtonElement>("#win-close"),
    winMin: $<HTMLButtonElement>("#win-min"),
    winMax: $<HTMLButtonElement>("#win-max"),
    statusCount: $<HTMLElement>("#status-count"),
    chipsMeta: $<HTMLElement>("#chips-meta"),
};

type Filter = "all" | "text" | "images" | "pinned";

let currentSettings: AppSettings | null = null;
let allItems: ClipItem[] = [];
let activeFilter: Filter = "all";

const search = createSearchBar(els.searchInput);
const confirm = createConfirm(els.app, els.mainContent);
const prompt = createPrompt(els.app, els.mainContent);

// Vault export: ask for the vault PIN (verified server-side; also becomes
// the ZIP password), then a save dialog picks the destination.
async function doExportVault() {
    const pin = await prompt.show({
        title: "Export Private Vault",
        message: "Enter your vault PIN/password. The ZIP will be encrypted with it — you'll need the same password to extract.",
        placeholder: "Vault PIN or password",
        confirmLabel: "Export",
        password: true,
    });
    if (!pin) return;
    nativeDialogOpen = true;
    try {
        const path = await api.exportVaultZip(pin);
        if (path) flash(`Vault exported to ${path}`);
    } catch (err) {
        flash("Vault export failed: " + String(err));
    } finally {
        nativeDialogOpen = false;
    }
}
const settings = createSettingsModal(els.settings, {
    load: () => api.getSettings(),
    save: async (s) => {
        const saved = await api.updateSettings(s);
        currentSettings = saved;
        settings.applyTheme(saved.theme);
        applyFilter(); // pinned placement may have changed
        return saved;
    },
    clearAll: async (keep) => {
        await api.clearAll(keep);
        await refresh();
    },
    systemInfo: () => api.systemInfo(),
    chooseDownloadDir: () => api.chooseDownloadDir(),
    chooseBackupDir: () => api.chooseBackupDir(),
    backupNow: async () => {
        try {
            const path = await api.runBackupNow();
            if (path) flash(`Backup saved to ${path}`);
        } catch (err) {
            flash("Backup failed: " + String(err));
        }
    },
    exportVault: doExportVault,
    chrome: { scrim: els.scrim, mainContent: els.mainContent },
    confirm,
});

// Shared "use this item" (paste + close), "copy" (clipboard only, stay open),
// and confirmed-delete handlers, reused by both the row and the context menu.
async function doPaste(item: ClipItem) {
    try {
        await api.pasteItem(item.id);
        flash("Copied to clipboard");
    } catch (err) {
        flash("Paste failed: " + String(err));
    }
}

async function doCopy(item: ClipItem) {
    try {
        await api.copyItem(item.id);
        flash("Copied to clipboard");
    } catch (err) {
        flash("Copy failed: " + String(err));
    }
}

async function doDelete(item: ClipItem) {
    const snippet = (item.preview || "").trim().slice(0, 80);
    const ok = await confirm.show({
        title: "Delete this item?",
        message: snippet ? `“${snippet}” will be permanently removed.` : "This item will be permanently removed.",
        confirmLabel: "Delete",
        cancelLabel: "Cancel",
        danger: true,
    });
    if (!ok) return;
    try {
        await api.deleteItem(item.id);
        await refresh();
        flash("Deleted");
    } catch (err) {
        flash("Delete failed: " + String(err));
    }
}

// While a native file dialog is up, the popup loses focus — suppress the
// hide-on-blur behaviour so the window doesn't vanish under the dialog.
let nativeDialogOpen = false;

async function doDownload(item: ClipItem) {
    nativeDialogOpen = true;
    try {
        const path = await api.downloadItem(item.id);
        if (path) flash(`Saved to ${path}`);
    } catch (err) {
        flash("Download failed: " + String(err));
    } finally {
        nativeDialogOpen = false;
    }
}

async function doExportZip() {
    nativeDialogOpen = true;
    try {
        const path = await api.exportAllZip();
        if (path) flash(`Exported to ${path}`);
    } catch (err) {
        flash("Export failed: " + String(err));
    } finally {
        nativeDialogOpen = false;
    }
}

function doPreview(item: ClipItem) {
    void previewModal.open(item);
}

async function doMoveToVault(item: ClipItem) {
    const unlocked = await vaultPanel.ensureUnlocked();
    if (!unlocked) return;
    try {
        await api.moveItemToVault(item.id);
        await refresh();
        await vaultPanel.refresh();
        flash("Moved to Private Vault");
    } catch (err) {
        flash("Move failed: " + String(err));
    }
}

const vaultPanel = createVaultPanel(els.vaultModal, {
    status: () => api.vaultStatus(),
    startSetup: () => api.startVaultSetup(),
    confirmSetup: (pin, confirmPIN, code) => api.confirmVaultSetup(pin, confirmPIN, code),
    unlockPIN: (pin) => api.unlockVaultWithPIN(pin),
    unlockCode: (code) => api.unlockVaultWithCode(code),
    resetPIN: (code, pin, confirmPIN) => api.resetVaultPIN(code, pin, confirmPIN),
    lock: () => api.lockVault(),
    list: () => api.listVaultItems(),
    copy: (id) => api.copyVaultItem(id),
    reveal: (id) => api.revealVaultItem(id),
    updateTitle: (id, title) => api.updateVaultItemTitle(id, title),
    delete: (id) => api.deleteVaultItem(id),
    flash,
    chrome: { scrim: els.scrim, mainContent: els.mainContent },
});

const previewModal = createPreviewModal(els.previewModal, {
    imageFor: (id) => api.getItemImage(id),
    copy: doCopy,
    download: doDownload,
    chrome: { scrim: els.scrim, mainContent: els.mainContent },
});

const list = createItemList(els.list, {
    onPaste: doPaste,
    onCopy: doCopy,
    onPinToggle: async (item) => {
        await api.pinItem(item.id, !item.pinned);
        await refresh();
    },
    onMoveToVault: doMoveToVault,
    onDownload: doDownload,
    onPreview: doPreview,
    onDelete: doDelete,
    onContextMenu: (at, item) => ctxMenu.open(at, item),
});

const ctxMenu = createContextMenu(els.ctxMenu, {
    onCopy: doCopy,
    onPinToggle: async (item) => {
        await api.pinItem(item.id, !item.pinned);
        await refresh();
    },
    onMoveToVault: doMoveToVault,
    onPreview: doPreview,
    onDownload: doDownload,
    onDelete: doDelete,
});

// ---------- Wiring ----------

search.onChange(async () => {
    await refresh();
});
search.onKeyToList((e) => list.handleKey(e));

els.list.addEventListener("keydown", (e) => list.handleKey(e));

els.openSettings.addEventListener("click", () => settings.open());
els.exportZip.addEventListener("click", () => void doExportZip());
els.openVault.addEventListener("click", () => vaultPanel.open());
// The clipboard watcher runs in the separate clipd-watch process, so manual refresh re-reads
// the DB on demand (the list also auto-refreshes whenever the popup is shown).
els.refreshList.addEventListener("click", () => {
    void refresh();
    els.refreshList.classList.add("spin");
    window.setTimeout(() => els.refreshList.classList.remove("spin"), 500);
});

// Tiny credit bar: open developer portfolio / GitHub repo in system browser.
// Falls back to copying the URL onto the clipboard if the browser can't be
// launched, and surfaces the result as a toast.
document.querySelectorAll<HTMLElement>(".credit-link").forEach((el) => {
    el.addEventListener("click", async (e) => {
        const trigger = e.currentTarget as HTMLElement;
        const url = trigger.dataset.url;
        if (!url) return;
        const label = trigger.dataset.label ?? "Link";
        try {
            const result = await api.openExternal(url);
            if (result === "copied") {
                flash(`Couldn't open browser · ${label} URL copied — paste it in your browser`);
            } else if (result === "failed") {
                flash(`Couldn't open ${label}`);
            }
        } catch (err) {
            flash(`Couldn't open ${label}: ${String(err)}`);
        }
    });
});

// Filter chips
els.chips.querySelectorAll<HTMLElement>(".chip").forEach((chip) => {
    chip.addEventListener("click", () => {
        els.chips.querySelectorAll(".chip").forEach((c) => c.classList.remove("active"));
        chip.classList.add("active");
        activeFilter = (chip.dataset.filter as Filter) ?? "all";
        applyFilter();
    });
});

// Window controls (macOS traffic lights)
els.winClose.addEventListener("click", () => api.hidePopup().catch(() => undefined));
els.winMin.addEventListener("click", () => api.minimizeWindow());
els.winMax.addEventListener("click", () => api.toggleMaximizeWindow());

document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
        if (confirm.isOpen()) return; // confirm dialog handles its own close
        if (settings.isOpen()) return; // overlay handles its own close
        if (vaultPanel.isOpen()) return; // overlay handles its own close
        if (previewModal.isOpen()) return; // overlay handles its own close
        if (!els.ctxMenu.classList.contains("hidden")) return; // ctx menu closes itself
        api.hidePopup();
    } else if ((e.ctrlKey || e.metaKey) && e.key === ",") {
        e.preventDefault();
        settings.open();
    } else if (e.key === "/" && document.activeElement !== els.searchInput) {
        e.preventDefault();
        search.focus();
    } else if (["ArrowDown", "ArrowUp", "PageDown", "PageUp", "Home", "End", "Enter"].includes(e.key)) {
        // Arrow-key navigation should work no matter where focus sits (on
        // show, focus often lands on <body>, not the search input). The
        // search input forwards its own keys via searchBar; other inputs,
        // textareas, and buttons keep their native key handling, and any
        // open overlay handles its own keys.
        if (settings.isOpen() || vaultPanel.isOpen() || previewModal.isOpen()) return;
        if (confirm.isOpen() || prompt.isOpen()) return;
        const t = document.activeElement;
        if (t === els.searchInput) return; // already forwarded by searchBar
        if (t === els.list || els.list.contains(t)) return; // list's own listener handles it
        if (
            t instanceof HTMLInputElement ||
            t instanceof HTMLTextAreaElement ||
            t instanceof HTMLButtonElement ||
            (t instanceof HTMLElement && t.isContentEditable)
        )
            return;
        list.handleKey(e);
    }
});

// Track whether the popup has actually held keyboard focus. On Wayland a
// frameless always-on-top popup is frequently shown WITHOUT the compositor
// granting it focus, which fires a spurious `blur` immediately after it
// appears — the old handler then hid the window ~150ms later, so it looked
// like it "opened and instantly closed". We only auto-hide on blur once the
// window has genuinely been focused, so that phantom blur is ignored.
let popupHadFocus = false;
window.addEventListener("focus", () => {
    popupHadFocus = true;
});

window.addEventListener("blur", () => {
    if (currentSettings?.windowFrame) return; // windowed mode lives in the taskbar
    if (currentSettings?.hideOnBlur === false) return;
    if (settings.isOpen()) return;
    if (vaultPanel.isOpen()) return;
    if (previewModal.isOpen()) return;
    if (nativeDialogOpen) return;
    if (!els.ctxMenu.classList.contains("hidden")) return;
    if (!popupHadFocus) return; // never focused (Wayland) → this blur is spurious
    // Slight delay so click-to-paste flow finishes first; re-check focus in
    // case the window regained it during the grace window.
    setTimeout(() => {
        if (document.hasFocus()) return;
        popupHadFocus = false; // require a fresh focus before the next auto-hide
        api.hidePopup().catch(() => undefined);
    }, 150);
});

// React to backend pushes.
onEvent("clipboard:new-item", async () => {
    await refresh();
});
onEvent("clipboard:cleared", async () => {
    await refresh();
});
onEvent("vault:changed", async () => {
    if (vaultPanel.isOpen()) await vaultPanel.refresh();
});

// ---------- Rendering ----------

async function refresh() {
    try {
        allItems = await api.listItems(search.value(), currentSettings?.maxItems ?? 200);
        applyFilter();
    } catch (err) {
        console.error("list items failed", err);
    }
    void updateStats();
}

/** Chips-row status: total items in the DB, plus process memory when the
 *  "Show memory usage" setting is on (opt-in — polling /proc costs power). */
async function updateStats() {
    const showMem = currentSettings?.showMemory ?? false;
    try {
        const s = await api.runtimeStats(showMem);
        const mb = (s.rssBytes / (1024 * 1024)).toFixed(1);
        els.chipsMeta.innerHTML = showMem
            ? `RAM: <strong>${mb}MB</strong> · <strong>${s.totalItems}</strong> items`
            : `<strong>${s.totalItems}</strong> items`;
    } catch {
        /* bindings unavailable (pure-Vite dev) — leave blank */
    }
}
// Periodic refresh only matters for the live RAM figure; the item count
// updates with every list refresh anyway.
window.setInterval(() => {
    if (currentSettings?.showMemory) void updateStats();
}, 5000);

/** Apply the active chip filter on top of the backend search result. */
function applyFilter() {
    // When "Pinned items on top" is off, pinned entries live exclusively in
    // the Pinned tab — hide them from All/Text/Images.
    const pinnedOnTop = currentSettings?.pinnedOnTop ?? true;
    const visible = allItems.filter((it) => {
        if (!pinnedOnTop && it.pinned && activeFilter !== "pinned") return false;
        switch (activeFilter) {
            case "text":
                return it.contentType === "text";
            case "images":
                return it.contentType === "image";
            case "pinned":
                return it.pinned;
            default:
                return true;
        }
    });
    const isFiltered = activeFilter !== "all" || search.value().trim() !== "";
    list.setItems(visible, { filtered: isFiltered });
    setCount(visible);
}

function setCount(items: ClipItem[]) {
    const total = items.length;
    const pinned = items.filter((i) => i.pinned).length;
    els.statusCount.textContent =
        total === 0 ? "No items" : pinned > 0 ? `${total} items · ${pinned} pinned` : `${total} items`;
}

let toastTimer: number | undefined;
function flash(msg: string) {
    els.toast.textContent = msg;
    els.toast.classList.remove("hidden");
    requestAnimationFrame(() => els.toast.classList.add("show"));
    if (toastTimer) window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => {
        els.toast.classList.remove("show");
        setTimeout(() => els.toast.classList.add("hidden"), 200);
    }, 1500);
}

// ---------- Boot ----------

async function boot() {
    try {
        currentSettings = await api.getSettings();
        settings.applyTheme(currentSettings.theme);
    } catch (err) {
        console.warn("initial settings load failed (running outside Wails?)", err);
    }
    await refresh();
    search.focus();
}

boot();
