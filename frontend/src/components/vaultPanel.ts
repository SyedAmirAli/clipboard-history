import type { VaultItem, VaultSecret, VaultSetupBundle, VaultStatus } from "../types";

const SVG_LOCK = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/></svg>`;
const SVG_UNLOCK = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="5" y="11" width="14" height="10" rx="2"/><path d="M16 11V7a4 4 0 0 0-7.8-1.2"/></svg>`;
const SVG_COPY = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`;
const SVG_DELETE = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 6h18"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/></svg>`;
const SVG_EYE = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>`;
const SVG_EYE_OFF = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m3 3 18 18"/><path d="M10.6 10.6A3 3 0 0 0 12 15a3 3 0 0 0 2.4-1.2"/><path d="M9.5 5.4A9.8 9.8 0 0 1 12 5c6.5 0 10 7 10 7a17.8 17.8 0 0 1-3.2 4.2"/><path d="M6.1 6.9C3.5 8.7 2 12 2 12s3.5 7 10 7a9.7 9.7 0 0 0 4-.8"/></svg>`;
const SVG_EDIT = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20h9"/><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z"/></svg>`;
const SVG_CHECK = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 6 9 17l-5-5"/></svg>`;
const SVG_CLOSE = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12"/></svg>`;

function renderPasswordField(opts: {
  field: string;
  label: string;
  placeholder: string;
  visible: boolean;
  toggleAct: string;
  autocomplete: string;
  disabled?: boolean;
}): string {
  const disabled = opts.disabled ? "disabled" : "";
  return `
    <label class="vault-field">
      <span class="vault-field-label">${opts.label}</span>
      <div class="vault-signin-input-wrap has-trailing">
        <input
          class="vault-signin-input vault-input"
          type="${opts.visible ? "text" : "password"}"
          data-field="${opts.field}"
          placeholder="${opts.placeholder}"
          autocomplete="${opts.autocomplete}"
          ${disabled}
        />
        <button
          type="button"
          class="vault-signin-eye"
          data-act="${opts.toggleAct}"
          aria-label="${opts.visible ? `Hide ${opts.label}` : `Show ${opts.label}`}"
          aria-pressed="${opts.visible}"
          title="${opts.visible ? "Hide" : "Show"}"
          ${disabled}
        >${opts.visible ? SVG_EYE_OFF : SVG_EYE}</button>
      </div>
    </label>
  `;
}

function renderLabeledField(opts: {
  field: string;
  label: string;
  placeholder: string;
  autocomplete?: string;
  inputmode?: string;
  type?: string;
  disabled?: boolean;
}): string {
  const disabled = opts.disabled ? "disabled" : "";
  return `
    <label class="vault-field">
      <span class="vault-field-label">${opts.label}</span>
      <div class="vault-signin-input-wrap">
        <input
          class="vault-signin-input vault-input"
          type="${opts.type ?? "text"}"
          data-field="${opts.field}"
          placeholder="${opts.placeholder}"
          autocomplete="${opts.autocomplete ?? "off"}"
          ${opts.inputmode ? `inputmode="${opts.inputmode}"` : ""}
          ${disabled}
        />
      </div>
    </label>
  `;
}

export interface VaultPanelActions {
  status(): Promise<VaultStatus>;
  startSetup(): Promise<VaultSetupBundle>;
  confirmSetup(pin: string, confirm: string, code: string): Promise<VaultStatus>;
  unlockPIN(pin: string): Promise<VaultStatus>;
  unlockCode(code: string): Promise<VaultStatus>;
  resetPIN(code: string, pin: string, confirm: string): Promise<VaultStatus>;
  lock(): Promise<void>;
  list(): Promise<VaultItem[]>;
  copy(id: number): Promise<void>;
  reveal(id: number): Promise<VaultSecret>;
  updateTitle(id: number, title: string): Promise<void>;
  delete(id: number): Promise<void>;
  flash(msg: string): void;
  chrome: {
    scrim: HTMLElement;
    mainContent: HTMLElement;
  };
}

