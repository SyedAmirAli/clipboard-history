# clipd GNOME Wayland Clipboard Resolution

This document records how the GNOME/Wayland clipboard flicker issue was
resolved and how the current architecture is meant to work.

## Problem

The original application mixed two responsibilities in the same process:

- the Wails GUI displayed clipboard history;
- the same GUI process also listened to clipboard changes.

On GNOME Wayland this caused repeated focus/activation side effects. The
visible symptoms were notification spam, dock/taskbar blinking, screen flicker,
and the `clipd` window behaving like it was constantly being reactivated.

`wl-paste --watch` was also tested, but GNOME Mutter does not support the
wlroots data-control watch mode. The observed error was:

```text
Watch mode requires a compositor that supports the wlroots data-control protocol
```

So a standalone `wl-paste --watch` daemon is not a reliable solution on GNOME.

## Final Architecture

The working solution separates capture and display:

- GNOME Shell extension: listens to clipboard text changes inside GNOME Shell.
- Python helper: short-lived DB writer called by the extension.
- SQLite DB: shared persistence at `~/.local/share/clipd/clipd.db`.
- Wails GUI: does not watch the clipboard. It only lists DB rows and writes a
  selected row back to the system clipboard.

This keeps the GUI idle while hidden and removes the clipboard-polling behavior
that triggered GNOME focus/notification issues.

## Installed Files

Packaged files:

```text
/usr/bin/clipd
/opt/clipd-watcher/main.py
/usr/share/gnome-shell/extensions/clipd-watcher@syedamirali/metadata.json
/usr/share/gnome-shell/extensions/clipd-watcher@syedamirali/extension.js
/usr/share/applications/clipd.desktop
/etc/xdg/autostart/clipd.desktop
```

There is intentionally no `clipd-watch.service` systemd unit. The old systemd
watcher path was removed because it still relied on external clipboard polling
and reproduced the GNOME issue.

## Login Startup

`/etc/xdg/autostart/clipd.desktop` starts the GUI resident process at login:

```ini
[Desktop Entry]
Type=Application
Name=clipd
GenericName=Clipboard History
Comment=Start clipd in the background
Exec=/usr/bin/clipd start
Icon=clipd
Terminal=false
Categories=Utility;
StartupNotify=false
X-GNOME-Autostart-enabled=true
```

At startup, `clipd` also performs best-effort user-session setup:

- enables `clipd-watcher@syedamirali` with `gnome-extensions enable`;
- falls back to adding the UUID to `org.gnome.shell enabled-extensions`;
- installs/updates the GNOME Super+V keybinding.

The GUI startup does not start a clipboard polling loop. Enabling the extension
only tells GNOME Shell to load the external listener.

## GNOME Shell Extension

Source path in this repo:

```text
build/linux/gnome-extension/clipd-watcher@syedamirali/
```

Installed path:

```text
/usr/share/gnome-shell/extensions/clipd-watcher@syedamirali/
```

`metadata.json`:

```json
{
  "uuid": "clipd-watcher@syedamirali",
  "name": "clipd Clipboard Watcher",
  "description": "Captures GNOME Shell clipboard text changes into clipd's SQLite database.",
  "shell-version": ["50"],
  "version": 1
}
```

`extension.js`:

```js
import Gio from 'gi://Gio';
import GLib from 'gi://GLib';
import St from 'gi://St';

import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';

const CLIPBOARD_TYPE = St.ClipboardType.CLIPBOARD;
const HELPER = '/opt/clipd-watcher/main.py';
const PYTHON = '/usr/bin/python3';
const POLL_SECONDS = 1;

export default class ClipdWatcherExtension extends Extension {
    enable() {
        this._clipboard = St.Clipboard.get_default();
        this._lastText = null;
        this._busy = false;
        this._timeoutId = GLib.timeout_add_seconds(
            GLib.PRIORITY_DEFAULT,
            POLL_SECONDS,
            () => {
                this._checkClipboard();
                return GLib.SOURCE_CONTINUE;
            });
        this._checkClipboard();
    }

    disable() {
        if (this._timeoutId) {
            GLib.source_remove(this._timeoutId);
            this._timeoutId = 0;
        }
        this._clipboard = null;
        this._lastText = null;
        this._busy = false;
    }

    _checkClipboard() {
        if (!this._clipboard || this._busy)
            return;
        this._busy = true;
        this._clipboard.get_text(CLIPBOARD_TYPE, (...args) => {
            this._busy = false;
            const text = args[args.length - 1];
            if (typeof text !== 'string' || text.length === 0)
                return;
            if (text === this._lastText)
                return;
            this._lastText = text;
            this._insertText(text);
        });
    }

    _insertText(text) {
        try {
            const proc = Gio.Subprocess.new(
                [PYTHON, HELPER, '--insert-text'],
                Gio.SubprocessFlags.STDIN_PIPE |
                Gio.SubprocessFlags.STDOUT_SILENCE |
                Gio.SubprocessFlags.STDERR_SILENCE);
            proc.communicate_utf8_async(text, null, (p, res) => {
                try {
                    p.communicate_utf8_finish(res);
                } catch (e) {
                    console.error(`clipd watcher helper failed: ${e}`);
                }
            });
        } catch (e) {
            console.error(`clipd watcher failed to spawn helper: ${e}`);
        }
    }
}
```

