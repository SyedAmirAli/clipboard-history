// Thin wrapper around the Wails-generated bindings (frontend/wailsjs/go/service/Service.*).
// We re-export typed functions so component code never has to look at
// the generated, untyped JS files directly. If the wails dev server
// isn't running and the bindings file is missing, we fall back to no-op
// stubs that surface a friendly error in the console.

import type { ClipItem, AppSettings, SystemInfo, VaultItem, VaultSecret, VaultSetupBundle, VaultStatus } from './types';

type WailsRuntime = {
  EventsOn(name: string, cb: (...data: unknown[]) => void): () => void;
  EventsOff(name: string): void;
  WindowHide(): void;
  WindowShow(): void;
  WindowMinimise(): void;
  WindowToggleMaximise(): void;
  BrowserOpenURL(url: string): void;
  Quit(): void;
};

declare global {
  interface Window {
    go?: {
      service?: {
        Service?: {
          ListItems(filter: string, limit: number): Promise<ClipItem[]>;
          PasteItem(id: number): Promise<void>;
          CopyItem(id: number): Promise<void>;
          PinItem(id: number, pinned: boolean): Promise<void>;
          DeleteItem(id: number): Promise<void>;
          ClearAll(keepPinned: boolean): Promise<void>;
          VaultStatus(): Promise<VaultStatus>;
          StartVaultSetup(): Promise<VaultSetupBundle>;
          ConfirmVaultSetup(pin: string, confirm: string, code: string): Promise<VaultStatus>;
          UnlockVaultWithPIN(pin: string): Promise<VaultStatus>;
          UnlockVaultWithCode(code: string): Promise<VaultStatus>;
          ResetVaultPIN(code: string, pin: string, confirm: string): Promise<VaultStatus>;
          LockVault(): Promise<void>;
          ListVaultItems(): Promise<VaultItem[]>;
          MoveItemToVault(id: number): Promise<void>;
          CopyVaultItem(id: number): Promise<void>;
          RevealVaultItem(id: number): Promise<VaultSecret>;
          UpdateVaultItemTitle(id: number, title: string): Promise<void>;
          DeleteVaultItem(id: number): Promise<void>;
          GetSettings(): Promise<AppSettings>;
          UpdateSettings(s: AppSettings): Promise<AppSettings>;
          SaveItemToFile(id: number): Promise<string>;
          PickSaveFolder(): Promise<string>;
          ExportAllToZip(): Promise<string>;
          ShowPopup(): Promise<void>;
          HidePopup(): Promise<void>;
          TogglePopup(): Promise<void>;
          IsVisible(): Promise<boolean>;
          SystemInfo(): Promise<Record<string, string>>;
        };
      };
    };
    runtime?: WailsRuntime;
  }
}

function svc() {
  const s = window.go?.service?.Service;
  if (!s) throw new Error('Wails bindings not available — is the app running inside Wails?');
  return s;
}

/**
 * Best-effort text → clipboard write used by openExternal() when the browser
 * launch can't be performed. Prefers the async Clipboard API and falls back
 * to a hidden <textarea> + execCommand on older WebViews.
 */
async function copyTextToClipboard(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const ta = document.createElement('textarea');
  ta.value = text;
  ta.setAttribute('readonly', '');
  ta.style.position = 'fixed';
  ta.style.top = '-1000px';
  ta.style.opacity = '0';
  document.body.appendChild(ta);
  ta.select();
  try {
    if (!document.execCommand('copy')) {
      throw new Error('execCommand("copy") returned false');
    }
  } finally {
    document.body.removeChild(ta);
  }
}

