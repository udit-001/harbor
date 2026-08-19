package db

import "testing"

// seedPageForComments inserts a workspace + page for comment tests. Returns the
// page slug.
func seedPageForComments(t *testing.T, s *Store, wsName, slug string) string {
	t.Helper()
	wsID := seedPageWorkspace(t, s, wsName)
	page, err := s.CreatePage(wsID, slug, "desc", "ctx", "", "", "", "body", nil)
	if err != nil {
		t.Fatalf("seed page %s: %v", slug, err)
	}
	return page.Slug
}

func TestCommentCreateListUpdateClose(t *testing.T) {
	s := newTestStore(t)
	slug := seedPageForComments(t, s, "ws", "monthly-totals")

	// Create three comments: a selection, an element, and a general one.
	created, err := s.CreateComment(slug, "#totals-chart", "the chart is clipped",
		CommentTypeSelection, "please widen the chart")
	if err != nil {
		t.Fatalf("create selection comment: %v", err)
	}
	if created.PageSlug != slug {
		t.Fatalf("view page = %q, want %q", created.PageSlug, slug)
	}
	if created.Type != CommentTypeSelection || created.Status != CommentStatusOpen {
		t.Fatalf("created = %+v, want type=selection status=open", created)
	}
	if created.ResolvedAt != nil {
		t.Fatalf("open comment should have no resolved_at: %+v", created)
	}
	selID := created.ID

	el, err := s.CreateComment(slug, "section.summary", "", CommentTypeElement, "bold the key metric")
	if err != nil {
		t.Fatalf("create element comment: %v", err)
	}
	gen, err := s.CreateComment(slug, "", "", CommentTypeGeneral, "overall this reads well")
	if err != nil {
		t.Fatalf("create general comment: %v", err)
	}
	if gen.PageSlug != slug || gen.Status != CommentStatusOpen {
		t.Fatalf("general comment view = %+v", gen)
	}

	// Default empty type normalizes to general; empty body is rejected.
	if _, err := s.CreateComment(slug, "", "", "", ""); err == nil {
		t.Fatalf("expected error for empty body")
	}
	defaulted, err := s.CreateComment(slug, "", "", "", "autotype")
	if err != nil {
		t.Fatalf("create comment with empty type: %v", err)
	}
	if defaulted.Type != CommentTypeGeneral {
		t.Fatalf("empty type defaulted to %q, want general", defaulted.Type)
	}
	// Invalid type/status rejected.
	if _, err := s.CreateComment(slug, "", "", "bogus", "x"); err == nil {
		t.Fatalf("expected error for invalid comment type")
	}
	if _, err := s.UpdateCommentStatus(el.ID, "bogus"); err == nil {
		t.Fatalf("expected error for invalid status transition target")
	}

	// List open comments (all four create-first), newest first; filter by page.
	open, err := s.ListComments(CommentFilter{Status: CommentStatusOpen})
	if err != nil {
		t.Fatalf("list open: %v", err)
	}
	if len(open) != 4 {
		t.Fatalf("open comments = %d, want 4", len(open))
	}
	if open[0].ID != defaulted.ID {
		t.Fatalf("first open comment = #%d, want #%d (newest first)", open[0].ID, defaulted.ID)
	}
	for i := 1; i < len(open); i++ {
		if open[i-1].ID < open[i].ID {
			t.Fatalf("open comments not newest-first: #%d before #%d", open[i-1].ID, open[i].ID)
		}
	}

	byPage, err := s.ListComments(CommentFilter{PageSlug: slug, Status: CommentStatusOpen})
	if err != nil || len(byPage) != 4 {
		t.Fatalf("list by page (expect 4): %d, err=%v", len(byPage), err)
	}
	noPage, err := s.ListComments(CommentFilter{PageSlug: "nope", Status: CommentStatusOpen})
	if err != nil || len(noPage) != 0 {
		t.Fatalf("list by missing page (expect 0): %d, err=%v", len(noPage), err)
	}

	// Status transition open → in-progress → done; done records resolved_at.
	inProg, err := s.UpdateCommentStatus(selID, CommentStatusInProgress)
	if err != nil {
		t.Fatalf("mark in-progress: %v", err)
	}
	if inProg.Status != CommentStatusInProgress || inProg.ResolvedAt != nil {
		t.Fatalf("in-progress = %+v", inProg)
	}
	if open, _ := s.ListComments(CommentFilter{Status: CommentStatusOpen}); len(open) != 3 {
		t.Fatalf("open after in-progress = %d, want 3", len(open))
	}

	done, err := s.CloseComment(selID)
	if err != nil {
		t.Fatalf("close comment: %v", err)
	}
	if done.Status != CommentStatusDone || done.ResolvedAt == nil {
		t.Fatalf("done comment missing resolved_at: %+v", done)
	}

	// Filtering by page + a later status.
	doneOnly, err := s.ListComments(CommentFilter{PageSlug: slug, Status: CommentStatusDone})
	if err != nil || len(doneOnly) != 1 || doneOnly[0].ID != selID {
		t.Fatalf("list done (expect 1): %+v, err=%v", doneOnly, err)
	}
}

