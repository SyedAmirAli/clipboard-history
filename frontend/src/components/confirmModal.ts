// Themed confirmation dialog matching the settings overlay's look (scrim +
// blurred main content + centered card). Returns a Promise<boolean> so call
// sites can `await confirm.show({...})` before a destructive action.

export interface ConfirmOptions {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
}

export interface ConfirmDialog {
  show(opts: ConfirmOptions): Promise<boolean>;
  isOpen(): boolean;
}

const SVG_WARN = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`;
const SVG_INFO = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`;

export function createConfirm(host: HTMLElement, mainContent?: HTMLElement): ConfirmDialog {
  const scrim = document.createElement('div');
  scrim.className = 'confirm-scrim';
  const dialog = document.createElement('div');
  dialog.className = 'confirm-dialog';
  dialog.setAttribute('role', 'alertdialog');
  dialog.setAttribute('aria-modal', 'true');
  host.appendChild(scrim);
  host.appendChild(dialog);

  let resolver: ((v: boolean) => void) | null = null;
  // Whether THIS dialog added the blur — so we don't strip a blur that the
  // settings overlay (which can sit underneath us) put there.
  let appliedBlur = false;
  const isOpen = () => scrim.classList.contains('open');

  const settle = (result: boolean) => {
    if (!resolver) return;
    scrim.classList.remove('open');
    dialog.classList.remove('open');
    if (appliedBlur) {
      mainContent?.classList.remove('blurred');
      appliedBlur = false;
    }
    document.removeEventListener('keydown', onKey, true);
    const r = resolver;
    resolver = null;
    r(result);
  };

  // Capture phase + stopPropagation so Esc/Enter resolve the dialog instead
  // of bubbling to the page handler (which would hide the whole window).
  const onKey = (e: KeyboardEvent) => {
    if (!isOpen()) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      settle(false);
    } else if (e.key === 'Enter') {
      e.preventDefault();
      e.stopPropagation();
      settle(true);
    }
  };

  scrim.addEventListener('mousedown', () => settle(false));

  return {
    isOpen,
    show(opts) {
      // If a dialog is somehow already open, cancel it first.
      settle(false);
      return new Promise<boolean>((resolve) => {
        resolver = resolve;
        const confirmLabel = opts.confirmLabel ?? 'Confirm';
        const cancelLabel = opts.cancelLabel ?? 'Cancel';
        dialog.innerHTML = `
          <div class="confirm-icon ${opts.danger ? 'danger' : ''}">${opts.danger ? SVG_WARN : SVG_INFO}</div>
          <h3 class="confirm-title">${escapeHtml(opts.title)}</h3>
          <p class="confirm-msg">${escapeHtml(opts.message)}</p>
          <div class="confirm-actions">
            <button class="s-btn" data-act="cancel">${escapeHtml(cancelLabel)}</button>
            <button class="s-btn ${opts.danger ? 'danger-solid' : 'primary'}" data-act="ok">${escapeHtml(confirmLabel)}</button>
          </div>`;
        dialog.querySelector<HTMLButtonElement>('[data-act="cancel"]')!
          .addEventListener('click', () => settle(false));
        dialog.querySelector<HTMLButtonElement>('[data-act="ok"]')!
          .addEventListener('click', () => settle(true));

        if (mainContent && !mainContent.classList.contains('blurred')) {
          mainContent.classList.add('blurred');
          appliedBlur = true;
        }
        scrim.classList.add('open');
        dialog.classList.add('open');
        document.addEventListener('keydown', onKey, true);
        dialog.querySelector<HTMLButtonElement>('[data-act="ok"]')!.focus();
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
