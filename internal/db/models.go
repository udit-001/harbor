package db

// Workspace represents a named body of work — the harbor organizing folder.
type Workspace struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"` // directory name (slug-ish, unique)
	Topic       string `db:"topic" json:"topic"`
	Description string `db:"description" json:"description"`
	Path        string `db:"path" json:"path"` // absolute path to workspace dir
	CreatedAt   string `db:"created_at" json:"createdAt"`
	LastStudied string `db:"last_studied" json:"lastStudied"`
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
	OriginPath  string `db:"origin_path" json:"originPath"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
	UpdatedAt   string `db:"updated_at" json:"updatedAt"`
}

// Scrap is one loose, unstructured capture in the global scratchpad. It is
// deliberately NOT workspace-scoped (global, sealed from workspaces). Title is
// required and drives the stable slug; Body is free text (URLs live inside).
// Status is exactly "active" (default agent read) or "done".
type Scrap struct {
	ID        int64  `db:"id" json:"id"`
	Slug      string `db:"slug" json:"slug"`
	Title     string `db:"title" json:"title"`
	Body      string `db:"body" json:"body"`
	Status    string `db:"status" json:"status"`
	CreatedAt string `db:"created_at" json:"createdAt"`
	UpdatedAt string `db:"updated_at" json:"updatedAt"`
}

// Tag is a first-class grouping object on the scratchpad: a name plus a
// description (the description is the semantic payload — a bare name adds no
// info over keyword-matching the body). Many-to-many with Scrap.
type Tag struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`
	Description string `db:"description" json:"description"`
	CreatedAt   string `db:"created_at" json:"createdAt"`
	UpdatedAt   string `db:"updated_at" json:"updatedAt"`
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
