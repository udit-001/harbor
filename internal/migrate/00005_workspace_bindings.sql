-- +goose Up

-- Workspace directory bindings: a project folder maps to the workspace its
-- artifacts belong in. resolveWorkspace consults this table (longest matching
-- prefix, walking up from the caller's cwd) before the global current-workspace
-- setting — per-project truth outranks global convenience. One binding row is
-- written by `harbor workspace bind`, and automatically when
-- `harbor workspace create` runs inside the folder.

CREATE TABLE workspace_bindings (
    folder_path  TEXT PRIMARY KEY, -- absolute, cleaned
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

-- +goose Down

DROP TABLE workspace_bindings;
