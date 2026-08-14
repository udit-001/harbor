package cli

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestResolveDataDirEnvOverride guards the HARB-14 fix: the daemon child must
// resolve the same data dir the parent resolved, via HARBOR_DATA_DIR.
func TestResolveDataDirEnvOverride(t *testing.T) {
	t.Setenv("HARBOR_DATA_DIR", "/tmp/override-harbor")
	if got := resolveDataDir(); got != "/tmp/override-harbor" {
		t.Fatalf("resolveDataDir = %q, want the HARBOR_DATA_DIR override", got)
	}
}

func TestResolveDataDirClearsAfterEnvUnset(t *testing.T) {
	// Without the env override, resolution falls back to config/default — and
	// must NOT remember the previous env value (no accidental leak).
	t.Setenv("HARBOR_DATA_DIR", "")
	if got := resolveDataDir(); got == "" {
		t.Fatalf("resolveDataDir must never return empty")
	}
}

// TestStopDoesNotOpenDatabase guards the fix that moved `stop` into the
// PersistentPreRunE skip list: stop only needs the pid file, so opening the
// DB is wasted work and can block up to busy_timeout (5s) when the running
// server holds the database. With an unopenable DB, stop must still succeed.
func TestStopDoesNotOpenDatabase(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg"))
	// Make the data dir a regular file so db.Open fails if it is invoked.
	bogus := filepath.Join(dir, "data")
	if err := os.WriteFile(bogus, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HARBOR_DATA_DIR", bogus)

	root := newRootForTest()
	root.SetArgs([]string{"stop"})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	if err := root.Execute(); err != nil {
		t.Fatalf("harbor stop with an unopenable DB must succeed: %v", err)
	}
}
