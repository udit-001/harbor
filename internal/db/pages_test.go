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
