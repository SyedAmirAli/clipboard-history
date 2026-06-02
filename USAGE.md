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
    libx11-dev libxfixes-dev xclip xdotool
  ```
  (`xclip` reads/writes the clipboard; `xdotool` powers auto‑paste.)
- For the `.deb` step: `nfpm` —
  `go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest`

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
already‑running instance over a Unix socket at `~/.local/share/clipd/clipd.sock`.
The path is derived from `$HOME` (not `$XDG_RUNTIME_DIR`) so it resolves
identically whether clipd is launched from a desktop session or from a bare,
env‑less call like `wsl.exe clipd toggle` bound to a Windows hotkey.

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
| Type anything | Live search the history (debounced, case‑insensitive substring match) |
| `/` | Jump focus to the search box |
| `↓` / `↑` | Move selection down / up the list |
| `Enter` | Paste the selected item (copies, hides, and auto‑pastes into your previous app) |
| `Ctrl` + `,` | Open Settings |
| `Esc` | Close the popup (or close an open dialog/menu first) |
| Right‑click an item | Context menu: Copy, Pin/Unpin, Delete |

Other UI:
- **Filter chips** at the top — All / Text / Images / Pinned.
- **Title bar** with macOS‑style traffic lights — red hides, amber minimizes
  (to the taskbar), green maximizes; drag the bar to move the window.
- **Row click** pastes (= `Enter`); the inline **Copy** button just copies to
  the clipboard *without* closing; **Pin** and **Delete** never close the window.

In *popup* mode the window also hides when it loses focus (configurable —
*Hide on blur*); in the default *taskbar* mode it stays put.

---

## 4. Features

A quick map of what's built in:

**Capture & history**
- Automatic clipboard monitoring — polls the X11 CLIPBOARD selection via
  `xclip` every 500 ms (pure‑Go, no cgo/libX11 link, keeps the binary small)
- Captures both **text** and **images** (PNG, when the clipboard offers an
  `image/png` target)
- **De‑duplication** by SHA‑256 content hash — re‑copying something bumps it to
  the top instead of creating a duplicate
- Configurable **max history size** (default 200 items)
- Image capture can be toggled off, with a per‑image size cap (default 5 MB)
- Image **thumbnails** generated for the list view

**Browse & use**
- Live **search** — case‑insensitive substring match over text entries
- **Filter chips**: All / Text / Images / Pinned
- **Pinned** section that survives "Clear history"
- **Row click / `Enter` = paste** — copies, hides, and (if enabled) auto‑pastes
  into the previously focused app via a synthesised `Ctrl+V`
- **Copy** action — puts an item on the clipboard *without* closing the window
- **Pin / Unpin**, **delete a single item**, **clear history (keeps pinned)**
- **Themed confirmation dialogs** for delete and clear (no native popups)
- Keyboard‑first navigation (arrows + Enter)
- Right‑click context menu per item

**Access & control**
- Global **Super+V hotkey** (X11 `XGrabKey`) to summon the popup
- **CLI control** — `clipd start|toggle|show|hide|quit|restart` (see §2); the
  hotkey alternative and a clean way to script clipd
- **Single‑instance guard** — running `clipd` twice toggles instead of duplicating
- **System tray** icon with menu: Open, Clear all, Settings, Quit
- **Frameless window with a custom macOS‑style title bar**; runs as either a
  normal **taskbar window** (default) or a floating always‑on‑top **popup**

**Persistence & integration**
- **SQLite** storage (WAL mode) at `~/.local/share/clipd/clipd.db`
- **Autostart** toggle — writes `~/.config/autostart/clipd.desktop`
- **Light/dark theme** (auto / light / dark)
- **Wayland guard** — detects a Wayland session and degrades gracefully to
  tray/CLI‑only mode instead of failing

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
| Hide on blur | `true` | Auto‑hide popup when it loses focus (popup mode only) |
| Launch at top | `false` | Position popup near the top of the screen |
| **Taskbar window** | `true` | On: normal window with a taskbar entry. Off: floating always‑on‑top popup that hides on blur. **Restart to apply.** |
| **Auto‑paste on select** | `true` | After selecting an item, synthesise `Ctrl+V` into the focused window (requires `xdotool`) |

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

# Stop / restart it cleanly
clipd quit                      # fully shuts down (or use the tray "Quit")
clipd restart                   # stop + start a fresh instance

# Where its data lives
~/.local/share/clipd/clipd.db          # history (SQLite)
~/.local/share/clipd/clipd.sock        # control socket (transient)
~/.config/autostart/clipd.desktop      # autostart entry (if enabled)
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
- **WSL2 / WSLg:** builds and runs; the window renders, **text** auto‑capture
  works, and the `clipd` CLI works. Known WSLg limitations (not clipd bugs):
  - The global **Super+V** grab and the **appindicator tray** don't work —
    bind `clipd toggle` to a Windows hotkey instead (section 7).
  - **Windows screenshots aren't captured**: WSLg doesn't expose a copied
    image to X11 as an `image/png` target, so there's nothing for clipd to
    grab (the image still pastes fine in Windows apps). Image capture works
    on a real Linux desktop, where screenshot tools set `image/png`.
  - **Auto‑paste** reaches Linux/XWayland apps but can't type into Windows
    apps (different display server) — the value is still on the clipboard.

  Treat WSL as a build/dev environment; do final QA on a real X11 desktop.

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
| Screenshots/images not in history | The clipboard must offer an `image/png` target (`xclip -selection clipboard -t TARGETS -o`). Windows screenshots under WSLg don't — see section 8. |
| Auto‑paste doesn't type into the field | Install `xdotool`; it targets the window focused *after* clipd hides. Terminals often need `Ctrl+Shift+V`, so toggle Auto‑paste off for those. |
| Corners look dark, not rounded | Your compositor isn't alpha‑compositing (common under WSLg). Looks correct on a real desktop with a compositor. |
