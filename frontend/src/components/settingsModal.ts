// Settings modal — renders a small form bound to AppSettings and
// persists changes through the supplied save callback. Includes a
// hotkey-capture widget that maps live keypresses into the
// "Mod+Mod+Key" string the Go-side parser expects.

import type { AppSettings, SystemInfo } from '../types';

export interface SettingsModalDeps {
  load: () => Promise<AppSettings>;
  save: (s: AppSettings) => Promise<AppSettings>;
  clearAll: (keepPinned: boolean) => Promise<void>;
  systemInfo: () => Promise<SystemInfo>;
}

export interface SettingsModal {
  open(): Promise<void>;
  close(): void;
  applyTheme(theme: AppSettings['theme']): void;
}

export function createSettingsModal(root: HTMLElement, deps: SettingsModalDeps): SettingsModal {
  let current: AppSettings | null = null;

  const close = () => {
    root.classList.add('hidden');
    root.setAttribute('aria-hidden', 'true');
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

  const render = (s: AppSettings, sys: SystemInfo) => {
    const wayland = (sys.sessionType || '').toLowerCase() === 'wayland';
    const session = wayland
      ? `<div class="session-info warn">Wayland session detected (${sys.desktop || 'unknown DE'}). Global hotkeys and background clipboard monitoring are disabled — open via the tray icon.</div>`
      : `<div class="session-info">Session: ${sys.sessionType || 'unknown'} · ${sys.desktop || ''}</div>`;
    root.innerHTML = `
      <div class="panel">
        <h2>Settings</h2>
        <div class="field">
          <label>Global hotkey</label>
          <div class="hotkey-capture" tabindex="0" data-role="hotkey">${s.hotkey}</div>
          <div class="hint">Click and press your desired shortcut (e.g. Super+V, Ctrl+Alt+H).</div>
        </div>
        <div class="field">
          <label for="f-max">Max history items</label>
          <input id="f-max" type="number" min="10" max="2000" value="${s.maxItems}" />
        </div>
        <div class="field-row">
          <div>
            <label>Capture images</label>
            <div class="hint">Store PNG screenshots in history</div>
          </div>
          <div class="switch ${s.keepImages ? 'on' : ''}" data-role="keepImages"></div>
        </div>
        <div class="field">
          <label for="f-mb">Max image size (MB)</label>
          <input id="f-mb" type="number" min="1" max="50" value="${s.maxImageMB}" />
        </div>
        <div class="field">
          <label for="f-theme">Theme</label>
          <select id="f-theme">
            <option value="auto" ${s.theme === 'auto' ? 'selected' : ''}>System default</option>
            <option value="light" ${s.theme === 'light' ? 'selected' : ''}>Light</option>
            <option value="dark" ${s.theme === 'dark' ? 'selected' : ''}>Dark</option>
          </select>
        </div>
        <div class="field-row">
          <div>
            <label>Auto-start on login</label>
            <div class="hint">Create ~/.config/autostart/clipd.desktop</div>
          </div>
          <div class="switch ${s.autostart ? 'on' : ''}" data-role="autostart"></div>
        </div>
        <div class="field-row">
          <div>
            <label>Hide on focus loss</label>
            <div class="hint">Auto-close popup when clicked away</div>
          </div>
          <div class="switch ${s.hideOnBlur ? 'on' : ''}" data-role="hideOnBlur"></div>
        </div>
        ${session}
        <div class="btn-row" style="margin-top:14px;">
          <button class="btn danger" data-action="clear-all">Clear history</button>
          <button class="btn" data-action="cancel">Cancel</button>
          <button class="btn primary" data-action="save">Save</button>
        </div>
      </div>
    `;

    // Switches
    root.querySelectorAll<HTMLElement>('.switch').forEach((sw) => {
      sw.addEventListener('click', () => {
        sw.classList.toggle('on');
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

    // Theme live-preview
    const themeSel = root.querySelector<HTMLSelectElement>('#f-theme')!;
    themeSel.addEventListener('change', () => applyTheme(themeSel.value as AppSettings['theme']));

    // Action buttons
    root.querySelector<HTMLButtonElement>('[data-action="cancel"]')!
      .addEventListener('click', () => {
        applyTheme(s.theme); // revert preview
        close();
      });

    root.querySelector<HTMLButtonElement>('[data-action="clear-all"]')!
      .addEventListener('click', async () => {
        if (!confirm('Clear all clipboard history? Pinned items will be kept.')) return;
        await deps.clearAll(true);
      });

    root.querySelector<HTMLButtonElement>('[data-action="save"]')!
      .addEventListener('click', async () => {
        if (!current) return;
        const out: AppSettings = {
          ...current,
          maxItems: Number((root.querySelector('#f-max') as HTMLInputElement).value) || current.maxItems,
          maxImageMB: Number((root.querySelector('#f-mb') as HTMLInputElement).value) || current.maxImageMB,
          theme: themeSel.value as AppSettings['theme'],
          keepImages: root.querySelector('[data-role="keepImages"]')!.classList.contains('on'),
          autostart: root.querySelector('[data-role="autostart"]')!.classList.contains('on'),
          hideOnBlur: root.querySelector('[data-role="hideOnBlur"]')!.classList.contains('on'),
        };
        try {
          const saved = await deps.save(out);
          applyTheme(saved.theme);
          close();
        } catch (err) {
          alert('Failed to save settings: ' + String(err));
        }
      });

    // Close on backdrop click
    root.addEventListener('mousedown', (e) => {
      if (e.target === root) {
        applyTheme(s.theme);
        close();
      }
    });
  };

  return {
    async open() {
      const [s, sys] = await Promise.all([deps.load(), deps.systemInfo()]);
      current = { ...s };
      render(current, sys);
      root.classList.remove('hidden');
      root.setAttribute('aria-hidden', 'false');
    },
    close,
    applyTheme,
  };
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
