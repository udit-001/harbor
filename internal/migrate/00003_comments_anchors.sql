-- +goose Up

-- Multi-anchor comments (HARB-29). Canonical `anchors` is a JSON array of
-- {kind, path, quote?, markerId?}; `reply_to` links a comment to the comment it
-- replies to (a flat, one-way thread); `updated_at` tracks edits. The legacy
-- single anchor/quote/type columns stay for backward-compat display and are
-- backfilled into `anchors`.

ALTER TABLE comments ADD COLUMN anchors   TEXT    NOT NULL DEFAULT '[]';
ALTER TABLE comments ADD COLUMN reply_to  INTEGER REFERENCES comments(id);
ALTER TABLE comments ADD COLUMN updated_at TEXT   NOT NULL DEFAULT '';

-- Backfill anchors from the legacy single-anchor shape. selection carried both
-- a selector path and a quote; element carried only a path; general was a
-- whole-document comment.
UPDATE comments SET
  anchors = CASE type
    WHEN 'selection' THEN json_array(json_object('kind','text','path',anchor,'quote',quote))
    WHEN 'element'   THEN json_array(json_object('kind','element','path',anchor))
    ELSE                  json_array(json_object('kind','document','path',''))
  END,
  updated_at = COALESCE(NULLIF(updated_at,''), created_at)
WHERE anchors = '[]';

-- +goose Down

ALTER TABLE comments DROP COLUMN updated_at;
ALTER TABLE comments DROP COLUMN reply_to;
ALTER TABLE comments DROP COLUMN anchors;