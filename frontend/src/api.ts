// Thin wrapper around the Wails-generated bindings (frontend/wailsjs/go/service/Service.*).
// We re-export typed functions so component code never has to look at
// the generated, untyped JS files directly. If the wails dev server
// isn't running and the bindings file is missing, we fall back to no-op
// stubs that surface a friendly error in the console.

import type { ClipItem, AppSettings, SystemInfo } from './types';

type WailsRuntime = {
  EventsOn(name: string, cb: (...data: unknown[]) => void): () => void;
  EventsOff(name: string): void;
  WindowHide(): void;
  WindowShow(): void;
  Quit(): void;
};

declare global {
  interface Window {
    go?: {
      service?: {
        Service?: {
          ListItems(filter: string, limit: number): Promise<ClipItem[]>;
          PasteItem(id: number): Promise<void>;
          PinItem(id: number, pinned: boolean): Promise<void>;
          DeleteItem(id: number): Promise<void>;
          ClearAll(keepPinned: boolean): Promise<void>;
          GetSettings(): Promise<AppSettings>;
          UpdateSettings(s: AppSettings): Promise<AppSettings>;
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

export const api = {
  listItems(filter = '', limit = 200): Promise<ClipItem[]> {
    return svc().ListItems(filter, limit);
  },
  pasteItem(id: number): Promise<void> {
    return svc().PasteItem(id);
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
  getSettings(): Promise<AppSettings> {
    return svc().GetSettings();
  },
  updateSettings(s: AppSettings): Promise<AppSettings> {
    return svc().UpdateSettings(s);
  },
  hidePopup(): Promise<void> {
    return svc().HidePopup();
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
