package db

import (
	"fmt"
	"strings"
)

// Comments domain. Comments are anchored human feedback on a page — the human
// side of the feedback loop. They live in the DB and never mutate the page
// file. The store sits on the Store seam directly (like pages); page scoping is
// resolved by slug and carried as page_id. The list surface returns
// CommentView (comment + page slug) so the CLI and the derived open-feedback
// state never resolve a raw page_id.

const commentColumns = `c.id, c.page_id, c.anchor, c.quote, c.type, c.body, c.status, c.created_at, COALESCE(c.resolved_at, '')`

const commentViewColumns = commentColumns + `, p.slug`

func scanComment(row interface{ Scan(...any) error }) (Comment, error) {
	var c Comment
	var resolved string
	err := row.Scan(
		&c.ID, &c.PageID, &c.Anchor, &c.Quote, &c.Type, &c.Body, &c.Status,
		&c.CreatedAt, &resolved,
	)
	if resolved != "" {
		c.ResolvedAt = &resolved
	}
	return c, err
}

func scanCommentView(row interface{ Scan(...any) error }) (CommentView, error) {
	var v CommentView
	var resolved string
	err := row.Scan(
		&v.ID, &v.PageID, &v.Anchor, &v.Quote, &v.Type, &v.Body, &v.Status,
		&v.CreatedAt, &resolved, &v.PageSlug,
	)
	if resolved != "" {
		v.ResolvedAt = &resolved
	}
	return v, err
}

func scanCommentViews(rows RowScanner) ([]CommentView, error) {
	return scanRows(rows, "comment", scanCommentView)
}

const changeColumns = `id, page_id, COALESCE(comment_id, 0), change_id, title, description, created_at`

func scanChange(row interface{ Scan(...any) error }) (Change, error) {
	var ch Change
	var commentID int64
	err := row.Scan(&ch.ID, &ch.PageID, &commentID, &ch.ChangeID, &ch.Title, &ch.Description, &ch.CreatedAt)
	if commentID != 0 {
		ch.CommentID = &commentID
	}
	return ch, err
}

func scanChanges(rows RowScanner) ([]Change, error) {
	return scanRows(rows, "change", scanChange)
}

// normalizeCommentStatus coerces an optional status to a valid value: empty
// becomes open; anything else must be a real status or it errors.
func normalizeCommentStatus(status string) (string, error) {
	switch status {
	case "":
		return CommentStatusOpen, nil
	case CommentStatusOpen, CommentStatusInProgress, CommentStatusDone:
		return status, nil
	default:
		return "", fmt.Errorf("comment status must be one of: open, in-progress, done (got %q)", status)
	}
}

// normalizeCommentType validates a comment type, defaulting empty to general.
func normalizeCommentType(typ string) (string, error) {
	switch typ {
	case "":
		return CommentTypeGeneral, nil
	case CommentTypeSelection, CommentTypeElement, CommentTypeGeneral:
		return typ, nil
	default:
		return "", fmt.Errorf("comment type must be one of: selection, element, general (got %q)", typ)
	}
}

// CommentFilter narrows ListComments. All fields are optional: zero values mean
// "no filter". PageSlug filters to comments on that page by slug.
type CommentFilter struct {
	PageSlug string // "" = all pages
	Status   string // "" = all statuses
}

