-- +goose Up

-- Feedback-loop data foundation (M2): anchored, page-scoped comments plus the
-- change markers the agent leaves in HTML when it acts on a comment. The human
-- writes comments from the shell (HARB-11); the agent reads the open queue, and
-- the changes table tracks which agent edit resolved which comment.

-- Comments: anchored human feedback on a page. type = selection | element |
-- general; anchor is the element's existing id or a stable computed CSS
-- selector (empty for general/page-level); quote is the optional selected text
-- snippet. status = open | in-progress | done; done records resolved_at. The
-- page file itself is never touched — comments live here.
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

-- Changes: a record of an agent edit that addresses feedback. change_id matches
-- a data-cf-change marker the agent embeds in the page HTML, so the what-changed
-- walkthrough (HARB-12) can map an edit back to the comment it resolves.
-- comment_id is nullable: a change may carry no comment (general tidying).
CREATE TABLE changes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id     INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    comment_id  INTEGER REFERENCES comments(id) ON DELETE SET NULL,
    change_id   TEXT NOT NULL,
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