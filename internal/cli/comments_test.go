package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/udit-001/harbor/internal/db"
)

func seedCommentsForCLI(t *testing.T, s *db.Store) {
	t.Helper()
	ws, err := s.CreateWorkspace("ws", "ws", "the work", t.TempDir())
	if err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	if _, err := s.CreatePage(ws.ID, "Monthly Totals", "d", "c", "", "", "body", nil); err != nil {
		t.Fatalf("seed page: %v", err)
	}
	if _, err := s.CreateComment("monthly-totals", "#chart", "the chart", db.CommentTypeSelection, "widen it"); err != nil {
		t.Fatalf("seed comment: %v", err)
	}
}

func TestCommentsListAndUpdate(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()
	seedCommentsForCLI(t, store)

	list := runWithStore(t, []string{"comments", "list"}, store)
	for _, want := range []string{"#1", "monthly-totals", "selection", "widen it"} {
		if !strings.Contains(list, want) {
			t.Fatalf("list missing %q:\n%s", want, list)
		}
	}

	// JSON list.
	jsonOut := runWithStore(t, []string{"comments", "list", "--json"}, store)
	if !strings.Contains(jsonOut, `"page": "monthly-totals"`) {
		t.Fatalf("json list missing page:\n%s", jsonOut)
	}

	// update status
	upd := runWithStore(t, []string{"comments", "update", "1", "--status", "in-progress"}, store)
	if !strings.Contains(upd, "in-progress") {
		t.Fatalf("update missing status:\n%s", upd)
	}
	// after moving, the open-list (default) is empty.
	empty := runWithStore(t, []string{"comments", "list"}, store)
	if strings.Contains(empty, "widen it") {
		t.Fatalf("list should not show non-open comment:\n%s", empty)
	}
	done := runWithStore(t, []string{"comments", "list", "--status", "in-progress"}, store)
	if !strings.Contains(done, "widen it") {
		t.Fatalf("list --status in-progress missing comment:\n%s", done)
	}
}

func TestCommentsListEmptyState(t *testing.T) {
	store, cleanup := newTestStore(t)
	defer cleanup()

	out := runWithStore(t, []string{"comments", "list"}, store)
	if !strings.Contains(out, "No open comments") {
		t.Fatalf("empty state missing:\n%s", out)
	}
}

// mkView builds a CommentView for watcher tests.
func mkView(id int64, page, quote, body string) db.CommentView {
	return db.CommentView{
		Comment: db.Comment{
			ID:    id,
			Quote: quote,
			Body:  body,
		},
		PageSlug: page,
	}
}

func TestCommentWatcherDedupesAndTails(t *testing.T) {
	var out bytes.Buffer
	poll := make(chan time.Time)
	done := make(chan struct{})

	fetches := [][]db.CommentView{
		{mkView(1, "a", "q1", "first body")},                               // initial echo
		{mkView(1, "a", "q1", "first body"), mkView(2, "b", "", "second")}, // new arrives
		{mkView(2, "b", "", "second")},                                     // quiet tick
	}
	i := 0
	fetch := func() ([]db.CommentView, error) {
		if i >= len(fetches) {
			return fetches[len(fetches)-1], nil
		}
		out := fetches[i]
		i++
		return out, nil
	}

	go func() {
		time.Sleep(5 * time.Millisecond)
		poll <- time.Now()
		time.Sleep(5 * time.Millisecond)
		poll <- time.Now()
		time.Sleep(5 * time.Millisecond)
		close(done)
	}()

	if err := commentWatcher(&out, fetch, poll, done); err != nil {
		t.Fatalf("watcher: %v", err)
	}

	got := out.String()
	for _, want := range []string{"[#1] a / \"q1\" / first body", "[#2] b / second"} {
		if !strings.Contains(got, want) {
			t.Fatalf("watcher missing %q:\n%s", want, got)
		}
	}
	// #1 must be echoed exactly once (deduped across ticks).
	if strings.Count(got, "[#1] a") != 1 {
		t.Fatalf("comment #1 echoed more than once:\n%s", got)
	}
}
