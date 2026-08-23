package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/udit-001/harbor/internal/db"
	"github.com/udit-001/harbor/internal/render"
	"github.com/udit-001/harbor/internal/web"
)

// NewMux builds the HTTP mux for the Harbor server: CSS serving, JSON API,
// page handlers. This is the testable internal seam — tests construct the mux
// and drive routes through httptest.NewRecorder without booting a real server.
//
// devCSS serves CSS from disk (no embed, no-cache) for `harbor dev`.
func NewMux(store *db.Store, dataDir string, devCSS bool) *http.ServeMux {
	mux := http.NewServeMux()
	broker := NewBroker()

	// Serve Tailwind CSS. In dev mode (DevCSS) read web/app.css from disk
	// on each request so styling changes are live without a Go rebuild;
	// disable caching so the browser always fetches the freshest file.
	mux.HandleFunc("GET /css/app.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		if devCSS {
			data, err := os.ReadFile("web/app.css")
			if err != nil {
				http.Error(w, "app.css not built — run `harbor dev` from the project root", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Write(data)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.CSS)
	})

	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.FaviconICO)
	})
	mux.HandleFunc("GET /favicon.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.FaviconPNG)
	})
	mux.HandleFunc("GET /favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.FaviconSVG)
	})

	mux.HandleFunc("GET /icon-192.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.Icon192)
	})
	mux.HandleFunc("GET /icon-512.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.Icon512)
	})

	mux.HandleFunc("GET /manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(web.Manifest)
	})

	// Vendored excalidraw viewer bundles (read-only app family, HARB-PLAN-4).
	// Path is sanitized by stripping the prefix; the remaining segments are
	// matched against the embedded FS so only vendored files can be served.
	mux.HandleFunc("GET /excalidraw/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/excalidraw/")
		if name == "" || strings.Contains(name, "..") {
			http.NotFound(w, r)
			return
		}
		data, err := web.Excalidraw.ReadFile("excalidraw/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ct := "application/octet-stream"
		switch {
		case strings.HasSuffix(name, ".js"):
			ct = "text/javascript; charset=utf-8"
		case strings.HasSuffix(name, ".woff2"):
			ct = "font/woff2"
		case strings.HasSuffix(name, ".md"):
			ct = "text/markdown; charset=utf-8"
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // content-pinned vendor dir
		_, _ = w.Write(data)
	})

	mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Service-Worker-Allowed", "/")
		w.Write(web.ServiceWorker)
	})

	mux.HandleFunc("GET /stopped.html", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(web.StoppedPage)
	})

	// JSON API
	mux.HandleFunc("GET /api/workspaces", jsonHandler(handleListWorkspaces(store)))
	mux.HandleFunc("GET /api/workspaces/{id}", jsonHandler(handleGetWorkspace(store)))
	mux.HandleFunc("GET /api/stats", jsonHandler(handleStats(store)))
	mux.HandleFunc("GET /api/search", jsonHandler(handleSearch(store)))
	mux.HandleFunc("GET /api/pages", jsonHandler(handlePagesJSON(store)))
	mux.HandleFunc("GET /api/pages/{slug}/comments", jsonHandler(handleListPageComments(store)))
	mux.HandleFunc("POST /api/pages/{slug}/comments", jsonHandler(handleCreatePageComment(store)))
	mux.HandleFunc("PATCH /api/pages/{slug}/comments/{id}", jsonHandler(handleUpdatePageComment(store)))
	mux.HandleFunc("GET /api/pages/{slug}/changes", jsonHandler(handleListPageChanges(store)))

	// App pages
	mux.HandleFunc("GET /", handleAppShell(store))
	mux.HandleFunc("GET /page/{slug}", handlePageView(store, dataDir))
	mux.HandleFunc("GET /page/{slug}/raw", handlePageRaw(store, dataDir))
	mux.HandleFunc("GET /page/{slug}/view", handlePageViewDoc(store, dataDir))
	mux.HandleFunc("GET /page/{slug}/assets/{path...}", handlePageAssets(store))
	mux.HandleFunc("GET /about", handleAboutPage(store))

	// Live-sync: CLI mutations broadcast through the broker; the dashboard
	// subscribes via SSE. See events.go.
	mux.HandleFunc("GET /api/events", handleSSE(broker))
	mux.HandleFunc("POST /api/notify", handleNotify(broker))

	return mux
}

