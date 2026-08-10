package db

import (
	"fmt"
	"os"
	"path/filepath"
)

const wsColumns = `id, name, topic, description, path, created_at, last_studied`

func scanWorkspace(row interface{ Scan(...any) error }) (Workspace, error) {
	var w Workspace
	err := row.Scan(&w.ID, &w.Name, &w.Topic, &w.Description, &w.Path, &w.CreatedAt, &w.LastStudied)
	return w, err
}

func scanWorkspaces(rows RowScanner) ([]Workspace, error) {
	return scanRows(rows, "workspace", scanWorkspace)
}

// GetWorkspaces returns all workspaces, newest first.
func (s *Store) GetWorkspaces() ([]Workspace, error) {
	rows, err := s.db.Query(fmt.Sprintf("SELECT %s FROM workspaces ORDER BY last_studied DESC", wsColumns))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWorkspaces(rows)
}

// GetWorkspace returns a single workspace by ID.
func (s *Store) GetWorkspace(id int64) (Workspace, error) {
	row := s.db.QueryRow(fmt.Sprintf("SELECT %s FROM workspaces WHERE id = ?", wsColumns), id)
	return scanWorkspace(row)
}

// GetWorkspaceByName returns a workspace by its name.
func (s *Store) GetWorkspaceByName(name string) (Workspace, error) {
	row := s.db.QueryRow(fmt.Sprintf("SELECT %s FROM workspaces WHERE name = ?", wsColumns), name)
	return scanWorkspace(row)
}

// AddWorkspace creates a new workspace row only. It does not create the
// on-disk directory or seed templates — for that use CreateWorkspace, which
// owns the full row ⇔ dir tree invariant. Tests use AddWorkspace to set up a
// row without the filesystem; production code uses CreateWorkspace.
func (s *Store) AddWorkspace(w Workspace) (Workspace, error) {
	now := nowTimestamp()
	result, err := s.db.Exec(
		`INSERT INTO workspaces (name, topic, description, path, created_at, last_studied)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		w.Name, w.Topic, w.Description, w.Path, now, now,
	)
	if err != nil {
		return Workspace{}, fmt.Errorf("add workspace: %w", err)
	}
	id, _ := result.LastInsertId()
	w.ID = id
	w.CreatedAt = now
	w.LastStudied = now
	return w, nil
}

// CreateWorkspace owns the full workspace lifecycle: it creates the directory
// tree (root + subdirs), seeds the default files (JS/CSS assets, RESOURCES/
// NOTES templates), and inserts the row. The "row ⇔ dir tree" invariant lives
// here — a created workspace always has both. wsPath is supplied by the caller
// (the CLI knows the data dir / --dir override); the store owns the scaffold.
func (s *Store) CreateWorkspace(name, topic, description, wsPath string) (Workspace, error) {
	layout := NewLayout(wsPath)

	for _, sub := range layout.Subdirs() {
		if err := os.MkdirAll(filepath.Join(layout.Root, sub), 0o755); err != nil {
			return Workspace{}, fmt.Errorf("create %s directory: %w", sub, err)
		}
	}

	displayName := topic
	if displayName == "" {
		displayName = name
	}
	if err := seedWorkspaceDefaults(layout, displayName); err != nil {
		return Workspace{}, fmt.Errorf("seed workspace: %w", err)
	}

	w := Workspace{Name: name, Topic: topic, Description: description, Path: wsPath}
	created, err := s.AddWorkspace(w)
	if err != nil {
		// Roll back the directory scaffold on DB failure so a retry
		// doesn't hit a duplicate-name error against an orphaned dir.
		_ = os.RemoveAll(wsPath)
		return Workspace{}, err
	}
	return created, nil
}

// DeleteWorkspaceByName removes a workspace's row, deletes its on-disk
// directory, and clears the current-workspace setting if it pointed at the
// deleted one. The inverse of CreateWorkspace — the row ⇔ dir tree invariant
// is torn down in one place. Confirmation prompting stays with the caller (a
// UI concern).
func (s *Store) DeleteWorkspaceByName(name string) error {
	w, err := s.GetWorkspaceByName(name)
	if err != nil {
		return fmt.Errorf("workspace %q not found: %w", name, err)
	}

	if err := s.DeleteWorkspace(w.ID); err != nil {
		return fmt.Errorf("delete workspace row: %w", err)
	}

	if w.Path != "" {
		if err := os.RemoveAll(w.Path); err != nil {
			return fmt.Errorf("remove workspace directory: %w", err)
		}
	}

	if current, _ := s.CurrentWorkspace(); current == w.Name {
		_ = s.SetCurrentWorkspace("")
	}
	return nil
}

// UpdateWorkspaceTopic updates the topic field.
func (s *Store) UpdateWorkspaceTopic(id int64, topic string) error {
	_, err := s.db.Exec("UPDATE workspaces SET topic = ? WHERE id = ?", topic, id)
	return err
}

// TouchWorkspace updates last_studied timestamp.
func (s *Store) TouchWorkspace(id int64) error {
	now := nowTimestamp()
	_, err := s.db.Exec("UPDATE workspaces SET last_studied = ? WHERE id = ?", now, id)
	return err
}

// DeleteWorkspace deletes a workspace.
func (s *Store) DeleteWorkspace(id int64) error {
	result, err := s.db.Exec("DELETE FROM workspaces WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("workspace %d not found", id)
	}
	return nil
}

// WorkspaceCount returns total number of workspaces.
func (s *Store) WorkspaceCount() (int, error) {
	var count int
	err := s.db.Get(&count, "SELECT COUNT(*) FROM workspaces")
	return count, err
}
