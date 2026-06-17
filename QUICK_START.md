# Quick Start for Developers

## TL;DR - Get Running in 5 Minutes

### 1. Install Prerequisites (one-time)
```bash
# Install Go 1.22+, Node.js, and Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 2. Clone & Setup
```bash
git clone https://github.com/yourusername/clipd.git
cd clipd
git checkout windows  # Get Windows support
npm install --prefix frontend
```

### 3. Start Dev Server
```powershell
# Windows
npm run dev:windows

# Linux
npm run dev:linux
```

App launches automatically with live reload. Press **Win+V** to test.

### 4. Make Changes
- **Frontend** (`frontend/src/`): Changes appear instantly ✨
- **Backend** (`internal/*/`): Restart dev server after changes

### 5. Build Release
```powershell
npm run build:windows
# Output: .\build\bin\clipd.exe
```

Done! 🎉

---

## One-Line Reference

| Task | Windows | Linux |
|------|---------|-------|
| **Start dev** | `npm run dev:windows` | `npm run dev:linux` |
| **Run binary** | `npm run start` | `npm run start:linux` |
| **Show popup** | `npm run show` | `npm run show:linux` |
| **Quit app** | `npm run stop` | `npm run stop:linux` |
| **Build binary** | `npm run build:windows` | `npm run build:bin` |
| **Build package** | - | `npm run build:deb` |
| **Run tests** | `npm test` | `npm test` |
| **Reset dev data** | `npm run reset:dev` | `npm run reset:dev:linux` |

---

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| **Win+V** / **Super+V** | Show/hide clipboard popup |
| **F12** (in dev window) | Open DevTools |
| **Ctrl+R** | Reload window |
| **Ctrl+Q** | Close app |
| **Enter** | Paste selected item |
| **Space** | Pin/unpin item |
| **Delete** | Delete item |

---

## Common Issues & Fixes

### "wails dev" won't start
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Clipboard not capturing
```bash
npm run stop           # Kill any running instance
npm run reset:dev      # Clear dev data
npm run dev:windows    # Fresh start
```

### Changes not appearing
- **Frontend**: Close dev window, wait 3 sec, restart
- **Backend**: Stop server (Ctrl+C), edit file, restart: `npm run dev:windows`

### "port in use" error
```bash
npm run stop           # Kill existing instance
# Wait 2-3 seconds
npm run dev:windows
```

---

## File You'll Edit Most

| File | Purpose |
|------|---------|
| `frontend/src/main.ts` | UI entry point |
| `frontend/src/components/*.ts` | UI components |
| `frontend/src/styles/main.css` | Styling |
| `internal/service/service.go` | Core business logic |
| `internal/clipboard/watcher_*.go` | Clipboard capture |
| `main.go` | App initialization |

---

## Debug Tips

### See Backend Logs
Logs appear in the dev server terminal. Look for:
```
2024/01/15 14:23:45 clipd: hotkey registered
2024/01/15 14:23:46 ingest text: hash abc123
```

### Check Frontend Console
Open DevTools in the dev window (F12) → Console tab

### Test from CLI
```powershell
# From another terminal while dev server runs:
.\build\bin\clipd.exe toggle
.\build\bin\clipd.exe show
.\build\bin\clipd.exe quit
```

### Force Fresh State
```bash
npm run reset:dev && npm run dev:windows
```

---

## Next Commands

- **Help**: See [DEV_SETUP.md](DEV_SETUP.md) for detailed guide
- **Architecture**: See [README.md](README.md)
- **Windows notes**: See [WINDOWS.md](WINDOWS.md)
- **User guide**: See [USAGE.md](USAGE.md)

---

**Happy coding!** 🚀
