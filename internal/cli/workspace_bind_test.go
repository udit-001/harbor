package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/udit-001/harbor/internal/db"
)

// Binding the caller's folder removes the --workspace requirement: page add
// resolves through the binding without the flag.
func TestPageAddResolvesBoundFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("boundws", "boundws", "", filepath.Join(home, "bws")); err != nil {
		t.Fatal(err)
	}

	cwd, _ := os.Getwd()
	if err := store.BindWorkspaceDir(cwd, "boundws"); err != nil {
		t.Fatal(err)
	}

	src := writeBodyFile(t, "<html><body><p>bound folder body</p></body></html>")
	out := runWithStore(t, []string{"page", "add", src, "--title", "no flag page"}, store)
	if !strings.Contains(out, "Page imported") {
		t.Fatalf("add failed:\n%s", out)
	}
	page, err := store.PageBySlug("no-flag-page")
	if err != nil {
		t.Fatalf("page missing after add: %v", err)
	}
	ws, err := store.GetWorkspace(page.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	if ws.Name != "boundws" {
		t.Fatalf("page landed in %q, want boundws", ws.Name)
	}
	_ = db.PageStatusDraft
}

// An explicit --workspace still wins over the folder binding.
func TestPageAddFlagBeatsBinding(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("boundws", "boundws", "", filepath.Join(home, "b1")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateWorkspace("other", "other", "", filepath.Join(home, "o1")); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	_ = store.BindWorkspaceDir(cwd, "boundws")

	src := writeBodyFile(t, "<html><body><p>x</p></body></html>")
	runWithStore(t, []string{"page", "add", src, "--title", "flag wins", "--workspace", "other"}, store)
	page, err := store.PageBySlug("flag-wins")
	if err != nil {
		t.Fatal(err)
	}
	ws, _ := store.GetWorkspace(page.WorkspaceID)
	if ws.Name != "other" {
		t.Fatalf("flag ignored: page in %q, want other", ws.Name)
	}
}
