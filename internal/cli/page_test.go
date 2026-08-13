package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	managed := managedPagePath("ws", "totals-chart")
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

func TestSearchCommandAndRebuild(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("ws", "ws", "the work", filepath.Join(home, "wsdir")); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	src := writeBodyFile(t, "<html><body><p>the quarterly revenue hero insights</p></body></html>")
	_ = runWithStore(t, []string{"page", "add", src, "--workspace", "ws",
		"--title", "Revenue Deep Dive", "--description", "revenue analysis", "--context", "built for board review"}, store)

	// Search matches body text and title/description.
	for _, q := range []string{"quarterly revenue hero", "deep dive", "revenue analysis"} {
		out := runWithStore(t, []string{"search", q}, store)
		if !strings.Contains(out, "revenue-deep-dive") {
			t.Fatalf("search %q missing page:\n%s", q, out)
		}
	}

	// Rebuild is idempotent-safe to run (page already has body text).
	rebuild := runWithStore(t, []string{"search", "--rebuild-index"}, store)
	if !strings.Contains(rebuild, "Reindexed") {
		t.Fatalf("rebuild missing summary:\n%s", rebuild)
	}
}

// jsonTime parses one top-level timestamp field out of a page --json record.
func jsonTime(t *testing.T, out, field string) time.Time {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("page --json output not parseable: %v\n%s", err, out)
	}
	v, ok := m[field].(string)
	if !ok {
		t.Fatalf("json field %q missing or not a string: %s", field, out)
	}
	ts, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		// Pages get datetime('now') ("2006-01-02 15:04:05") on insert via
		// schema default but RFC3339Nano on update — parse both (both UTC).
		if ts2, err2 := time.Parse("2006-01-02 15:04:05", v); err2 == nil {
			return ts2
		}
		t.Fatalf("json field %q not a timestamp %q: %v", field, v, err)
	}
	return ts
}

func TestPageUpdateFileReplacesContent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("ws", "ws", "the work", filepath.Join(home, "wsdir")); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	old := writeBodyFile(t, "<html><body><p>old body marker text</p></body></html>")
	_ = runWithStore(t, []string{"page", "add", old, "--workspace", "ws",
		"--title", "totals chart", "--description", "a chart"}, store)

	managed := managedPagePath("ws", "totals-chart")
	data, err := os.ReadFile(managed)
	if err != nil {
		t.Fatalf("managed file missing after add: %v", err)
	}
	if !strings.Contains(string(data), "old body marker text") {
		t.Fatalf("managed file missing original content:\n%s", data)
	}

	before := jsonTime(t, runWithStore(t, []string{"page", "read", "totals-chart", "--json"}, store), "updatedAt")

	newSrc := writeBodyFile(t, "<html><body><p>brand new body marker text</p></body></html>")
	out := runWithStore(t, []string{"page", "update", "totals-chart", "--file", newSrc}, store)
	if !strings.Contains(out, "Page updated") {
		t.Fatalf("update missing confirmation:\n%s", out)
	}

	// The managed file — the content harbor serves — is the new HTML.
	data, err = os.ReadFile(managed)
	if err != nil {
		t.Fatalf("managed file missing after update: %v", err)
	}
	if !strings.Contains(string(data), "brand new body marker text") {
		t.Fatalf("managed file not replaced with new content:\n%s", data)
	}
	if strings.Contains(string(data), "old body marker text") {
		t.Fatalf("managed file still holds stale content:\n%s", data)
	}

	// FTS index refreshed: search finds the new body text and not the old.
	if got := runWithStore(t, []string{"page", "list", "--search", "brand new body"}, store); !strings.Contains(got, "totals-chart") {
		t.Fatalf("search for new content missed the page:\n%s", got)
	}
	if got := runWithStore(t, []string{"page", "list", "--search", "old body marker"}, store); strings.Contains(got, "totals-chart") {
		t.Fatalf("search still returns stale body text:\n%s", got)
	}

	// updated_at bumped (nanosecond precision makes same-second flake impossible).
	after := jsonTime(t, runWithStore(t, []string{"page", "read", "totals-chart", "--json"}, store), "updatedAt")
	if !after.After(before) {
		t.Fatalf("updated_at not bumped: before=%s after=%s", before, after)
	}
}

func TestPageUpdateFileAloneIsValid(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("ws", "ws", "the work", filepath.Join(home, "wsdir")); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	src := writeBodyFile(t, "<html><body><p>content</p></body></html>")
	_ = runWithStore(t, []string{"page", "add", src, "--workspace", "ws",
		"--title", "totals chart"}, store)

	newSrc := writeBodyFile(t, "<html><body><p>fresh content</p></body></html>")
	out := runWithStore(t, []string{"page", "update", "totals-chart", "--file", newSrc}, store)
	if !strings.Contains(out, "Page updated") {
		t.Fatalf("--file alone should be a valid update, got:\n%s", out)
	}
}

func TestPageUpdateFileMissingPathErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("ws", "ws", "the work", filepath.Join(home, "wsdir")); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	src := writeBodyFile(t, "<html><body><p>content</p></body></html>")
	_ = runWithStore(t, []string{"page", "add", src, "--workspace", "ws",
		"--title", "totals chart"}, store)

	missing := filepath.Join(home, "does-not-exist.html")
	_, err := runWithStoreErr(t, []string{"page", "update", "totals-chart", "--file", missing}, store)
	if err == nil {
		t.Fatalf("expected error for missing --file path")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Fatalf("error should name the missing path %q, got: %s", missing, err.Error())
	}
}

func TestPageUpdateFileEmptyErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("ws", "ws", "the work", filepath.Join(home, "wsdir")); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	src := writeBodyFile(t, "<html><body><p>content</p></body></html>")
	_ = runWithStore(t, []string{"page", "add", src, "--workspace", "ws",
		"--title", "totals chart"}, store)

	empty := writeBodyFile(t, "   \n  ")
	_, err := runWithStoreErr(t, []string{"page", "update", "totals-chart", "--file", empty}, store)
	if err == nil {
		t.Fatalf("expected error for empty --file content")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error should call out empty content, got: %s", err.Error())
	}
}

// A metadata-only update (no --file) must NOT touch body text: without the
// else-branch re-extraction, update only touches metadata, so the search
// index must still find the original content.
func TestPageUpdateMetadataOnlyKeepsBodyText(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("ws", "ws", "the work", filepath.Join(home, "wsdir")); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	src := writeBodyFile(t, "<html><body><p>metadata only keeps me searchable</p></body></html>")
	_ = runWithStore(t, []string{"page", "add", src, "--workspace", "ws",
		"--title", "totals chart"}, store)

	// Metadata-only update: status flips, body text must stay indexed.
	out := runWithStore(t, []string{"page", "update", "totals-chart", "--status", "published"}, store)
	if !strings.Contains(out, "Page updated") {
		t.Fatalf("update missing confirmation:\n%s", out)
	}
	if got := runWithStore(t, []string{"page", "list", "--search", "metadata only keeps"}, store); !strings.Contains(got, "totals-chart") {
		t.Fatalf("metadata-only update dropped body text from search:\n%s", got)
	}
}

// The documented combo --file + --status must apply both halves: content push
// AND the metadata flip, in one command.
func TestPageUpdateFileWithStatusCombo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	store, cleanup := newTestStore(t)
	defer cleanup()

	if _, err := store.CreateWorkspace("ws", "ws", "the work", filepath.Join(home, "wsdir")); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}

	src := writeBodyFile(t, "<html><body><p>old body marker text</p></body></html>")
	_ = runWithStore(t, []string{"page", "add", src, "--workspace", "ws",
		"--title", "totals chart"}, store)

	newSrc := writeBodyFile(t, "<html><body><p>combo new body marker text</p></body></html>")
	out := runWithStore(t, []string{"page", "update", "totals-chart", "--file", newSrc, "--status", "published"}, store)
	if !strings.Contains(out, "Page updated") {
		t.Fatalf("update missing confirmation:\n%s", out)
	}

	// Managed file replaced with new content.
	managed := managedPagePath("ws", "totals-chart")
	data, err := os.ReadFile(managed)
	if err != nil {
		t.Fatalf("managed file missing after combo update: %v", err)
	}
	if !strings.Contains(string(data), "combo new body marker text") {
		t.Fatalf("managed file not replaced with new content:\n%s", data)
	}

	// Search finds the new body, not the old.
	if got := runWithStore(t, []string{"page", "list", "--search", "combo new body"}, store); !strings.Contains(got, "totals-chart") {
		t.Fatalf("search for new content missed the page:\n%s", got)
	}
	if got := runWithStore(t, []string{"page", "list", "--search", "old body marker"}, store); strings.Contains(got, "totals-chart") {
		t.Fatalf("search still returns stale body text:\n%s", got)
	}

	// Status flipped to published.
	read := runWithStore(t, []string{"page", "read", "totals-chart", "--json"}, store)
	if !strings.Contains(read, "\"status\": \"published\"") {
		t.Fatalf("status not flipped to published:\n%s", read)
	}
}
