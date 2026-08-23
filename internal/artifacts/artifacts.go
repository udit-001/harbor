package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
)

// The managed store is harbor's artifact home:
//
//	<dataDir>/store/<workspace>/<slug>.<format>
//
// One directory per workspace, one file per page — the stored file IS the
// artifact; everything else (DB rows, FTS index) is derived. This module is
// the single seam for that layout and for its write invariant:
//
//	Stage writes to a temp file beside the final path (same filesystem, so the
//	later rename is atomic), the caller updates the database, and only then
//	does Commit flip the file into place. A crash between the two can never
//	leave the served content ahead of the index — view and search can't
//	disagree.
//
// Both writers (the CLI today, server-side endpoints when they exist) go
// through here; nothing else may compose store paths by hand.

// FileName is the managed-store name for an artifact: <slug>.<format>.
func FileName(slug, format string) string {
	return slug + "." + format
}

// WorkspaceDir is where a workspace's artifacts live under the data dir.
func WorkspaceDir(dataDir, workspaceName string) string {
	return filepath.Join(dataDir, "store", workspaceName)
}

// Path resolves the served copy of an artifact.
func Path(dataDir, workspaceName, slug, format string) string {
	return filepath.Join(WorkspaceDir(dataDir, workspaceName), FileName(slug, format))
}

// Stage writes data to a temp file beside the artifact's final path, ready to
// be renamed into place by Commit. Returns the temp path; callers own cleanup
// on failure (os.Remove).
func Stage(dataDir, workspaceName, slug, format string, data []byte) (string, error) {
	dir := WorkspaceDir(dataDir, workspaceName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create managed store directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, FileName(slug, format)+"-*.tmp")
	if err != nil {
		return "", fmt.Errorf("stage artifact: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("stage artifact: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("stage artifact: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("stage artifact: %w", err)
	}
	return tmpPath, nil
}

// Commit atomically renames a staged temp file into place. Call only after
// the database write has committed — see the invariant above.
func Commit(dataDir, workspaceName, slug, format, tmpPath string) error {
	if err := os.Rename(tmpPath, Path(dataDir, workspaceName, slug, format)); err != nil {
		return fmt.Errorf("commit artifact: %w", err)
	}
	return nil
}
