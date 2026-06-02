# clipd — Linux Clipboard History

A lightweight clipboard manager for X11 Linux (Debian / Ubuntu / Mint /
Pop!\_OS) inspired by the Windows + V panel.

-   Text **and** image (PNG) history with thumbnails
-   Pin items so they survive eviction; live search and All/Text/Images/Pinned filters
-   **Click / `Enter` to paste** — copies the item and auto-pastes it into your
    previous app (via `xdotool`); separate **Copy** action that just copies
-   Global hotkey (default `Super + V`) **and** a full CLI
    (`clipd start|toggle|show|hide|quit|restart`) as the hotkey alternative
-   System tray icon, auto-start on login, light / dark / auto theme
-   Frameless window with a custom macOS-style title bar; runs as a normal
    taskbar window or a floating popup
-   Themed delete / clear-history confirmation dialogs
-   Built with Go + Wails v2, vanilla TypeScript frontend, pure-Go SQLite
-   ~13 MB single binary

> 📖 Full usage, CLI reference, settings, and troubleshooting: **[USAGE.md](USAGE.md)**.

## Install (recommended)

```bash
sudo apt install ./clipd_0.1.0_amd64.deb
```

This installs:

-   Binary at `/usr/bin/clipd`
-   Desktop entry at `/usr/share/applications/clipd.desktop`
-   Icon at `/usr/share/icons/hicolor/256x256/apps/clipd.png`

It pulls in the runtime dependencies automatically (`libgtk-3-0`,
`libwebkit2gtk-4.1-0`, `libayatana-appindicator3-1`, `libx11-6`, `xclip`,
`xdotool`). Launch from your application menu or with:

```bash
clipd start
```

## Usage

Open the popup with the global **`Super + V`** hotkey, or from the command line:

```bash
clipd start      # start the daemon (idempotent)
clipd toggle     # show/hide the popup — bind this to a key if the hotkey can't grab
clipd quit       # fully shut down   (clipd restart = stop + start)
clipd help       # full command list
```

Type to search, use the All/Text/Images/Pinned chips to filter, and press
`Enter` (or click a row) to paste an item back. See **[USAGE.md](USAGE.md)** for
the complete reference, settings, keyboard shortcuts, and platform notes.

## Build from source

```bash
# One-time prerequisites
sudo apt-get update && sudo apt-get install -y build-essential pkg-config \
  libgtk-3-dev libwebkit2gtk-4.1-dev libayatana-appindicator3-dev \
  libx11-dev libxfixes-dev xclip xdotool
go install github.com/wailsapp/wails/v2/cmd/wails@latest
go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest   # for the .deb only

# Build
./scripts/build.sh        # → ./build/bin/clipd  (binary only)
./scripts/build.sh deb    # → dist/clipd_<version>_amd64.deb
```

> **Note:** clipd targets **X11**. Under Wayland the global hotkey, tray, and
> background capture are disabled (use `clipd toggle`). Under WSL2/WSLg the
> hotkey, tray, and Windows-screenshot image capture don't work — it's a build
> environment, not a place to judge the full experience.
