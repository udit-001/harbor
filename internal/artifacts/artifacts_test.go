package artifacts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStageCommitRoundtrip(t *testing.T) {
	dataDir := t.TempDir()
	tmp, err := Stage(dataDir, "ws", "my-page", "markdown", []byte("# hello"))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp) // no-op after commit
	if filepath.Dir(tmp) != WorkspaceDir(dataDir, "ws") {
		t.Fatalf("staged outside workspace dir: %s", tmp)
	}
	// final path must not exist before commit…
	if _, err := os.Stat(Path(dataDir, "ws", "my-page", "markdown")); !os.IsNotExist(err) {
		t.Fatal("final path exists before commit")
	}
	if err := Commit(dataDir, "ws", "my-page", "markdown", tmp); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(Path(dataDir, "ws", "my-page", "markdown"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# hello" {
		t.Fatalf("content mismatch: %q", got)
	}
}

func TestCommitReplacesAtomically(t *testing.T) {
	dataDir := t.TempDir()

	tmp1, err := Stage(dataDir, "ws", "p", "html", []byte("v1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Commit(dataDir, "ws", "p", "html", tmp1); err != nil {
		t.Fatal(err)
	}

	tmp2, err := Stage(dataDir, "ws", "p", "html", []byte("v2"))
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp2)
	if err := Commit(dataDir, "ws", "p", "html", tmp2); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(Path(dataDir, "ws", "p", "html"))
	if string(got) != "v2" {
		t.Fatalf("replace failed: %q", got)
	}
	entries, _ := os.ReadDir(WorkspaceDir(dataDir, "ws"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp litter: %s", e.Name())
		}
	}
}
