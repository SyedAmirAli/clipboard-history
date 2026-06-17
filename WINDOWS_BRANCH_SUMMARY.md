# Windows Branch Summary

This document summarizes the Windows edition of clipd - a complete Windows-native port with Windows-first documentation and build scripts.

## 🎯 Branch Purpose

The `windows` branch is **Windows-dedicated** while keeping the codebase cross-platform. This means:

- ✅ **Windows primary:** All docs, scripts, and examples are Windows-first
- ✅ **Linux support:** Cross-platform Go code with build tags still works on Linux  
- ✅ **Clean separation:** Different branches for different platforms
- ✅ **Master branch unchanged:** Linux users stay on master

---

## 📚 Documentation (Windows-Focused)

### For Users
- **[README.md](README.md)** — Start here! Overview, features, quick start
- **[GETTING_STARTED_WINDOWS.md](GETTING_STARTED_WINDOWS.md)** — Step-by-step user guide
- **[USAGE.md](USAGE.md)** — Complete features and keyboard shortcuts
- **[WINDOWS.md](WINDOWS.md)** — Build, install, troubleshooting

### For Developers
- **[RUN_DEV_WINDOWS.md](RUN_DEV_WINDOWS.md)** — 5-step dev server setup
- **[QUICK_START.md](QUICK_START.md)** — Command reference
- **[DEV_SETUP.md](DEV_SETUP.md)** — Comprehensive development guide
- **[DOCS.md](DOCS.md)** — Documentation index & navigation

### For Understanding Architecture
- **[WINDOWS_PORT_SUMMARY.md](WINDOWS_PORT_SUMMARY.md)** — Windows port details

---

## 🔧 Build System (Windows-Only)

### Updated `package.json`
Simple, Windows-native npm scripts:

```json
{
  "scripts": {
    "dev": "wails dev",
    "build": "wails build", 
    "start": ".\\build\\bin\\clipd.exe",
    "show": ".\\build\\bin\\clipd.exe show",
    "stop": ".\\build\\bin\\clipd.exe quit",
    "test": "go test ./... -v",
    "vault:reset": ".\\build\\bin\\clipd.exe reset-vault",
    "reset:dev": "rmdir /s /q %TEMP%\\clipd-dev-data 2>nul || true"
  }
}
```

All paths use Windows backslashes (`.\`), no Linux environment variables.

### Build Script
- **New:** `scripts/build-windows.bat` — Native Windows batch script
- **Old:** `scripts/build.sh` — Linux script (can be removed on windows branch)

---

## 🏗️ Cross-Platform Code

The codebase supports both Windows and Linux using **Go build tags**:

### Platform-Specific Files
```
internal/
├── autostart/
│   ├── autostart.go           (common types)
│   ├── autostart_windows.go   (Windows Registry)
│   └── autostart_linux.go     (XDG .desktop files)
├── clipboard/
│   ├── watcher.go             (common types)
│   ├── watcher_windows.go     (Win32 API)
│   ├── watcher_linux.go       (xclip/wl-paste)
│   ├── paster_windows.go      (keybd_event)
│   ├── paster_linux.go        (xdotool)
│   ├── writer_windows.go      (Win32 clipboard)
│   └── writer_linux.go        (xclip/wl-copy)
├── ipc/
│   ├── ipc.go                 (common types)
│   ├── ipc_windows.go         (named pipes)
│   └── ipc_linux.go           (Unix sockets)
└── shortcut/
    ├── gnome.go               (common types)
    ├── shortcut_windows.go    (no-op, uses hotkey)
    └── shortcut_linux.go      (GNOME gsettings)
```

**How it works:** Go compiler automatically selects `*_windows.go` or `*_linux.go` based on build target.

---

## 🚀 How to Use

### Run clipd
```powershell
# Download and run
.\clipd.exe

# Or use npm
npm run start
```

### Start Development
```powershell
npm run dev
# App opens with live reload, press Win+V to test
```

### Build for Release
```powershell
npm run build
# Output: .\build\bin\clipd.exe
```

### Use CLI Commands
```powershell
.\build\bin\clipd.exe show       # Show clipboard popup
.\build\bin\clipd.exe toggle     # Show/hide
.\build\bin\clipd.exe hide       # Hide popup
.\build\bin\clipd.exe quit       # Quit app
.\build\bin\clipd.exe restart    # Restart
.\build\bin\clipd.exe reset-vault # Clear vault
```

---

## 📋 What's On This Branch

### ✅ Complete
- ✅ Full Windows port (Win32 APIs)
- ✅ Windows-native npm scripts (PowerShell-compatible)
- ✅ Windows-native build script (`build-windows.bat`)
- ✅ Comprehensive user documentation
- ✅ Complete developer guides
- ✅ GETTING_STARTED guide for end users
- ✅ Package.json simplified for Windows
- ✅ All platform-specific code with build tags

### ⚠️ Optional Cleanup (Safe to Remove)
On the windows branch, these Linux-only files can be removed if desired:

```
scripts/build.sh               # Linux build script (has build-windows.bat alternative)
internal/autostart/autostart_linux.go
internal/clipboard/paster_linux.go
internal/clipboard/watcher_linux.go
internal/clipboard/writer_linux.go
internal/ipc/ipc_linux.go
internal/shortcut/shortcut_linux.go
```

**Why:** These files use `//go:build linux` tags, so they won't compile on Windows anyway. Removing them keeps the Windows branch "clean" and Windows-focused.

**Note:** Keep them if you want the ability to build Linux binaries from this branch. Removing them forces the branch to be Windows-only.

---

## 🎯 To Remove Linux Files from Windows Branch

```powershell
git rm scripts/build.sh
git rm internal/autostart/autostart_linux.go
git rm internal/clipboard/paster_linux.go
git rm internal/clipboard/watcher_linux.go
git rm internal/clipboard/writer_linux.go
git rm internal/ipc/ipc_linux.go
git rm internal/shortcut/shortcut_linux.go
git commit -m "clean: remove Linux-specific code from windows branch

This branch is now Windows-only. Linux users should use the master branch."
```

---

## 🚀 Next Steps

1. **Clone and test:**
   ```powershell
   git clone https://github.com/yourusername/clipd.git
   cd clipd
   git checkout windows
   npm run dev
   ```

2. **Try the hotkey:** Press **Win+V**

3. **Explore settings:** Click the gear icon

4. **Use the CLI:**
   ```powershell
   npm run show  # Show popup
   npm run stop  # Quit
   ```

5. **Share feedback** on [GitHub Issues](https://github.com/yourusername/clipd/issues)

---

## ✨ Summary

The `windows` branch is a **complete, production-ready Windows edition of clipd** with:
- Full Windows support with native APIs
- Windows-first documentation and examples
- Simplified npm scripts for PowerShell
- Same high-quality code as master branch
- Easy to build, easy to use, easy to develop

**Get started:** `npm run dev` → Press **Win+V** → Enjoy! 🎉

---

**Version:** 2.0.0 (Windows Edition)  
**Status:** ✅ Production Ready  
**Platform:** Windows 10+ (Primary)  
**Last Updated:** June 2026
