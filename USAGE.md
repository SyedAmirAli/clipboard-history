# clipd — Usage & Management Guide

A lightweight (~13 MB) clipboard‑history manager for X11 Linux (Debian/Ubuntu),
built with Go + Wails v2 + vanilla TypeScript. Keeps a searchable history of the
text and images you copy, lets you pin favourites, and pastes any past entry back
to the clipboard with a click or a keystroke.

---

## 1. Quick start

### Run a build you already have
```bash
./build/bin/clipd          # start clipd (lives in the tray / background)
```
It starts **hidden**. Pop the window open with the CLI trigger (below) or the
configured hotkey.

### Build from source
Prerequisites:
- Go 1.25+
- Wails v2 CLI — `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Node + Yarn (frontend bundling)
- System libs:
  ```bash
  sudo apt-get install -y build-essential pkg-config \
    libgtk-3-dev libwebkit2gtk-4.1-dev libayatana-appindicator3-dev \
    libx11-dev libxfixes-dev xclip
  ```

Build:
```bash
wails build -tags webkit2_41 -clean -ldflags="-s -w"
# → ./build/bin/clipd   (~13 MB)
```
Or use the helper:
```bash
./scripts/build.sh        # binary only
./scripts/build.sh deb    # also produces dist/clipd_<v>_amd64.deb (needs nfpm)
```

### Install the .deb (on a real Debian/Ubuntu desktop)
```bash
sudo apt install ./dist/clipd_0.1.0_amd64.deb
```
Installs the binary to `/usr/bin/clipd`, an app icon, and a `.desktop` launcher.

---

## 2. Command-line interface

`clipd` doubles as its own remote control. A second invocation talks to the
already‑running instance over a Unix socket at `$XDG_RUNTIME_DIR/clipd.sock`.

| Command | What it does |
|---------|--------------|
| `clipd` | Start clipd. **If one is already running, it toggles that window instead of launching a second copy** (single‑instance guard). |
| `clipd start` | Explicitly start clipd. Idempotent — if one is already running it just reports that, without toggling the window. |
| `clipd toggle` | Show the popup if hidden, hide it if shown. **The command‑line alternative to the Super+V hotkey.** |
| `clipd show` | Force the popup to show. |
| `clipd hide` | Force the popup to hide. |
| `clipd quit` | **Fully shut down** the running instance (aliases: `exit`, `stop`). |
| `clipd restart` | Shut the running instance down and start a fresh one. |
| `clipd help` (`-h`, `--help`) | Print usage. |

If you run a control command with no instance running, clipd tells you so and
exits non‑zero — nothing to control yet.

---

## 3. In-app keyboard shortcuts

Once the popup is open:

| Key | Action |
|-----|--------|
| Type anything | Live search the history (debounced, FTS5‑backed) |
| `/` | Jump focus to the search box |
| `↓` / `↑` | Move selection down / up the list |
| `Enter` | Paste the selected item back to the clipboard (and hide) |
| `Ctrl` + `,` | Open Settings |
| `Esc` | Close the popup |
| Right‑click an item | Context menu: Pin/Unpin, Delete, Copy |

The window also hides automatically when it loses focus (configurable —
*Hide on blur*).

---

## 4. Features

A quick map of what's built in:

**Capture & history**
- Automatic clipboard monitoring via the X11 **XFixes** extension (event‑driven,
  with a 500 ms polling fallback)
- Captures both **text** and **images** (PNG)
- **De‑duplication** by SHA‑256 content hash — re‑copying something bumps it to
  the top instead of creating a duplicate
- Configurable **max history size** (default 200 items)
- Image capture can be toggled off, with a per‑image size cap (default 5 MB)
- Image **thumbnails** generated for the list view

**Browse & use**
- Live **full‑text search** (SQLite FTS5)
- **Pinned** section that survives "Clear all"
- One‑click / `Enter` **paste‑back** to the system clipboard
- **Pin/unpin**, **delete single item**, **clear all (optionally keep pinned)**
- Keyboard‑first navigation (arrows + Enter)
- Right‑click context menu per item

**Access**
- Global **Super+V hotkey** (X11 `XGrabKey`) to summon the popup
- **CLI trigger** (`clipd toggle/show/hide`) as the hotkey alternative ★ added
- **Single‑instance guard** — running `clipd` twice toggles instead of duplicating ★ added
- **System tray** icon with menu: Open, Clear all, Settings, Quit
- Frameless, always‑on‑top, auto‑centered popup window

**Persistence & integration**
- **SQLite** storage (WAL mode) at `~/.local/share/clipd/clipd.db`
- **Autostart** toggle — writes `~/.config/autostart/clipd.desktop`
- **Light/dark theme** (auto / light / dark)
- **Wayland guard** — detects a Wayland session and degrades gracefully to
  tray/CLI‑only mode instead of failing

★ = features added on top of the original plan in this working copy.

---

## 5. Settings

Open with `Ctrl+,` or the tray/Settings menu. Stored in the `settings` table of
the SQLite DB. Defaults:

| Setting | Default | Meaning |
|---------|---------|---------|
| Hotkey | `Super+V` | Global shortcut to open the popup |
| Max items | `200` | History cap; oldest unpinned trimmed |
| Keep images | `true` | Capture image copies, not just text |
| Max image MB | `5` | Reject images larger than this |
| Theme | `auto` | `auto` / `light` / `dark` |
| Autostart | `false` | Launch clipd at login |
| Hide on blur | `true` | Auto‑hide popup when it loses focus |
| Launch at top | `false` | Position popup near the top of the screen |

---

## 6. Managing the process

```bash
# Start (background)
./build/bin/clipd &

