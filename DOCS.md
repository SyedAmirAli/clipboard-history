# clipd Windows Documentation

Welcome to the clipd documentation! This guide covers everything about running and developing clipd on Windows.

## Quick Navigation

### 👤 For Users
- **[README.md](README.md)** — Overview, features, quick start
- **[WINDOWS.md](WINDOWS.md)** — Installation, configuration, troubleshooting
- **[USAGE.md](USAGE.md)** — Full user guide, keyboard shortcuts, settings

### 👨‍💻 For Developers
- **[RUN_DEV_WINDOWS.md](RUN_DEV_WINDOWS.md)** — 5-minute dev setup, commands, debugging
- **[QUICK_START.md](QUICK_START.md)** — One-liner reference for common tasks
- **[DEV_SETUP.md](DEV_SETUP.md)** — Comprehensive development guide

### 🔧 Architecture & Porting
- **[WINDOWS_PORT_SUMMARY.md](WINDOWS_PORT_SUMMARY.md)** — Windows port details, platform-specific implementations
- **[main.go](main.go)** — Application entry point
- **[internal/](internal/)** — Core modules (platform-specific code with build tags)

---

## Getting Started

### Running clipd

**Download & Run (Easiest):**
1. Download `clipd.exe` from [Releases](https://github.com/yourusername/clipd/releases)
2. Double-click to run
3. Press **Win+V** to open clipboard history

**From Terminal:**
```powershell
.\build\bin\clipd.exe
```

**Using npm (If building from source):**
```powershell
npm run start
```

### Developing clipd

**Start dev server (takes 30 seconds):**
```powershell
npm run dev
```

App opens automatically with live reload. Edit code and restart to see changes.

---

## Key Features

✅ **Clipboard History** — Text and images  
✅ **Global Hotkey** — Win+V (customizable)  
✅ **System Tray** — Quick access menu  
✅ **Private Vault** — Encrypted storage  
✅ **Auto-Start** — Launches on login  
✅ **CLI Control** — `clipd toggle`, `clipd show`, etc.  
✅ **Theming** — Light/dark/auto  
✅ **Zero Dependencies** — Single native binary  

---

## Common Commands

### Running
| Command | What It Does |
|---------|-------------|
| `npm run dev` | Start dev server (auto-opens window) |
| `npm run start` | Run built binary from terminal |
| `npm run show` | Show clipboard popup |
| `npm run stop` | Quit running instance |

### Building & Testing
| Command | What It Does |
|---------|-------------|
| `npm run build` | Build release binary |
| `npm test` | Run all tests |
| `npm run reset:dev` | Clear clipboard history |

### Vault & Reset
| Command | What It Does |
|---------|-------------|
| `npm run vault:reset` | Clear private vault |

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| **Win+V** | Show/hide clipboard |
| **Enter** | Paste item |
| **Space** | Pin/unpin item |
| **Delete** | Delete item |
| **F12** (in dev) | Open DevTools |
| **Ctrl+Q** (in dev) | Close app |

---

## Troubleshooting

### Clipboard not capturing?
```powershell
npm run stop
npm run reset:dev
npm run dev
```

### Dev server won't start?
```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
npm run dev
```

### Changes not showing?
Restart dev server: Stop (Ctrl+C) → `npm run dev`

See [WINDOWS.md](WINDOWS.md) for more troubleshooting.

---

## Build from Source

```powershell
# Prerequisites: Go 1.22+, Node.js, Wails v2
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone and build
git clone https://github.com/yourusername/clipd.git
cd clipd
git checkout windows
npm --prefix frontend install
npm run build

# Output: .\build\bin\clipd.exe
```

See [WINDOWS.md](WINDOWS.md) for detailed build instructions.

---

## Project Structure

```
clipd/
├── README.md               ← Start here
├── WINDOWS.md              ← Windows features & troubleshooting
├── USAGE.md                ← User manual
├── RUN_DEV_WINDOWS.md      ← Dev setup
├── QUICK_START.md          ← Command reference
├── DOCS.md                 ← This file
├── package.json            ← npm scripts (Windows-focused)
├── main.go                 ← App entry point
├── go.mod, go.sum          ← Go dependencies
├── wails.json              ← Wails configuration
├── internal/               ← Core modules
│   ├── autostart/          ← Windows Registry startup
│   ├── clipboard/          ← Clipboard capture (Windows API)
│   ├── config/             ← Config paths (AppData)
│   ├── db/                 ← SQLite database
│   ├── hotkey/             ← Global hotkey
│   ├── ipc/                ← Named pipes IPC
│   ├── service/            ← Business logic
│   ├── shortcut/           ← Desktop shortcuts
│   ├── tray/               ← System tray
│   └── vault/              ← Encrypted storage
├── frontend/               ← TypeScript/React UI
│   ├── src/                ← Frontend source
│   ├── package.json
│   └── vite.config.ts
├── build/                  ← Build output
│   ├── bin/
│   ├── windows/            ← Windows resources
│   └── appicon.png
└── scripts/
    ├── build.sh            ← Linux build (not used on Windows)
    └── build-windows.bat   ← Windows build script
```

---

## Platform-Specific Code

This is a **cross-platform** codebase that supports Windows and Linux. Platform-specific code uses Go build tags:

```go
//go:build windows
// Windows implementation
```

```go
//go:build linux
// Linux implementation
```

Key platform-specific modules:
- `internal/clipboard/watcher_*.go` — Clipboard access
- `internal/clipboard/paster_*.go` — Synthetic input
- `internal/clipboard/writer_*.go` — Clipboard writing
- `internal/ipc/*_*.go` — Process communication
- `internal/autostart/*_*.go` — Startup registration
- `internal/shortcut/*_*.go` — Global shortcuts

See [WINDOWS_PORT_SUMMARY.md](WINDOWS_PORT_SUMMARY.md) for implementation details.

---

## Development Workflow

1. **Start dev server** → `npm run dev`
2. **Make changes** → Frontend changes auto-reload, backend needs restart
3. **Test** → Press Win+V to test clipboard, F12 for DevTools
4. **Commit** → `git add . && git commit -m "..."`
5. **Build release** → `npm run build`

---

## Contributing

- Report bugs via [GitHub Issues](https://github.com/yourusername/clipd/issues)
- Follow Git commit conventions
- Test on Windows before submitting PR
- Update docs if adding features

---

## Resources

- **Wails Docs** — https://wails.io
- **Go Docs** — https://golang.org/doc
- **TypeScript Docs** — https://www.typescriptlang.org/docs

---

**Questions?** Check the relevant documentation file or search the codebase!

**Version:** 2.0.0 (Windows Edition)  
**Platform:** Windows (also works on Linux via master branch)  
**Status:** Production Ready ✅
