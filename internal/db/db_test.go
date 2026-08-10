package db

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestOpenPoolSurvivesOneLeakedConnection is the regression test for the
// "server starts but never responds" deadlock.
//
// Root cause: Open previously set SetMaxOpenConns(1). With a single
// connection, ANY code path that checks out a connection and fails to
// return it (a *sql.Rows whose Close was skipped, an uncommitted tx, a
// goroutine that died mid-query) permanently wedges the pool — every later
// query blocks forever on pool acquisition. busy_timeout cannot help: it
// governs the SQLite file lock, not Go's pool checkout. The dashboard
// boots, the listener accepts connections, and every DB-touching route
// hangs with zero bytes returned.
//
// This test models the exact failure: it leaks ONE connection (the common
// case — a single unclosed Rows) and asserts a normal store query still
// completes within a deadline. Under SetMaxOpenConns(1) GetWorkspaces
// deadlocks until the test timeout; with a pool of >=2 it returns promptly.
func TestOpenPoolSurvivesOneLeakedConnection(t *testing.T) {
	store := newTestStore(t)
	db := store.SQL()

	// Leak exactly one connection and hold it for the whole test, mimicking
	// a *sql.Rows that was never closed. Do NOT call Close until cleanup.
	leaked, err := db.Conn(context.Background())
	if err != nil {
		t.Fatalf("checkout leak conn: %v", err)
	}
	t.Cleanup(func() { _ = leaked.Close() })

	// A normal store query must still complete. Under MaxOpenConns(1) this
	// blocks forever (the one conn is leaked); with a pool of >=2 it
	// returns fast on a different connection.
	done := make(chan error, 1)
	go func() {
		_, err := store.GetWorkspaces()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("GetWorkspaces after 1 leaked conn: %v", err)
		}
		// Success: the server keeps serving despite one leaked connection.
	case <-time.After(3 * time.Second):
		t.Fatalf("GetWorkspaces deadlocked after leaking 1 connection — " +
			"pool is sized so a single leak wedges everything (see maxOpenConns in db.go)")
	}
}

// TestOpenSetsSynchronousNormal asserts the WAL-recommended synchronous=NORMAL
// pragma is set on Open. NORMAL skips the per-commit fsync (fsync happens at
// checkpoint instead), shortening each write-lock hold — smaller collision
// window between agent CLI writes and human web writes on the shared file
// (LEARN-103). Values: 0=OFF, 1=NORMAL, 2=FULL, 3=EXTRA.
func TestOpenSetsSynchronousNormal(t *testing.T) {
	store := newTestStore(t)
	var sync int
	if err := store.SQL().QueryRow("PRAGMA synchronous").Scan(&sync); err != nil {
		t.Fatalf("query synchronous: %v", err)
	}
	if sync != 1 {
		t.Fatalf("PRAGMA synchronous = %d, want 1 (NORMAL)", sync)
	}
}

// TestFTSUpdateTriggersScopedToIndexedColumns asserts the pages_au trigger is
// scoped via AFTER UPDATE OF title, description, context, body_text so that
// non-indexed UPDATEs (status, updated_at, origin_path) skip the FTS
// delete+insert. If a future migration recreates the trigger un-scoped, this
// fails.
func TestFTSUpdateTriggersScopedToIndexedColumns(t *testing.T) {
	store := newTestStore(t)
	var sql string
	if err := store.SQL().QueryRow(
		"SELECT sql FROM sqlite_master WHERE type='trigger' AND name='pages_au'",
	).Scan(&sql); err != nil {
		t.Fatalf("query pages_au definition: %v", err)
	}
	if !strings.Contains(sql, "UPDATE OF") {
		t.Errorf("pages_au not scoped to indexed columns — missing 'UPDATE OF' in:\n%s", sql)
	}
}

// TestBaselineMigratesHarborSchema asserts the fresh Harbor-only migration set
// creates the library schema and nothing Pharos: the page library tables
// (workspaces/tags/pages/comments/changes) exist, the scratchpad (scraps) does
// NOT, and pages_fts is external-content over pages.
func TestBaselineMigratesHarborSchema(t *testing.T) {
	store := newTestStore(t)
	dbc := store.SQL()

	have := func(table string) bool {
		var n int
		if err := dbc.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&n); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		return n == 1
	}
	for _, table := range []string{"workspaces", "pages", "page_tags", "tags", "comments", "changes"} {
		if !have(table) {
			t.Errorf("baseline missing table %q", table)
		}
	}
	if have("scraps") {
		t.Error("scraps table should not exist in the Harbor baseline (Pharos cruft)")
	}

	// pages_fts must be external-content over pages (content-rowid sync).
	var ftsCreate string
	if err := dbc.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='pages_fts'").Scan(&ftsCreate); err != nil {
		t.Fatalf("query pages_fts: %v", err)
	}
	if !strings.Contains(ftsCreate, "content=pages") {
		t.Errorf("pages_fts not external content over pages:\n%s", ftsCreate)
	}

	// Foreign-key support is on, so join rows cascade.
	var fkOn int
	if err := dbc.QueryRow("PRAGMA foreign_keys").Scan(&fkOn); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if fkOn != 1 {
		t.Errorf("foreign_keys = %d, want 1", fkOn)
	}
}