// dateShort returns the YYYY-MM-DD prefix of a timestamp.
func dateShort(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// ── helpers ──

func jsonHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fn(w, r)
	}
}

func jsonResponse(w http.ResponseWriter, v any) {
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// writePage renders a full page and writes it to the response. The sidebar
// data is emptied until later tickets add workspace-of-pages navigation.
func writePage(w http.ResponseWriter, title, activeWS string, content string) {
	f := render.Frame{Title: title, ActiveWS: activeWS}
	fmt.Fprint(w, render.Page(f, content))
}

// writeNotFound renders a styled 404 page inside the app frame.
func writeNotFound(w http.ResponseWriter, title, message string) {
	w.WriteHeader(http.StatusNotFound)
	writePage(w, title, "", render.NotFound(title, message))
}

// ── API Handlers ──

func handleListWorkspaces(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, _ := store.GetWorkspaces()
		if ws == nil {
			ws = []db.Workspace{}
		}
		jsonResponse(w, ws)
	}
}

func handleGetWorkspace(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		ws, err := store.GetWorkspace(id)
		if err != nil {
			jsonError(w, "not found", 404)
			return
		}
		jsonResponse(w, ws)
	}
}

func handleStats(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, _ := store.GetWorkspaces()
		jsonResponse(w, map[string]any{"totalWorkspaces": len(ws)})
	}
}

// handleSearch performs full-text search over the page library (title,
// description, context, body, tag names/descriptions). Returns flat results.
func handleSearch(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			jsonError(w, "missing q", 400)
			return
		}

		pages, err := store.SearchPages(q, db.PageFilter{})
		if err != nil {
			jsonError(w, "search failed", http.StatusInternalServerError)
			return
		}
		type apiResult struct {
			Type    string `json:"type"`
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet,omitempty"`
		}
		results := make([]apiResult, 0, len(pages))
		for _, p := range pages {
			results = append(results, apiResult{
				Type:    "page",
				Title:   p.Title,
				URL:     "/page/" + p.Slug,
				Snippet: truncateSnippet(p.Description, 200),
			})
		}
		jsonResponse(w, results)
	}
}

func truncateSnippet(s string, maxLen int) string {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) <= maxLen {
		return trimmed
	}
	cut := strings.LastIndex(trimmed[:maxLen], " ")
	if cut < 1 {
		cut = maxLen
	}
	return strings.TrimSpace(trimmed[:cut]) + "..."
}

// ── Pages ──

func handleAboutPage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writePage(w, "About", "", render.About())
	}
}

// handleAppShell serves the Harbor Library home ("decide" surface): sidebar of
// workspaces + tags, status segment, live-search box, and the page list. Query
// params (status/workspace/tag/q) narrow the list via server-side render.
func handleAppShell(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			writeNotFound(w, "Page not found", "The page you're looking for doesn't exist.")
			return
		}
		data := buildLibraryData(store, r)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(render.Library(data)))
	}
}

// handlePagesJSON serves the complete page list as JSON — the dataset the
// library's client-side filters (status pills, search, sidebar workspace/tag)
// run against in the browser. No filtering here: the client holds the full set
// and narrows it locally for instant feedback without reloads.
func handlePagesJSON(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows := libraryRows(store, db.PageFilter{}, "")
		if rows == nil {
			rows = []render.PageRow{}
		}
		jsonResponse(w, rows)
	}
}

