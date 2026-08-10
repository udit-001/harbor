package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// WorkspaceStore is a scoped view over the database bound to a single workspace.
// Created via Store.Workspace(name) — the workspace is resolved once at the
// seam, so callers never thread workspaceID through every call.
//
// WorkspaceStore is the Harbor reuse substrate that later tickets extend with
// page/tag/workspace-of-pages operations. It currently owns the workspace
// identity seam and asset management (see assets.go).
type WorkspaceStore struct {
	store *Store
	ws    Workspace
}

// Workspace returns the resolved workspace.
func (w *WorkspaceStore) Workspace() Workspace { return w.ws }

// Layout returns the on-disk layout for this workspace.
func (w *WorkspaceStore) Layout() Layout { return NewLayout(w.ws.Path) }

// db returns the underlying *sqlx.DB for direct query access.
func (w *WorkspaceStore) db() *sqlx.DB { return w.store.db }

// Touch updates the last_studied timestamp for this workspace.
func (w *WorkspaceStore) Touch() error {
	return w.store.TouchWorkspace(w.ws.ID)
}

// UpdateTopic updates the topic field for this workspace.
func (w *WorkspaceStore) UpdateTopic(topic string) error {
	return w.store.UpdateWorkspaceTopic(w.ws.ID, topic)
}

// ── Construction ──

// Workspace returns a WorkspaceStore scoped to the named workspace. The
// workspace is resolved (name→ID) once here; subsequent calls need not
// pass the ID. Returns an error if the workspace does not exist.
func (s *Store) Workspace(name string) (*WorkspaceStore, error) {
	ws, err := s.GetWorkspaceByName(name)
	if err != nil {
		return nil, fmt.Errorf("workspace %q not found: %w", name, err)
	}
	return &WorkspaceStore{store: s, ws: ws}, nil
}

// WorkspaceByID returns a WorkspaceStore scoped to the workspace with the
// given ID. Used when the ID is already known.
func (s *Store) WorkspaceByID(id int64) (*WorkspaceStore, error) {
	ws, err := s.GetWorkspace(id)
	if err != nil {
		return nil, fmt.Errorf("workspace %d not found: %w", id, err)
	}
	return &WorkspaceStore{store: s, ws: ws}, nil
}
