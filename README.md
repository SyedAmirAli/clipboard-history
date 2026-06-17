# clipd — Clipboard History (Windows Edition)

A lightweight, native clipboard manager for **Windows** inspired by the Windows + V panel. Full-featured with clipboard history, global hotkey, system tray, and encrypted vault.

> **Note:** This is the Windows edition of clipd. For Linux/Debian/Ubuntu, see the [master branch](https://github.com/yourusername/clipd/tree/master).

## Features

-   **Text and image history** with thumbnails (PNG support)
-   **Pin items** so they survive eviction; live search with All/Text/Images/Pinned filters
-   **Global hotkey** — **Win+V** (customizable) opens the clipboard popup instantly
-   **Full CLI** — `clipd toggle|show|hide|quit|restart` for command-line control or keybindings
-   **System tray icon** with quick-access menu
-   **Auto-start on login** — Registry-based startup
-   **Light / Dark / Auto theme** matching Windows appearance
-   **Frameless window** with custom title bar; runs as floating popup or normal window
-   **Private Vault** — PIN/password-protected storage for sensitive clipboard items
-   **Built with** Go + Wails v2, vanilla TypeScript frontend, pure-Go SQLite
-   **~30 MB** single native binary (includes everything)

> 📖 **Quick Start:** See [RUN_DEV_WINDOWS.md](RUN_DEV_WINDOWS.md)  
> 📖 **Full Reference:** See [WINDOWS.md](WINDOWS.md) for features, build instructions, troubleshooting

## Quick Start (30 seconds)

### Download & Run
1. Download `clipd.exe` from [Releases](https://github.com/yourusername/clipd/releases)
2. Double-click `clipd.exe`
3. Press **Win+V** to open clipboard history

### First Time Setup
- Choose light/dark theme
- Customize hotkey if desired (default: Win+V)
- Enable "Auto-start" to launch on login
- Done! ✨

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

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| **Win+V** | Show/hide clipboard history |
| **Enter** | Paste selected item |
| **Space** | Pin/unpin item |
| **Delete** | Delete item |
| **Ctrl+A** | Select all (in search) |
| **Escape** | Close popup |

## Build from Source

### Prerequisites
- **Go 1.22+** — [Download](https://golang.org/dl/)
- **Node.js + npm/yarn** — For frontend build
- **Wails v2** — `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Build Steps
```powershell
# Clone and checkout Windows branch
git clone https://github.com/yourusername/clipd.git
cd clipd
git checkout windows

# Build frontend
npm --prefix frontend install
npm --prefix frontend run build

# Build binary
npm run build:windows
# Output: .\build\bin\clipd.exe
```

Or use the build script directly:
```powershell
.\scripts\build-windows.bat
```

See [WINDOWS.md](WINDOWS.md) for detailed build instructions and troubleshooting.

## Development

To develop or modify clipd:

```powershell
npm run dev
```

This starts a dev server with live reload. Press **Win+V** to test. See [RUN_DEV_WINDOWS.md](RUN_DEV_WINDOWS.md) for development guide.

## Configuration

Settings stored in: `%APPDATA%\clipd\`
- `clipd.db` — SQLite clipboard history database
- Settings: Theme, hotkey, retention, auto-paste preference

## Support

- **Bug reports**: [GitHub Issues](https://github.com/yourusername/clipd/issues)
- **Feature requests**: Create an issue
- **Windows-specific issues**: See [WINDOWS.md](WINDOWS.md) troubleshooting section

## License

See LICENSE file for details.
