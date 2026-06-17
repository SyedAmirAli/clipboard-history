# Development Setup Guide

This guide covers setting up a development environment for clipd on Windows and Linux.

## Prerequisites

### All Platforms
- **Go 1.22+** — [Download](https://golang.org/dl/)
- **Node.js + npm/yarn** — For frontend
- **Git** — For version control
- **Wails v2** — `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Linux (Ubuntu/Mint/Pop!_OS)
```bash
sudo apt-get update && sudo apt-get install -y \
  build-essential pkg-config \
  libgtk-3-dev libwebkit2gtk-4.1-dev \
  libayatana-appindicator3-dev \
  libx11-dev libxfixes-dev \
  xclip xdotool wl-clipboard
```

### Windows
- **Visual C++ Build Tools** — Required for some native dependencies
- No additional system packages needed

## Development Commands

### Cross-Platform (Auto-detects OS)
```bash
npm run dev          # Start dev server with live reload
npm run build        # Build optimized binary
npm test             # Run all tests
npm run reset:dev    # Clear dev data (clipboard history)
```

### Platform-Specific Commands

#### Windows
```powershell
npm run dev:windows       # Dev server on Windows
npm run build:windows     # Build Windows binary
npm run start            # Run binary from build/bin/
npm run show             # Show clipboard popup
npm run stop             # Quit running instance
npm run vault:reset      # Clear vault
npm run reset:dev        # Clear all dev data
```

#### Linux
```bash
npm run dev:linux        # Dev server on Linux
npm run build            # Build .deb package (deb by default)
npm run build:bin        # Build binary only
npm run build:deb        # Build .deb package
npm run start:linux      # Run binary
npm run show:linux       # Show clipboard popup
npm run stop:linux       # Quit running instance
npm run vault:reset:linux # Clear vault
npm run reset:dev:linux  # Clear all dev data
```

## Development Workflow

### 1. Start Development Server
Live reload with frontend and backend changes:

**Windows:**
```powershell
npm run dev:windows
```

**Linux:**
```bash
npm run dev:linux
```

The app opens automatically in a dev window with hot reload.

### 2. Make Changes

**Frontend:** Edit `frontend/src/` — changes reflect instantly
**Backend:** Edit `internal/*/` — requires manual restart (Ctrl+C, then re-run)

### 3. Test Your Changes
```bash
npm test          # Run Go tests
npm run show      # Trigger clipboard popup (if running)
npm run stop      # Stop the dev instance
```

### 4. Build Release Binary

**Windows:**
```powershell
npm run build:windows
# Output: .\build\bin\clipd.exe
```

**Linux:**
```bash
npm run build:bin    # Just binary
npm run build:deb    # Package as .deb
```

### 5. Reset Development State
Clear clipboard history and start fresh:

**Windows:**
```powershell
npm run reset:dev
```

**Linux:**
```bash
npm run reset:dev:linux
```

## Directory Structure

```
clipd/
├── frontend/              # TypeScript UI
│   ├── src/
│   │   ├── main.ts       # Entry point
│   │   ├── components/   # UI components
│   │   └── styles/       # CSS
│   └── package.json
├── internal/             # Go backend
│   ├── autostart/        # Auto-start integration
│   ├── clipboard/        # Clipboard access (platform-specific)
│   ├── config/           # Configuration
│   ├── db/              # SQLite database
│   ├── hotkey/          # Global hotkey
│   ├── ipc/             # Inter-process communication
│   ├── service/         # Business logic
│   ├── shortcut/        # Desktop shortcuts
│   ├── thumbnail/       # Image thumbnails
│   ├── tray/            # System tray
│   └── vault/           # Encrypted storage
├── build/               # Build output
│   ├── bin/            # Compiled binaries
│   ├── appicon.png
│   ├── linux/          # Linux build assets
│   └── windows/        # Windows build assets
├── scripts/
│   ├── build.sh        # Linux build script
│   └── build-windows.bat # Windows build script
├── main.go             # Application entry point
├── wails.json          # Wails configuration
├── go.mod, go.sum      # Go dependencies
├── package.json        # npm scripts
└── README.md
```

## Common Tasks

### Add a New Feature

1. Plan the change (backend + frontend)
2. Edit files in `internal/*/` (backend) and `frontend/src/` (frontend)
3. Test: `npm run dev` → use the popup → `npm test`
4. Build: `npm run build:windows` (or `:bin` on Linux)
5. Commit with clear message

### Debug an Issue

**Frontend (TypeScript):**
- Dev window includes Chrome DevTools (F12)
- Check Console tab for errors
- Edit `frontend/src/` and see changes live

**Backend (Go):**
- Add logging: `log.Printf("debug: %v", value)`
- Run: `npm run dev:linux` or `npm run dev:windows`
- View logs in terminal where dev server is running

### Run Tests
```bash
npm test              # All tests
npm run test:verbose  # Verbose output
```

Tests run on both Linux and Windows automatically via build tags.

## Keyboard Shortcuts (Dev Window)

- **Win+V / Super+V** — Toggle popup (default hotkey)
- **Ctrl+Shift+I** — Open DevTools (frontend debugging)
- **Ctrl+R** — Reload window
- **Ctrl+Q** — Close app (or use CLI: `npm run stop`)

## Troubleshooting

### "wails dev" fails to start
```bash
# Ensure wails is installed
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Check wails version
wails version
```

### Clipboard not capturing
- Close any other clipboard managers
- Run: `npm run stop && npm run dev:windows` (or `:linux`)
- Check console for errors

### Hot reload not working
- Only frontend files (TypeScript) auto-reload
- Backend changes require restart: Stop dev server → restart
- Delete `build/` folder and rebuild if stuck

### Build fails
```bash
# Clean build cache
npm run reset:dev

# Rebuild from scratch
npm run build:windows  # or appropriate command
```

### Port already in use
Dev server uses multiple ports. If you see "port in use":
```bash
# Kill any lingering clipd processes
npm run stop

# Wait a few seconds
# Restart: npm run dev:windows
```

## Environment Variables (Advanced)

### Windows
```powershell
$env:APPDATA = "C:\Users\YOU\AppData\Roaming"  # Config location
$env:GOCACHE = "..."  # Go build cache
$env:GOMODCACHE = "..."  # Go modules cache
```

### Linux
```bash
export XDG_DATA_HOME=/tmp/clipd-dev-data      # Data location
export XDG_CONFIG_HOME=/tmp/clipd-dev-config  # Config location
export GOCACHE=/tmp/clipd-go-cache
export GOMODCACHE=/tmp/clipd-go-mod
```

## Next Steps

- Read [README.md](README.md) for architecture overview
- Read [WINDOWS.md](WINDOWS.md) for Windows-specific notes
- Check [USAGE.md](USAGE.md) for user-facing documentation
- Review code in `internal/service/service.go` for core logic

## Getting Help

- Run `clipd --help` for CLI commands
- Check GitHub Issues for known problems
- Enable debug logging: Look at `main.go` for log configuration
