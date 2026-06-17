# clipd for Windows

This guide covers building and running clipd on Windows.

## Features

- Clipboard history with text and image support
- Global hotkey (Win+V by default)
- System tray icon with quick actions
- Dark/light/auto theme
- Private Vault for sensitive items
- Auto-start on login

## Building from Source

### Prerequisites

- **Go 1.22+** — [Download](https://golang.org/dl/)
- **Wails v2** — Install with: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Node.js + npm/yarn** — For the frontend build
- **Git** — For version control

### Build Steps

```powershell
# Clone the repository
git clone https://github.com/yourusername/clipd.git
cd clipd

# Switch to windows branch
git checkout windows

# Build the application
.\scripts\build-windows.bat

# The executable will be at: .\build\bin\clipd.exe
```

## Running clipd

Double-click `clipd.exe` or run from PowerShell:

```powershell
.\build\bin\clipd.exe
```

### Command-line Options

```
clipd                    Start clipd (or toggle if already running)
clipd start             Start the daemon
clipd toggle            Show/hide the clipboard popup
clipd show              Show the popup
clipd hide              Hide the popup
clipd quit              Fully shut down
clipd restart           Restart the application
clipd reset-vault       Delete private vault setup
```

## Configuration

Settings are stored in: `%APPDATA%\clipd\`

- `clipd.db` — SQLite database with clipboard history
- Theme, hotkey, and other preferences are managed in-app

## Default Hotkey

- **Win+V** — Show/hide clipboard history

You can customize this in the app's Settings.

## Auto-start on Login

Enable "Auto-start on login" in the Settings window. This adds an entry to Windows Startup folder.

## Troubleshooting

### Clipboard not capturing?

- Make sure another clipboard manager isn't running (e.g., Ditto, ClipClip)
- Restart clipd with `clipd restart`

### Hotkey not working?

- Try using `clipd toggle` from a scheduled task or bind it to a Windows hotkey manually via Settings > Keyboard Shortcuts
- Check that no other application has claimed the Win+V hotkey

### Images not showing?

- Windows clipboard for images is supported (DIB format)
- PNG format support is currently limited; convert images to standard formats

## Performance

- Memory usage: ~30-50 MB typical
- CPU: <1% at rest
- Polls clipboard every 500ms (configurable)

## Limitations (vs Linux)

1. Image format support is limited to DIB (standard Windows clipboard bitmap)
2. No support for synthetic input (Ctrl+V) like xdotool on Linux — clipboard is copied but you must paste manually or use auto-paste if supported by your app
3. Tray icon always shows (no minimize-to-tray-only mode yet)

## Development

To modify the code:

1. Edit Go backend code in `internal/*/`
2. Edit TypeScript frontend code in `frontend/src/`
3. Run `wails build` to test
4. Use `wails dev` for live reload during development

See main README.md for architecture details.

## Uninstall

Simply delete `clipd.exe` and the config folder (`%APPDATA%\clipd\`).

To remove auto-start, disable it in Settings before uninstalling.
