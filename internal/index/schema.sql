-- MailCli local message index schema
-- SQLite + FTS5, version 2

PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA synchronous = NORMAL;

-- ── messages ────────────────────────────────────────────────────────────────
-- One row per (account, message-id). body_json holds the full StandardMessage
-- as JSON; all other columns are denormalized copies for efficient filtering.

CREATE TABLE IF NOT EXISTS messages (
    account      TEXT    NOT NULL,
    mailbox      TEXT    NOT NULL DEFAULT '',
    id           TEXT    NOT NULL,          -- Message-ID
    thread_id    TEXT    NOT NULL DEFAULT '',
    date         TEXT    NOT NULL DEFAULT '', -- RFC3339
    indexed_at   TEXT    NOT NULL DEFAULT '',
    from_addr    TEXT    NOT NULL DEFAULT '',
    subject      TEXT    NOT NULL DEFAULT '',
    category     TEXT    NOT NULL DEFAULT '',
    labels       TEXT    NOT NULL DEFAULT '[]', -- JSON array of strings
    action_types TEXT    NOT NULL DEFAULT '[]', -- JSON array of strings
    has_codes    INTEGER NOT NULL DEFAULT 0,
    code_count   INTEGER NOT NULL DEFAULT 0,
    snippet      TEXT    NOT NULL DEFAULT '',
    body_text    TEXT    NOT NULL DEFAULT '', -- plain text body for FTS
    body_json    TEXT    NOT NULL,            -- full StandardMessage JSON
    PRIMARY KEY (account, id)
);

CREATE INDEX IF NOT EXISTS idx_messages_thread  ON messages(thread_id);
CREATE INDEX IF NOT EXISTS idx_messages_date    ON messages(date);
CREATE INDEX IF NOT EXISTS idx_messages_account ON messages(account, mailbox);

-- ── messages_fts ─────────────────────────────────────────────────────────────
-- Standalone FTS5 table (no content= to avoid rowid mapping issues on upsert).
-- We manually sync this table inside each Upsert call using a transaction.

CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    account  UNINDEXED,
    msg_id   UNINDEXED,
    subject,
    from_addr,
    body_text,
    tokenize = "unicode61 remove_diacritics 2"
);
