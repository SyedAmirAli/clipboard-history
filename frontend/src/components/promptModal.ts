// Small themed input dialog (scrim + centered card), matching confirmModal's
// look. Used to collect the vault PIN/password before an encrypted export.
// Resolves to the entered string, or null when cancelled.

export interface PromptOptions {
  title: string;
  message: string;
  placeholder?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  /** Render the input as a password field. */
  password?: boolean;
}

export interface PromptDialog {
  show(opts: PromptOptions): Promise<string | null>;
  isOpen(): boolean;
}

const SVG_KEY = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3m-3.5 3.5L19 4"/></svg>`;

export function createPrompt(host: HTMLElement, mainContent?: HTMLElement): PromptDialog {
  const scrim = document.createElement('div');
  scrim.className = 'confirm-scrim';
  const dialog = document.createElement('div');
  dialog.className = 'confirm-dialog prompt-dialog';
  dialog.setAttribute('role', 'dialog');
  dialog.setAttribute('aria-modal', 'true');
  host.appendChild(scrim);
  host.appendChild(dialog);

  let resolver: ((v: string | null) => void) | null = null;
  let appliedBlur = false;
  const isOpen = () => scrim.classList.contains('open');

  const settle = (result: string | null) => {
    if (!resolver) return;
    scrim.classList.remove('open');
    dialog.classList.remove('open');
    if (appliedBlur) {
      mainContent?.classList.remove('blurred');
      appliedBlur = false;
    }
    document.removeEventListener('keydown', onKey, true);
    dialog.innerHTML = ''; // don't keep the secret in the DOM
    const r = resolver;
    resolver = null;
    r(result);
  };

  const currentValue = () =>
    dialog.querySelector<HTMLInputElement>('.prompt-input')?.value ?? '';

  const onKey = (e: KeyboardEvent) => {
    if (!isOpen()) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      settle(null);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      e.stopPropagation();
      settle(currentValue());
    }
  };

  scrim.addEventListener('mousedown', () => settle(null));

  return {
    isOpen,
    show(opts) {
      settle(null);
      return new Promise<string | null>((resolve) => {
        resolver = resolve;
        dialog.innerHTML = `
          <div class="confirm-icon">${SVG_KEY}</div>
          <h3 class="confirm-title">${escapeHtml(opts.title)}</h3>
          <p class="confirm-msg">${escapeHtml(opts.message)}</p>
          <input class="prompt-input" type="${opts.password ? 'password' : 'text'}"
                 placeholder="${escapeHtml(opts.placeholder ?? '')}"
                 autocomplete="${opts.password ? 'current-password' : 'off'}" />
          <div class="confirm-actions">
            <button class="s-btn" data-act="cancel">${escapeHtml(opts.cancelLabel ?? 'Cancel')}</button>
            <button class="s-btn primary" data-act="ok">${escapeHtml(opts.confirmLabel ?? 'Continue')}</button>
          </div>`;
        dialog.querySelector<HTMLButtonElement>('[data-act="cancel"]')!
          .addEventListener('click', () => settle(null));
        dialog.querySelector<HTMLButtonElement>('[data-act="ok"]')!
          .addEventListener('click', () => settle(currentValue()));

        if (mainContent && !mainContent.classList.contains('blurred')) {
          mainContent.classList.add('blurred');
          appliedBlur = true;
        }
        scrim.classList.add('open');
        dialog.classList.add('open');
        document.addEventListener('keydown', onKey, true);
        dialog.querySelector<HTMLInputElement>('.prompt-input')!.focus();
      });
    },
  };
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
