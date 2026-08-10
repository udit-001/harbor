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
	store   *db.Store
	mux     *http.ServeMux
	wsDir   string
	dataDir string
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

	dataDir := filepath.Join(dir, "store")
	return &testEnv{store: store, mux: NewMux(store, dataDir, false), wsDir: wsDir, dataDir: dataDir}
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

func TestLibraryHome(t *testing.T) {
	env := newTestEnv(t)
	rec := env.get(t, "/")
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	checks := []struct{ name, want string }{
		{"brand", "harbor"},
		{"page title", "All pages"},
		{"empty prompt", "harbor page add"},
		{"search box", `id="q"`},
		{"status segment", "Published"},
	}
	for _, c := range checks {
		if !strings.Contains(body, c.want) {
			t.Errorf("library HTML missing %s (%q)", c.name, c.want)
		}
	}
}

func TestLibraryListsSeededPages(t *testing.T) {
	env := newTestEnv(t)
	wsID := env.workspaceID(t)
	if _, err := env.store.CreateTag("finance", "money"); err != nil {
		t.Fatal(err)
	}
	if _, err := env.store.CreatePage(wsID, "Revenue Deep Dive", "quarterly analysis", "board", "",
		"", "revenue hero body", []string{"finance"}); err != nil {
		t.Fatalf("create page: %v", err)
	}

	// Home renders the row with workspace + tag + status.
	body := env.get(t, "/").Body.String()
	for _, want := range []string{`href="/page/revenue-deep-dive"`, "Revenue Deep Dive", "alpha", "finance"} {
		if !strings.Contains(body, want) {
			t.Errorf("library home missing %q", want)
		}
	}

	// Status segment renders backend: published shows the row, draft hides it.
	if got := env.get(t, "/?status=published").Body.String(); strings.Contains(got, "Revenue Deep Dive") {
		t.Error("published filter should not show a draft page")
	}
	if got := env.get(t, "/?status=draft").Body.String(); !strings.Contains(got, "Revenue Deep Dive") {
		t.Error("draft filter should show the draft page")
	}

	// Live-search fragment endpoint returns the row by body text.
	frag := env.get(t, "/api/pages?q=revenue+hero").Body.String()
	if !strings.Contains(frag, "Revenue Deep Dive") {
		t.Errorf("live search fragment missing page: %s", frag)
	}
	// And the empty-state fragment for a no-match query.
	fragEmpty := env.get(t, "/api/pages?q=zzzqqq").Body.String()
	if !strings.Contains(fragEmpty, "No pages yet") {
		t.Errorf("empty fragment missing empty state: %s", fragEmpty)
	}
}

func TestPageViewAndRaw(t *testing.T) {
	env := newTestEnv(t)
	wsID := env.workspaceID(t)

	const html = `<!doctype html><html><head><title>deep</title></head><body><h1>Revenue Deep Dive</h1><p>raw hero</p></body></html>`
	if _, err := env.store.CreatePage(wsID, "Revenue Deep Dive", "quarterly", "board", "", "", "raw hero", nil); err != nil {
		t.Fatalf("create page: %v", err)
	}
	// Managed file byte-for-byte (no injection/restyle): the server serves it as-is.
	dir := filepath.Join(env.dataDir, "store", "alpha")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "revenue-deep-dive.html"), []byte(html), 0644)

	// Raw serves the exact bytes.
	raw := env.get(t, "/page/revenue-deep-dive/raw")
	if raw.Code != 200 {
		t.Fatalf("raw status = %d, want 200", raw.Code)
	}
	if got := raw.Body.String(); got != html {
		t.Errorf("raw body not byte-for-byte:\n%q", got)
	}

	// Page view renders the header + iframe onto the raw URL, title intact.
	view := env.get(t, "/page/revenue-deep-dive").Body.String()
	for _, want := range []string{`src="/page/revenue-deep-dive/raw"`, "Revenue Deep Dive", `data-slug="revenue-deep-dive"`, "Pop out", "alpha"} {
		if !strings.Contains(view, want) {
			t.Errorf("page view missing %q", want)
		}
	}

	// Missing page/raw → 404.
	if env.get(t, "/page/nope/raw").Code != 404 {
		t.Error("missing raw page should 404")
	}
	if env.get(t, "/page/nope").Code != 404 {
		t.Error("missing page view should 404")
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
