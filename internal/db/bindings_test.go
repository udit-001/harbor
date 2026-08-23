package db

import (
	"path/filepath"
	"testing"
)

func TestBindAndResolveDeepest(t *testing.T) {
	store := newTestStore(t)
	seedWorkspace(t, store, "alpha")
	root := t.TempDir()
	sub := filepath.Join(root, "sub")

	if err := store.BindWorkspaceDir(root, "alpha"); err != nil {
		t.Fatal(err)
	}
	// bind a second workspace deeper
	if _, err := store.AddWorkspace(Workspace{Name: "beta", Topic: "b", Path: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if err := store.BindWorkspaceDir(sub, "beta"); err != nil {
		t.Fatal(err)
	}

	// deepest binding wins; ancestors inherit
	for dir, want := range map[string]string{
		root:                         "alpha",
		sub:                          "beta",
		filepath.Join(sub, "deeper"): "beta",
	} {
		got, err := store.WorkspaceForDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("WorkspaceForDir(%s) = %q, want %q", dir, got, want)
		}
	}

	// unrelated folder: no binding
	got, _ := store.WorkspaceForDir(t.TempDir())
	if got != "" {
		t.Errorf("unbound folder resolved to %q", got)
	}

	// re-binding replaces
	if err := store.BindWorkspaceDir(sub, "alpha"); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.WorkspaceForDir(sub); got != "alpha" {
		t.Errorf("re-bind failed: got %q", got)
	}
}

func TestUnbindAndUnknownWorkspace(t *testing.T) {
	store := newTestStore(t)
	seedWorkspace(t, store, "alpha")
	dir := t.TempDir()
	if err := store.BindWorkspaceDir(dir, "no-such-ws"); err == nil {
		t.Fatal("bound to unknown workspace")
	}
	if err := store.BindWorkspaceDir(dir, "alpha"); err != nil {
		t.Fatal(err)
	}
	if err := store.UnbindWorkspaceDir(dir); err != nil {
		t.Fatal(err)
	}
	if got, _ := store.WorkspaceForDir(dir); got != "" {
		t.Errorf("unbind failed: still resolves to %q", got)
	}
}
