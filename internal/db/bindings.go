package db

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Workspace directory bindings map a project folder to the workspace its
// artifacts belong in. resolveWorkspace consults them before the global
// current-workspace setting: per-project truth outranks global convenience.
// One binding covers the bound folder and everything below it.

// BindWorkspaceDir maps folder (cleaned, absolute) to the named workspace.
// Re-binding a folder replaces the old mapping.
func (s *Store) BindWorkspaceDir(folder, workspaceName string) error {
	wsStore, err := s.Workspace(workspaceName)
	if err != nil {
		return fmt.Errorf("workspace %q not found\n  Use 'harbor workspace list' to see available workspaces", workspaceName)
	}
	wsID := wsStore.Workspace().ID
	folder = normalizeFolder(folder)
	if folder == "" {
		return fmt.Errorf("cannot bind an empty folder")
	}
	_, err = s.db.Exec(`INSERT INTO workspace_bindings (folder_path, workspace_id)
		VALUES (?, ?)
		ON CONFLICT(folder_path) DO UPDATE SET workspace_id = excluded.workspace_id`,
		folder, wsID)
	return err
}

// UnbindWorkspaceDir removes the binding for folder.
func (s *Store) UnbindWorkspaceDir(folder string) error {
	folder = normalizeFolder(folder)
	_, err := s.db.Exec(`DELETE FROM workspace_bindings WHERE folder_path = ?`, folder)
	return err
}

// WorkspaceForDir returns the name of the workspace bound to dir or any of
// its ancestors — the deepest (longest path) match wins. Empty string when
// nothing binds.
func (s *Store) WorkspaceForDir(dir string) (string, error) {
	dir = normalizeFolder(dir)
	if dir == "" {
		return "", nil
	}
	type binding struct {
		path string
		name string
	}
	rows, err := s.db.Query(`SELECT b.folder_path, w.name FROM workspace_bindings b
		JOIN workspaces w ON w.id = b.workspace_id`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var best *binding
	for rows.Next() {
		var b binding
		if err := rows.Scan(&b.path, &b.name); err != nil {
			return "", err
		}
		if dir == b.path || strings.HasPrefix(dir, b.path+string(filepath.Separator)) {
			if best == nil || len(b.path) > len(best.path) {
				b := b // copy for the pointer
				best = &b
			}
		}
	}
	if best == nil {
		return "", rows.Err()
	}
	return best.name, rows.Err()
}

// BindCwd binds the caller's current working directory to the named
// workspace. Convenience for `workspace bind`.
func (s *Store) BindCwd(workspaceName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if err := s.BindWorkspaceDir(cwd, workspaceName); err != nil {
		return "", err
	}
	return normalizeFolder(cwd), nil
}

// normalizeFolder makes folder paths comparable: absolute and cleaned.
func normalizeFolder(folder string) string {
	if folder == "" {
		return ""
	}
	abs, err := filepath.Abs(folder)
	if err != nil {
		return ""
	}
	return filepath.Clean(abs)
}
