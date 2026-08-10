package db

import (
	"fmt"
	"strings"
)

// Pages domain. Pages are the atomic artifact in harbor — workspace-scoped
// (unlike scraps, which are global and sealed). The page store lives on the
// Store seam directly; workspace-scoped helpers are not required because a
// page already carries workspace_id.

const pageColumns = `id, slug, workspace_id, title, description, context, status, origin_path, COALESCE(body_text, ''), created_at, updated_at`

func scanPage(row interface{ Scan(...any) error }) (Page, error) {
	var p Page
	err := row.Scan(
		&p.ID, &p.Slug, &p.WorkspaceID, &p.Title, &p.Description, &p.Context,
		&p.Status, &p.OriginPath, &p.BodyText, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func scanPages(rows RowScanner) ([]Page, error) {
	return scanRows(rows, "page", scanPage)
}

// normalizePageStatus coerces an optional status to a valid value: empty
// becomes draft; anything else must be a real status or it errors. Mirrors the
// "exactly N states" invariant of scraps.
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
// tag names are attached; each must already exist as a Tag (create-first rule,
// mirroring scraps). Returns the created page by slug.
func (s *Store) CreatePage(workspaceID int64, title, description, context, status, originPath, bodyText string, tagNames []string) (Page, error) {
	slug := Slugify(title)
	if slug == "" {
		return Page{}, fmt.Errorf("page title must produce a slug")
	}
	status, err := normalizePageStatus(status)
	if err != nil {
		return Page{}, err
	}

	res, err := s.db.Exec(
		`INSERT INTO pages (slug, workspace_id, title, description, context, status, origin_path, body_text)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		slug, workspaceID, title, description, context, status, originPath, bodyText,
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

// UpdatePage mutates a page in place. Pointers are optional: nil means
// "unchanged"; status (when non-nil) must be a valid status; tags (when non-nil)
// replaces the full tag set. The slug never changes (stable find-then-update
// handle); a changed title does not regenerate it.
func (s *Store) UpdatePage(slug string, title, description, context, status, originPath, bodyText *string, tags *[]string) (Page, error) {
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
	newOrigin := current.OriginPath
	if originPath != nil {
		newOrigin = *originPath
	}
	newBody := current.BodyText
	if bodyText != nil {
		newBody = *bodyText
	}

	_, err = s.db.Exec(
		`UPDATE pages SET title = ?, description = ?, context = ?, status = ?, origin_path = ?, body_text = ?, updated_at = ?
		 WHERE id = ?`,
		newTitle, newDesc, newCtx, newStatus, newOrigin, newBody, nowTimestamp(), current.ID,
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
