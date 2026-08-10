package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/udit-001/harbor/internal/db"
)

// runWithStoreErr is like runWithStore but returns the error instead of
// fatalling, so fail-fast behavior can be asserted.
func runWithStoreErr(t *testing.T, args []string, store *db.Store) (string, error) {
	t.Helper()
	root := newRootForTest()
	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), ctxStore{}, store))
		return nil
	}
	ctx := context.WithValue(context.Background(), ctxStore{}, store)

	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	err := root.ExecuteContext(ctx)
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String(), err
}

func TestPageAddFailsFastOnMissingWorkspace(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // sandbox the managed-store data dir
	store, cleanup := newTestStore(t)
	defer cleanup()

	src := writeBodyFile(t, "<html><body><h1>hi</h1></body></html>")
	out, err := runWithStoreErr(t, []string{"page", "add", src, "--workspace", "missing"}, store)
	if err == nil {
		t.Fatalf("expected error for missing workspace:\n%s", out)
	}
	if !strings.Contains(err.Error(), "workspace create") {
		t.Fatalf("error should teach the workspace create command, got: %s", err.Error())
	}
}

func TestPageAddImportRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("ws", "ws", "the work", filepath.Join(home, "wsdir")); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	_ = runWithStore(t, []string{"tag", "create", "finance", "--description", "money"}, store)
	_ = runWithStore(t, []string{"tag", "create", "chart", "--description", "charts"}, store)

	src := writeBodyFile(t, "<html><body><p>monthly totals chart body</p><script>x</script></body></html>")
	out := runWithStore(t, []string{"page", "add", src, "--workspace", "ws",
		"--title", "totals chart", "--description", "totals chart", "--context", "built from /api", "--tag", "finance"}, store)

	if !strings.Contains(out, "Page imported") {
		t.Fatalf("add missing confirmation:\n%s", out)
	}

	// Managed copy landed in the sandboxed store, and body text was extracted
	// from it (script content must NOT leak into the FTS body).
	managed := filepath.Join(home, ".harbor", "store", "ws", "totals-chart.html")
	if _, err := os.Stat(managed); err != nil {
		t.Fatalf("managed file not written at %s: %v", managed, err)
	}

	// list shows it; search hits the extracted body text; read shows provenance.
	list := runWithStore(t, []string{"page", "list", "--workspace", "ws"}, store)
	if !strings.Contains(list, "totals-chart") {
		t.Fatalf("list missing page:\n%s", list)
	}
	search := runWithStore(t, []string{"page", "list", "--search", "totals chart body"}, store)
	if !strings.Contains(search, "totals-chart") {
		t.Fatalf("search missing page:\n%s", search)
	}
	read := runWithStore(t, []string{"page", "read", "totals-chart"}, store)
	for _, want := range []string{"totals chart", "built from /api", "finance"} {
		if !strings.Contains(read, want) {
			t.Fatalf("read missing %q:\n%s", want, read)
		}
	}

	// update status + full tag replace; slug stable.
	update := runWithStore(t, []string{"page", "update", "totals-chart", "--status", "published", "--tag", "chart"}, store)
	if !strings.Contains(update, "Page updated") {
		t.Fatalf("update missing confirmation:\n%s", update)
	}

	// delete removes the row.
	del := runWithStore(t, []string{"page", "delete", "totals-chart"}, store)
	if !strings.Contains(del, "Page deleted") {
		t.Fatalf("delete missing confirmation:\n%s", del)
	}
}