Current limitation: this extension captures text only. Image capture is not
implemented yet.

## Python DB Helper

Source path in this repo:

```text
build/linux/clipd-watcher/main.py
```

Installed path:

```text
/opt/clipd-watcher/main.py
```

The helper is not a daemon. It receives clipboard payloads over stdin, writes to
SQLite, and exits. Its important entry points are:

```text
/usr/bin/python3 /opt/clipd-watcher/main.py --insert-text
/usr/bin/python3 /opt/clipd-watcher/main.py --insert-png
```

The GNOME extension currently uses `--insert-text`.

Manual verification:

```bash
printf 'manual-helper-test' | /usr/bin/python3 /opt/clipd-watcher/main.py --insert-text
sqlite3 "file:$HOME/.local/share/clipd/clipd.db?mode=ro" \
  "SELECT id, content_type, text_content FROM items ORDER BY id DESC LIMIT 5;"
```

## GUI Responsibilities

The Wails GUI process:

- opens `~/.local/share/clipd/clipd.db`;
- lists rows through `Service.ListItems`;
- refreshes from DB on demand;
- writes selected text/image entries back to the system clipboard through
  `wl-copy`;
- installs the GNOME keybinding for `Super+V`;
- stays resident when started as `clipd start`.

The GUI process does not listen to clipboard changes.

## Super+V and Auto Paste

GNOME owns the global shortcut because Wayland applications cannot grab global
keys themselves. `clipd` registers a GNOME custom keybinding for Super+V that
runs:

```text
/usr/bin/clipd
```

If no GUI is running, this starts the hidden resident process and opens the
popup. If it is already running, the single-instance socket toggles the popup.

When a history item is selected, `clipd`:

1. writes the item to the clipboard using `wl-copy`;
2. hides the popup;
3. if Auto-paste is enabled, sends Ctrl+V to the previously focused X11/XWayland
   window using `xdotool`.

Auto-paste is best-effort. GNOME Wayland blocks arbitrary synthetic input into
native Wayland windows, so this works for X11/XWayland clients but may not work
for native Wayland applications. Supporting all native Wayland apps would need a
separate virtual-input daemon such as `ydotool`, which adds privilege and setup
complexity.

## Commands Used During Debugging

Check extension state:

```bash
gsettings get org.gnome.shell enabled-extensions
gnome-extensions enable clipd-watcher@syedamirali
```

Check recent DB rows:

```bash
sqlite3 "file:$HOME/.local/share/clipd/clipd.db?mode=ro" \
  "SELECT id, content_type, substr(text_content,1,80), datetime(last_used_at,'unixepoch') FROM items ORDER BY last_used_at DESC LIMIT 10;"
```

Test live clipboard capture:

```bash
TEST_TEXT="extension-live-test-$(date +%s)"
printf '%s' "$TEST_TEXT" | wl-copy
sleep 3
sqlite3 "file:$HOME/.local/share/clipd/clipd.db?mode=ro" \
  "SELECT id, content_type, text_content FROM items WHERE text_content = '$TEST_TEXT' ORDER BY id DESC LIMIT 1;"
```

Check old systemd watcher state:

```bash
systemctl --user status clipd-watch
systemctl status clipd-watch
```

Expected result: the unit should not exist.

## Why This Fixed the Issue

GNOME Shell already has privileged access to clipboard state, so the extension
can observe text changes without creating an external polling process that
activates/focuses windows. The short-lived helper writes directly to the same
SQLite database used by the GUI. Since the GUI no longer polls or listens while
hidden, opening and closing the Clipd window no longer triggers the repeated
GNOME notification and blinking behavior.
