# clipd — Linux Clipboard History

A lightweight clipboard manager for X11 Linux (Debian / Ubuntu / Mint /
Pop!\_OS) inspired by the Windows + V panel.

-   Text **and** image (PNG) history
-   Pin items so they survive eviction
-   Live search across the whole history
-   Global hotkey (default `Super + V`) to summon the popup anywhere
-   System tray icon with quick actions
-   Auto-start on login (toggleable)
-   Light / Dark / Auto theme
-   Built with Go + Wails v2, vanilla TypeScript frontend, pure-Go SQLite
-   ~12-18 MB single binary, ~50 MB resident memory typical

## Install (recommended)

```bash
sudo apt install ./clipd_0.1.0_amd64.deb
```

This installs:

-   Binary at `/usr/bin/clipd`
-   Desktop entry at `/usr/share/applications/clipd.desktop`
-   Icon at `/usr/share/icons/hicolor/256x256/apps/clipd.png`

Launch from your application menu or with:

```bash
clipd
```

## Usage

```shell
sudo apt-get update && sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev libayatana-appindicator3-dev libx11-dev libxfixes-dev
```
