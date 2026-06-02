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