// CreateComment creates a comment on a page by slug. anchor/quote/type describe
// the selection the human anchored; body is the actual feedback. Returns the
// created comment's display view.
func (s *Store) CreateComment(pageSlug, anchor, quote, typ, body string) (CommentView, error) {
	page, err := s.PageBySlug(pageSlug)
	if err != nil {
		return CommentView{}, fmt.Errorf("comment: %w", err)
	}
	typ, err = normalizeCommentType(typ)
	if err != nil {
		return CommentView{}, err
	}
	if strings.TrimSpace(body) == "" {
		return CommentView{}, fmt.Errorf("comment body must not be empty")
	}

	res, err := s.db.Exec(
		`INSERT INTO comments (page_id, anchor, quote, type, body, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		page.ID, anchor, quote, typ, body, CommentStatusOpen, nowTimestamp(),
	)
	if err != nil {
		return CommentView{}, fmt.Errorf("insert comment: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return CommentView{}, err
	}
	return s.CommentByID(id)
}

// CommentByID returns a single comment view by id.
func (s *Store) CommentByID(id int64) (CommentView, error) {
	row := s.db.QueryRow(
		"SELECT "+commentViewColumns+" FROM comments c JOIN pages p ON p.id = c.page_id WHERE c.id = ?",
		id,
	)
	v, err := scanCommentView(row)
	if err != nil {
		return CommentView{}, fmt.Errorf("comment %d not found: %w", id, err)
	}
	return v, nil
}

// ListComments lists comment views, optionally filtered by page slug and
// status. Results are ordered oldest-first (stable thread order).
func (s *Store) ListComments(filter CommentFilter) ([]CommentView, error) {
	q := "SELECT " + commentViewColumns + " FROM comments c JOIN pages p ON p.id = c.page_id"
	var args []any
	var where []string

	if filter.PageSlug != "" {
		where = append(where, "p.slug = ?")
		args = append(args, filter.PageSlug)
	}
	if filter.Status != "" {
		where = append(where, "c.status = ?")
		args = append(args, filter.Status)
	}

	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY c.created_at ASC, c.id ASC"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanCommentViews(rows)
}

// OpenCommentCounts returns page slug → number of open comments, the input to
// the derived open-feedback state (Library rows + page chrome). Derived at read
// time from the comments queue — no persisted "has feedback" flag exists.
func (s *Store) OpenCommentCounts() (map[string]int, error) {
	rows, err := s.db.Query(
		`SELECT p.slug, COUNT(c.id) FROM pages p
		 JOIN comments c ON c.page_id = p.id
		 WHERE c.status = ? GROUP BY p.id`,
		CommentStatusOpen,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var slug string
		var n int
		if err := rows.Scan(&slug, &n); err != nil {
			return nil, err
		}
		out[slug] = n
	}
	return out, rows.Err()
}

// UpdateCommentStatus transitions a comment's lifecycle state
// (open → in-progress → done). Setting done records resolved_at. Returns the
// updated view.
func (s *Store) UpdateCommentStatus(id int64, status string) (CommentView, error) {
	normalized, err := normalizeCommentStatus(status)
	if err != nil {
		return CommentView{}, err
	}
	current, err := s.CommentByID(id)
	if err != nil {
		return CommentView{}, err
	}

	var resolved any
	if normalized == CommentStatusDone {
		resolved = nowTimestamp()
	} else {
		resolved = current.ResolvedAt
	}

	_, err = s.db.Exec(
		`UPDATE comments SET status = ?, resolved_at = ? WHERE id = ?`,
		normalized, resolved, id,
	)
	if err != nil {
		return CommentView{}, fmt.Errorf("update comment %d: %w", id, err)
	}
	return s.CommentByID(id)
}

// CloseComment resolves a comment (status → done).
func (s *Store) CloseComment(id int64) (CommentView, error) {
	return s.UpdateCommentStatus(id, CommentStatusDone)
}

// CreateChange records an agent edit tied to a page and, optionally, a comment.
// ChangeID matches a data-cf-change marker embedded in the page HTML.
func (s *Store) CreateChange(pageSlug, changeID string, commentID int64, title, description string) (Change, error) {
	page, err := s.PageBySlug(pageSlug)
	if err != nil {
		return Change{}, fmt.Errorf("change: %w", err)
	}
	if strings.TrimSpace(changeID) == "" {
		return Change{}, fmt.Errorf("change_id must not be empty")
	}

	var commentRef any
	if commentID != 0 {
		c, err := s.CommentByID(commentID)
		if err != nil {
			return Change{}, err
		}
		if c.PageID != page.ID {
			return Change{}, fmt.Errorf("comment %d does not belong to page %q", commentID, pageSlug)
		}
		commentRef = commentID
	}

	res, err := s.db.Exec(
		`INSERT INTO changes (page_id, comment_id, change_id, title, description, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		page.ID, commentRef, changeID, title, description, nowTimestamp(),
	)
	if err != nil {
		return Change{}, fmt.Errorf("insert change: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Change{}, err
	}
	return s.ChangeByID(id)
}

// ChangeByID returns a single change by id.
func (s *Store) ChangeByID(id int64) (Change, error) {
	row := s.db.QueryRow("SELECT "+changeColumns+" FROM changes WHERE id = ?", id)
	ch, err := scanChange(row)
	if err != nil {
		return Change{}, fmt.Errorf("change %d not found: %w", id, err)
	}
	return ch, nil
}

// ListChanges lists change records for a page by slug (all), or for a single
// comment when commentID is non-zero.
func (s *Store) ListChanges(pageSlug string, commentID int64) ([]Change, error) {
	q := "SELECT " + changeColumns + " FROM changes WHERE page_id IN (SELECT id FROM pages WHERE slug = ?)"
	var args []any = []any{pageSlug}
	if commentID != 0 {
		q += " AND comment_id = ?"
		args = append(args, commentID)
	}
	q += " ORDER BY created_at ASC, id ASC"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanChanges(rows)
}
