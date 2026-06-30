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