export const api = {
  listItems(filter = '', limit = 200): Promise<ClipItem[]> {
    return svc().ListItems(filter, limit);
  },
  pasteItem(id: number): Promise<void> {
    return svc().PasteItem(id);
  },
  copyItem(id: number): Promise<void> {
    return svc().CopyItem(id);
  },
  pinItem(id: number, pinned: boolean): Promise<void> {
    return svc().PinItem(id, pinned);
  },
  deleteItem(id: number): Promise<void> {
    return svc().DeleteItem(id);
  },
  clearAll(keepPinned = true): Promise<void> {
    return svc().ClearAll(keepPinned);
  },
  vaultStatus(): Promise<VaultStatus> {
    return svc().VaultStatus();
  },
  startVaultSetup(): Promise<VaultSetupBundle> {
    return svc().StartVaultSetup();
  },
  confirmVaultSetup(pin: string, confirm: string, code: string): Promise<VaultStatus> {
    return svc().ConfirmVaultSetup(pin, confirm, code);
  },
  unlockVaultWithPIN(pin: string): Promise<VaultStatus> {
    return svc().UnlockVaultWithPIN(pin);
  },
  unlockVaultWithCode(code: string): Promise<VaultStatus> {
    return svc().UnlockVaultWithCode(code);
  },
  resetVaultPIN(code: string, pin: string, confirm: string): Promise<VaultStatus> {
    return svc().ResetVaultPIN(code, pin, confirm);
  },
  lockVault(): Promise<void> {
    return svc().LockVault();
  },
  listVaultItems(): Promise<VaultItem[]> {
    return svc().ListVaultItems();
  },
  moveItemToVault(id: number): Promise<void> {
    return svc().MoveItemToVault(id);
  },
  copyVaultItem(id: number): Promise<void> {
    return svc().CopyVaultItem(id);
  },
  revealVaultItem(id: number): Promise<VaultSecret> {
    return svc().RevealVaultItem(id);
  },
  updateVaultItemTitle(id: number, title: string): Promise<void> {
    return svc().UpdateVaultItemTitle(id, title);
  },
  deleteVaultItem(id: number): Promise<void> {
    return svc().DeleteVaultItem(id);
  },
  getSettings(): Promise<AppSettings> {
    return svc().GetSettings();
  },
  updateSettings(s: AppSettings): Promise<AppSettings> {
    return svc().UpdateSettings(s);
  },
  saveItemToFile(id: number): Promise<string> {
    return svc().SaveItemToFile(id);
  },
  pickSaveFolder(): Promise<string> {
    return svc().PickSaveFolder();
  },
  exportAllToZip(): Promise<string> {
    return svc().ExportAllToZip();
  },
  hidePopup(): Promise<void> {
    return svc().HidePopup();
  },
  minimizeWindow(): void {
    window.runtime?.WindowMinimise();
  },
  toggleMaximizeWindow(): void {
    window.runtime?.WindowToggleMaximise();
  },
  /**
   * Try to open a URL in the user's default browser. If that fails for any
   * reason (no Wails runtime, popup blocked, native call throws, etc.) we
   * fall back to copying the URL onto the system clipboard so the user can
   * paste it themselves.
   *
   * Returns:
   *   "opened" — handed off to the browser
   *   "copied" — could not open; URL was copied to the clipboard instead
   *   "failed" — could not open and could not copy
   */
  async openExternal(url: string): Promise<'opened' | 'copied' | 'failed'> {
    const rt = window.runtime;
    if (rt?.BrowserOpenURL) {
      try {
        rt.BrowserOpenURL(url);
        return 'opened';
      } catch (e) {
        console.warn('BrowserOpenURL failed', e);
      }
    }
    try {
      const w = window.open(url, '_blank', 'noopener,noreferrer');
      if (w) return 'opened';
    } catch (e) {
      console.warn('window.open failed', e);
    }
    try {
      await copyTextToClipboard(url);
      return 'copied';
    } catch (e) {
      console.warn('clipboard fallback failed', e);
    }
    return 'failed';
  },
  async systemInfo(): Promise<SystemInfo> {
    const raw = await svc().SystemInfo();
    return {
      sessionType: raw.sessionType ?? '',
      desktop: raw.desktop ?? '',
    };
  },
};

export function onEvent(name: string, cb: (...data: unknown[]) => void): () => void {
  const rt = window.runtime;
  if (!rt) {
    console.warn('Wails runtime missing; events disabled');
    return () => {};
  }
  return rt.EventsOn(name, cb);
}
