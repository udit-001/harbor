package db

import (
	"fmt"
	"github.com/udit-001/harbor/internal/extract"
	"sort"
	"strings"
)

// Pages domain. Pages are the atomic artifact in harbor — workspace-scoped
// standalone HTML pages. The page store lives on the
// Store seam directly; workspace-scoped helpers are not required because a
// page already carries workspace_id.

const pageColumns = `id, slug, workspace_id, title, description, context, status, COALESCE(format, 'html'), origin_path, COALESCE(body_text, ''), created_at, updated_at`

func scanPage(row interface{ Scan(...any) error }) (Page, error) {
	var p Page
	err := row.Scan(
		&p.ID, &p.Slug, &p.WorkspaceID, &p.Title, &p.Description, &p.Context,
		&p.Status, &p.Format, &p.OriginPath, &p.BodyText, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func scanPages(rows RowScanner) ([]Page, error) {
	return scanRows(rows, "page", scanPage)
}

// normalizePageStatus coerces an optional status to a valid value: empty
// becomes draft; anything else must be a real status or it errors. Mirrors the
// exactly N states" invariant.
func normalizePageStatus(status string) (string, error) {
	switch status {
	case "":
		return PageStatusDraft, nil
	case PageStatusDraft, PageStatusPublished, PageStatusArchived:
		return status, nil
	default:
		return "", fmt.Errorf("page status must be one of: draft, published, archived (got %q)", status)
	}
}

// normalizePageFormat coerces an optional format to a valid value: empty
// becomes html (the original page kind); anything else must be a known format
// or it errors — same closed-enum discipline as status.
func normalizePageFormat(format string) (string, error) {
	switch format {
	case "":
		return extract.ArtifactHTML, nil
	default:
		if !extract.ValidArtifactFormat(format) {
			return "", fmt.Errorf("page format must be one of: html, markdown, pdf, text, svg, image, excalidraw (got %q)", format)
		}
		return format, nil
	}
}

// PageFilter narrows ListPages. All fields are optional: zero values mean
// "no filter". TagName filters to pages carrying that tag by name; WorkspaceSlug
// filters to pages in that workspace by name. Query/FTS filtering lands with
// the search slice (HARB-5).
type PageFilter struct {
	Status        string // "" = all
	WorkspaceSlug string // "" = all
	TagName       string // "" = all
}

// CreatePage creates a new page. The workspace and status are validated; title
// is required and drives the stable slug (derived once via Slugify). The listed
// tag names are attached; each must already exist as a Tag (create-first rule).
// Returns the created page by slug.
func (s *Store) CreatePage(workspaceID int64, title, description, context, status, format, originPath, bodyText string, tagNames []string) (Page, error) {
	slug := Slugify(title)
	if slug == "" {
		return Page{}, fmt.Errorf("page title must produce a slug")
	}
	status, err := normalizePageStatus(status)
	if err != nil {
		return Page{}, err
	}
	format, ferr := normalizePageFormat(format)
	if ferr != nil {
		return Page{}, ferr
	}

	res, err := s.db.Exec(
		`INSERT INTO pages (slug, workspace_id, title, description, context, status, format, origin_path, body_text)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		slug, workspaceID, title, description, context, status, format, originPath, bodyText,
	)
	if err != nil {
		return Page{}, fmt.Errorf("insert page: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Page{}, err
	}

	if len(tagNames) > 0 {
		if err := s.setPageTags(id, tagNames); err != nil {
			return Page{}, err
		}
	}

	return s.PageBySlug(slug)
}

// PageBySlug returns a single page by its stable slug.
func (s *Store) PageBySlug(slug string) (Page, error) {
	row := s.db.QueryRow("SELECT "+pageColumns+" FROM pages WHERE slug = ?", slug)
	p, err := scanPage(row)
	if err != nil {
		return Page{}, fmt.Errorf("page %q not found: %w", slug, err)
	}
	return p, nil
}

// ListPages lists pages, optionally filtered by status, workspace, and tag.
// Results are ordered newest-updated first.
func (s *Store) ListPages(filter PageFilter) ([]Page, error) {
	q := "SELECT " + pageColumns + " FROM pages"
	var args []any
	var where []string

	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.WorkspaceSlug != "" {
		where = append(where, "workspace_id IN (SELECT id FROM workspaces WHERE name = ?)")
		args = append(args, filter.WorkspaceSlug)
	}
	if filter.TagName != "" {
		where = append(where, "id IN (SELECT pt.page_id FROM page_tags pt JOIN tags t ON t.id = pt.tag_id WHERE t.name = ?)")
		args = append(args, filter.TagName)
	}

	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY updated_at DESC"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPages(rows)
}

// ListPageRows returns the library list in ONE query: display columns only
// (no body_text), workspace name via LEFT JOIN, tag names via a correlated
// GROUP_CONCAT, and the derived open-feedback count via a correlated COUNT.
// This replaces the old per-page GetWorkspace + TagsForPage follow-ups
// (N+1) that made every library render and page-view navigation cost ~2N
// queries — visibly slow on Windows once the library grows past a few
// hundred pages.
//
// q (when set) searches via the page FTS index plus tag name/description
// LIKE, mirroring SearchPages. Results are ordered newest-updated first,
// like ListPages.
func (s *Store) ListPageRows(filter PageFilter, q string) ([]PageListRow, error) {
	where := []string{"1=1"}
	args := []any{CommentStatusOpen}

	if q != "" {
		fts := buildFTSQuery(q)
		if fts == "" {
			return []PageListRow{}, nil
		}
		tagLike, likeArgs := ftsTermsLike(q)
		where = append(where,
			"(p.id IN (SELECT rowid FROM pages_fts WHERE pages_fts MATCH ?)"+
				" OR p.id IN (SELECT DISTINCT pt.page_id FROM page_tags pt JOIN tags t ON t.id = pt.tag_id WHERE "+tagLike+"))")
		args = append(args, fts)
		args = append(args, likeArgs...)
	}
	if filter.Status != "" {
		where = append(where, "p.status = ?")
		args = append(args, filter.Status)
	}
	if filter.WorkspaceSlug != "" {
		where = append(where, "p.workspace_id IN (SELECT id FROM workspaces WHERE name = ?)")
		args = append(args, filter.WorkspaceSlug)
	}
	if filter.TagName != "" {
		where = append(where, "p.id IN (SELECT pt.page_id FROM page_tags pt JOIN tags t ON t.id = pt.tag_id WHERE t.name = ?)")
		args = append(args, filter.TagName)
	}

	query := `SELECT p.slug, p.title, p.description, p.status, COALESCE(p.format, 'html'), p.updated_at,
		COALESCE(w.name, ''),
		COALESCE((SELECT GROUP_CONCAT(t.name, ',') FROM page_tags pt JOIN tags t ON t.id = pt.tag_id WHERE pt.page_id = p.id), ''),
		(SELECT COUNT(*) FROM comments c WHERE c.page_id = p.id AND c.status = ?)
	FROM pages p
	LEFT JOIN workspaces w ON w.id = p.workspace_id
	WHERE ` + strings.Join(where, " AND ") + `
	ORDER BY p.updated_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []PageListRow{}
	for rows.Next() {
		var r PageListRow
		if err := rows.Scan(&r.Slug, &r.Title, &r.Description, &r.Status, &r.Format, &r.UpdatedAt,
			&r.Workspace, &r.Tags, &r.FeedbackOpen); err != nil {
			return nil, fmt.Errorf("scan page list row: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// PageListRow is one row of the library list: a page's display fields plus
// the derived data the list shows (workspace name, tag names, open feedback
// count) — and crucially NOT body_text, which can be hundreds of KB per page.
// Fetching these as one query is what keeps library navigation O(1) queries
// instead of O(pages).
type PageListRow struct {
	Slug         string
	Title        string
	Description  string
	Status       string
	Format       string // artifact format: html, markdown, pdf, text, svg, image, excalidraw
	UpdatedAt    string
	Workspace    string // workspace name; "" only for a dangling workspace_id
	Tags         string // comma-joined tag names
	FeedbackOpen int    // derived: open comments on the page
}

// TagList splits the joined tag names into a name-sorted slice (same order
// TagsForPage returns), never nil.
func (r PageListRow) TagList() []string {
	if r.Tags == "" {
		return []string{}
	}
	tags := strings.Split(r.Tags, ",")
	sort.Strings(tags)
	return tags
}

// UpdatePage mutates a page in place. Pointers are optional: nil means
// "unchanged"; status (when non-nil) must be a valid status; tags (when non-nil)
// replaces the full tag set. The slug never changes (stable find-then-update
// handle); a changed title does not regenerate it.
func (s *Store) UpdatePage(slug string, title, description, context, status, format, originPath, bodyText *string, tags *[]string) (Page, error) {
	current, err := s.PageBySlug(slug)
	if err != nil {
		return Page{}, err
	}

	newTitle := current.Title
	if title != nil {
		newTitle = *title
	}
	newDesc := current.Description
	if description != nil {
		newDesc = *description
	}
	newCtx := current.Context
	if context != nil {
		newCtx = *context
	}
	newStatus := current.Status
	if status != nil {
		newStatus, err = normalizePageStatus(*status)
		if err != nil {
			return Page{}, err
		}
	}
	newFormat := current.Format
	if format != nil {
		newFormat, err = normalizePageFormat(*format)
		if err != nil {
			return Page{}, err
		}
	}
	newOrigin := current.OriginPath
	if originPath != nil {
		newOrigin = *originPath
	}
	newBody := current.BodyText
	if bodyText != nil {
		newBody = *bodyText
	}

	_, err = s.db.Exec(
		`UPDATE pages SET title = ?, description = ?, context = ?, status = ?, format = ?, origin_path = ?, body_text = ?, updated_at = ?
		 WHERE id = ?`,
		newTitle, newDesc, newCtx, newStatus, newFormat, newOrigin, newBody, nowTimestamp(), current.ID,
	)
	if err != nil {
		return Page{}, fmt.Errorf("update page %q: %w", slug, err)
	}

	if tags != nil {
		if err := s.setPageTags(current.ID, *tags); err != nil {
			return Page{}, err
		}
	}

	return s.PageBySlug(slug)
}

// DeletePage permanently removes a page (join rows + FTS cascade).
func (s *Store) DeletePage(slug string) error {
	res, err := s.db.Exec("DELETE FROM pages WHERE slug = ?", slug)
	if err != nil {
		return fmt.Errorf("delete page %q: %w", slug, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("page %q not found", slug)
	}
	return nil
}

// PageCountForWorkspace returns how many pages belong to a workspace by name.
// Used for workspace stats; an unknown workspace yields zero with no error.
func (s *Store) PageCountForWorkspace(workspaceSlug string) (int, error) {
	var count int
	err := s.db.Get(&count,
		"SELECT COUNT(*) FROM pages WHERE workspace_id IN (SELECT id FROM workspaces WHERE name = ?)",
		workspaceSlug,
	)
	return count, err
}

// PagesNeedingReindex returns pages whose body_text is empty — the ones the
// FTS index is missing body coverage for. Indexing is idempotent: rebuild
// harvests only these, so already-indexed pages are left untouched.
func (s *Store) PagesNeedingReindex() ([]Page, error) {
	rows, err := s.db.Query("SELECT " + pageColumns + " FROM pages WHERE body_text = ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPages(rows)
}

// ClearPageBodies empties body_text for every page, forcing a full re-index on
// the next rebuild. The scoped FTS trigger updates the index accordingly.
func (s *Store) ClearPageBodies() error {
	_, err := s.db.Exec("UPDATE pages SET body_text = ''")
	return err
}

// PagesCountByWorkspace returns workspace name → page count, for sidebar counts.
func (s *Store) PagesCountByWorkspace() (map[string]int, error) {
	rows, err := s.db.Query(
		"SELECT w.name, COUNT(p.id) FROM workspaces w LEFT JOIN pages p ON p.workspace_id = w.id GROUP BY w.id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			return nil, err
		}
		out[name] = n
	}
	return out, rows.Err()
}

// PagesCountByTag returns tag name → page count, for sidebar counts.
func (s *Store) PagesCountByTag() (map[string]int, error) {
	rows, err := s.db.Query(
		"SELECT t.name, COUNT(pt.page_id) FROM tags t LEFT JOIN page_tags pt ON pt.tag_id = t.id GROUP BY t.id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			return nil, err
		}
		out[name] = n
	}
	return out, rows.Err()
}

// setPageTags replaces the tag association set for a page. Each name must
// already exist as a Tag — the agent creates tags deliberately with a
// description (create-first rule). Detaching a tag does NOT delete the Tag; it
// only severs the association.
func (s *Store) setPageTags(pageID int64, tagNames []string) error {
	if _, err := s.db.Exec("DELETE FROM page_tags WHERE page_id = ?", pageID); err != nil {
		return fmt.Errorf("clear page tags: %w", err)
	}
	for _, name := range tagNames {
		if strings.TrimSpace(name) == "" {
			continue
		}
		tag, err := s.TagByName(name)
		if err != nil {
			return fmt.Errorf("tag %q does not exist — create it first with 'harbor tag create %q --description ...'",
				name, name)
		}
		if _, err := s.db.Exec(
			"INSERT OR IGNORE INTO page_tags (page_id, tag_id) VALUES (?, ?)",
			pageID, tag.ID,
		); err != nil {
			return fmt.Errorf("attach tag %q: %w", name, err)
		}
	}
	return nil
}

// TagsForPage returns the tags attached to a page by slug, ordered by name.
func (s *Store) TagsForPage(slug string) ([]Tag, error) {
	rows, err := s.db.Query(
		"SELECT t.id, t.name, t.description, t.created_at, t.updated_at FROM tags t "+
			"JOIN page_tags pt ON pt.tag_id = t.id "+
			"JOIN pages p ON p.id = pt.page_id "+
			"WHERE p.slug = ? ORDER BY t.name ASC",
		slug,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTags(rows)
}

// SearchPages performs full-text search across page title, description,
// context, body text, plus the descriptions of attached tags, narrowed by the
// same filters as ListPages. A page surfaces if it matches its own text OR any
// attached tag's name/description.
func (s *Store) SearchPages(query string, filter PageFilter) ([]Page, error) {
	q := buildFTSQuery(query)
	if q == "" {
		return []Page{}, nil
	}

	titleBody := "SELECT rowid FROM pages_fts WHERE pages_fts MATCH ?"

	tagLike, likeArgs := ftsTermsLike(query)
	tagIdsSQL := fmt.Sprintf(
		"SELECT DISTINCT pt.page_id FROM page_tags pt JOIN tags t ON t.id = pt.tag_id WHERE %s",
		tagLike,
	)

	args := []any{q}
	args = append(args, likeArgs...)

	where := []string{"(id IN (%s) OR id IN (%s))"}

	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, filter.Status)
	}
	if filter.WorkspaceSlug != "" {
		where = append(where, "workspace_id IN (SELECT id FROM workspaces WHERE name = ?)")
		args = append(args, filter.WorkspaceSlug)
	}
	if filter.TagName != "" {
		where = append(where, "id IN (SELECT pt.page_id FROM page_tags pt JOIN tags t ON t.id = pt.tag_id WHERE t.name = ?)")
		args = append(args, filter.TagName)
	}

	combined := fmt.Sprintf(
		"SELECT %s FROM pages WHERE %s ORDER BY updated_at DESC",
		pageColumns,
		fmt.Sprintf(strings.Join(where, " AND "), titleBody, tagIdsSQL),
	)
	rows, err := s.db.Query(combined, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPages(rows)
}
