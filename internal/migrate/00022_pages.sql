-- +goose Up

-- Harbor domain foundation. Pages are the atomic artifact — workspace-scoped,
-- standalone HTML pages the agent produced and imported. Unlike scraps (which
-- stay global and sealed), a page belongs to exactly one workspace.

-- Workspace semantic payload: the description powers disambiguation and search
-- ("which body of work is this?"). Ship it alongside the scaffolding work.
ALTER TABLE workspaces ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- Pages: atomic HTML artifact. Title drives the stable slug (find-then-update);
-- the slug never changes on rename. status = draft | published | archived.
-- origin_path is a soft, informational breadcrumb (never dereferenced).
CREATE TABLE pages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    slug        TEXT NOT NULL UNIQUE,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    context     TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'draft',
    origin_path TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_pages_workspace ON pages(workspace_id);
CREATE INDEX idx_pages_status ON pages(status);

-- Many-to-many between pages and tags. Deleting a page or a tag removes its
-- join rows (cascade). The tag description is the semantic payload searched
-- alongside the page text.
CREATE TABLE page_tags (
    page_id INTEGER NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (page_id, tag_id)
);

CREATE INDEX idx_page_tags_tag ON page_tags(tag_id);

-- FTS5 index over page title + description + context (metadata fields). Body
-- text extraction + indexing lands with the search slice. Tag name/description
-- is NOT indexed here (tags are a separate table) — tag search is a LIKE-join
-- in the store layer, mirroring the scraps pattern.
CREATE VIRTUAL TABLE pages_fts USING fts5(
    title, description, context,
    content=pages,
    content_rowid=id,
    tokenize = 'porter unicode61'
);

-- +goose StatementBegin
CREATE TRIGGER pages_ai AFTER INSERT ON pages BEGIN
    INSERT INTO pages_fts(rowid, title, description, context)
    VALUES (new.id, new.title, new.description, new.context);
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER pages_ad AFTER DELETE ON pages BEGIN
    INSERT INTO pages_fts(pages_fts, rowid, title, description, context)
    VALUES('delete', old.id, old.title, old.description, old.context);
END;
-- +goose StatementEnd

-- Scoped to indexed columns so non-indexed UPDATEs (status, updated_at,
-- origin_path) skip the FTS delete+insert (the scrappad pattern).
-- +goose StatementBegin
CREATE TRIGGER pages_au AFTER UPDATE OF title, description, context ON pages BEGIN
    INSERT INTO pages_fts(pages_fts, rowid, title, description, context)
    VALUES('delete', old.id, old.title, old.description, old.context);
    INSERT INTO pages_fts(rowid, title, description, context)
    VALUES (new.id, new.title, new.description, new.context);
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
DROP TABLE IF EXISTS pages;
ALTER TABLE workspaces DROP COLUMN description;