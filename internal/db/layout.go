package db

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Layout owns the on-disk directory structure of a workspace. One place
// defines where assets and workspace documents live — callers ask it instead
// of reconstructing paths from convention. Later tickets rework it to the
// Harbor workspace-of-pages layout.
type Layout struct {
	Root string // absolute path to the workspace directory
}

// NewLayout creates a Layout for the given workspace root path.
func NewLayout(root string) Layout {
	return Layout{Root: root}
}

// Subdirs returns the workspace subdirectory names in creation order.
func (l Layout) Subdirs() []string {
	return []string{"assets"}
}

// AssetPath returns the absolute path for an asset file.
func (l Layout) AssetPath(filename string) string {
	return filepath.Join(l.Root, "assets", filename)
}

// SafeJoin resolves filename into subdir inside the workspace root, rejecting
// traversal (.., absolute paths) and the directory itself. It is the single
// path guard for every user-supplied filename (assets) — one place owns the
// rule so the copies can't drift.
func (l Layout) SafeJoin(subdir, filename string) (string, error) {
	dir := filepath.Join(l.Root, subdir)
	target := filepath.Join(dir, filename)
	rel, err := filepath.Rel(dir, target)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(filename) {
		return "", fmt.Errorf("invalid path %q", filename)
	}
	return target, nil
}
