-- +goose Up

-- Page body text: the plain-text extraction of a page's HTML, indexed for FTS.
-- Harvested idempotently when a page is added/updated; powers page textual
-- search. Extending the pages_fts index to include it requires recreating the
-- virtual table (external-content FTS columns are fixed at creation).
ALTER TABLE pages ADD COLUMN body_text TEXT NOT NULL DEFAULT '';

DROP TRIGGER IF EXISTS pages_au;
DROP TRIGGER IF EXISTS pages_ad;
DROP TRIGGER IF EXISTS pages_ai;
DROP TABLE IF EXISTS pages_fts;

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
ALTER TABLE pages DROP COLUMN body_text;