# Is it running?
pgrep -af build/bin/clipd

# Open / close the window without touching the process
clipd show
clipd hide
clipd toggle

# Stop it
pkill -f build/bin/clipd        # or use the tray "Quit" item

# Where its data lives
~/.local/share/clipd/clipd.db          # history (SQLite)
~/.config/autostart/clipd.desktop      # autostart entry (if enabled)
$XDG_RUNTIME_DIR/clipd.sock            # control socket (transient)
```

Inspect history directly (optional, needs `sqlite3`):
```bash
sqlite3 ~/.local/share/clipd/clipd.db \
  "SELECT id, content_type, substr(text_content,1,50), pinned FROM items ORDER BY id DESC LIMIT 10;"
```

---

## 7. Binding a shortcut to `clipd toggle`

Because `clipd toggle` is just a command, you can bind it to any key your
environment offers — handy when the built‑in X11 hotkey can't grab the key.

**Native Linux desktop (GNOME example):**
Settings → Keyboard → Custom Shortcuts → add command `clipd toggle`, assign a key.
(XFCE, KDE, etc. have equivalent "custom shortcut" panels.)

**WSL / Windows side** (since global X11 hotkeys don't grab under WSLg):
bind a Windows shortcut that runs the command inside WSL — e.g. with
Microsoft PowerToys (Keyboard Manager → run program) or AutoHotkey:
```autohotkey
; AutoHotkey v2 — Win+V runs clipd toggle inside WSL
#v::Run('wsl.exe -e clipd toggle', , 'Hide')
```

---

## 8. Platform notes

- **Target platform:** X11 on Debian/Ubuntu. This is the supported environment
  for the full feature set.
- **Wayland:** clipboard monitoring and the X11 hotkey are disabled; use the
  tray icon or `clipd toggle`. clipd detects this and won't hard‑fail.
- **WSL2 / WSLg:** verified to build and run; the window renders, clipboard
  auto‑capture works, and the `clipd toggle/show/hide` CLI works. The global
  Super+V grab and the appindicator tray are unreliable under WSLg — use the
  CLI trigger (section 7) instead. Not a substitute for QA on a real desktop.

---

## 9. Troubleshooting

| Symptom | Fix |
|---------|-----|
| `clipd: no running instance to "toggle"` | Start clipd first (`./build/bin/clipd &`). |
| Window never appears | Check it's running (`pgrep -af clipd`); look at logs (run it in a terminal, not detached). |
| `libEGL ... DRI3` warnings | Harmless — just means no GPU acceleration (common in WSLg). |
| Build fails: `webkit2gtk-4.1 MISSING` | Install the dev libs in section 1. |
| Build fails: `pattern all:frontend/dist: no matching files` | Run the frontend build — `wails build` does this for you. |
| Nothing gets captured | Confirm you're on X11 (`echo $XDG_SESSION_TYPE`); `xclip` installed; clipd running. |
| Super+V does nothing | Expected under Wayland/WSLg — bind `clipd toggle` instead (section 7). |
