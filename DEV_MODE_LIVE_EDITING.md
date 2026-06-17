# Dev Mode with Live Editing - Complete Guide

This guide shows you how to run clipd in development mode and see your changes instantly.

---

## 🚀 Quick Start (3 Steps)

### Step 1: Prerequisites Setup (One Time)

Install frontend dependencies:
```powershell
cd C:\Users\syeda\Desktop\development\clipboard-history
npm --prefix frontend install
npm --prefix frontend run build
```

This installs all npm packages and builds the frontend. Do this only once (or when dependencies change).

### Step 2: Start Dev Server

```powershell
npm run dev
```

**What happens:**
- ✅ Wails dev server starts
- ✅ App window opens automatically
- ✅ Browser DevTools available (F12)
- ✅ Live reload enabled for frontend

### Step 3: Test It

Press **Win+V** in the dev window to show clipboard history. Copy something to your clipboard and watch it appear!

---

## 📝 Live Editing Setup

### Frontend Changes (TypeScript/CSS) - Auto Reload ✨

These changes appear **instantly** without restarting:

**File locations:**
```
frontend/src/
├── main.ts              ← Main app entry
├── components/
│   ├── searchBar.ts     ← Search box
│   ├── itemList.ts      ← Clipboard items display
│   ├── settingsModal.ts ← Settings dialog
│   ├── vaultPanel.ts    ← Vault panel
│   └── ...
└── styles/
    ├── main.css         ← Global styles
    └── components.css   ← Component styles
```

**How to edit:**
1. Open file in your editor (VS Code recommended)
2. Make changes
3. Save file (Ctrl+S)
4. Watch the dev window update **instantly!** ✨

**Example:** Change app title in `frontend/src/main.ts`:

```typescript
// Find this line:
document.title = "clipd"

// Change to:
document.title = "My Clipboard Manager"

// Save → Title changes in dev window immediately!
```

---

### Backend Changes (Go) - Requires Restart

These changes need a restart:

**File locations:**
```
internal/
├── service/
│   └── service.go       ← Core business logic
├── clipboard/
│   └── watcher_windows.go   ← Clipboard monitoring
├── config/
│   └── config.go        ← Configuration
└── ...
```

**How to edit:**
1. Open file in your editor
2. Make changes
3. **Stop dev server** (Press Ctrl+C in PowerShell)
4. Run `npm run dev` again
5. Changes take effect

**Example:** Add logging to understand clipboard capture:

```go
// In internal/clipboard/watcher_windows.go, add:
log.Printf("DEBUG: Found clipboard item: %s", text)

// Save → Stop dev server (Ctrl+C) → npm run dev → See logs in console
```

---

## 🎯 Common Dev Tasks

### Task 1: Edit UI (Frontend)

**Goal:** Change how items are displayed

**Steps:**
1. Open `frontend/src/components/itemList.ts`
2. Find the HTML template
3. Change styling or structure
4. Save
5. Dev window updates instantly! ✨

**Example - Increase item height:**
```typescript
// In itemList.ts, find:
className="item h-12"

// Change to:
className="item h-16"

// Save → Items are now taller in dev window!
```

---

### Task 2: Edit Styles (CSS)

**Goal:** Change colors, spacing, layout

**Steps:**
1. Open `frontend/src/styles/main.css` or `components.css`
2. Modify CSS
3. Save
4. Dev window updates instantly! ✨

**Example - Dark theme adjustments:**
```css
/* In main.css, find: */
:root {
  --bg: #161718;
  --text: #ffffff;
}

/* Change to: */
:root {
  --bg: #0a0a0a;
  --text: #e0e0e0;
}

/* Save → Dev window instantly shows new colors! */
```

---

### Task 3: Add Logging (Backend)

**Goal:** Debug clipboard capture behavior

**Steps:**
1. Open `internal/clipboard/watcher_windows.go`
2. Add `log.Printf()` statements
3. Save
4. Stop dev server (Ctrl+C)
5. Run `npm run dev` again
6. Logs appear in PowerShell console

