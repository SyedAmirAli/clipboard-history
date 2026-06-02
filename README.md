# clipd — Linux Clipboard History

A lightweight clipboard manager for X11 Linux (Debian / Ubuntu / Mint /
Pop!_OS) inspired by the Windows + V panel.

- Text **and** image (PNG) history
- Pin items so they survive eviction
- Live search across the whole history
- Global hotkey (default `Super + V`) to summon the popup anywhere
- System tray icon with quick actions
- Auto-start on login (toggleable)
- Light / Dark / Auto theme
- Built with Go + Wails v2, vanilla TypeScript frontend, pure-Go SQLite
- ~12-18 MB single binary, ~50 MB resident memory typical

## Install (recommended)

```bash
sudo apt install ./clipd_0.1.0_amd64.deb
```

This installs:
- Binary at `/usr/bin/clipd`
- Desktop entry at `/usr/share/applications/clipd.desktop`
- Icon at `/usr/share/icons/hicolor/256x256/apps/clipd.png`

Launch from your application menu or with:

```bash
clipd
```

## Usage

| Action | Shortcut |
| --- | --- |
| Open / close the popup | `Super + V` (configurable in Settings) |
| Focus the search box | `/` |
| Move selection | `↑` / `↓` / `PageUp` / `PageDown` / `Home` / `End` |
| Paste selected item | `Enter` or click |
| Open the right-click menu | Right click on an item |
| Pin / unpin | Right-click → Pin to top, or click the pin icon |
| Open Settings | Tray icon → Settings, or `Ctrl + ,` |
| Close popup | `Esc` (or click outside, if "Hide on focus loss" is on) |

After clicking / Enter, the popup closes and the item is on the system
clipboard — paste with `Ctrl + V` in any application.

## Build from source

### Prerequisites

System libraries (one-time install):

```bash
sudo apt install \
  build-essential pkg-config \
  libgtk-3-dev libwebkit2gtk-4.1-dev \
  libx11-dev libayatana-appindicator3-dev \
  xclip
```

> On Ubuntu 22.04 or older you may need `libwebkit2gtk-4.0-dev` instead.
> Use `-tags webkit2_40` in that case (the default config targets 4.1).

Toolchain:

- Go 1.22 or newer
- [Wails v2 CLI](https://wails.io): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Yarn (`sudo apt install yarnpkg` or via Corepack)
- (Optional, for `.deb`) [nfpm](https://nfpm.goreleaser.com): `go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest`

### Build

```bash
# fetch JS deps and produce the binary at ./build/bin/clipd
./scripts/build.sh

# or, additionally produce dist/clipd_*.deb
./scripts/build.sh deb
```

Equivalent manual commands:

```bash
cd frontend && yarn install && cd ..
wails build -tags webkit2_41 -clean -ldflags="-s -w"
```

### Development

```bash
wails dev -tags webkit2_41
```

Hot-reloads both the Go backend and the TypeScript frontend.

## Project layout

```
.
├── main.go                          # Wails entry, wires services
├── go.mod
├── wails.json                       # Wails config (uses yarn)
├── nfpm.yaml                        # .deb package manifest
├── scripts/build.sh                 # one-shot build + package
├── build/
│   ├── appicon.png                  # application icon
│   └── linux/clipd.desktop          # freedesktop entry
├── internal/
│   ├── clipboard/                   # X11 watcher + writer + types
│   ├── db/                          # SQLite store (modernc.org/sqlite)
│   ├── config/                      # XDG paths
│   ├── hotkey/                      # global hotkey registration
│   ├── thumbnail/                   # in-process PNG downscaler
│   ├── tray/                        # AppIndicator system tray
│   ├── autostart/                   # ~/.config/autostart manager
│   └── service/                     # Wails-bound methods (JS bridge)
├── tools/genicon/                   # tiny helper to regenerate tray icon
└── frontend/
    ├── index.html
    ├── package.json                 # yarn-managed
    ├── tsconfig.json
    ├── vite.config.ts
    └── src/
        ├── main.ts                  # bootstrap
        ├── api.ts                   # typed wrappers around Wails bindings
        ├── types.ts                 # ClipItem, AppSettings, SystemInfo
        ├── styles/                  # main.css + components.css
        └── components/
            ├── searchBar.ts
            ├── itemList.ts
            ├── itemContextMenu.ts
            └── settingsModal.ts
```

## Data location

clipd follows XDG conventions:

- History database: `${XDG_DATA_HOME:-~/.local/share}/clipd/clipd.db`
- Autostart entry (when enabled): `${XDG_CONFIG_HOME:-~/.config}/autostart/clipd.desktop`

To wipe state completely:

```bash
rm -rf ~/.local/share/clipd ~/.config/autostart/clipd.desktop
```

## Troubleshooting

**Hotkey doesn't open the popup.**
Another app may already own that combo. Open Settings, click the hotkey
box, and pick a different combination (e.g. `Ctrl+Alt+V`).

**Nothing happens when I copy.**
clipd uses `xclip` under the hood. Confirm it's installed:
`which xclip`. Reinstall with `sudo apt install xclip` if missing.

**It says "Wayland session detected".**
Global hotkeys and background clipboard monitoring don't work portably
on Wayland — this is a Wayland design choice, not a clipd limitation.
Workarounds:
- Log in via the "Ubuntu on Xorg" session at the login screen, OR
- Use the tray icon to summon the popup (paste/pin/delete still work),
- Native Wayland support is planned for a future release (via XDG
  portals + `wlr-data-control`).

**Build fails with `X11/Xlib.h: No such file or directory`.**
Install `libx11-dev` (see prerequisites above).

**Build fails with `webkit2gtk-4.0` not found.**
Install either `libwebkit2gtk-4.1-dev` (Ubuntu 24.04+) or
`libwebkit2gtk-4.0-dev` (older). Use `-tags webkit2_41` for the 4.1
variant (already set in `scripts/build.sh`).

**Tray icon doesn't show on GNOME.**
GNOME removed legacy tray support; install the
[AppIndicator extension](https://extensions.gnome.org/extension/615/appindicator-support/).

## Roadmap

- Wayland support via `org.freedesktop.portal.GlobalShortcuts` +
  `wlr-data-control`
- macOS and Windows builds
- Rich-text preservation (HTML / formatting)
- Sync across devices (opt-in, end-to-end encrypted)

## License

MIT