// handleListPageComments serves the full comment thread for a page (oldest
// first). The shell's comment panel fetches this when it opens.
func handleListPageComments(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if _, err := store.PageBySlug(slug); err != nil {
			jsonError(w, "page not found", http.StatusNotFound)
			return
		}
		comments, err := store.ListComments(db.CommentFilter{PageSlug: slug, Status: r.URL.Query().Get("status")})
		if err != nil {
			jsonError(w, "failed to load comments", http.StatusInternalServerError)
			return
		}
		if comments == nil {
			comments = []db.CommentView{}
		}
		jsonResponse(w, comments)
	}
}

// handleListPageChanges serves the changes recorded for a page — what the
// what-changed walkthrough tours. The shell fetches this to know which
// data-cf-change markers exist and their title/description.
func handleListPageChanges(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		if _, err := store.PageBySlug(slug); err != nil {
			jsonError(w, "page not found", http.StatusNotFound)
			return
		}
		changes, err := store.ListChanges(slug, 0)
		if err != nil {
			jsonError(w, "failed to load changes", http.StatusInternalServerError)
			return
		}
		if changes == nil {
			changes = []db.Change{}
		}
		jsonResponse(w, changes)
	}
}

// handleCreatePageComment writes a new comment to the DB for a page. The page
// file is never touched. The multi-anchor `anchors` list is canonical (HARB-29);
// the legacy type/anchor/quote fields are accepted as a single-anchor fallback
// so old clients keep working.
func handleCreatePageComment(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slug := r.PathValue("slug")
		var body struct {
			Type    string      `json:"type"`
			Anchor  string      `json:"anchor"`
			Quote   string      `json:"quote"`
			Body    string      `json:"body"`
			Anchors []db.Anchor `json:"anchors"`
			ReplyTo int64       `json:"replyTo"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		var (
			created db.CommentView
			err     error
		)
		if len(body.Anchors) > 0 {
			created, err = store.CreateCommentAnchors(slug, body.Body, body.Anchors, body.ReplyTo)
		} else {
			created, err = store.CreateComment(slug, body.Anchor, body.Quote, body.Type, body.Body)
		}
		if err != nil {
			if _, perr := store.PageBySlug(slug); perr != nil {
				jsonError(w, "page not found", http.StatusNotFound)
				return
			}
			jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		jsonResponse(w, created)
	}
}

// handleUpdatePageComment edits an open comment (body + anchors) and/or
// transitions its status. Editing a done comment is rejected (HARB-20).
func handleUpdatePageComment(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			jsonError(w, "invalid comment id", http.StatusBadRequest)
			return
		}
		var body struct {
			Body    *string     `json:"body"`
			Anchors []db.Anchor `json:"anchors"`
			Status  *string     `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		var out db.CommentView
		if body.Body != nil || body.Anchors != nil {
			b := ""
			if body.Body != nil {
				b = *body.Body
			}
			out, err = store.UpdateComment(id, b, body.Anchors)
			if err != nil {
				jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if body.Status != nil {
			out, err = store.UpdateCommentStatus(id, *body.Status)
			if err != nil {
				jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if body.Body == nil && body.Anchors == nil && body.Status == nil {
			jsonError(w, "nothing to update", http.StatusBadRequest)
			return
		}
		jsonResponse(w, out)
	}
}

func pageFilterFromQuery(r *http.Request) db.PageFilter {
	return db.PageFilter{
		Status:        r.URL.Query().Get("status"),
		WorkspaceSlug: r.URL.Query().Get("workspace"),
		TagName:       r.URL.Query().Get("tag"),
	}
}

// libraryRows builds the render rows for a filter. One query via
// ListPageRows (workspace name, tags, and open-feedback counts are all folded
// into it — no per-page follow-ups). q (when set) goes through the page FTS;
// otherwise it's a plain filtered list.
func libraryRows(store *db.Store, filter db.PageFilter, q string) []render.PageRow {
	list, err := store.ListPageRows(filter, q)
	if err != nil {
		return []render.PageRow{}
	}
	rows := make([]render.PageRow, 0, len(list))
	for _, p := range list {
		rows = append(rows, render.PageRow{
			Slug:         p.Slug,
			Title:        p.Title,
			Desc:         p.Description,
			Workspace:    p.Workspace,
			Format:       p.Format,
			Status:       p.Status,
			Tags:         p.TagList(),
			Updated:      shortDate(p.UpdatedAt),
			FeedbackOpen: p.FeedbackOpen,
		})
	}
	return rows
}

func buildLibraryData(store *db.Store, r *http.Request) render.LibraryData {
	filter := pageFilterFromQuery(r)
	q := r.URL.Query().Get("q")

	data := render.LibraryData{
		Filter: render.LibraryFilter{
			Status:    filter.Status,
			Workspace: filter.WorkspaceSlug,
			Tag:       filter.TagName,
			Q:         q,
		},
	}

	if byWs, err := store.PagesCountByWorkspace(); err == nil {
		for _, w := range mustWorkspaces(store) {
			data.Workspaces = append(data.Workspaces, render.LibrarySection{Name: w.Name, Count: byWs[w.Name]})
		}
	}
	if byTag, err := store.PagesCountByTag(); err == nil {
		for _, t := range mustTags(store) {
			data.Tags = append(data.Tags, render.LibrarySection{Name: t.Name, Count: byTag[t.Name]})
		}
	}

	data.Pages = libraryRows(store, filter, q)
	data.Total = len(data.Pages)
	data.HrefQuery = filterQueryString(filter, q)
	return data
}

// filterQueryString builds the "?status=…&workspace=…&tag=…&q=…" suffix that
// carries the current set across a page-link navigation.
func filterQueryString(filter db.PageFilter, q string) string {
	var parts []string
	add := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+url.QueryEscape(v))
		}
	}
	add("status", filter.Status)
	add("workspace", filter.WorkspaceSlug)
	add("tag", filter.TagName)
	add("q", q)
	if len(parts) == 0 {
		return ""
	}
	return "?" + strings.Join(parts, "&")
}

// managedArtifactPath resolves the managed file for a page under the given
// data dir. The stored filename is <slug>.<format> — the file IS the artifact.
// Workspace names and slugs are slugified/stable, so the path is
// deterministic and safe.
func managedArtifactPath(dataDir, workspaceName, slug, format string) string {
	return filepath.Join(dataDir, "store", workspaceName, slug+"."+format)
}

// formatContentTypes maps each page format to the content type its raw bytes
// are served as. Only native-family formats reach the raw endpoint; the view
// endpoint renders text-frame formats from these same bytes.
func formatContentType(format string, data []byte) string {
	switch format {
	case db.FormatPDF:
		return "application/pdf"
	case db.FormatSVG:
		return "image/svg+xml"
	case db.FormatImage:
		ct := http.DetectContentType(data) // png/jpeg/gif/webp/bmp
		if !strings.HasPrefix(ct, "image/") {
			return "application/octet-stream"
		}
		return ct
	case db.FormatMarkdown:
		return "text/markdown; charset=utf-8"
	case db.FormatText:
		return "text/plain; charset=utf-8"
	case db.FormatExcalidraw:
		return "application/json"
	default:
		return "text/html; charset=utf-8"
	}
}

// handlePageRaw serves a page's file byte-for-byte, as the agent wrote it:
// native formats (html/pdf/svg/image) are the iframe src and pop-out target;
// markdown/text bytes feed the view endpoint. No restyle, no injection, no
// theme toggle. 404 when the page or its managed file is missing.
func handlePageRaw(store *db.Store, dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := store.PageBySlug(r.PathValue("slug"))
		if err != nil {
			writeRawNotFound(w)
			return
		}
		wsName := ""
		if ws, werr := store.GetWorkspace(page.WorkspaceID); werr == nil {
			wsName = ws.Name
		}
		data, err := os.ReadFile(managedArtifactPath(dataDir, wsName, page.Slug, page.Format))
		if err != nil {
			writeRawNotFound(w)
			return
		}
		w.Header().Set("Content-Type", formatContentType(page.Format, data))
		_, _ = w.Write(data)
	}
}

