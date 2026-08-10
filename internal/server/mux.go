package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/udit-001/harbor/internal/db"
	"github.com/udit-001/harbor/internal/docutil"
	"github.com/udit-001/harbor/internal/markdown"
	"github.com/udit-001/harbor/internal/render"
	"github.com/udit-001/harbor/internal/web"
)

// NewMux builds the HTTP mux for the Harbor server: CSS serving, JSON API,
// page handlers. This is the testable internal seam — tests construct the mux
// and drive routes through httptest.NewRecorder without booting a real server.
//
// devCSS serves CSS from disk (no embed, no-cache) for `harbor dev`.
func NewMux(store *db.Store, devCSS bool) *http.ServeMux {
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

	// App pages
	mux.HandleFunc("GET /", handleAppShell(store))
	mux.HandleFunc("GET /workspace/{name}", handleWorkspacePage(store))
	mux.HandleFunc("GET /workspace/{name}/resources", handleDocPage(store, "resources"))
	mux.HandleFunc("GET /workspace/{name}/notes", handleDocPage(store, "notes"))
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

// handleSearch performs full-text search over the global scratchpad (scraps
// and their tag descriptions). Returns flat results.
func handleSearch(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			jsonError(w, "missing q", 400)
			return
		}

		scraps, _ := store.SearchScraps(q, "")
		type apiResult struct {
			Type    string `json:"type"`
			Title   string `json:"title"`
			URL     string `json:"url"`
			Snippet string `json:"snippet,omitempty"`
		}
		results := make([]apiResult, 0, len(scraps))
		for _, s := range scraps {
			results = append(results, apiResult{
				Type:    "scrap",
				Title:   s.Title,
				Snippet: truncateSnippet(s.Body, 200),
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

// handleAppShell serves the dashboard home page: a workspace grid. It is
// rebuilt by later tickets as Harbor's page/tag/workspace-of-pages surface.
func handleAppShell(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			writeNotFound(w, "Page not found", "The page you're looking for doesn't exist.")
			return
		}

		ws, _ := store.GetWorkspaces()
		data := render.DashboardData{Stats: render.Stats{Workspaces: len(ws)}}
		for _, w := range ws {
			data.Workspaces = append(data.Workspaces, render.WorkspaceCard{
				Name: w.Name, Topic: w.Topic, LastStudied: w.LastStudied,
			})
		}

		writePage(w, "Dashboard", "", render.Dashboard(data))
	}
}

// handleWorkspacePage serves a workspace landing page.
func handleWorkspacePage(store *db.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}
		ws := wsStore.Workspace()

		data := render.WorkspaceData{
			Workspace: render.Workspace{Name: ws.Name, Topic: ws.Topic},
		}
		writePage(w, ws.DisplayName(), name, render.WorkspacePage(data))
	}
}

// ── Workspace Document Page (Resources, Notes) ──

// docKind describes one workspace document readable from disk.
type docKind struct {
	title string
	path  func(db.Layout) string
}

var docKinds = map[string]docKind{
	"resources": {title: "Resources", path: db.Layout.ResourcesPath},
	"notes":     {title: "Notes", path: db.Layout.NotesPath},
}

func handleDocPage(store *db.Store, kind string) http.HandlerFunc {
	dk, ok := docKinds[kind]
	if !ok {
		panic("unknown doc kind: " + kind)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		wsStore, err := store.Workspace(name)
		if err != nil {
			writeNotFound(w, "Workspace not found", fmt.Sprintf("Workspace %q doesn't exist.", name))
			return
		}

		data := render.DocumentData{Title: dk.title, Kind: kind}

		raw, err := os.ReadFile(dk.path(wsStore.Layout()))
		if err == nil {
			trimmed := strings.TrimSpace(string(raw))
			if !docutil.IsTemplate(trimmed, kind) {
				if body := docutil.StripH1(trimmed); body != "" {
					data.BodyHTML = markdown.Render(body)
				}
			}
		}
		wsStore.Touch()
		if data.BodyHTML == "" {
			data.Empty = true
		}

		writePage(w, dk.title, name, render.Document(data))
	}
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
