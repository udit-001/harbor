package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/udit-001/harbor/internal/db"
)

func seedCommentPage(t *testing.T, env *testEnv) string {
	t.Helper()
	wsID := env.workspaceID(t)
	if _, err := env.store.CreatePage(wsID, "Revenue Deep Dive", "quarterly", "board", "", "", "body", nil); err != nil {
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