**Example:**
```go
func readTextWindows(ctx context.Context) (string, error) {
    // Add logging
    log.Printf("DEBUG: Reading text from clipboard...")
    
    if !openClipboard() {
        log.Printf("DEBUG: Failed to open clipboard")
        return "", fmt.Errorf("failed to open clipboard")
    }
    defer closeClipboard()
    
    log.Printf("DEBUG: Clipboard opened successfully")
    // ... rest of function
}

// When you copy text to clipboard:
// Logs appear in PowerShell:
// 2024/06/17 15:23:45 DEBUG: Reading text from clipboard...
// 2024/06/17 15:23:45 DEBUG: Clipboard opened successfully
```

---

### Task 4: Change App Settings

**Goal:** Modify default configuration

**Steps:**
1. Open `internal/clipboard/types.go` (look for `DefaultSettings()`)
2. Change default values
3. Save
4. Stop dev server (Ctrl+C)
5. Run `npm run dev` again
6. Restart clipboard app (close and reopen)
7. New defaults apply

**Example - Change default hotkey:**
```go
// In DefaultSettings():
return Settings{
    Hotkey:      "Ctrl+Shift+V",  // Changed from "Super+V"
    MaxItems:    300,               // Changed from 200
    Theme:       "dark",            // Changed from "auto"
    AutoPaste:   false,             // Changed from true
    // ... rest
}

// Save → Stop dev server → npm run dev → Restart app → New defaults!
```

---

## 🔍 Debugging Tips

### See Console Logs

**Frontend logs:**
- Press **F12** in dev window
- Go to **Console** tab
- See all JavaScript logs and errors

**Backend logs:**
- Check **PowerShell** terminal where dev server is running
- All Go `log.Printf()` statements appear there

**Example workflow:**
```powershell
# Terminal shows:
2024/06/17 15:23:45 clipd: hotkey registered
2024/06/17 15:23:46 watcher start: clipboard monitor started
2024/06/17 15:23:47 DEBUG: Reading text from clipboard...
2024/06/17 15:23:47 ingest text: hash abc123
```

### Open DevTools for UI Debugging

**Steps:**
1. Press **F12** in dev window
2. **Elements** tab: Inspect HTML structure
3. **Console** tab: See JavaScript errors and logs
4. **Network** tab: See API calls (desktop ↔ backend)
5. **Application** tab: Check LocalStorage, cookies

**Example - Debug why search isn't working:**
1. Press F12
2. Console tab
3. You'll see any JavaScript errors
4. Fix the error in frontend code
5. Save → Console shows it's fixed! ✨

---

## 🔄 Dev Workflow Example

Let's say you want to **make the search bar bigger and change its placeholder text**:

### Step 1: Start Dev Server
```powershell
npm run dev
# App window opens with live reload
```

### Step 2: Edit Frontend (Live Updates)
```powershell
# Open editor: frontend/src/components/searchBar.ts

# Find:
<input type="text" placeholder="Search history...">

# Change to:
<input type="text" placeholder="🔍 Type to search..." style="font-size: 1.2em; padding: 12px;">

# Save (Ctrl+S)
```

### Step 3: Watch It Update
- ✨ Dev window instantly shows larger search bar
- ✨ New placeholder text appears
- ✨ No restart needed!

### Step 4: Press Win+V to Test
- Open clipboard history
- Type in search bar
- See your changes in real-time!

---

## 🛠️ Full Dev Cycle Example

**Scenario:** You want to add a "Clear All" button to the history

### Step 1: Backend Changes (Needs Restart)
```powershell
# Stop current dev server (Ctrl+C)

# Edit: internal/service/service.go
# Add a new method: ClearAllHistory()

# Then:
npm run dev
# Server restarts, backend code updated
```

### Step 2: Frontend Changes (Live Update)
```powershell
# Edit: frontend/src/components/itemList.ts
# Add button HTML: <button onclick="clearAll()">Clear All</button>

# Edit: frontend/src/main.ts
# Add click handler

# Save (Ctrl+S)
# Dev window updates instantly! ✨
```

### Step 3: Test It
- Press Win+V
- Click "Clear All" button
- See it work in real-time!

