// Settings overlay — an in-popup panel (blurs the main content behind a
// dim scrim) bound to AppSettings. Persists changes through the supplied
// save callback. Custom controls replace the native ones: a segmented
// theme picker, [−] value [+] steppers with hold-to-repeat, toggle
// switches, and a hotkey-capture widget that maps live keypresses into
// the "Mod+Mod+Key" string the Go-side parser expects.

import type { AppSettings, SystemInfo } from '../types';
import type { ConfirmDialog } from './confirmModal';

export interface SettingsChrome {
  scrim: HTMLElement;
  mainContent: HTMLElement;
}

export interface SettingsModalDeps {
  load: () => Promise<AppSettings>;
  save: (s: AppSettings) => Promise<AppSettings>;
  clearAll: (keepPinned: boolean) => Promise<void>;
  systemInfo: () => Promise<SystemInfo>;
  chrome: SettingsChrome;
  confirm: ConfirmDialog;
}

export interface SettingsModal {
  open(): Promise<void>;
  close(): void;
  isOpen(): boolean;
  applyTheme(theme: AppSettings['theme']): void;
}

const SVG_MINUS = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="5" y1="12" x2="19" y2="12"/></svg>`;
const SVG_PLUS = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>`;
const SVG_TRASH = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6"/><path d="M10 11v6M14 11v6"/></svg>`;
const SVG_X = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`;
const SVG_CHECK = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>`;

interface StepCfg {
  min: number;
  max: number;
  step: number;
}
const STEP_CFG: Record<string, StepCfg> = {
  's-max-items': { min: 10, max: 2000, step: 10 },
  's-max-mb': { min: 1, max: 50, step: 1 },
};

