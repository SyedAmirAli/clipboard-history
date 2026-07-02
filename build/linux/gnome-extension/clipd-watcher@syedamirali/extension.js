import Gio from 'gi://Gio';
import GLib from 'gi://GLib';
import St from 'gi://St';

import {Extension} from 'resource:///org/gnome/shell/extensions/extension.js';

const CLIPBOARD_TYPE = St.ClipboardType.CLIPBOARD;
const HELPER = '/opt/clipd-watcher/main.py';
const PYTHON = '/usr/bin/python3';
const POLL_SECONDS = 1;

// Image mimetypes we can hand to the helper, in preference order. The helper
// stores PNG blobs, so PNG first; GNOME's clipboard (mutter) converts between
// image formats on demand, so requesting image/png usually works even when
// the source app advertised only image/jpeg or image/bmp.
const IMAGE_MIMES = ['image/png'];

export default class ClipdWatcherExtension extends Extension {
    enable() {
        this._clipboard = St.Clipboard.get_default();
        this._lastText = null;
        this._lastImageHash = null;
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
        this._lastImageHash = null;
        this._busy = false;
    }

    _checkClipboard() {
        if (!this._clipboard || this._busy)
            return;
        // When the clipboard holds an image, capture the image and skip the
        // text poll for this tick — image copies often carry a text/html or
        // file-path alternate that we don't want as a separate history entry.
        const mimes = this._clipboard.get_mimetypes(CLIPBOARD_TYPE) ?? [];
        const imageMime = IMAGE_MIMES.find(m => mimes.includes(m)) ??
            (mimes.some(m => m.startsWith('image/')) ? 'image/png' : null);
        if (imageMime) {
            this._checkImage(imageMime);
            return;
        }
        this._lastImageHash = null;
        this._checkText();
    }

    _checkText() {
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

    _checkImage(mime) {
        this._busy = true;
        this._clipboard.get_content(CLIPBOARD_TYPE, mime, (...args) => {
            this._busy = false;
            const bytes = args[args.length - 1];
            if (!bytes || bytes.get_size() === 0)
                return;
            const hash = GLib.compute_checksum_for_bytes(
                GLib.ChecksumType.SHA256, bytes);
            if (hash === this._lastImageHash)
                return;
            this._lastImageHash = hash;
            this._lastText = null;
            this._insertImage(bytes);
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

    _insertImage(bytes) {
        try {
            const proc = Gio.Subprocess.new(
                [PYTHON, HELPER, '--insert-png'],
                Gio.SubprocessFlags.STDIN_PIPE |
                Gio.SubprocessFlags.STDOUT_SILENCE |
                Gio.SubprocessFlags.STDERR_SILENCE);
            proc.communicate_async(bytes, null, (p, res) => {
                try {
                    p.communicate_finish(res);
                } catch (e) {
                    console.error(`clipd watcher image helper failed: ${e}`);
                }
            });
        } catch (e) {
            console.error(`clipd watcher failed to spawn image helper: ${e}`);
        }
    }
}
