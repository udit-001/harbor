-- +goose Up

-- Changes get a human-readable summary so the what-changed walkthrough (HARB-12)
-- can tour a change by its title + description, not just the raw change_id
-- marker. The agent writes these when it records a change addressing a comment.
ALTER TABLE changes ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE changes ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE changes DROP COLUMN description;
ALTER TABLE changes DROP COLUMN title;