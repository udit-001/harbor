package db

import (
	"testing"
)

// seedPageWorkspace inserts a workspace row so page tests can reference a real
// workspace_id. Returns the workspace ID.
func seedPageWorkspace(t *testing.T, s *Store, name string) int64 {
	t.Helper()
	w, err := s.AddWorkspace(Workspace{Name: name, Path: "/tmp/" + name})
	if err != nil {
		t.Fatalf("seed workspace %s: %v", name, err)
	}
	return w.ID
}

func TestPageCreateGetListUpdateDelete(t *testing.T) {
	s := newTestStore(t)
	wsID := seedPageWorkspace(t, s, "income-tracker")

	if _, err := s.CreateTag("finance", "money stuff"); err != nil {
		t.Fatalf("create tag: %v", err)
	}

	created, err := s.CreatePage(wsID, "Monthly Totals", "monthly totals chart",
		"built from /api/reports; prototype v2", "", "/origin/report.html", "monthly report body", []string{"finance"})
	if err != nil {
		t.Fatalf("create page: %v", err)
	}
	if created.Slug != "monthly-totals" {
		t.Fatalf("slug = %q, want %q", created.Slug, "monthly-totals")
	}
	if created.Status != PageStatusDraft {
		t.Fatalf("default status = %q, want draft", created.Status)
	}
	if created.WorkspaceID != wsID {
		t.Fatalf("workspace_id = %d, want %d", created.WorkspaceID, wsID)
	}
	if created.OriginPath != "/origin/report.html" {
		t.Fatalf("origin_path = %q", created.OriginPath)
	}

	got, err := s.PageBySlug("monthly-totals")
	if err != nil {
		t.Fatalf("get by slug: %v", err)
	}
	if got.Title != "Monthly Totals" || got.Description != "monthly totals chart" {
		t.Fatalf("got = %+v", got)
	}

	tags, err := s.TagsForPage("monthly-totals")
	if err != nil {
		t.Fatalf("tags for page: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "finance" {
		t.Fatalf("tags = %+v, want [finance]", tags)
	}

	// List by status / workspace / tag filters.
	if pages, err := s.ListPages(PageFilter{Status: PageStatusPublished}); err != nil || len(pages) != 0 {
		t.Fatalf("list published (expect 0): %+v, err=%v", pages, err)
	}
	if pages, err := s.ListPages(PageFilter{Status: PageStatusDraft}); err != nil || len(pages) != 1 {
		t.Fatalf("list draft (expect 1): %+v, err=%v", pages, err)
	}
	if pages, err := s.ListPages(PageFilter{WorkspaceSlug: "income-tracker"}); err != nil || len(pages) != 1 {
		t.Fatalf("list by workspace (expect 1): %+v, err=%v", pages, err)
	}
	if pages, err := s.ListPages(PageFilter{TagName: "finance"}); err != nil || len(pages) != 1 {
		t.Fatalf("list by tag (expect 1): %+v, err=%v", pages, err)
	}
	if pages, err := s.ListPages(PageFilter{TagName: "missing"}); err != nil || len(pages) != 0 {
		t.Fatalf("list by missing tag (expect 0): %+v, err=%v", pages, err)
	}

	// Search across body text and tag descriptions.
	if pages, err := s.SearchPages("report body", PageFilter{}); err != nil || len(pages) != 1 {
		t.Fatalf("search body (expect 1): %+v, err=%v", pages, err)
	}
	if pages, err := s.SearchPages("money", PageFilter{}); err != nil || len(pages) != 1 {
		t.Fatalf("search by tag description (expect 1): %+v, err=%v", pages, err)
	}
	if pages, err := s.SearchPages("monthly totals", PageFilter{Status: PageStatusDraft}); err != nil || len(pages) != 1 {
		t.Fatalf("search by title w/ filter (expect 1): %+v, err=%v", pages, err)
	}

	// Update: rename + status; slug must stay stable.
	published := PageStatusPublished
	foo := "foo"
	updated, err := s.UpdatePage("monthly-totals", &foo, nil, nil, &published, nil, nil, nil)
	if err != nil {
		t.Fatalf("update page: %v", err)
	}
	if updated.Slug != "monthly-totals" {
		t.Fatalf("slug changed on rename: %q", updated.Slug)
	}
	if updated.Title != "foo" || updated.Status != PageStatusPublished {
		t.Fatalf("updated = %+v, want title=foo status=published", updated)
	}
	if updated.Description != "monthly totals chart" || updated.Context != "built from /api/reports; prototype v2" {
		t.Fatalf("unchanged pointer fields were clobbered: %+v", updated)
	}

	// Replace the tag set on update (create-first: must exist first).
	if _, err := s.CreateTag("chart", "charts"); err != nil {
		t.Fatalf("create tag chart: %v", err)
	}
	if _, err := s.UpdatePage("monthly-totals", nil, nil, nil, nil, nil, nil, &[]string{"chart"}); err != nil {
		t.Fatalf("replace tags: %v", err)
	}
	tags, _ = s.TagsForPage("monthly-totals")
	if len(tags) != 1 || tags[0].Name != "chart" {
		t.Fatalf("after replace tags = %+v, want [chart]", tags)
	}

	if err := s.DeletePage("monthly-totals"); err != nil {
		t.Fatalf("delete page: %v", err)
	}
	if _, err := s.PageBySlug("monthly-totals"); err == nil {
		t.Fatalf("expected not-found after delete")
	}
}

func TestPageCreateValidations(t *testing.T) {
	s := newTestStore(t)
	wsID := seedPageWorkspace(t, s, "work")

	if _, err := s.CreatePage(wsID, "", "d", "c", "", "", "", nil); err == nil {
		t.Fatalf("expected error for empty title")
	}

	if _, err := s.CreatePage(wsID, "ok", "d", "c", "in-progress", "", "", nil); err == nil {
		t.Fatalf("expected error for invalid status")
	}

	// Create-first tag rule: attaching a tag that doesn't exist must fail.
	if _, err := s.CreatePage(wsID, "p", "d", "c", "", "", "", []string{"nope"}); err == nil {
		t.Fatalf("expected error for missing tag")
	}
}

func TestPageReindexHelpers(t *testing.T) {
	s := newTestStore(t)
	wsID := seedPageWorkspace(t, s, "ws")

	if _, err := s.CreatePage(wsID, "with-body", "d", "c", "", "", "hello body", nil); err != nil {
		t.Fatalf("create with-body: %v", err)
	}
	if _, err := s.CreatePage(wsID, "empty-body", "d", "c", "", "", "", nil); err != nil {
		t.Fatalf("create empty-body: %v", err)
	}

	// Only the empty-bodied page needs reindexing (idempotency: the indexed
	// page is left alone).
	needs, err := s.PagesNeedingReindex()
	if err != nil {
		t.Fatalf("pages needing reindex: %v", err)
	}
	if len(needs) != 1 || needs[0].Slug != "empty-body" {
		t.Fatalf("needs reindex = %+v, want [empty-body]", needs)
	}

	// Clearing forces a full re-index (both pages now need it).
	if err := s.ClearPageBodies(); err != nil {
		t.Fatalf("clear bodies: %v", err)
	}
	needs, _ = s.PagesNeedingReindex()
	if len(needs) != 2 {
		t.Fatalf("after clear, needs = %d pages, want 2", len(needs))
	}
}

func TestPageWorkspaceCascadeDelete(t *testing.T) {
	s := newTestStore(t)
	wsID := seedPageWorkspace(t, s, "work-cascade")

	if _, err := s.CreatePage(wsID, "page-a", "d", "c", "", "", "", nil); err != nil {
		t.Fatalf("create page: %v", err)
	}
	// Deleting the workspace cascades to its pages.
	if err := s.DeleteWorkspace(wsID); err != nil {
		t.Fatalf("delete workspace: %v", err)
	}
	if rows, err := s.ListPages(PageFilter{}); err != nil || len(rows) != 0 {
		t.Fatalf("pages after workspace delete (expect 0): %+v, err=%v", rows, err)
	}
}

func TestWorkspaceDescriptionRoundTrip(t *testing.T) {
	s := newTestStore(t)
	_, err := s.AddWorkspace(Workspace{Name: "w", Description: "the work", Path: "/tmp/w"})
	if err != nil {
		t.Fatalf("add workspace: %v", err)
	}
	got, err := s.GetWorkspaceByName("w")
	if err != nil {
		t.Fatalf("get workspace: %v", err)
	}
	if got.Description != "the work" {
		t.Fatalf("description = %q, want %q", got.Description, "the work")
	}
	// CreateWorkspace path persists description too.
	created, err := s.CreateWorkspace("cw", "cw", "created description", t.TempDir())
	if err != nil {
		t.Fatalf("create workspace: %v", err)
	}
	if created.Description != "created description" {
		t.Fatalf("created description = %q", created.Description)
	}
}

func TestListPageRowsSingleQueryParity(t *testing.T) {
	s := newTestStore(t)
	wsA := seedPageWorkspace(t, s, "income")
	wsB := seedPageWorkspace(t, s, "health")

	for _, tag := range []struct{ name, desc string }{
		{"finance", "money stuff"},
		{"zeta", "last tag name"},
		{"alpha", "first tag name"},
	} {
		if _, err := s.CreateTag(tag.name, tag.desc); err != nil {
			t.Fatalf("create tag: %v", err)
		}
	}

	mk := func(ws int64, title, desc, status, body string, tags []string) string {
		p, err := s.CreatePage(ws, title, desc, "ctx", status, "/o.html", body, tags)
		if err != nil {
			t.Fatalf("create page %s: %v", title, err)
		}
		return p.Slug
	}
	budget := mk(wsA, "Budget", "monthly budget chart", PageStatusPublished, "budget numbers table", []string{"zeta", "finance", "alpha"})
	trend := mk(wsA, "Trend", "spending trend line", PageStatusDraft, "trend over months", nil)
	vitals := mk(wsB, "Vitals", "resting heart rate", PageStatusPublished, "bpm readings", []string{"alpha"})

	// Two open + one done comment on budget: FeedbackOpen must be 2.
	if _, err := s.CreateComment(budget, "", "", "general", "widen the chart"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateComment(budget, "#hdr", "the header", "element", "fix header"); err != nil {
		t.Fatal(err)
	}
	doneC, _ := s.CreateComment(budget, "", "", "general", "already handled")
	if _, err := s.UpdateCommentStatus(doneC.ID, CommentStatusDone); err != nil {
		t.Fatal(err)
	}

	rows, err := s.ListPageRows(PageFilter{}, "")
	if err != nil {
		t.Fatalf("ListPageRows: %v", err)
	}
	bySlug := map[string]PageListRow{}
	for _, r := range rows {
		bySlug[r.Slug] = r
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	b := bySlug[budget]
	if b.Workspace != "income" {
		t.Errorf("budget workspace = %q, want income", b.Workspace)
	}
	gotTags := b.TagList()
	wantTags := []string{"alpha", "finance", "zeta"} // name-sorted, matches TagsForPage
	if len(gotTags) != len(wantTags) {
		t.Errorf("budget tags = %v, want %v", gotTags, wantTags)
	} else {
		for i := range wantTags {
			if gotTags[i] != wantTags[i] {
				t.Errorf("budget tags = %v, want %v", gotTags, wantTags)
				break
			}
		}
	}
	if b.FeedbackOpen != 2 {
		t.Errorf("budget FeedbackOpen = %d, want 2 (open only)", b.FeedbackOpen)
	}
	if bySlug[trend].Tags != "" || len(bySlug[trend].TagList()) != 0 {
		t.Errorf("trend tags = %q, want empty (and TagList length 0)", bySlug[trend].Tags)
	}
	if bySlug[vitals].Workspace != "health" {
		t.Errorf("vitals workspace = %q, want health", bySlug[vitals].Workspace)
	}

	// Filters.
	st, _ := s.ListPageRows(PageFilter{Status: PageStatusPublished}, "")
	if len(st) != 2 {
		t.Errorf("status filter: got %d, want 2", len(st))
	}
	wf, _ := s.ListPageRows(PageFilter{WorkspaceSlug: "health"}, "")
	if len(wf) != 1 || wf[0].Slug != vitals {
		t.Errorf("workspace filter: got %v, want [%s]", wf, vitals)
	}
	tf, _ := s.ListPageRows(PageFilter{TagName: "zeta"}, "")
	if len(tf) != 1 || tf[0].Slug != budget {
		t.Errorf("tag filter: got %v, want [%s]", tf, budget)
	}

	// Search parity with the old SearchPages path: same slug set, plus the
	// derived fields the list needs.
	oldPages, err := s.SearchPages("budget", PageFilter{})
	if err != nil {
		t.Fatal(err)
	}
	sr, err := s.ListPageRows(PageFilter{}, "budget")
	if err != nil {
		t.Fatal(err)
	}
	if len(sr) != len(oldPages) {
		t.Fatalf("search: ListPageRows got %d, SearchPages got %d", len(sr), len(oldPages))
	}
	if sr[0].Slug != oldPages[0].Slug || sr[0].Workspace != "income" || sr[0].FeedbackOpen != 2 {
		t.Errorf("search row = %+v", sr[0])
	}

	// Tag-description search path (LIKE fallback): zeta's description "last
	// tag name" matches "last" even though no page body contains it.
	tr, err := s.ListPageRows(PageFilter{}, "last")
	if err != nil {
		t.Fatal(err)
	}
	if len(tr) != 1 || tr[0].Slug != budget {
		t.Errorf("tag-desc search: got %v, want [%s]", tr, budget)
	}
}