// handlePageAssets serves a page's workspace assets: GET /page/{slug}/assets/*
// maps to <the page's workspace>/assets/* — so an artifact referencing
// assets/foo.css relatively just works, byte-for-byte, with no rewrite.
// Workspace-scoped on purpose: two workspaces may hold different files under
// the same name; the page's own workspace wins. Assets are raw by model (no
// DB row), so this is a plain guarded file serve.
func handlePageAssets(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := store.PageBySlug(r.PathValue("slug"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		ws, err := store.GetWorkspace(page.WorkspaceID)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		root := filepath.Join(ws.Path, "assets")
		req := filepath.Clean(filepath.Join(root, r.PathValue("path")))
		if req != root && !strings.HasPrefix(req, root+string(filepath.Separator)) {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, req)
	}
}

// handlePageViewDoc serves derived views the pageview iframe can host:
// text-frame formats render from their source (markdown via goldmark, text as
// styled <pre>), excalidraw gets the vendored read-only viewer shell. The
// stored file is never touched — the view is derived on read, raw bytes remain
// at /raw (pop-out shows them). Native formats don't route here; the iframe
// points at /raw directly.
func handlePageViewDoc(store *db.Store, dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := store.PageBySlug(r.PathValue("slug"))
		if err != nil {
			writeRawNotFound(w)
			return
		}

		// App family: the shell only needs the raw URL — it fetches the scene
		// itself, so no file read here.
		if page.Format == db.FormatExcalidraw {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			rawURL := "/page/" + page.Slug + "/raw"
			_, _ = w.Write([]byte(render.ExcalidrawView(render.PageMeta{Slug: page.Slug, Title: page.Title, Format: page.Format}, rawURL)))
			return
		}

		wsName := ""
		if ws, werr := store.GetWorkspace(page.WorkspaceID); werr == nil {
			wsName = ws.Name
		}
		data, err := os.ReadFile(managedArtifactPath(dataDir, wsName, page.Slug, page.Format))
		if err != nil {
			writeRawNotFound(w)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(render.TextFrameView(render.PageMeta{Slug: page.Slug, Title: page.Title, Format: page.Format}, string(data))))
	}
}

func writeRawNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte("page not found"))
}

// handlePageView serves the page view (consume surface): one shared header for
// container ⇄ full modes, an iframe onto /page/{slug}/raw. prev/next are scoped
// to the current Library set (the filter query params).
func handlePageView(store *db.Store, dataDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := store.PageBySlug(r.PathValue("slug"))
		if err != nil {
			writeRawNotFound(w)
			return
		}

		filter := pageFilterFromQuery(r)
		q := r.URL.Query().Get("q")
		set := libraryRows(store, filter, q) // ordered current set
		prevURL, nextURL := "", ""
		wsName := ""
		tags := []string{}
		openFb := 0
		for i, row := range set {
			if row.Slug != page.Slug {
				continue
			}
			// The row already carries everything the header needs (workspace,
			// tags, open feedback) — no follow-up queries for the current page.
			wsName, tags, openFb = row.Workspace, row.Tags, row.FeedbackOpen
			if i > 0 {
				prevURL = "/page/" + set[i-1].Slug + filterQueryString(filter, q)
			}
			if i < len(set)-1 {
				nextURL = "/page/" + set[i+1].Slug + filterQueryString(filter, q)
			}
			break
		}
		if wsName == "" {
			// Direct visit with a filter that excludes this page (or a dangling
			// workspace): fall back to direct lookups for the header.
			if ws, werr := store.GetWorkspace(page.WorkspaceID); werr == nil {
				wsName = ws.Name
			}
			if ts, terr := store.TagsForPage(page.Slug); terr == nil {
				tags = []string{}
				for _, t := range ts {
					tags = append(tags, t.Name)
				}
			}
			if m, merr := store.OpenCommentCounts(); merr == nil {
				openFb = m[page.Slug]
			}
		}

		// The iframe src follows the view family: native formats (html/pdf/svg/
		// image) serve their raw bytes; text-frame formats (markdown/text) get
		// the derived view. Pop-out always targets the raw bytes.
		iframeURL := "/page/" + page.Slug + "/raw"
		switch page.Format {
		case db.FormatMarkdown, db.FormatText, db.FormatExcalidraw:
			iframeURL = "/page/" + page.Slug + "/view"
		}
		data := render.PageViewData{
			Slug:         page.Slug,
			Title:        page.Title,
			Status:       page.Status,
			Format:       page.Format,
			Workspace:    wsName,
			Tags:         tags,
			IframeURL:    iframeURL,
			RawURL:       "/page/" + page.Slug + "/raw",
			BackURL:      "/" + filterQueryString(filter, q),
			PrevURL:      prevURL,
			NextURL:      nextURL,
			FeedbackOpen: openFb,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(render.PageView(data)))
	}
}

