-- +goose Up

-- Artifacts (HARB-59): pages generalize from HTML-only to 7 formats across
-- 3 view families (native: html/pdf/svg/image · text-frame: markdown/text ·
-- app: excalidraw). `format` is an attribute of the page, not a type: the
-- stored file is the artifact; rendering is a view derived from format.
-- Existing rows are all HTML pages, hence the default.

ALTER TABLE pages ADD COLUMN format TEXT NOT NULL DEFAULT 'html';

-- +goose Down

ALTER TABLE pages DROP COLUMN format;
