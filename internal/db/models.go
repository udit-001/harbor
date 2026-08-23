package db

// Workspace represents a named body of work — the harbor organizing folder.
type Workspace struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"` // directory name (slug-ish, unique)
	Topic       string `db:"topic" json:"topic"`
	Description string `db:"description" json:"description"`
	Path        string `db:"path" json:"path"` // absolute path to workspace dir
	CreatedAt   string `db:"created_at" json:"createdAt"`
}

// PageStatus is the set of valid page statuses. It describes the page's
// readiness — NOT whether it has open feedback (that is derived from the
// comments queue, never stored here).
const (
	PageStatusDraft     = "draft"
	PageStatusPublished = "published"
	PageStatusArchived  = "archived"
)

// Page is the atomic artifact in harbor: a standalone HTML page the agent
// produced, imported into a managed store and owned by exactly one workspace.
// Title is required and drives the stable slug; the slug never changes on
// rename. Description describes what the page shows; Context describes where it
// came from / why it exists — both are searchable provenance. OriginPath is a
// soft, informational breadcrumb (never dereferenced).
type Page struct {
	ID          int64  `db:"id" json:"id"`
	Slug        string `db:"slug" json:"slug"`
	WorkspaceID int64  `db:"workspace_id" json:"workspaceId"`
	Title       string `db:"title" json:"title"`
	Description string `db:"description" json:"description"`
	Context     string `db:"context" json:"context"`
	Status      string `db:"status" json:"status"`
	Format      string `db:"format" json:"format"`
	OriginPath  string `db:"origin_path" json:"originPath"`
	BodyText    string `db:"body_text" json:"bodyText"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
	UpdatedAt   string `db:"updated_at" json:"updatedAt"`
}

// Page formats live in internal/extract (ArtifactHTML … ArtifactExcalidraw):
// one vocabulary for detection, storage validation, and serving. db stores
// the string and validates through extract.ValidArtifactFormat.

// Tag is a first-class grouping object shared by pages: a name plus a
// description (the description is the semantic payload — a bare name adds no
// info over keyword-matching the page text). Many-to-many with pages via
// page_tags.
type Tag struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
	UpdatedAt   string `db:"updated_at" json:"updatedAt"`
}

// CommentType is the set of valid comment anchor kinds.
const (
	CommentTypeSelection = "selection" // quoted text selection
	CommentTypeElement   = "element"   // a specific element, anchored by selector
	CommentTypeGeneral   = "general"   // page-level feedback, no anchor
)

// CommentStatus is the set of valid comment lifecycle states.
const (
	CommentStatusOpen       = "open"
	CommentStatusInProgress = "in-progress"
	CommentStatusDone       = "done"
)

// Anchor is a single reference to a spot in the page a comment is attached to.
// Kind mirrors the legacy CommentType: text (a quoted selection), element (a
// selector), or document (whole page). MarkerID optionally binds the anchor to
// a stable data-cf-change marker so it survives edits (HARB-24/27).
type Anchor struct {
	Kind     string `json:"kind"`
	Path     string `json:"path,omitempty"`
	Quote    string `json:"quote,omitempty"`
	MarkerID string `json:"markerId,omitempty"`
}

// Anchor kinds (canonical). These map onto the legacy comment types for the
// old single-anchor display columns: text→selection, element→element,
// document→general.
const (
	AnchorKindText     = "text"
	AnchorKindElement  = "element"
	AnchorKindDocument = "document"
)

// Comment is anchored human feedback on a page. The page file is never touched
// — comments live here, keyed by page_id and (optionally) an anchor. Quote is
// the selected text snippet; Anchor is the element's existing id or a stable
// computed CSS selector. Status drives the derived open-feedback surface.
type Comment struct {
	ID         int64   `db:"id" json:"id"`
	PageID     int64   `db:"page_id" json:"pageId"`
	Anchor     string  `db:"anchor" json:"anchor"`
	Quote      string  `db:"quote" json:"quote"`
	Type       string  `db:"type" json:"type"`
	Body       string  `db:"body" json:"body"`
	Status     string  `db:"status" json:"status"`
	CreatedAt  string  `db:"created_at" json:"createdAt"`
	ResolvedAt *string `db:"resolved_at" json:"resolvedAt"`
	// Anchors is the canonical multi-anchor list (JSON column). The legacy
	// Anchor/Quote/Type columns remain for backward-compat display and are
	// derived from the first anchor on write.
	Anchors   []Anchor `json:"anchors"`
	ReplyTo   *int64   `db:"reply_to" json:"replyTo,omitempty"`
	UpdatedAt string   `db:"updated_at" json:"updatedAt"`
}

// CommentView pairs a Comment with its page slug — the READY-TO-DISPLAY shape
// for the CLI (and the derived open-feedback surface). Listing joins on
// pages.slug so the shell and the agent never have to resolve a raw page_id.
type CommentView struct {
	Comment
	PageSlug string `json:"page"`
}

// Change records an agent edit that addresses feedback. ChangeID matches a
// data-cf-change marker embedded in the page HTML, so the what-changed
// walkthrough can map an edit back to the comment it resolves. CommentID is
// nullable: a change may carry no comment (general tidying).
type Change struct {
	ID          int64  `db:"id" json:"id"`
	PageID      int64  `db:"page_id" json:"pageId"`
	CommentID   *int64 `db:"comment_id" json:"commentId"`
	ChangeID    string `db:"change_id" json:"changeId"`
	Title       string `db:"title" json:"title"`
	Description string `db:"description" json:"description"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
}

// Settings holds user preferences.
type Settings struct {
	ID                  int64  `db:"id" json:"id"`
	DefaultView         string `db:"default_view" json:"defaultView"`
	ItemsPerPage        int    `db:"items_per_page" json:"itemsPerPage"`
	AssetsDir           string `db:"assets_dir" json:"assetsDir"`
	LastActiveWorkspace string `db:"last_active_workspace" json:"lastActiveWorkspace"`
}

// DisplayName returns the user-friendly topic if set, else the directory name.
// Used everywhere a human reads the name; URLs and keys must still use Name.
func (w Workspace) DisplayName() string {
	if w.Topic != "" {
		return w.Topic
	}
	return w.Name
}
