// Search bar wires the input element to a debounced query callback.
// The input element itself lives in index.html so the topbar can act
// as a drag handle for the frameless window with the input cut-out.

export interface SearchBar {
  /** Read or set the current query. */
  value(): string;
  setValue(v: string): void;
  /** Move keyboard focus to the input. */
  focus(): void;
  /** Subscribe to debounced change events. */
  onChange(cb: (q: string) => void): void;
  /** Forward keydown events from the input that affect the list. */
  onKeyToList(cb: (e: KeyboardEvent) => void): void;
}

export function createSearchBar(input: HTMLInputElement, debounceMs = 100): SearchBar {
  let changeCb: (q: string) => void = () => {};
  let keyCb: (e: KeyboardEvent) => void = () => {};
  let t: number | undefined;

  input.addEventListener('input', () => {
    if (t) clearTimeout(t);
    t = window.setTimeout(() => changeCb(input.value), debounceMs);
  });

  input.addEventListener('keydown', (e) => {
    // Forward navigation keys to the list handler. Backspace etc. are
    // left to the input itself.
    if (['ArrowDown', 'ArrowUp', 'Enter', 'PageDown', 'PageUp', 'Home', 'End'].includes(e.key)) {
      keyCb(e);
    }
  });

  return {
    value: () => input.value,
    setValue(v: string) {
      input.value = v;
      changeCb(v);
    },
    focus() {
      input.focus();
      input.select();
    },
    onChange(cb) {
      changeCb = cb;
    },
    onKeyToList(cb) {
      keyCb = cb;
    },
  };
}
