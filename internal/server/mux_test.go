package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/udit-001/harbor/internal/db"
)

// testEnv bundles a store and mux for server tests. The store uses a real
// SQLite temp file (matching db test pattern). Workspace files are written
// to a real temp dir so handlers that read from disk work correctly.
type testEnv struct {
	store *db.Store
	mux   *http.ServeMux
	wsDir string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	store, err := db.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	wsDir := filepath.Join(dir, "alpha")
	os.MkdirAll(filepath.Join(wsDir, "assets"), 0755)

	store.AddWorkspace(db.Workspace{Name: "alpha", Topic: "Alpha", Path: wsDir})

	// Workspace documents.
	os.WriteFile(filepath.Join(wsDir, "RESOURCES.md"), []byte("{some placeholder}"), 0644)
	os.WriteFile(filepath.Join(wsDir, "NOTES.md"), []byte("# Notes\n\nReal notes"), 0644)

	return &testEnv{store: store, mux: NewMux(store, false), wsDir: wsDir}
}

func (e *testEnv) get(t *testing.T, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", target, nil)
	e.mux.ServeHTTP(rec, req)
	return rec
}

func (e *testEnv) workspaceID(t *testing.T) int64 {
	t.Helper()
	rec := e.get(t, "/api/workspaces")
	var wsList []db.Workspace
	json.Unmarshal(rec.Body.Bytes(), &wsList)
	if len(wsList) == 0 {
		t.Fatal("no workspaces in test store")
	}
	return wsList[0].ID
}

// ── Smoke tests: surviving routes return 200 + correct content-type ──

func TestSmokeAPIRoutes(t *testing.T) {
	env := newTestEnv(t)
	wsID := env.workspaceID(t)
	id := strconv.FormatInt(wsID, 10)

	cases := []struct {
		name        string
		path        string
		wantContent string
	}{
		{"workspaces", "/api/workspaces", "application/json"},
		{"workspace-by-id", "/api/workspaces/" + id, "application/json"},
		{"stats", "/api/stats", "application/json"},
		{"search", "/api/search?q=alpha", "application/json"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := env.get(t, c.path)
			if rec.Code != 200 {
				t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, c.wantContent) {
				t.Errorf("content-type = %q, want prefix %q", ct, c.wantContent)
			}
		})
	}
}

func TestSmokePageRoutes(t *testing.T) {
	env := newTestEnv(t)

	cases := []struct {
		name string
		path string
	}{
		{"css", "/css/app.css"},
		{"dashboard", "/"},
		{"workspace", "/workspace/alpha"},
		{"about", "/about"},
		{"resources", "/workspace/alpha/resources"},
		{"notes", "/workspace/alpha/notes"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := env.get(t, c.path)
			if rec.Code != 200 {
				t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body.String()[:min(200, rec.Body.Len())])
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, "text/") {
				t.Errorf("content-type = %q, want text/ prefix", ct)
			}
		})
	}
}

func TestPWAStaticRoutes(t *testing.T) {
	env := newTestEnv(t)
	cases := []struct {
		name, path, wantContent string
	}{
		{"manifest", "/manifest.webmanifest", "application/manifest+json"},
		{"service-worker", "/sw.js", "application/javascript"},
		{"stopped-page", "/stopped.html", "text/html"},
		{"icon-192", "/icon-192.png", "image/png"},
		{"icon-512", "/icon-512.png", "image/png"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := env.get(t, c.path)
			if rec.Code != 200 {
				t.Errorf("status = %d, want 200", rec.Code)
			}
			ct := rec.Header().Get("Content-Type")
			if !strings.HasPrefix(ct, c.wantContent) {
				t.Errorf("content-type = %q, want prefix %q", ct, c.wantContent)
			}
		})
	}
}

func TestPWAHeadTags(t *testing.T) {
	env := newTestEnv(t)
	rec := env.get(t, "/")
	body := rec.Body.String()

	checks := []struct{ name, want string }{
		{"manifest link", `rel="manifest" href="/manifest.webmanifest"`},
		{"sentinel meta", `<meta name="harbor-app" content="1">`},
		{"theme-color", `<meta name="theme-color" id="theme-color" content="#eceff4">`},
		{"apple-touch-icon", `rel="apple-touch-icon" href="/icon-192.png"`},
		{"sw registration", "navigator.serviceWorker.register('/sw.js')"},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.want) {
			t.Errorf("dashboard HTML missing %s", c.name)
		}
	}
}

func TestDocPagePlaceholderDetection(t *testing.T) {
	env := newTestEnv(t)

	// Notes has real content → should render.
	rec := env.get(t, "/workspace/alpha/notes")
	body := rec.Body.String()
	if !strings.Contains(body, "Real notes") {
		t.Error("notes page should render real content")
	}

	// Resources has {some placeholder} → should render empty state, not raw template.
	rec = env.get(t, "/workspace/alpha/resources")
	body = rec.Body.String()
	if strings.Contains(body, "{some placeholder}") {
		t.Error("resources page should not render raw placeholder template content")
	}
}

func TestWorkspaceNotFound(t *testing.T) {
	env := newTestEnv(t)

	rec := env.get(t, "/workspace/nonexistent")
	if rec.Code != 404 {
		t.Errorf("nonexistent workspace should 404; got %d", rec.Code)
	}
}

func TestSearchAPIOverScraps(t *testing.T) {
	env := newTestEnv(t)

	// Seed a scrap (scratchpad is global — no workspace needed).
	if _, err := env.store.CreateScrap("machine learning roadmap", "linear algebra first", []string{}); err != nil {
		t.Fatalf("create scrap: %v", err)
	}

	rec := env.get(t, "/api/search?q=algebra")
	if rec.Code != 200 {
		t.Fatalf("search status = %d; body: %s", rec.Code, rec.Body.String())
	}
	var results []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("search for 'algebra' returned no scrap results")
	}
	if results[0]["title"] != "machine learning roadmap" {
		t.Errorf("first result title = %v, want the seeded scrap", results[0]["title"])
	}
}
