-- +goose Up

-- The feedback loop: anchored human comments on pages, plus the agent's
-- change records that the "what changed" walkthrough tours. The human writes
-- comments from the shell; the agent responds by editing the page, marking the
-- changed element (data-cf-change) and recording a change.

CREATE TABLE comments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id     INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    anchor      TEXT NOT NULL DEFAULT '',
    quote       TEXT NOT NULL DEFAULT '',
    type        TEXT NOT NULL DEFAULT 'general',
    body        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'open',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    resolved_at TEXT
);

CREATE INDEX idx_comments_page ON comments(page_id);
CREATE INDEX idx_comments_status ON comments(status);

CREATE TABLE changes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id     INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    comment_id  INTEGER REFERENCES comments(id) ON DELETE SET NULL,
    change_id   TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_changes_page ON changes(page_id);
CREATE INDEX idx_changes_comment ON changes(comment_id);

-- +goose Down

DROP INDEX IF EXISTS idx_changes_comment;
DROP INDEX IF EXISTS idx_changes_page;
DROP TABLE IF EXISTS changes;
DROP INDEX IF EXISTS idx_comments_status;
DROP INDEX IF EXISTS idx_comments_page;
DROP TABLE IF EXISTS comments;