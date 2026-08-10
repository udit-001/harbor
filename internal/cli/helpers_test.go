package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/udit-001/harbor/internal/db"
)

// newTestStore opens a fresh temp SQLite database with migrations applied.
// Returned cleanup closes the store and removes the temp file.
func newTestStore(t *testing.T) (*db.Store, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	store, err := db.Open(path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return store, func() { _ = store.Close() }
}

// newRootForTest returns the shared root command (with all init()-registered
// subcommands), but with every flag reset to its registered default so state
// never leaks between test executions on the shared command objects. Callers
// set SetArgs + PersistentPreRunE before Execute.
func newRootForTest() *cobra.Command {
	resetCommandFlags(rootCmd)
	return rootCmd
}

// resetCommandFlags resets every flag (local and persistent) on a command and
// its whole subtree to its DefValue. cobra re-uses the same command objects
// across test runs, so a flag left set by a previous Execute (e.g. --tag)
// would otherwise poison later runs.
func resetCommandFlags(c *cobra.Command) {
	resetFlagSet(c.Flags())
	resetFlagSet(c.PersistentFlags())
	for _, sub := range c.Commands() {
		resetCommandFlags(sub)
	}
}

func resetFlagSet(fs *pflag.FlagSet) {
	fs.VisitAll(func(f *pflag.Flag) {
		switch f.Value.Type() {
		case "stringSlice", "stringArray":
			// Composite pflag values append rather than replace on Set, so a
			// previous run's elements (e.g. --tag ml) would otherwise leak.
			// Their concrete type exposes Replace, which resets them cleanly.
			if r, ok := f.Value.(interface{ Replace([]string) error }); ok {
				_ = r.Replace(nil)
			}
		default:
			_ = f.Value.Set(f.DefValue)
		}
	})
}

// runWithStore executes the named cobra subcommand with the given store
// injected via context (the seam created in LEARN-9). Captures stdout and
// fails the test on command error.
func runWithStore(t *testing.T, args []string, store *db.Store) string {
	t.Helper()
	root := newRootForTest()
	root.SetArgs(args)
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), ctxStore{}, store))
		return nil
	}
	root.PersistentPostRunE = nil
	ctx := context.WithValue(context.Background(), ctxStore{}, store)

	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	err := root.ExecuteContext(ctx)
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if err != nil {
		t.Fatalf("run %v: %v\n%s", args, err, buf.String())
	}
	return buf.String()
}