func TestOpenCommentCountsDerived(t *testing.T) {
	s := newTestStore(t)
	a := seedPageForComments(t, s, "ws", "page-a")
	b := seedPageForComments(t, s, "ws2", "page-b")
	c := seedPageForComments(t, s, "ws3", "page-c") // no comments

	if _, err := s.CreateComment(a, "", "", CommentTypeGeneral, "one"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.CreateComment(a, ".hero", "", CommentTypeElement, "two"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.CreateComment(b, "", "", CommentTypeGeneral, "one on b"); err != nil {
		t.Fatalf("create: %v", err)
	}

	counts, err := s.OpenCommentCounts()
	if err != nil {
		t.Fatalf("open counts: %v", err)
	}
	if counts[a] != 2 || counts[b] != 1 {
		t.Fatalf("counts = %+v, want page-a=2 page-b=1", counts)
	}
	if _, ok := counts[c]; ok {
		t.Fatalf("page-c with no open comments should be absent, got %+v", counts)
	}

	// Closing a comment drops it from the derived count (read-time derivation).
	if _, err := s.CloseComment(2); err != nil {
		t.Fatalf("close: %v", err)
	}
	counts, _ = s.OpenCommentCounts()
	if counts[a] != 1 {
		t.Fatalf("after close, page-a = %d, want 1", counts[a])
	}
}

func TestCommentChangeRecord(t *testing.T) {
	s := newTestStore(t)
	slug := seedPageForComments(t, s, "ws", "landing")

	c, err := s.CreateComment(slug, ".hero", "", CommentTypeElement, "tighten the hero spacing")
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}

	// Record a change addressing that comment.
	ch, err := s.CreateChange(slug, "cf-7", c.ID, "Tighten hero spacing", "reduced the gap between the headline and the call-to-action")
	if err != nil {
		t.Fatalf("create change: %v", err)
	}
	if ch.ChangeID != "cf-7" || ch.CommentID == nil || *ch.CommentID != c.ID {
		t.Fatalf("change = %+v, want change_id=cf-7 comment=%d", ch, c.ID)
	}
	if ch.Title != "Tighten hero spacing" || ch.Description != "reduced the gap between the headline and the call-to-action" {
		t.Fatalf("change title/description = %q / %q", ch.Title, ch.Description)
	}

	byComment, err := s.ListChanges(slug, c.ID)
	if err != nil || len(byComment) != 1 {
		t.Fatalf("changes for comment (expect 1): %+v, err=%v", byComment, err)
	}
	all, err := s.ListChanges(slug, 0)
	if err != nil || len(all) != 1 {
		t.Fatalf("all page changes (expect 1): %+v, err=%v", all, err)
	}

	// A change may carry no comment (general tidying).
	if _, err := s.CreateChange(slug, "cf-8", 0, "", ""); err != nil {
		t.Fatalf("create standalone change: %v", err)
	}
	if all, _ := s.ListChanges(slug, 0); len(all) != 2 {
		t.Fatalf("all page changes after standalone (expect 2): %d", len(all))
	}

	// A comment belonging to a different page can't be referenced.
	other := seedPageForComments(t, s, "ws2", "other-page")
	oc, err := s.CreateComment(other, "", "", CommentTypeGeneral, "on other")
	if err != nil {
		t.Fatalf("create comment on other: %v", err)
	}
	if _, err := s.CreateChange(slug, "cf-9", oc.ID, "", ""); err == nil {
		t.Fatalf("expected error referencing comment from another page")
	}
}

func TestCommentCascadeOnPageDelete(t *testing.T) {
	s := newTestStore(t)
	slug := seedPageForComments(t, s, "ws", "doomed")
	if _, err := s.CreateComment(slug, "", "", CommentTypeGeneral, "feedback"); err != nil {
		t.Fatalf("create comment: %v", err)
	}

	// Deleting the page cascades to its comments.
	if err := s.DeletePage(slug); err != nil {
		t.Fatalf("delete page: %v", err)
	}
	if open, _ := s.ListComments(CommentFilter{Status: CommentStatusOpen}); len(open) != 0 {
		t.Fatalf("comments after page delete (expect 0): %d", len(open))
	}
}

func TestCreateCommentAnchorsMultiAndReply(t *testing.T) {
	s := newTestStore(t)
	slug := seedPageForComments(t, s, "ws", "multi-anchor")

	// A multi-anchor comment: two text selections flagged together.
	anchors := []Anchor{
		{Kind: AnchorKindText, Path: "#row-a", Quote: "growth is off"},
		{Kind: AnchorKindText, Path: "#row-b", Quote: "same issue here"},
	}
	created, err := s.CreateCommentAnchors(slug, "flag both rows", anchors, 0)
	if err != nil {
		t.Fatalf("create multi-anchor comment: %v", err)
	}
	if len(created.Anchors) != 2 {
		t.Fatalf("anchors round-trip = %d, want 2", len(created.Anchors))
	}
	if created.Anchors[0].Path != "#row-a" || created.Anchors[1].Quote != "same issue here" {
		t.Fatalf("anchor contents wrong: %+v", created.Anchors)
	}
	// Legacy display columns derive from the first anchor.
	if created.Type != CommentTypeSelection || created.Anchor != "#row-a" || created.Quote != "growth is off" {
		t.Fatalf("legacy columns not derived: type=%q anchor=%q quote=%q", created.Type, created.Anchor, created.Quote)
	}

	// Reply: a new open comment linked back to the parent, same page.
	reply, err := s.CreateCommentAnchors(slug, "agree, also row-c", []Anchor{{Kind: AnchorKindElement, Path: "#row-c"}}, created.ID)
	if err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if reply.ReplyTo == nil || *reply.ReplyTo != created.ID {
		t.Fatalf("reply.ReplyTo = %v, want %d", reply.ReplyTo, created.ID)
	}

	// Reply to a comment on another page is rejected.
	other := seedPageForComments(t, s, "ws2", "other-page")
	if _, err := s.CreateCommentAnchors(other, "cross-page reply", anchors, created.ID); err == nil {
		t.Fatalf("expected error replying to a comment on another page")
	}

	// Empty anchor list defaults to a whole-document anchor.
	def, err := s.CreateCommentAnchors(slug, "overall", nil, 0)
	if err != nil {
		t.Fatalf("create with no anchors: %v", err)
	}
	if len(def.Anchors) != 1 || def.Anchors[0].Kind != AnchorKindDocument || def.Type != CommentTypeGeneral {
		t.Fatalf("default anchor wrong: %+v", def)
	}
}

func TestUpdateCommentEditOpenOnly(t *testing.T) {
	s := newTestStore(t)
	slug := seedPageForComments(t, s, "ws", "editable")

	c, err := s.CreateCommentAnchors(slug, "original", []Anchor{{Kind: AnchorKindElement, Path: ".hero"}}, 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if c.UpdatedAt == "" {
		t.Fatalf("updated_at should be set on create")
	}

	// Editing an open comment updates body + anchors.
	edited, err := s.UpdateComment(c.ID, "revised wording", []Anchor{{Kind: AnchorKindText, Path: "#kpi", Quote: "revised"}})
	if err != nil {
		t.Fatalf("edit open comment: %v", err)
	}
	if edited.Body != "revised wording" || len(edited.Anchors) != 1 ||
		edited.Anchors[0].Kind != AnchorKindText || edited.Quote != "revised" {
		t.Fatalf("edit not applied: %+v", edited)
	}

	// Empty body is rejected.
	if _, err := s.UpdateComment(c.ID, " ", nil); err == nil {
		t.Fatalf("expected error for empty body on edit")
	}

	// A done comment can never be revised.
	if _, err := s.UpdateCommentStatus(c.ID, CommentStatusDone); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := s.UpdateComment(c.ID, "late revision", nil); err == nil {
		t.Fatalf("expected error editing a done comment")
	}
}