export interface VaultPanel {
  open(): Promise<void>;
  close(): void;
  isOpen(): boolean;
  refresh(): Promise<void>;
  ensureUnlocked(): Promise<boolean>;
}

type Mode = "locked" | "setup" | "reset" | "unlocked";

export function createVaultPanel(root: HTMLElement, actions: VaultPanelActions): VaultPanel {
  let status: VaultStatus = { configured: false, unlocked: false, failedAttempts: 0 };
  let setup: VaultSetupBundle | null = null;
  let items: VaultItem[] = [];
  let mode: Mode = "locked";
  let authCodeVisible = false;
  let pinVisible = false;
  let setupPinVisible = false;
  let setupConfirmVisible = false;
  let resetPinVisible = false;
  let resetConfirmVisible = false;
  let editingTitleID: number | null = null;
  let unlockResolvers: Array<(ok: boolean) => void> = [];
  const revealed = new Map<number, string>();
  const { scrim, mainContent } = actions.chrome;

  const isOpen = () => root.classList.contains("open");

  async function open() {
    await refresh();
    mainContent.classList.add("blurred");
    scrim.classList.add("open");
    root.classList.add("open");
    root.setAttribute("aria-hidden", "false");
    scrim.onclick = close;
    document.addEventListener("keydown", escHandler, true);
    requestAnimationFrame(() => {
      root.querySelector<HTMLInputElement>(".vault-input")?.focus();
    });
  }

  function close() {
    mainContent.classList.remove("blurred");
    scrim.classList.remove("open");
    root.classList.remove("open");
    root.setAttribute("aria-hidden", "true");
    scrim.onclick = null;
    document.removeEventListener("keydown", escHandler, true);
    revealed.clear();
    pinVisible = false;
    setupPinVisible = false;
    setupConfirmVisible = false;
    resetPinVisible = false;
    resetConfirmVisible = false;
    authCodeVisible = false;
    editingTitleID = null;
    resolveUnlockWaiters(false);
  }

  function escHandler(e: KeyboardEvent) {
    if (e.key !== "Escape" || !isOpen()) return;
    e.preventDefault();
    e.stopPropagation();
    if (editingTitleID !== null) {
      editingTitleID = null;
      renderUnlocked();
      return;
    }
    close();
  }

  async function refresh() {
    try {
      status = await actions.status();
      mode = status.configured ? (status.unlocked ? "unlocked" : "locked") : "setup";
      if (status.unlocked) {
        items = await actions.list();
        if (editingTitleID !== null && !items.some((item) => item.id === editingTitleID)) {
          editingTitleID = null;
        }
      } else {
        items = [];
        revealed.clear();
        editingTitleID = null;
      }
      if (mode === "setup" && !setup) setup = await actions.startSetup();
      render();
    } catch (err) {
      root.innerHTML = `<div class="vault-head"><span>${SVG_LOCK}</span><strong>Private Vault</strong></div><div class="vault-error">${escapeHtml(String(err))}</div>`;
    }
  }

  async function ensureUnlocked(): Promise<boolean> {
    await refresh();
    if (status.unlocked) return true;
    await open();
    return new Promise((resolve) => {
      unlockResolvers.push(resolve);
    });
  }

  function resolveUnlockWaiters(ok: boolean) {
    if (unlockResolvers.length === 0) return;
    const resolvers = unlockResolvers;
    unlockResolvers = [];
    resolvers.forEach((resolve) => resolve(ok));
  }

  function completeUnlockFlow() {
    if (unlockResolvers.length > 0) {
      resolveUnlockWaiters(true);
      close();
    }
  }

  function render() {
    root.classList.toggle("vault-unlocked", status.unlocked);
    root.classList.toggle("vault-mode-locked", mode === "locked");
    if (mode === "setup") renderSetup();
    if (mode === "locked") renderLocked();
    if (mode === "reset") renderReset();
    if (mode === "unlocked") renderUnlocked();
  }

  function renderSetup() {
    const qr = setup?.qrCodeSvg ?? "";
    root.innerHTML = `
      <div class="vault-head"><span>${SVG_LOCK}</span><strong>Private Vault</strong></div>
      <div class="vault-setup">
        <div class="vault-setup-content">
          <div class="vault-qr-col">
            <div class="vault-qr">${qr}</div>
            <div class="vault-key-wrap">
              <div class="vault-key">${escapeHtml(setup?.manualKey ?? "")}</div>
              <button type="button" class="vault-key-copy" data-act="copy-manual-key" title="Copy recovery key" aria-label="Copy recovery key">${SVG_COPY}</button>
            </div>
          </div>
          <div class="vault-form-col">
            <div class="vault-instructions">
              <strong>Set up authenticator access</strong>
              <p>Scan the QR code in Google Authenticator, or enter the manual key. Then create a PIN/password and enter the current 6-digit code.</p>
            </div>
            ${renderPasswordField({
              field: "pin",
              label: "PIN or password",
              placeholder: "Enter PIN or password",
              visible: setupPinVisible,
              toggleAct: "toggle-setup-pin",
              autocomplete: "new-password",
            })}
            ${renderPasswordField({
              field: "confirm",
              label: "Confirm PIN or password",
              placeholder: "Re-enter PIN or password",
              visible: setupConfirmVisible,
              toggleAct: "toggle-setup-confirm",
              autocomplete: "new-password",
            })}
            ${renderLabeledField({
              field: "code",
              label: "Authenticator code",
              placeholder: "6-digit code",
              inputmode: "numeric",
              autocomplete: "one-time-code",
            })}
            <button class="s-btn primary vault-full" data-act="confirm-setup">Create Vault</button>
          </div>
        </div>
      </div>
    `;
  }

  function renderLocked() {
    const lockout = status.lockedUntil && status.lockedUntil * 1000 > Date.now();
    const lockIcon = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>`;
    const keyIcon = `<svg class="vault-signin-input-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>`;
    const phoneIcon = `<svg class="vault-signin-input-icon" viewBox="0 0 24 24" aria-hidden="true"><rect x="5" y="2" width="14" height="20" rx="2"/><line x1="12" y1="18" x2="12.01" y2="18"/></svg>`;
    const unlockBtnIcon = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 9.9-1"/></svg>`;
    const resetIcon = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>`;
    const phoneBtnIcon = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="2" width="14" height="20" rx="2"/><line x1="12" y1="18" x2="12.01" y2="18"/></svg>`;
    const shieldIcon = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>`;

    const authCode = authCodeVisible
      ? `
        <div class="vault-signin-auth-wrap">
          <div class="vault-signin-input-wrap">
            ${phoneIcon}
            <input class="vault-signin-input vault-input" data-field="code" placeholder="Authenticator code" inputmode="numeric" autocomplete="one-time-code" ${lockout ? "disabled" : ""}/>
          </div>
          <button class="vault-signin-primary" data-act="unlock-code" ${lockout ? "disabled" : ""}>${unlockBtnIcon}<span>Unlock with Authenticator</span></button>
        </div>
      `
      : "";

    root.innerHTML = `
      <button type="button" class="vault-signin-close" data-act="close-panel" aria-label="Close" title="Close">${SVG_CLOSE}</button>
      <div class="vault-signin-card" role="document">
        <div class="vault-signin-header">
          <div class="vault-signin-icon" aria-hidden="true">${lockIcon}</div>
          <div class="vault-signin-titles">
            <h1 class="vault-signin-title">Private Vault</h1>
            <p class="vault-signin-subtitle">Enter your credentials to continue</p>
          </div>
        </div>

        <div class="vault-signin-input-wrap has-trailing">
          ${keyIcon}
          <input class="vault-signin-input vault-input" type="${pinVisible ? "text" : "password"}" data-field="pin" placeholder="PIN or password" autocomplete="current-password" ${lockout ? "disabled" : ""}/>
          <button type="button" class="vault-signin-eye" data-act="toggle-pin" aria-label="${pinVisible ? "Hide PIN" : "Show PIN"}" aria-pressed="${pinVisible}" title="${pinVisible ? "Hide" : "Show"}" ${lockout ? "disabled" : ""}>${pinVisible ? SVG_EYE_OFF : SVG_EYE}</button>
        </div>

        <button class="vault-signin-primary" data-act="unlock-pin" ${lockout ? "disabled" : ""}>${unlockBtnIcon}<span>Unlock vault</span></button>

        <div class="vault-signin-grid">
          <button class="vault-signin-ghost danger" data-act="reset">${resetIcon}<span>Reset PIN</span></button>
          <button class="vault-signin-ghost" data-act="show-auth">${phoneBtnIcon}<span>Auth code</span></button>
        </div>

        ${authCode}

        <hr class="vault-signin-divider"/>

        <div class="vault-signin-footer">${shieldIcon}<span>End-to-end encrypted</span></div>
      </div>
    `;
  }

  function renderReset() {
    root.innerHTML = `
      <div class="vault-head"><span>${SVG_LOCK}</span><strong>Reset PIN</strong><button class="s-btn" data-act="cancel-reset">Cancel</button></div>
      <div class="vault-reset">
        ${renderLabeledField({
          field: "code",
          label: "Authenticator code",
          placeholder: "6-digit code",
          inputmode: "numeric",
          autocomplete: "one-time-code",
        })}
        ${renderPasswordField({
          field: "pin",
          label: "New PIN or password",
          placeholder: "Enter new PIN or password",
          visible: resetPinVisible,
          toggleAct: "toggle-reset-pin",
          autocomplete: "new-password",
        })}
        ${renderPasswordField({
          field: "confirm",
          label: "Confirm new PIN or password",
          placeholder: "Re-enter new PIN or password",
          visible: resetConfirmVisible,
          toggleAct: "toggle-reset-confirm",
          autocomplete: "new-password",
        })}
        <button class="s-btn primary vault-full" data-act="reset-pin">Save New PIN</button>
      </div>
    `;
  }

  function renderUnlocked() {
    const body = items.length
      ? items.map(renderVaultItem).join("")
      : `<div class="vault-empty">No private items</div>`;
    root.innerHTML = `
      <div class="vault-head">
        <span>${SVG_UNLOCK}</span>
        <strong>Private Vault</strong>
        <div class="vault-head-actions">
          <button class="s-btn" data-act="lock">Lock</button>
          <button class="s-btn" data-act="close-panel">Close</button>
        </div>
      </div>
      <div class="vault-list">${body}</div>
    `;
  }

  function renderVaultItem(item: VaultItem) {
    const meta = item.contentType === "image" ? imageMeta(item) : "Private text item";
    const title = item.title?.trim() || meta;
    const revealedText = revealed.get(item.id);
    const canReveal = item.contentType === "text";
    const editingTitle = editingTitleID === item.id;
    const titleView = editingTitle
      ? `<input class="vault-title-input" data-field="vault-title" value="${escapeHtml(title)}" maxlength="120" aria-label="Vault item title"/>`
      : `<div class="vault-preview">${escapeHtml(title)}</div>`;
    const actionButtons = editingTitle
      ? `
        <button class="ab save-title" title="Save title" aria-label="Save title">${SVG_CHECK}</button>
        <button class="ab cancel-title" title="Cancel title" aria-label="Cancel title">${SVG_CLOSE}</button>
      `
      : `
        <button class="ab reveal-vault" title="${revealedText ? "Hide" : "Show"}" ${canReveal ? "" : "disabled"}>${revealedText ? SVG_EYE_OFF : SVG_EYE}</button>
        <button class="ab edit-title" title="Edit title" aria-label="Edit title">${SVG_EDIT}</button>
        <button class="ab copy-vault" title="Copy">${SVG_COPY}</button>
        <button class="ab danger delete-vault" title="Delete">${SVG_DELETE}</button>
      `;
    return `
      <div class="vault-item ${editingTitle ? "editing-title" : ""}" data-id="${item.id}">
        <div class="vault-icon">${SVG_LOCK}</div>
        <div class="vault-item-main">
          <div class="vault-secret ${revealedText ? "revealed" : ""}">${escapeHtml(revealedText ?? "••••••••••••")}</div>
          ${titleView}
        </div>
        ${actionButtons}
      </div>
    `;
  }

  async function submitConfirmSetup() {
    await actions.confirmSetup(value("pin"), value("confirm"), value("code"));
    setup = null;
    await refresh();
    actions.flash("Private vault created");
    completeUnlockFlow();
  }

  async function submitUnlockPin() {
    await actions.unlockPIN(value("pin"));
    await refresh();
    completeUnlockFlow();
  }

  async function submitUnlockCode() {
    await actions.unlockCode(value("code"));
    authCodeVisible = false;
    await refresh();
    completeUnlockFlow();
  }

  async function submitResetPin() {
    await actions.resetPIN(value("code"), value("pin"), value("confirm"));
    authCodeVisible = false;
    await refresh();
    actions.flash("PIN/password reset");
    completeUnlockFlow();
  }

  root.addEventListener("click", async (e) => {
    const target = e.target as HTMLElement;
    const act = target.closest<HTMLElement>("[data-act]")?.dataset.act;
    try {
      if (act === "close-panel") {
        close();
      } else if (act === "confirm-setup") {
        await submitConfirmSetup();
      } else if (act === "unlock-pin") {
        await submitUnlockPin();
      } else if (act === "unlock-code") {
        await submitUnlockCode();
      } else if (act === "reset") {
        mode = "reset";
        render();
      } else if (act === "cancel-reset") {
        mode = "locked";
        render();
      } else if (act === "show-auth") {
        authCodeVisible = true;
        renderLocked();
        requestAnimationFrame(() => root.querySelector<HTMLInputElement>('[data-field="code"]')?.focus());
      } else if (act === "toggle-pin") {
        togglePasswordVisibility("pin", () => {
          pinVisible = !pinVisible;
        }, renderLocked);
      } else if (act === "toggle-setup-pin") {
        togglePasswordVisibility("pin", () => {
          setupPinVisible = !setupPinVisible;
        }, renderSetup);
      } else if (act === "toggle-setup-confirm") {
        togglePasswordVisibility("confirm", () => {
          setupConfirmVisible = !setupConfirmVisible;
        }, renderSetup);
      } else if (act === "toggle-reset-pin") {
        togglePasswordVisibility("pin", () => {
          resetPinVisible = !resetPinVisible;
        }, renderReset);
      } else if (act === "toggle-reset-confirm") {
        togglePasswordVisibility("confirm", () => {
          resetConfirmVisible = !resetConfirmVisible;
        }, renderReset);
      } else if (act === "copy-manual-key") {
        const text = setup?.secret ?? setup?.manualKey?.replace(/\s+/g, "") ?? "";
        if (!text) return;
        await copyTextToClipboard(text);
        actions.flash("Recovery key copied");
      } else if (act === "reset-pin") {
        await submitResetPin();
      } else if (act === "lock") {
        await actions.lock();
        revealed.clear();
        await refresh();
      }

      const itemEl = target.closest<HTMLElement>(".vault-item");
      if (itemEl) {
        const id = Number(itemEl.dataset.id);
        if (target.closest(".reveal-vault")) {
          if (revealed.has(id)) {
            revealed.delete(id);
          } else {
            const secret = await actions.reveal(id);
            if (secret.contentType !== "text" || !secret.text) {
              actions.flash("Only text vault items can be revealed");
            } else {
              revealed.set(id, secret.text);
            }
          }
          renderUnlocked();
        } else if (target.closest(".edit-title")) {
          editingTitleID = id;
          renderUnlocked();
          requestAnimationFrame(() => {
            const input = root.querySelector<HTMLInputElement>(`.vault-item[data-id="${id}"] .vault-title-input`);
            input?.focus();
            input?.select();
          });
        } else if (target.closest(".save-title")) {
          await saveTitle(id);
        } else if (target.closest(".cancel-title")) {
          editingTitleID = null;
          renderUnlocked();
        } else if (target.closest(".copy-vault")) {
          actions.flash("Copying private item...");
          await actions.copy(id);
          actions.flash("Copied private item");
        } else if (target.closest(".delete-vault")) {
          await actions.delete(id);
          revealed.delete(id);
          await refresh();
        }
      }
    } catch (err) {
      actions.flash(String(err));
    }
  });

  root.addEventListener("keydown", (e) => {
    if (e.key !== "Enter" || !isOpen()) return;
    const target = e.target;
    if (!(target instanceof HTMLInputElement)) return;
    const field = target.dataset.field;
    if (!field) return;

    e.preventDefault();
    e.stopPropagation();

    void (async () => {
      try {
        if (mode === "locked") {
          if (field === "pin") await submitUnlockPin();
          else if (field === "code" && authCodeVisible) await submitUnlockCode();
        } else if (mode === "setup") {
          await submitConfirmSetup();
        } else if (mode === "reset") {
          await submitResetPin();
        } else if (mode === "unlocked" && field === "vault-title") {
          const itemEl = target.closest<HTMLElement>(".vault-item");
          const id = Number(itemEl?.dataset.id);
          if (Number.isFinite(id)) await saveTitle(id);
        }
      } catch (err) {
        actions.flash(String(err));
      }
    })();
  });

  function value(field: string): string {
    return root.querySelector<HTMLInputElement>(`[data-field="${field}"]`)?.value ?? "";
  }

  async function saveTitle(id: number) {
    const input = root.querySelector<HTMLInputElement>(`.vault-item[data-id="${id}"] .vault-title-input`);
    if (!input) return;
    await actions.updateTitle(id, input.value);
    editingTitleID = null;
    actions.flash("Vault title saved");
    await refresh();
  }

  function collectInputs(): Record<string, string> {
    const out: Record<string, string> = {};
    root.querySelectorAll<HTMLInputElement>("[data-field]").forEach((el) => {
      const field = el.dataset.field;
      if (field) out[field] = el.value;
    });
    return out;
  }

  function togglePasswordVisibility(
    focusField: string,
    toggle: () => void,
    rerender: () => void
  ) {
    const saved = collectInputs();
    const input = root.querySelector<HTMLInputElement>(`[data-field="${focusField}"]`);
    const caret = input?.selectionStart ?? (saved[focusField]?.length ?? 0);
    toggle();
    rerender();
    requestAnimationFrame(() => {
      for (const [field, val] of Object.entries(saved)) {
        const el = root.querySelector<HTMLInputElement>(`[data-field="${field}"]`);
        if (el) el.value = val;
      }
      const next = root.querySelector<HTMLInputElement>(`[data-field="${focusField}"]`);
      if (!next) return;
      next.focus();
      try {
        next.setSelectionRange(caret, caret);
      } catch {}
    });
  }

  return { open, close, isOpen, refresh, ensureUnlocked };
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function imageMeta(item: VaultItem): string {
  if (item.imageW && item.imageH) return `Private image ${item.imageW}×${item.imageH}`;
  return "Private image item";
}

async function copyTextToClipboard(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const ta = document.createElement("textarea");
  ta.value = text;
  ta.setAttribute("readonly", "");
  ta.style.position = "fixed";
  ta.style.top = "-1000px";
  ta.style.opacity = "0";
  document.body.appendChild(ta);
  ta.select();
  try {
    if (!document.execCommand("copy")) {
      throw new Error('execCommand("copy") returned false');
    }
  } finally {
    document.body.removeChild(ta);
  }
}
