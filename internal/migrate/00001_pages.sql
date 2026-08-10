-- +goose Up

-- Harbor domain foundation (fresh baseline; no Pharos/learn-tool surface).
-- The atomic artifact is the Page: a workspace-scoped, standalone HTML page
-- imported with provenance (description/context), labeled with status, and
-- searchable via an external-content FTS index.

CREATE TABLE workspaces (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    topic       TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    path        TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Settings singleton: holds user preferences (including the last-active
-- workspace used to resolve a bare `harbor workspace` read).
CREATE TABLE settings (
    id                    INTEGER PRIMARY KEY CHECK (id = 1),
    default_view          TEXT NOT NULL DEFAULT 'dashboard',
    items_per_page        INTEGER NOT NULL DEFAULT 25,
    assets_dir            TEXT NOT NULL DEFAULT 'assets',
    last_active_workspace TEXT NOT NULL DEFAULT ''
);
INSERT OR IGNORE INTO settings (id) VALUES (1);

-- Tags: first-class grouping objects shared across pages, with a semantic
-- description (the payload that powers tag-description search).
CREATE TABLE tags (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE pages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    slug         TEXT NOT NULL UNIQUE,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    context      TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'draft',
    origin_path  TEXT NOT NULL DEFAULT '',
    body_text    TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_pages_workspace ON pages(workspace_id);
CREATE INDEX idx_pages_status ON pages(status);

-- Many-to-many between pages and tags (cascade removes join rows).
CREATE TABLE page_tags (
    page_id INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (page_id, tag_id)
);
CREATE INDEX idx_page_tags_tag ON page_tags(tag_id);

-- External-content FTS over page title/description/context/body_text.
CREATE VIRTUAL TABLE pages_fts USING fts5(
    title, description, context, body_text,
    content=pages,
    content_rowid=id,
    tokenize = 'porter unicode61'
);

-- +goose StatementBegin
CREATE TRIGGER pages_ai AFTER INSERT ON pages BEGIN
    INSERT INTO pages_fts(rowid, title, description, context, body_text)
    VALUES (new.id, new.title, new.description, new.context, new.body_text);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER pages_ad AFTER DELETE ON pages BEGIN
    INSERT INTO pages_fts(pages_fts, rowid, title, description, context, body_text)
    VALUES('delete', old.id, old.title, old.description, old.context, old.body_text);
END;
-- +goose StatementEnd

-- Scoped to indexed columns so non-indexed UPDATEs (status, updated_at,
-- origin_path) skip the FTS delete+insert.
-- +goose StatementBegin
CREATE TRIGGER pages_au AFTER UPDATE OF title, description, context, body_text ON pages BEGIN
    INSERT INTO pages_fts(pages_fts, rowid, title, description, context, body_text)
    VALUES('delete', old.id, old.title, old.description, old.context, old.body_text);
    INSERT INTO pages_fts(rowid, title, description, context, body_text)
    VALUES (new.id, new.title, new.description, new.context, new.body_text);
END;
-- +goose StatementEnd

INSERT INTO pages_fts(pages_fts) VALUES('rebuild');

-- +goose Down

DROP TRIGGER IF EXISTS pages_au;
DROP TRIGGER IF EXISTS pages_ad;
DROP TRIGGER IF EXISTS pages_ai;
DROP TABLE IF EXISTS pages_fts;
DROP INDEX IF EXISTS idx_page_tags_tag;
DROP TABLE IF EXISTS page_tags;
DROP INDEX IF EXISTS idx_pages_status;
DROP INDEX IF EXISTS idx_pages_workspace;
DROP TABLE IF EXISTS pages;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS workspaces;