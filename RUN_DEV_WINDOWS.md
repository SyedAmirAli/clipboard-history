# Running Dev Server on Windows

## Quick Start

### Step 1: Build the Frontend
```powershell
npm --prefix frontend install
npm --prefix frontend run build
```

### Step 2: Start Dev Server
```powershell
npm run dev
```

Or explicitly:
```powershell
npm run dev:windows
```

**The app will open automatically in a dev window!**

### Step 3: Test the Popup
Press **Win+V** in the dev window to show/hide the clipboard popup.

### Step 4: Make Changes & Test
- **Frontend** (`frontend/src/`): Restart dev server to see changes
- **Backend** (`internal/*/`): Restart dev server to see changes

---

## Commands for Windows

### Running the App

**Start dev server (opens app window):**
```powershell
npm run dev
```

**Run built binary from terminal:**
```powershell
npm run start
```

**Show clipboard popup (if already running):**
```powershell
npm run show
```

**Quit running instance:**
```powershell
npm run stop
```

### Building

**Build binary:**
```powershell
npm run build
```

Output: `.\build\bin\clipd.exe`

### Testing & Cleanup

**Run tests:**
```powershell
npm test
```

**Clear dev data:**
```powershell
npm run reset:dev
```

---

## Troubleshooting

### "wails dev" won't start

**Solution:** Install Wails v2
```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### App window won't open

**Solution 1:** Check for errors in terminal
```powershell
npm run dev
# Look for error messages
```

**Solution 2:** Kill any existing clipd process
```powershell
npm run stop
npm run reset:dev
npm run dev
```

### Clipboard not capturing

**Solution:** Restart dev server
```powershell
npm run stop
npm run reset:dev
npm run dev
```

### "Port already in use" error

**Solution:** Wait a few seconds and try again
```powershell
npm run stop
# Wait 3-5 seconds
npm run dev
```

### Changes not appearing

**For frontend changes:** Restart dev server
```powershell
# Press Ctrl+C in dev window terminal
npm run dev
```

**For backend changes:** Restart dev server
```powershell
# Press Ctrl+C in dev window terminal
npm run dev
```

---

## Working with Dev Window

Once `npm run dev` opens the dev window:

### Keyboard Shortcuts
| Key | Action |
|-----|--------|
| **Win+V** | Show/hide clipboard popup |
| **F12** | Open DevTools (Chrome developer tools) |
| **Ctrl+R** | Reload window |
| **Ctrl+Shift+I** | Toggle DevTools |
| **Ctrl+Q** | Close app |

### Using DevTools
- **Console**: See JavaScript errors and logs
- **Elements**: Inspect HTML structure
- **Network**: Check backend API calls
- **Application**: View LocalStorage, cookies

### Testing Workflow
1. Copy something to your clipboard
2. Press **Win+V** to show clipd popup
3. Check if item appears in history
4. Click item to paste it back
5. Check console for any errors (F12)

---

## Development Tips

### Add Backend Logging
Edit `internal/service/service.go` or relevant file:
```go
log.Printf("debug: value = %v", value)
```

Logs appear in the terminal where you ran `npm run dev`.

### Frontend Changes Need Restart
Edit `frontend/src/main.ts` or any component:
```
Ctrl+C (stop dev server)
npm run dev (restart)
```

### CLI Testing While Dev Server Runs
Open **another PowerShell window** in the same directory:
```powershell
# Test clipboard commands
.\build\bin\clipd.exe show
.\build\bin\clipd.exe hide
.\build\bin\clipd.exe toggle
.\build\bin\clipd.exe quit
```

---

## Building Release Binary

When ready to release:

```powershell
npm run build
```

Output: `.\build\bin\clipd.exe`

This is a production binary ready to distribute.

---

## Next Steps

1. ✅ Start dev server: `npm run dev`
2. ✅ Test popup: Press **Win+V**
3. ✅ Make changes to code
4. ✅ Restart dev server to see changes
5. ✅ Run tests: `npm test`
6. ✅ Build release: `npm run build`

Need help? See [QUICK_START.md](QUICK_START.md) or [DEV_SETUP.md](DEV_SETUP.md).
