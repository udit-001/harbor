package db

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func seedWorkspace(t *testing.T, store *Store, name string) *WorkspaceStore {
	t.Helper()
	_, err := store.AddWorkspace(Workspace{Name: name, Path: "/tmp/" + name})
	if err != nil {
		t.Fatalf("seed workspace %s: %v", name, err)
	}
	wsStore, err := store.Workspace(name)
	if err != nil {
		t.Fatalf("get workspace %s: %v", name, err)
	}
	return wsStore
}
