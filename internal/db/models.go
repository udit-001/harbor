package db

// Workspace represents a learning workspace.
type Workspace struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`   // directory name
	Topic       string `db:"topic" json:"topic"` // user-friendly topic
	Path        string `db:"path" json:"path"`   // absolute path to workspace dir
	CreatedAt   string `db:"created_at" json:"createdAt"`
	LastStudied string `db:"last_studied" json:"lastStudied"`
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
