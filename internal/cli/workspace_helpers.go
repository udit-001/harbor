package cli

import (
	"fmt"

	"github.com/udit-001/harbor/internal/db"
)

// resolveWorkspace returns a WorkspaceStore for the named (or current
// auto-selected) workspace. It mirrors the legacy behaviour that used to live
// next to the lesson commands, preserved here because several surviving
// workspace-scoped commands depend on it. The error messages deliberately name
// the exact follow-up command to run.
func resolveWorkspace(s *db.Store, name string) (*db.WorkspaceStore, error) {
	if name != "" {
		ws, err := s.Workspace(name)
		if err != nil {
			return nil, fmt.Errorf("workspace %q not found\n  Use 'harbor workspace list' to see available workspaces", name)
		}
		return ws, nil
	}

	current, err := s.CurrentWorkspace()
	if err != nil {
		return nil, formatError("failed to get current workspace", err)
	}
	if current != "" {
		ws, err := s.Workspace(current)
		if err == nil {
			return ws, nil
		}
		_ = s.SetCurrentWorkspace("")
	}

	workspaces, err := s.GetWorkspaces()
	if err != nil {
		return nil, formatError("failed to list workspaces", err)
	}

	switch len(workspaces) {
	case 0:
		return nil, fmt.Errorf("no workspaces found\n  Use 'harbor init' to create one")
	case 1:
		return s.Workspace(workspaces[0].Name)
	default:
		return nil, fmt.Errorf("no current workspace set. You have %d workspaces:\n  Use 'harbor workspace use <name>' to set one, or pass -w to override", len(workspaces))
	}
}
