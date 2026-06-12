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
import { createVaultPanel } from "./components/vaultPanel";
import type { AppSettings, ClipItem } from "./types";

const $ = <T extends HTMLElement>(sel: string) => document.querySelector<T>(sel)!;

const els = {
    app: $<HTMLElement>("#app"),
    searchInput: $<HTMLInputElement>("#search"),
    list: $<HTMLElement>("#list"),
    vaultModal: $<HTMLElement>("#vault-modal"),
    chips: $<HTMLElement>("#chips"),
    ctxMenu: $<HTMLElement>("#ctx-menu"),
    settings: $<HTMLElement>("#settings-modal"),
    scrim: $<HTMLElement>("#settings-scrim"),
    mainContent: $<HTMLElement>("#main-content"),
    toast: $<HTMLElement>("#toast"),
    openSettings: $<HTMLButtonElement>("#open-settings"),
    openVault: $<HTMLButtonElement>("#open-vault"),
    winClose: $<HTMLButtonElement>("#win-close"),
    winMin: $<HTMLButtonElement>("#win-min"),
    winMax: $<HTMLButtonElement>("#win-max"),
    statusCount: $<HTMLElement>("#status-count"),
};

type Filter = "all" | "text" | "images" | "pinned";

let currentSettings: AppSettings | null = null;
let allItems: ClipItem[] = [];
let activeFilter: Filter = "all";

const search = createSearchBar(els.searchInput);
const confirm = createConfirm(els.app, els.mainContent);
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

const list = createItemList(els.list, {
    onPaste: doPaste,
    onCopy: doCopy,
    onPinToggle: async (item) => {
        await api.pinItem(item.id, !item.pinned);
        await refresh();
    },
    onMoveToVault: doMoveToVault,
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
    onDelete: doDelete,
});

// ---------- Wiring ----------

search.onChange(async () => {
    await refresh();
});
search.onKeyToList((e) => list.handleKey(e));

els.list.addEventListener("keydown", (e) => list.handleKey(e));

els.openSettings.addEventListener("click", () => settings.open());
els.openVault.addEventListener("click", () => vaultPanel.open());

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
        if (!els.ctxMenu.classList.contains("hidden")) return; // ctx menu closes itself
        api.hidePopup();
    } else if ((e.ctrlKey || e.metaKey) && e.key === ",") {
        e.preventDefault();
        settings.open();
    } else if (e.key === "/" && document.activeElement !== els.searchInput) {
        e.preventDefault();
        search.focus();
    }
});

window.addEventListener("blur", () => {
    if (currentSettings?.windowFrame) return; // windowed mode lives in the taskbar
    if (currentSettings?.hideOnBlur === false) return;
    if (settings.isOpen()) return;
    if (vaultPanel.isOpen()) return;
    if (!els.ctxMenu.classList.contains("hidden")) return;
    // Slight delay so click-to-paste flow finishes first.
    setTimeout(() => api.hidePopup().catch(() => undefined), 150);
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
}

/** Apply the active chip filter on top of the backend search result. */
function applyFilter() {
    const visible = allItems.filter((it) => {
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
