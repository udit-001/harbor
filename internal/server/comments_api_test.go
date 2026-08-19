package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udit-001/harbor/internal/db"
)

func seedCommentPage(t *testing.T, env *testEnv) string {
	t.Helper()
	wsID := env.workspaceID(t)
	if _, err := env.store.CreatePage(wsID, "Revenue Deep Dive", "quarterly", "board", "", "", "", "body", nil); err != nil {
		t.Fatalf("create page: %v", err)
	}
	return "revenue-deep-dive"
}

// postJSON issues a POST with the given JSON body and returns the recorder.
func postJSON(env *testEnv, target, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	env.mux.ServeHTTP(rec, req)
	return rec
}

func TestCommentsAPIEmptyThenCreate(t *testing.T) {
	env := newTestEnv(t)
	slug := seedCommentPage(t, env)
	target := "/api/pages/" + slug + "/comments"

	// Empty list first.
	var list []db.CommentView
	if err := json.Unmarshal(env.get(t, target).Body.Bytes(), &list); err != nil {
		t.Fatalf("decode empty list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty comment list, got %d", len(list))
	}

	// POST a selection comment → 201 + the created view enriched with page slug.
	rec := postJSON(env, target, `{"type":"selection","anchor":"h1","quote":"Revenue Deep Dive","body":"widen the hero"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201; body: %s", rec.Code, rec.Body.String())
	}
	var created db.CommentView
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.PageSlug != slug || created.Type != "selection" ||
		created.Anchor != "h1" || created.Quote != "Revenue Deep Dive" ||
		created.Status != db.CommentStatusOpen {
		t.Fatalf("created = %+v", created)
	}

	// List now reflects the write (the DB round trip).
	var after []db.CommentView
	json.Unmarshal(env.get(t, target).Body.Bytes(), &after)
	if len(after) != 1 || after[0].ID != created.ID {
		t.Fatalf("after create = %+v, want 1 comment #%d", after, created.ID)
	}

	// A general comment defaults type to general.
	g := postJSON(env, target, `{"body":"overall great"}`)
	if g.Code != http.StatusCreated {
		t.Fatalf("general POST status = %d; body: %s", g.Code, g.Body.String())
	}
	var gen db.CommentView
	json.Unmarshal(g.Body.Bytes(), &gen)
	if gen.Type != db.CommentTypeGeneral {
		t.Fatalf("empty type defaulted to %q, want general", gen.Type)
	}
}

func TestChangesAPI(t *testing.T) {
	env := newTestEnv(t)
	slug := seedCommentPage(t, env)
	target := "/api/pages/" + slug + "/changes"

	// Empty before any change is recorded.
	var empty []db.Change
	json.Unmarshal(env.get(t, target).Body.Bytes(), &empty)
	if len(empty) != 0 {
		t.Fatalf("expected empty changes, got %d", len(empty))
	}

	// Record a change via the store; the endpoint reports it with title/description.
	if _, err := env.store.CreateChange(slug, "cf-1", 0, "Widen hero", "expanded the hero to full width"); err != nil {
		t.Fatalf("create change: %v", err)
	}
	var list []db.Change
	json.Unmarshal(env.get(t, target).Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("changes = %d, want 1", len(list))
	}
	if list[0].ChangeID != "cf-1" || list[0].Title != "Widen hero" || list[0].Description != "expanded the hero to full width" {
		t.Fatalf("change = %+v", list[0])
	}

	// Missing page → 404.
	if env.get(t, "/api/pages/nope/changes").Code != http.StatusNotFound {
		t.Fatal("changes for missing page should 404")
	}
}

func TestOpenFeedbackIndicator(t *testing.T) {
	env := newTestEnv(t)
	slug := seedCommentPage(t, env) // "revenue-deep-dive"

	// No open comments → /api/pages carries feedbackOpen 0, no badge anywhere.
	var rows []map[string]any
	json.Unmarshal(env.get(t, "/api/pages").Body.Bytes(), &rows)
	for _, r := range rows {
		if r["slug"] == slug {
			if n := r["feedbackOpen"]; n != float64(0) {
				t.Fatalf("feedbackOpen before comment = %v, want 0", n)
			}
		}
	}
	const fbBadge = `<span class="fb"><i></i>1 open</span>`
	if got := env.get(t, "/").Body.String(); strings.Contains(got, fbBadge) {
		t.Fatalf("library should not show an open-feedback badge without comments")
	}
	if got := env.get(t, "/page/"+slug).Body.String(); strings.Contains(got, `id="pvFb"`) {
		t.Fatalf("page chrome should not show a feedback badge without comments")
	}

	// Open a comment → derived indicator appears everywhere (no persisted flag).
	rec := postJSON(env, "/api/pages/"+slug+"/comments", `{"type":"general","body":"widen the hero"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("post status = %d; body: %s", rec.Code, rec.Body.String())
	}

	json.Unmarshal(env.get(t, "/api/pages").Body.Bytes(), &rows)
	for _, r := range rows {
		if r["slug"] == slug {
			if n := r["feedbackOpen"]; n != float64(1) {
				t.Fatalf("feedbackOpen after comment = %v, want 1", n)
			}
		}
	}
	if got := env.get(t, "/").Body.String(); !strings.Contains(got, fbBadge) {
		t.Fatalf("library row missing open-feedback indicator:\n%s", got)
	}
	if got := env.get(t, "/page/"+slug).Body.String(); !strings.Contains(got, `id="pvFb"`) || !strings.Contains(got, "1 open") {
		t.Fatalf("page chrome missing feedback badge:\n%s", got)
	}
}

func TestCommentsAPIValidation(t *testing.T) {
	env := newTestEnv(t)
	slug := seedCommentPage(t, env)
	target := "/api/pages/" + slug + "/comments"

	// Empty body → 400.
	if rec := postJSON(env, target, `{"type":"general","body":""}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty body status = %d, want 400", rec.Code)
	}
	// Invalid type → 400.
	if rec := postJSON(env, target, `{"type":"bogus","body":"x"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad type status = %d, want 400", rec.Code)
	}
	// Malformed JSON → 400.
	if rec := postJSON(env, target, `{not json`); rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed status = %d, want 400", rec.Code)
	}
	// Missing page → 404 on both list and create.
	if env.get(t, "/api/pages/nope/comments").Code != http.StatusNotFound {
		t.Fatal("list comments on missing page should 404")
	}
	if rec := postJSON(env, "/api/pages/nope/comments", `{"body":"x"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("create comment on missing page status = %d, want 404", rec.Code)
	}
}

// patchJSON issues a PATCH with the given JSON body and returns the recorder.
func patchJSON(env *testEnv, target, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	env.mux.ServeHTTP(rec, req)
	return rec
}

func TestCommentsAPIMultiAnchorCreateAndEdit(t *testing.T) {
	env := newTestEnv(t)
	slug := seedCommentPage(t, env)
	target := "/api/pages/" + slug + "/comments"

	rec := postJSON(env, target,
		`{"body":"flag both rows","anchors":[{"kind":"text","path":"#row-a","quote":"off"},{"kind":"text","path":"#row-b","quote":"off2"}]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create code = %d: %s", rec.Code, rec.Body.String())
	}
	var created db.CommentView
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if len(created.Anchors) != 2 || created.Anchor != "#row-a" {
		t.Fatalf("multi-anchor create failed: %+v", created)
	}

	id := created.ID
	crec := patchJSON(env, fmt.Sprintf("/api/pages/%s/comments/%d", slug, id),
		`{"body":"revised","anchors":[{"kind":"element","path":".hero"}]}`)
	if crec.Code != http.StatusOK {
		t.Fatalf("edit code = %d: %s", crec.Code, crec.Body.String())
	}
	var edited db.CommentView
	if err := json.Unmarshal(crec.Body.Bytes(), &edited); err != nil {
		t.Fatalf("decode edited: %v", err)
	}
	if edited.Body != "revised" || len(edited.Anchors) != 1 || edited.Anchors[0].Kind != db.AnchorKindElement {
		t.Fatalf("edit not applied: %+v", edited)
	}
}
