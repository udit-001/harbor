package db

import (
	"fmt"
	"strings"
)

// Tags are first-class grouping objects shared by pages (via page_tags). The
// description is the semantic payload that powers tag-description search.

const tagColumns = `id, name, description, created_at, updated_at`

func scanTag(row interface{ Scan(...any) error }) (Tag, error) {
	var t Tag
	err := row.Scan(&t.ID, &t.Name, &t.Description, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func scanTags(rows RowScanner) ([]Tag, error) {
	return scanRows(rows, "tag", scanTag)
}

// TagByName returns a single tag by its unique name.
func (s *Store) TagByName(name string) (Tag, error) {
	row := s.db.QueryRow("SELECT "+tagColumns+" FROM tags WHERE name = ?", name)
	tag, err := scanTag(row)
	if err != nil {
		return Tag{}, err
	}
	return tag, nil
}

// CreateTag adds a NEW tag with the given description. Fails if a tag with
// that name already exists — description mutation belongs to UpdateTag, and
// add-or-update would silently clobber an existing tag's semantic payload.
func (s *Store) CreateTag(name, description string) (Tag, error) {
	if _, err := s.TagByName(name); err == nil {
		return Tag{}, fmt.Errorf("tag %q already exists — use 'tag update' to change its description", name)
	}
	if _, err := s.db.Exec(
		"INSERT INTO tags (name, description) VALUES (?, ?)",
		name, description,
	); err != nil {
		return Tag{}, fmt.Errorf("create tag %q: %w", name, err)
	}
	return s.TagByName(name)
}

// ListTags lists all tags ordered by name.
func (s *Store) ListTags() ([]Tag, error) {
	rows, err := s.db.Query("SELECT " + tagColumns + " FROM tags ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTags(rows)
}

// UpdateTagDescription sets a tag's description (the semantic payload that
// powers tag-description search). Returns an error if the tag is missing.
func (s *Store) UpdateTagDescription(name, description string) error {
	res, err := s.db.Exec(
		"UPDATE tags SET description = ?, updated_at = ? WHERE name = ?",
		description, nowTimestamp(), name,
	)
	if err != nil {
		return fmt.Errorf("update tag %q: %w", name, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tag %q not found", name)
	}
	return nil
}

// DeleteTag permanently removes a tag and its associations (cascade).
func (s *Store) DeleteTag(name string) error {
	res, err := s.db.Exec("DELETE FROM tags WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("delete tag %q: %w", name, err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tag %q not found", name)
	}
	return nil
}

// ftsTermsLike builds a LIKE-based matcher over tag name + description for the
// raw query terms (used for tag-description search). Returns a SQL fragment
// and the args to bind, in placeholder order.
func ftsTermsLike(query string) (string, []any) {
	var parts []string
	var args []any
	for _, tok := range strings.Fields(query) {
		tok = strings.TrimRight(tok, "*")
		if tok == "" {
			continue
		}
		parts = append(parts,
			"(t.name LIKE ? OR t.description LIKE ?)")
		args = append(args, "%"+tok+"%", "%"+tok+"%")
	}
	if len(parts) == 0 {
		return "0", args
	}
	return "(" + strings.Join(parts, " OR ") + ")", args
}