export function createSettingsModal(root: HTMLElement, deps: SettingsModalDeps): SettingsModal {
  let current: AppSettings | null = null;
  // Original theme captured on open, so Cancel/Escape can revert a preview.
  let openedTheme: AppSettings['theme'] = 'auto';
  const { scrim, mainContent } = deps.chrome;

  const isOpen = () => root.classList.contains('open');

  const cancel = () => {
    applyTheme(openedTheme); // revert any live theme preview
    close();
  };

  // Escape closes the overlay (captured before main.ts sees it). If a
  // confirm dialog is layered on top, let it handle Escape itself.
  const escHandler = (e: KeyboardEvent) => {
    if (e.key === 'Escape' && isOpen()) {
      if (deps.confirm.isOpen()) return;
      e.stopPropagation();
      e.preventDefault();
      cancel();
    }
  };

  const open = async () => {
    const [s, sys] = await Promise.all([deps.load(), deps.systemInfo()]);
    current = { ...s };
    openedTheme = s.theme;
    render(current, sys);
    mainContent.classList.add('blurred');
    scrim.classList.add('open');
    root.classList.add('open');
    root.setAttribute('aria-hidden', 'false');
    scrim.onclick = cancel;
    document.addEventListener('keydown', escHandler, true);
  };

  const close = () => {
    mainContent.classList.remove('blurred');
    scrim.classList.remove('open');
    root.classList.remove('open');
    root.setAttribute('aria-hidden', 'true');
    scrim.onclick = null;
    document.removeEventListener('keydown', escHandler, true);
  };

  const applyTheme = (theme: AppSettings['theme']) => {
    const html = document.documentElement;
    if (theme === 'auto') html.removeAttribute('data-theme');
    else html.setAttribute('data-theme', theme);
  };

  const captureNextHotkey = (display: HTMLElement, onCapture: (spec: string) => void) => {
    display.classList.add('capturing');
    display.textContent = 'Press a shortcut…';
    const handler = (e: KeyboardEvent) => {
      e.preventDefault();
      e.stopPropagation();
      const mods: string[] = [];
      if (e.metaKey) mods.push('Super');
      if (e.ctrlKey) mods.push('Ctrl');
      if (e.altKey) mods.push('Alt');
      if (e.shiftKey) mods.push('Shift');
      const keyName = friendlyKeyName(e);
      if (!keyName) return; // ignore lone modifier presses
      const spec = [...mods, keyName].join('+');
      display.classList.remove('capturing');
      display.textContent = spec;
      document.removeEventListener('keydown', handler, true);
      onCapture(spec);
    };
    document.addEventListener('keydown', handler, true);
  };

  const stepperHtml = (id: string, value: number) => `
    <div class="stepper">
      <button class="step-btn" data-step="${id}" data-dir="-1" tabindex="-1">${SVG_MINUS}</button>
      <input class="step-input" id="${id}" type="number" value="${value}"
             min="${STEP_CFG[id].min}" max="${STEP_CFG[id].max}" />
      <button class="step-btn" data-step="${id}" data-dir="1" tabindex="-1">${SVG_PLUS}</button>
    </div>`;

  const switchRow = (
    role: keyof AppSettings,
    on: boolean,
    label: string,
    hint?: string,
  ) => `
    <div class="s-row">
      <div class="s-info">
        <div class="s-lbl">${label}</div>
        ${hint ? `<div class="s-hint">${hint}</div>` : ''}
      </div>
      <div class="switch ${on ? 'on' : ''}" data-role="${role}"></div>
    </div>`;

  const render = (s: AppSettings, sys: SystemInfo) => {
    const wayland = (sys.sessionType || '').toLowerCase() === 'wayland';
    const session = wayland
      ? `<div class="session-info warn">Wayland session detected (${sys.desktop || 'unknown DE'}). Global hotkeys and background clipboard monitoring are disabled — open via the tray icon.</div>`
      : `<div class="session-info">Session: ${sys.sessionType || 'unknown'} · ${sys.desktop || ''}</div>`;

    root.innerHTML = `
      <div class="settings-body">
        <h2>Configure your clipboard experience</h2>

        <div class="s-section">Shortcuts</div>
        <div class="s-field">
          <div class="s-lbl">Global hotkey</div>
          <div class="hotkey" data-role="hotkey" tabindex="0">${escapeHtml(s.hotkey)}</div>
          <div class="s-hint">Click and press your desired shortcut (e.g. Super+V, Ctrl+Alt+H).</div>
        </div>

        <div class="s-section">History</div>
        <div class="s-row">
          <div class="s-info"><div class="s-lbl">Max history items</div></div>
          ${stepperHtml('s-max-items', s.maxItems)}
        </div>
        ${switchRow('keepImages', s.keepImages, 'Capture images', 'Store PNG screenshots in history')}
        <div class="s-row">
          <div class="s-info"><div class="s-lbl">Max image size (MB)</div></div>
          ${stepperHtml('s-max-mb', s.maxImageMB)}
        </div>

        <div class="s-section">Appearance</div>
        <div class="s-row">
          <div class="s-info"><div class="s-lbl">Theme</div></div>
          <div class="segmented" data-role="theme">
            <div class="seg-opt ${s.theme === 'auto' ? 'active' : ''}" data-val="auto">Auto</div>
            <div class="seg-opt ${s.theme === 'dark' ? 'active' : ''}" data-val="dark">Dark</div>
            <div class="seg-opt ${s.theme === 'light' ? 'active' : ''}" data-val="light">Light</div>
          </div>
        </div>

        <div class="s-section">Behaviour</div>
        ${switchRow('autostart', s.autostart, 'Auto-start on login', 'Launch clipd automatically when you sign in to Windows')}
        ${switchRow('hideOnBlur', s.hideOnBlur, 'Hide on focus loss', 'Auto-close popup when clicked away')}
        ${switchRow('launchAtTop', s.launchAtTop, 'Launch at screen top', 'Position the popup near the top of the screen')}
        ${switchRow('autoPaste', s.autoPaste, 'Auto-paste on select', 'Paste straight into the focused field (sends Ctrl+V)')}
        ${switchRow('showInTaskbar', s.showInTaskbar, 'Show in taskbar', 'Give the window a taskbar button so it minimises to the taskbar. Off: a taskbar-less popup.')}
        ${switchRow('windowFrame', s.windowFrame, 'Taskbar window', 'On: a normal window with a taskbar entry. Off: floating always-on-top popup that hides on focus loss. Restart to apply.')}

        ${session}
      </div>

      <div class="settings-footer">
        <button class="s-btn danger" data-action="clear-all">${SVG_TRASH}Clear</button>
        <button class="s-btn" data-action="cancel">${SVG_X}Cancel</button>
        <button class="s-btn primary" data-action="save">${SVG_CHECK}Save</button>
      </div>
    `;

    wireControls();
  };

  const wireControls = () => {
    // Switches
    root.querySelectorAll<HTMLElement>('.switch').forEach((sw) => {
      sw.addEventListener('click', () => sw.classList.toggle('on'));
    });

    // Segmented theme picker — live preview
    root.querySelectorAll<HTMLElement>('[data-role="theme"] .seg-opt').forEach((opt) => {
      opt.addEventListener('click', () => {
        root.querySelectorAll('[data-role="theme"] .seg-opt').forEach((o) => o.classList.remove('active'));
        opt.classList.add('active');
        applyTheme(opt.dataset.val as AppSettings['theme']);
      });
    });

    // Steppers (click + hold-to-repeat)
    let holdTimeout: number | undefined;
    let holdInterval: number | undefined;
    const stepChange = (id: string, dir: number) => {
      const cfg = STEP_CFG[id];
      const el = root.querySelector<HTMLInputElement>(`#${id}`);
      if (!cfg || !el) return;
      const next = Math.max(cfg.min, Math.min(cfg.max, (Number(el.value) || 0) + dir * cfg.step));
      el.value = String(next);
    };
    root.querySelectorAll<HTMLButtonElement>('.step-btn').forEach((btn) => {
      const id = btn.dataset.step!;
      const dir = Number(btn.dataset.dir);
      const start = () => {
        stepChange(id, dir);
        holdTimeout = window.setTimeout(() => {
          holdInterval = window.setInterval(() => stepChange(id, dir), 70);
        }, 350);
      };
      const stop = () => {
        window.clearTimeout(holdTimeout);
        window.clearInterval(holdInterval);
      };
      btn.addEventListener('mousedown', start);
      btn.addEventListener('mouseup', stop);
      btn.addEventListener('mouseleave', stop);
    });
    // Editable step inputs — clamp on blur, arrow keys step
    root.querySelectorAll<HTMLInputElement>('.step-input').forEach((inp) => {
      const cfg = STEP_CFG[inp.id];
      if (!cfg) return;
      inp.addEventListener('focus', () => inp.select());
      inp.addEventListener('blur', () => {
        let v = parseInt(inp.value, 10);
        if (isNaN(v)) v = cfg.min;
        inp.value = String(Math.max(cfg.min, Math.min(cfg.max, v)));
      });
      inp.addEventListener('keydown', (e) => {
        if (e.key === 'ArrowUp') {
          e.preventDefault();
          stepChange(inp.id, 1);
        } else if (e.key === 'ArrowDown') {
          e.preventDefault();
          stepChange(inp.id, -1);
        }
      });
    });

    // Hotkey capture
    const hotkeyEl = root.querySelector<HTMLElement>('[data-role="hotkey"]')!;
    hotkeyEl.addEventListener('click', () => {
      captureNextHotkey(hotkeyEl, (spec) => {
        if (current) current.hotkey = spec;
      });
    });
    hotkeyEl.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault();
        hotkeyEl.click();
      }
    });

    // Footer actions
    root.querySelector<HTMLButtonElement>('[data-action="cancel"]')!.addEventListener('click', cancel);

    root.querySelector<HTMLButtonElement>('[data-action="clear-all"]')!.addEventListener('click', async () => {
      const ok = await deps.confirm.show({
        title: 'Clear clipboard history?',
        message: 'All unpinned items will be permanently removed. Pinned items are kept.',
        confirmLabel: 'Clear history',
        cancelLabel: 'Cancel',
        danger: true,
      });
      if (!ok) return;
      await deps.clearAll(true);
    });

    root.querySelector<HTMLButtonElement>('[data-action="save"]')!.addEventListener('click', async () => {
      if (!current) return;
      const themeOpt = root.querySelector<HTMLElement>('[data-role="theme"] .seg-opt.active');
      const out: AppSettings = {
        ...current,
        maxItems: Number((root.querySelector('#s-max-items') as HTMLInputElement).value) || current.maxItems,
        maxImageMB: Number((root.querySelector('#s-max-mb') as HTMLInputElement).value) || current.maxImageMB,
        theme: (themeOpt?.dataset.val as AppSettings['theme']) ?? current.theme,
        keepImages: roleOn('keepImages'),
        autostart: roleOn('autostart'),
        hideOnBlur: roleOn('hideOnBlur'),
        launchAtTop: roleOn('launchAtTop'),
        autoPaste: roleOn('autoPaste'),
        showInTaskbar: roleOn('showInTaskbar'),
        windowFrame: roleOn('windowFrame'),
      };
      try {
        const saved = await deps.save(out);
        applyTheme(saved.theme);
        close();
      } catch (err) {
        alert('Failed to save settings: ' + String(err));
      }
    });
  };

  const roleOn = (role: string) =>
    root.querySelector(`[data-role="${role}"]`)!.classList.contains('on');

  return { open, close, isOpen, applyTheme };
}

function friendlyKeyName(e: KeyboardEvent): string | null {
  if (['Shift', 'Control', 'Alt', 'Meta', 'OS'].includes(e.key)) return null;
  if (e.key === ' ' || e.code === 'Space') return 'Space';
  if (e.key === 'Enter') return 'Enter';
  if (e.key === 'Tab') return 'Tab';
  if (e.key === 'Escape') return 'Escape';
  if (e.key.length === 1) return e.key.toUpperCase();
  return null;
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
