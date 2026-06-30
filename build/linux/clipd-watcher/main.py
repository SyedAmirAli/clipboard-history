#!/usr/bin/env python3
"""Database writer helper for the clipd GNOME Shell extension.

The GNOME Shell extension is the clipboard listener. This helper is intentionally
short-lived: it receives clipboard payloads over stdin, writes the clipd SQLite
database, and exits. It has no dependency on the Wails GUI process.
"""

from __future__ import annotations

import base64
import hashlib
import os
import sqlite3
import struct
import sys
import time
from pathlib import Path


PNG_SIG = b"\x89PNG\r\n\x1a\n"


SCHEMA = """
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS items (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  content_type  TEXT    NOT NULL,
  text_content  TEXT,
  image_blob    BLOB,
  image_thumb   TEXT,
  image_w       INTEGER NOT NULL DEFAULT 0,
  image_h       INTEGER NOT NULL DEFAULT 0,
  content_hash  TEXT    NOT NULL UNIQUE,
  pinned        INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL,
  last_used_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_items_order
  ON items(pinned DESC, last_used_at DESC);

CREATE INDEX IF NOT EXISTS idx_items_hash
  ON items(content_hash);

CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS vault_entries (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  content_type  TEXT    NOT NULL,
  payload       BLOB    NOT NULL,
  nonce         BLOB    NOT NULL,
  content_hash  TEXT    NOT NULL,
  created_at    INTEGER NOT NULL,
  last_used_at  INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_vault_entries_order
  ON vault_entries(last_used_at DESC);

CREATE INDEX IF NOT EXISTS idx_vault_entries_hash
  ON vault_entries(content_hash);
"""


def db_path() -> Path:
    base = os.environ.get("XDG_DATA_HOME")
    if base:
        root = Path(base)
    else:
        root = Path.home() / ".local" / "share"
    return root / "clipd" / "clipd.db"


def open_db() -> sqlite3.Connection:
    path = db_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(path, timeout=5)
    conn.executescript(SCHEMA)
    return conn


def hash_text(text: str) -> str:
    return "t:" + hashlib.sha256(text.encode()).hexdigest()


def hash_image(png: bytes) -> str:
    if not png.startswith(PNG_SIG):
        return "i:" + hashlib.sha256(png).hexdigest()
    h = hashlib.sha256()
    p = 8
    saw = False
    while p + 8 <= len(png):
        length = struct.unpack(">I", png[p : p + 4])[0]
        end = p + 12 + length
        if end > len(png):
            break
        chunk = png[p + 4 : p + 8]
        if chunk in {b"IHDR", b"PLTE", b"IDAT"}:
            h.update(chunk)
            h.update(png[p + 8 : p + 8 + length])
            saw = True
        p = end
    return "i:" + (h.hexdigest() if saw else hashlib.sha256(png).hexdigest())


def png_size(png: bytes) -> tuple[int, int]:
    if len(png) >= 24 and png.startswith(PNG_SIG) and png[12:16] == b"IHDR":
        return struct.unpack(">II", png[16:24])
    return 0, 0


def settings(conn: sqlite3.Connection) -> tuple[int, bool, int]:
    rows = dict(conn.execute("SELECT key, value FROM settings").fetchall())
    max_items = int(rows.get("max_items", "200") or "200")
    keep_images = rows.get("keep_images", "1") == "1"
    max_image_mb = int(rows.get("max_image_mb", "5") or "5")
    return max_items, keep_images, max_image_mb


def trim(conn: sqlite3.Connection, max_items: int) -> None:
    if max_items <= 0:
        return
    conn.execute(
        """
        DELETE FROM items
        WHERE pinned = 0
          AND id IN (
            SELECT id FROM items
            WHERE pinned = 0
            ORDER BY last_used_at DESC
            LIMIT -1 OFFSET ?
          )
        """,
        (max_items,),
    )


def add_text(conn: sqlite3.Connection, text: str, content_hash: str) -> None:
    now = int(time.time())
    with conn:
        row = conn.execute("SELECT id FROM items WHERE content_hash = ?", (content_hash,)).fetchone()
        if row:
            conn.execute("UPDATE items SET last_used_at = ? WHERE id = ?", (now, row[0]))
        else:
            conn.execute(
                """
                INSERT INTO items (content_type, text_content, content_hash, pinned, created_at, last_used_at)
                VALUES ('text', ?, ?, 0, ?, ?)
                """,
                (text, content_hash, now, now),
            )
        trim(conn, settings(conn)[0])


def add_image(conn: sqlite3.Connection, png: bytes, content_hash: str) -> None:
    max_items, keep_images, max_image_mb = settings(conn)
    if not keep_images or len(png) > max_image_mb * 1024 * 1024:
        return
    now = int(time.time())
    w, h = png_size(png)
    thumb = "data:image/png;base64," + base64.b64encode(png).decode("ascii") if len(png) <= 256_000 else ""
    with conn:
        row = conn.execute("SELECT id FROM items WHERE content_hash = ?", (content_hash,)).fetchone()
        if row:
            conn.execute("UPDATE items SET last_used_at = ? WHERE id = ?", (now, row[0]))
        else:
            conn.execute(
                """
                INSERT INTO items (content_type, image_blob, image_thumb, image_w, image_h,
                                   content_hash, pinned, created_at, last_used_at)
                VALUES ('image', ?, ?, ?, ?, ?, 0, ?, ?)
                """,
                (png, thumb, w, h, content_hash, now, now),
            )
        trim(conn, max_items)


def insert_text_from_stdin() -> int:
    data = sys.stdin.buffer.read()
    if not data:
        return 0
    conn = open_db()
    try:
        try:
            text = data.decode("utf-8")
        except UnicodeDecodeError:
            return 0
        if text:
            add_text(conn, text, hash_text(text))
    finally:
        conn.close()
    return 0


def insert_png_from_stdin() -> int:
    data = sys.stdin.buffer.read()
    if not data:
        return 0
    conn = open_db()
    try:
        add_image(conn, data, hash_image(data))
    finally:
        conn.close()
    return 0


def main() -> int:
    if len(sys.argv) > 1 and sys.argv[1] == "--insert-text":
        return insert_text_from_stdin()
    if len(sys.argv) > 1 and sys.argv[1] == "--insert-png":
        return insert_png_from_stdin()
    print("clipd-watch: this helper is called by the GNOME Shell extension", file=sys.stderr)
    print(f"clipd-watch: database path is {db_path()}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        print("clipd-watch: stopped", file=sys.stderr)
        raise SystemExit(0)
