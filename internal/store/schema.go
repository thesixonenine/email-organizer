package store

// schema is the database schema SQL, embedded directly for simplicity.
// The reference copy lives at migrations/001_init.sql.
const schema = `
CREATE TABLE IF NOT EXISTS mailboxes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL DEFAULT 'imap',
    email TEXT NOT NULL UNIQUE,
    imap_server TEXT,
    imap_port INTEGER DEFAULT 993,
    exchange_type TEXT,
    ews_endpoint TEXT,
    tenant_id TEXT,
    client_id TEXT,
    auth_code TEXT NOT NULL,
    fetch_count INTEGER NOT NULL DEFAULT 500,
    last_sync_at TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mailbox_id INTEGER NOT NULL REFERENCES mailboxes(id),
    uid TEXT NOT NULL,
    folder TEXT NOT NULL,
    subject TEXT,
    body_preview TEXT,
    body TEXT,
    from_name TEXT,
    from_email TEXT,
    to_list TEXT,
    cc_list TEXT,
    received_at TEXT NOT NULL,
    flags TEXT,
    is_deleted INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(mailbox_id, uid, folder)
);

CREATE TABLE IF NOT EXISTS rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    trigger TEXT NOT NULL CHECK(trigger IN ('on_arrival','on_delete')),
    condition_json TEXT NOT NULL,
    action_json TEXT NOT NULL,
    priority INTEGER DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS rule_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_id INTEGER NOT NULL REFERENCES rules(id),
    mailbox_id INTEGER NOT NULL REFERENCES mailboxes(id),
    message_id INTEGER REFERENCES messages(id),
    action TEXT NOT NULL,
    result TEXT NOT NULL CHECK(result IN ('success','failed')),
    error_msg TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_messages_mailbox_folder ON messages(mailbox_id, folder);
CREATE INDEX IF NOT EXISTS idx_mailboxes_email ON mailboxes(email);
CREATE INDEX IF NOT EXISTS idx_rules_enabled ON rules(enabled);
`