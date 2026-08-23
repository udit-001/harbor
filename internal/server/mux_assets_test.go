package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Pages reference workspace assets relatively (assets/foo.css); the
// /page/{slug}/assets/ route resolves them against the page's own workspace.
func TestPageAssetsServesFromPageWorkspace(t *testing.T) {
	env := newTestEnv(t)

	if err := os.WriteFile(filepath.Join(env.wsDir, "assets", "theme.css"), []byte("body{color:red}"), 0o644); err != nil {
		t.Fatal(err)
	}
	wsID := env.workspaceID(t)
	src := filepath.Join(t.TempDir(), "p.html")
	if err := os.WriteFile(src, []byte("<html><body>x</body></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.CreatePage(wsID, "asset page", "", "", "", "", src, "", nil); err != nil {
		t.Fatal(err)
	}

	if r := env.get(t, "/page/asset-page/assets/theme.css"); r.Code != http.StatusOK || r.Body.String() != "body{color:red}" {
		t.Fatalf("asset fetch: got %d %q", r.Code, r.Body.String())
	}

	// traversal is rejected
	if r := env.get(t, "/page/asset-page/assets/../NOTES.md"); r.Code == http.StatusOK && strings.Contains(r.Body.String(), "Real notes") {
		t.Fatal("traversal served workspace file")
	}
	// unknown asset 404s
	if r := env.get(t, "/page/asset-page/assets/nope.css"); r.Code != http.StatusNotFound {
		t.Fatalf("missing asset: got %d", r.Code)
	}
}