func mustWorkspaces(store *db.Store) []db.Workspace {
	ws, err := store.GetWorkspaces()
	if err != nil {
		return nil
	}
	return ws
}

func mustTags(store *db.Store) []db.Tag {
	tags, err := store.ListTags()
	if err != nil {
		return nil
	}
	return tags
}

func shortDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// ── Live-sync: SSE subscribe + CLI notify broadcast ──

// handleSSE opens a Server-Sent Events stream that delivers broker
// broadcasts to the dashboard client. The topic query param selects
// which event stream to subscribe to ("workspace:<name>" or "home").
// The handler blocks until the client disconnects (r.Context cancelled),
// then unsubscribes to release the broker's channel.
func handleSSE(b *Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topic := r.URL.Query().Get("topic")
		if topic == "" {
			jsonError(w, "missing topic", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		// Prevent the response from being buffered by intermediaries.
		// Localhost has none, but the header is cheap insurance.
		w.Header().Set("X-Accel-Buffering", "no")

		// SSE connections are long-lived; the server's global WriteTimeout
		// (30s) would otherwise kill them. Clear the deadline for this
		// connection only — other routes keep the timeout backstop.
		rc := http.NewResponseController(w)
		_ = rc.SetWriteDeadline(time.Time{})

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Write a comment line so the client knows the stream is live
		// before any broadcast arrives.
		fmt.Fprintf(w, ":connected\n\n")
		flusher.Flush()

		ch, unsub := b.Subscribe(topic)
		defer unsub()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-ch:
				if !ok {
					return
				}
				data, _ := json.Marshal(ev)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}

// handleNotify accepts a JSON payload from the CLI and broadcasts it to
// dashboard clients. For topic-scoped events ("changed", "page-changed"),
// it broadcasts to the named topic's subscribers. For "navigate" events
// (agent-driven URL navigation), it broadcasts to all subscribers
// regardless of topic. Returns JSON with the delivered subscriber count.
func handleNotify(b *Broker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Topic    string `json:"topic"`
			Type     string `json:"type"`
			PageType string `json:"pageType"`
			Seq      int    `json:"seq"`
			Slug     string `json:"slug"`
			URL      string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if body.Type == "" {
			jsonError(w, "type is required", http.StatusBadRequest)
			return
		}
		ev := Event{
			Type:     body.Type,
			PageType: body.PageType,
			Seq:      body.Seq,
			Slug:     body.Slug,
			URL:      body.URL,
		}
		var delivered int
		if body.Type == "navigate" {
			delivered = b.BroadcastAll(ev)
		} else {
			if body.Topic == "" {
				jsonError(w, "topic is required for non-navigate events", http.StatusBadRequest)
				return
			}
			delivered = b.Broadcast(body.Topic, ev)
		}
		jsonResponse(w, map[string]int{"delivered": delivered})
	}
}
