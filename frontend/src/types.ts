// Mirror of clipd/internal/clipboard.Item in TS. Wails generates a copy
// of this shape during build, but we keep our own canonical types so
// the UI compiles even before the first wails build (in pure-Vite dev).

export type ContentType = 'text' | 'image';

export interface ClipItem {
  id: number;
  contentType: ContentType;
  textContent?: string;
  preview: string;
  imageThumb?: string;
  imageW?: number;
  imageH?: number;
  contentHash: string;
  pinned: boolean;
  createdAt: number;
  lastUsedAt: number;
}

export interface AppSettings {
  hotkey: string;
  maxItems: number;
  keepImages: boolean;
  maxImageMB: number;
  theme: 'light' | 'dark' | 'auto';
  autostart: boolean;
  hideOnBlur: boolean;
  launchAtTop: boolean;
  windowFrame: boolean;
  autoPaste: boolean;
  showInTaskbar: boolean;
  saveFolder: string;
  showPinnedInHistory: boolean;
}

export interface SystemInfo {
  sessionType: string;
  desktop: string;
}

export interface VaultStatus {
  configured: boolean;
  unlocked: boolean;
  failedAttempts: number;
  lockedUntil?: number;
}

export interface VaultSetupBundle {
  secret: string;
  otpauthUrl: string;
  qrCodeSvg: string;
  manualKey: string;
  accountName: string;
  issuer: string;
}

export interface VaultItem {
  id: number;
  title?: string;
  contentType: ContentType;
  preview: string;
  imageThumb?: string;
  imageW?: number;
  imageH?: number;
  createdAt: number;
  lastUsedAt: number;
}

export interface VaultSecret {
  contentType: ContentType;
  text?: string;
}