---

## 📋 Typical Dev Session

```powershell
# 1. Start dev server
npm run dev
# App window opens...

# 2. Edit frontend files (instant updates)
# - Open frontend/src/main.ts
# - Make changes
# - Save (Ctrl+S)
# - Watch dev window update instantly!

# 3. Need to change backend?
# - Press Ctrl+C in PowerShell to stop
# - Edit internal/service/service.go
# - npm run dev to restart
# - Changes take effect

# 4. Testing
# - Press Win+V in dev window
# - Copy items to clipboard
# - See changes in real-time
# - Use F12 DevTools if needed

# 5. When done
# - Press Ctrl+C to stop dev server
# - Make release build: npm run build
```

---

## 🚀 Advanced: Edit Multiple Files Simultaneously

**VS Code Setup (Recommended):**

1. Open folder: `C:\Users\syeda\Desktop\development\clipboard-history`
2. Split editor (Click and drag tab to split)
3. Left side: `frontend/src/main.ts` (frontend code)
4. Right side: `frontend/src/styles/main.css` (styles)
5. Edit left → Changes appear instantly on right
6. Edit right → Styles update instantly

**PowerShell:**
- Keep running in background
- Watch for errors/logs
- Press Ctrl+C to stop dev server

---

## ⚡ Pro Tips

### Tip 1: Use VS Code
- IntelliSense for TypeScript
- Live preview of CSS
- Integrated terminal
- Keyboard shortcuts

### Tip 2: Browser DevTools
- Press F12 in dev window
- Inspect HTML structure
- Edit CSS live (doesn't save, just for testing)
- Check console for errors

### Tip 3: Hot Reload
- Only frontend reloads automatically
- Backend needs restart
- Plan edits accordingly

### Tip 4: Check File Paths
- Frontend: `frontend/src/` (TypeScript/CSS)
- Backend: `internal/*/` (Go)
- Make sure you're editing the right one!

---

## 🔧 Troubleshooting Dev Mode

### Dev Server Won't Start
```powershell
# Make sure Wails is installed:
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Then try again:
npm run dev
```

### Changes Not Appearing
**Frontend (TypeScript/CSS):**
- Make sure you saved the file (Ctrl+S)
- Check that you're editing in `frontend/src/`
- Refresh dev window (Ctrl+R)

**Backend (Go):**
- Restart dev server (Ctrl+C → npm run dev)
- Check PowerShell console for errors
- Make sure you edited the right file

### App Window Won't Open
```powershell
# Kill any existing clipd processes
npm run stop

# Clear dev data
npm run reset:dev

# Try again
npm run dev
```

### DevTools Not Opening (F12)
- Make sure dev server is running
- F12 might be captured by another app
- Try in different dev window positions
- Check for VS Code shortcuts conflict

---

## 📚 Quick Reference

| Action | Result | Time |
|--------|--------|------|
| Edit `frontend/src/` → Save | Changes appear instantly | < 1 sec ✨ |
| Edit `internal/` → Save | Nothing changes yet | - |
| Stop dev server (Ctrl+C) | Backend ready to reload | ~1 sec |
| `npm run dev` again | Backend changes take effect | ~5-10 sec |
| Press Win+V | Show clipboard (test your changes) | Instant |
| Press F12 | Open DevTools (debug UI) | Instant |
| Ctrl+Shift+I | Toggle DevTools | Instant |

---

## 🎯 Now You're Ready!

```powershell
# 1. Install dependencies (one time)
npm --prefix frontend install
npm --prefix frontend run build

# 2. Start dev server
npm run dev

# 3. Edit frontend files in your editor
# Files are in: frontend/src/

# 4. Save (Ctrl+S)
# Changes appear instantly in dev window!

# 5. Test with Win+V
# Try your clipboard history feature!

# 6. Done? Stop with Ctrl+C
```

**Enjoy live editing! 🎉**

---

**Pro Tip:** Keep a split screen:
- **Left:** VS Code with your editor
- **Right:** Dev window showing live changes
- **Bottom:** PowerShell showing logs

Now when you edit code, you see changes instantly!
