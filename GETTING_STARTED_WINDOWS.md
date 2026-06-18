# Getting Started with clipd on Windows

Welcome! This guide shows you how to **download, run, and use** clipd on Windows.

---

## 📥 Installation (Easiest Way)

### 1. Download

-   Go to [Releases](https://github.com/yourusername/clipd/releases)
-   Download `clipd.exe` (Windows 11 recommended)

### 2. Run

-   Double-click `clipd.exe`
-   Windows may ask for permission — click **Run**

### 3. Done! ✨

-   Icon appears in system tray (bottom-right corner)
-   Press **Win+V** to open clipboard history

That's it! No installation required, no administrator privileges needed.

---

## 🎮 Using clipd

### Opening Clipboard History

| Method            | How                                              |
| ----------------- | ------------------------------------------------ |
| **Global Hotkey** | Press **Win+V** (default)                        |
| **System Tray**   | Right-click tray icon → "Open clipboard history" |
| **Command**       | Open PowerShell: `.\build\bin\clipd.exe show`    |

### Keyboard Shortcuts (In Popup)

| Shortcut   | Action                            |
| ---------- | --------------------------------- |
| **Win+V**  | Show/hide history                 |
| **Enter**  | Paste selected item               |
| **Space**  | Pin/unpin item (keeps it forever) |
| **Delete** | Delete item from history          |
| **Escape** | Close popup                       |
| **Ctrl+A** | Select all (in search box)        |

### Mouse Actions

| Action               | Result                                     |
| -------------------- | ------------------------------------------ |
| **Click item**       | Paste it into your app                     |
| **Right-click item** | Show options (copy, delete, move to vault) |
| **Click "Copy"**     | Copy without pasting                       |
| **Scroll**           | Scroll through history                     |

---

## ⚙️ Settings

### Open Settings

-   Click the gear icon in top-right of clipboard popup
-   Or: Right-click tray icon → Settings

### Available Settings

| Setting                 | Options                                  | Default  |
| ----------------------- | ---------------------------------------- | -------- |
| **Hotkey**              | Any key combination (e.g., Ctrl+Shift+V) | Win+V    |
| **Theme**               | Light / Dark / Auto                      | Auto     |
| **Max Items**           | 50-500                                   | 200      |
| **Keep Images**         | On/Off                                   | On       |
| **Max Image Size**      | 1-10 MB                                  | 5 MB     |
| **Auto-paste**          | On/Off                                   | On       |
| **Window Style**        | Floating popup / Normal window           | Floating |
| **Hide on blur**        | On/Off                                   | On       |
| **Auto-start on login** | On/Off                                   | Off      |

---

## 🔐 Private Vault

Store sensitive data (passwords, API keys, credit cards) securely encrypted.

### Set Up Vault

1. In clipboard history, click the **lock icon** (top-right)
2. Choose password OR PIN + authenticator
3. Confirm setup

### Add Item to Vault

1. Copy sensitive data to clipboard
2. It appears in history
3. Click → "Move to vault"
4. Item is encrypted and removed from regular history

### Access Vault

1. Click **lock icon** in popup
2. Enter password/PIN (or scan authenticator)
3. View encrypted items

---

## 💡 Tips & Tricks

### Pin Items

Items get deleted after 200 entries (default). To keep items forever:

-   Click **space bar** while selecting item (adds a pin)
-   Pinned items never get deleted

### Search History

-   Type in search box at top
-   Filters all items by text
-   Use "All/Text/Images/Pinned" chips to narrow down

### Multi-Select

-   Hold **Shift** and click to select range
-   Hold **Ctrl** and click to add/remove individual items
-   Bulk delete with **Delete** key

### Copy Without Pasting

-   Right-click item → "Copy only"
-   OR: Click "Copy" button (doesn't auto-paste)

### Clear History

-   Click menu (three dots) → "Clear history"
-   Confirms before deleting (pinned items stay)

---

## 🚀 Command Line

Use the `clipd` command from PowerShell for automation or scripting.

### Available Commands

```powershell
.\build\bin\clipd.exe toggle      # Show/hide popup
.\build\bin\clipd.exe show        # Show popup
.\build\bin\clipd.exe hide        # Hide popup
.\build\bin\clipd.exe quit        # Quit app
.\build\bin\clipd.exe restart     # Restart app
.\build\bin\clipd.exe reset-vault # Clear vault (not history)
```

### Add to PATH (Optional)

To run `clipd` from anywhere:

1. Find path to `clipd.exe` (e.g., `C:\Users\You\Desktop\clipd.exe`)
2. Right-click **Start** → Settings → System → About → Advanced system settings
3. Click **Environment Variables**
4. Under "User variables", click **New**
5. Variable name: `PATH`
6. Variable value: `C:\path\to\clipd` (the folder containing clipd.exe)
7. Click **OK** × 3
8. Open new PowerShell window
9. Now you can run: `clipd toggle`

---

## 🔧 Troubleshooting

### Clipboard Not Capturing?

```powershell
# Restart clipd
.\build\bin\clipd.exe quit
.\build\bin\clipd.exe    # Restart
```

### Hotkey Not Working?

1. Check Settings → make sure hotkey is set
2. Try changing hotkey (maybe another app uses Win+V)
3. Restart: `.\build\bin\clipd.exe restart`

### App Won't Start?

-   Check if another clipd is running: `.\build\bin\clipd.exe quit`
-   Delete config folder: `%APPDATA%\clipd\` (backup first!)
-   Reinstall clipd.exe

### Vault Locked Out?

-   If you forget password: `.\build\bin\clipd.exe reset-vault`
-   This deletes vault but keeps history

### Auto-paste Not Working?

-   Some apps don't allow simulated Ctrl+V (for security)
-   Workaround: Copy item, then manually Ctrl+V
-   Or toggle "Auto-paste" in Settings

---

## 📁 Where Stuff Lives

| Item              | Location                   |
| ----------------- | -------------------------- |
| Clipboard history | `%APPDATA%\clipd\clipd.db` |
| Settings          | Same database              |
| Vault             | Same database (encrypted)  |
| Logs              | Command line output        |

To backup:

```powershell
# Copy entire folder
xcopy "%APPDATA%\clipd" "C:\Backup\clipd" /S /Y
```

---

## 🎛️ Auto-Start on Login

Enable so clipd runs automatically when you boot:

### Method 1: In App Settings

-   Open clipboard popup (Win+V)
-   Click gear icon
-   Toggle "Auto-start on login" → ON
-   Done!

### Method 2: Windows Settings

1. Right-click **Start** → Settings → Apps → Startup
2. Toggle **clipd** → ON

### Check If It's Working

-   Restart computer
-   After login, check system tray for clipd icon

---

## 📊 Data & Privacy

### What clipd Collects

-   **Only your clipboard contents** — stored locally in `%APPDATA%\clipd\`
-   No internet connection
-   No tracking or analytics
-   No data sent anywhere

### What clipd Doesn't Do

-   ❌ Upload to cloud
-   ❌ Track you
-   ❌ Show ads
-   ❌ Collect telemetry
-   ❌ Access other files

### Clearing Everything

```powershell
# Remove entire clipboard history
rmdir "%APPDATA%\clipd" /s
# clipd will recreate on next run with empty history
```

---

## 🆘 Getting Help

### Common Questions

-   **Q:** Can I customize the hotkey?  
     **A:** Yes! Settings → Hotkey → pick any combination

-   **Q:** Does it work with images?  
     **A:** Yes! PNG images are captured and shown as thumbnails

-   **Q:** Is it safe?  
     **A:** Totally! Local-only, no internet, open source

-   **Q:** Can I use it on Mac/Linux?  
     **A:** Linux has a separate version. Mac is not yet supported.

### Report Issues

-   Found a bug? Go to [GitHub Issues](https://github.com/yourusername/clipd/issues)
-   Include: Windows version, what you were doing, error message

### More Docs

-   **[README.md](README.md)** — Overview
-   **[USAGE.md](USAGE.md)** — Complete user manual
-   **[WINDOWS.md](WINDOWS.md)** — Windows-specific features

---

## 🚀 Enjoy

You're all set. Start using clipd:

1. **Press Win+V** to open history
2. **Explore settings** to customize
3. **Use vault** for sensitive data
4. **Enable auto-start** if you like

Any questions? See the docs or file an issue on GitHub.

**Happy clipboard managing!** 📋✨
