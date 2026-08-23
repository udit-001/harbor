package render

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PageViewData drives the Harbor page view — the "consume" surface. One shared
// header for both container and full modes; the page renders in an iframe that
// fills the content column (container) or the whole viewport (full).
type PageViewData struct {
	Slug         string
	Title        string
	Status       string
	Format       string // artifact format; drives which chrome is meaningful
	Workspace    string
	Tags         []string
	IframeURL    string // iframe src: /raw for native formats, /view for text-frame
	RawURL       string // /page/<slug>/raw — the pop-out target (raw bytes)
	BackURL      string // library URL preserving the current filter
	PrevURL      string // "" when no previous page in the current set
	NextURL      string // "" when no next page in the current set
	FeedbackOpen int    // open comments on this page (derived; drives the chrome badge)
}

// PageView renders the full page-view document. Mode is decided client-side:
// localStorage keyed by slug, defaulting to container. Switching modes only
// toggles the envelope class — the iframe src (RawURL) is never reloaded or
// restyled, so the page stays byte-for-byte as the agent wrote it.
func PageView(d PageViewData) string {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">`)
	b.WriteString(`<title>` + esc(d.Title) + ` — harbor</title>`)
	b.WriteString(ThemeBootScript)
	b.WriteString(`<link rel="stylesheet" href="/css/app.css?v=21">`) // Tailwind utilities for new pageview markup (HARB-32 seed)
	b.WriteString(pageViewCSS())
	b.WriteString(`</head><body data-slug="` + e(d.Slug) + `">`)
	b.WriteString(pageViewHeader(d))
	b.WriteString(`<main class="wrap"><iframe id="frame" src="` + e(d.IframeURL) + `" title="` + esc(d.Title) + `"></iframe></main>`)
	b.WriteString(commentPanelMarkup())
	b.WriteString(changeTourMarkup())
	// Per-page data seam for the extracted JS files (HARB-36). Go writes the
	// dynamic context; the //go:embed'd pageview js reads window.__harbor.
	if ctx, cerr := json.Marshal(struct {
		Slug      string `json:"slug"`
		Format    string `json:"format"`
		Workspace string `json:"workspace"`
		Live      struct {
			Topic string `json:"topic"`
			Mode  string `json:"mode"`
			Slug  string `json:"slug"`
		} `json:"live"`
	}{Slug: d.Slug, Format: d.Format, Workspace: d.Workspace, Live: struct {
		Topic string `json:"topic"`
		Mode  string `json:"mode"`
		Slug  string `json:"slug"`
	}{Topic: "workspace:" + d.Workspace, Mode: "workspace", Slug: d.Slug}}); cerr == nil {
		b.WriteString(`<script>window.__harbor=`)
		b.WriteString(string(ctx))
		b.WriteString(`;</script>` + "\n")
	}
	b.WriteString(`<script>` + pageviewShellJS + `</script>` + "\n")
	b.WriteString(`<script>` + pageviewAnnotationsJS + `</script>` + "\n")
	b.WriteString(`<script>` + pageviewTourJS + `</script>` + "\n")
	if d.Workspace != "" {
		b.WriteString(`<script>` + harborLiveSyncJS + `</script>` + "\n")
	}
	b.WriteString(`</body></html>`)
	return b.String()
}

func pageViewHeader(d PageViewData) string {
	var b strings.Builder
	b.WriteString(`<header class="pv" id="pv">`)
	// Breadcrumb: back to the library (parent), then the page title as the
	// current crumb — one integrated location region, not chrome + heading in
	// two separate rows.
	b.WriteString(`<a class="icon-btn back" href="` + e(d.BackURL) + `" data-tooltip="Back to library" aria-label="Back to library">` + iconChevronLeft() + `</a>`)
	b.WriteString(`<a class="crumb" href="` + e(d.BackURL) + `">Library</a>`)
	b.WriteString(`<span class="sep">/</span>`)
	b.WriteString(`<h1 class="title">` + esc(d.Title) + `</h1>`)
	b.WriteString(`<span class="badge ` + statusClass(d.Status) + `">` + esc(d.Status) + `</span>`)
	if d.FeedbackOpen > 0 {
		fmt.Fprintf(&b, `<span class="pv-fb" id="pvFb" data-n="%d" title="%d open comment(s)"><i></i>%d open</span>`, d.FeedbackOpen, d.FeedbackOpen, d.FeedbackOpen)
	}
	b.WriteString(`<div class="chips">`)
	b.WriteString(`<span class="tag">` + e(d.Workspace) + `</span>`)
	for _, t := range d.Tags {
		b.WriteString(`<span class="tag">` + e(t) + `</span>`)
	}
	b.WriteString(`</div><div class="navr">`)
	// prev/next within the current set
	if d.PrevURL != "" {
		b.WriteString(`<a class="icon-btn" href="` + e(d.PrevURL) + `" data-tooltip="Previous" aria-label="Previous">` + iconPrev() + `</a>`)
	} else {
		b.WriteString(`<span class="icon-btn disabled" title="No previous">` + iconPrev() + `</span>`)
	}
	if d.NextURL != "" {
		b.WriteString(`<a class="icon-btn" href="` + e(d.NextURL) + `" data-tooltip="Next" aria-label="Next">` + iconNext() + `</a>`)
	} else {
		b.WriteString(`<span class="icon-btn disabled" title="No next">` + iconNext() + `</span>`)
	}
	b.WriteString(`<button class="icon-btn" id="commentBtn" aria-haspopup="dialog" aria-expanded="false" aria-label="Comments" data-tooltip="Comments">` + iconComment() + `</button>`)
	b.WriteString(`<button class="icon-btn" id="modeBtn" data-tooltip="Toggle full / container" aria-label="Toggle view mode">` + iconExpand() + `</button>`)
	b.WriteString(`<a class="icon-btn" href="` + e(d.RawURL) + `" target="_blank" rel="noopener" data-tooltip="Pop out" aria-label="Pop out">` + iconPopOut() + `</a>`)
	b.WriteString(`</div></header>`)
	return b.String()
}

func iconPrev() string {
	return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M15 18l-6-6 6-6"/></svg>`
}
func iconNext() string {
	return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"><path d="M9 6l6 6-6 6"/></svg>`
}
func iconExpand() string {
	return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3"/></svg>`
}
func iconPopOut() string {
	return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><path d="M15 3h6v6M10 14 21 3"/></svg>`
}
func iconComment() string {
	return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>`
}
func iconCollect() string {
	return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 8v8M8 12h8"/><circle cx="12" cy="12" r="9"/></svg>`
}

// commentPanelMarkup is the shell-side comment panel: a right-hand drawer where
// the human composes anchored feedback about the page. It lives entirely in the
// shell — the agent's HTML (in the iframe) is never injected into. Type/anchor/
// quote are captured client-side from the same-origin iframe and submitted via
// the JSON API; the page file itself is never touched.
func commentPanelMarkup() string {
	return `<div class="comment-panel" id="commentPanel" role="dialog" aria-modal="true" aria-labelledby="commentPanelTitle" aria-hidden="true" inert>
  <div class="cp-head">
    <span class="cp-title" id="commentPanelTitle">Comment</span>
    <button class="icon-btn cp-close" id="commentClose" aria-label="Close comments" title="Close">` + iconClose() + `</button>
  </div>
  <div class="cp-body">
    <!-- List-first pane (default): review history + filters, compose is opt-in. -->
    <div class="cp-pane" id="cpListPane">
      <div class="cp-toolbar">
        <div class="cp-filters" role="group" aria-label="Filter comments"><span class="cp-thumb" aria-hidden="true"></span>
          <button type="button" class="cp-chip" data-filter="open">Open</button>
          <button type="button" class="cp-chip" data-filter="done">Done</button>
          <button type="button" class="cp-chip" data-filter="all">All</button>
        </div>
        <button type="button" class="cp-newbtn" id="cpNew">+ New comment</button>
      </div>
      <div class="cp-list-head" id="cpListHead" hidden>Comments</div>
      <ol class="cp-list" id="cpList" aria-live="polite"></ol>
    </div>
    <!-- Compose pane (whole-doc via New comment; anchored via the pill). -->
    <div class="cp-pane" id="cpComposePane" hidden>
      <div class="cp-capture" id="cpCapture">Click an element in the page, or select its text, to anchor your comment — then type and post.</div>
      <div class="cp-replyto" id="cpReplyTo" hidden></div>
      <div class="cp-row">
        <select id="cpType" aria-label="Comment type">
          <option value="general">Whole page</option>
          <option value="selection">Text selection</option>
          <option value="element">Element</option>
        </select>
      </div>
      <div class="cp-fields">
        <div class="cp-anchor"><span class="cp-label">Where</span><code id="cpAnchor">whole page</code><button type="button" class="cp-clear" id="cpClear" hidden aria-label="Clear selection" title="Clear">` + iconClear() + `</button></div>
        <div class="cp-quote" id="cpQuoteWrap" hidden><span class="cp-label">Quote</span><blockquote id="cpQuote"></blockquote></div>
      </div>
      <form id="commentForm" class="cp-form">
        <label class="cp-label" for="cpBody">Your feedback</label>
        <textarea id="cpBody" rows="4" required placeholder="Tell the agent what to change…"></textarea>
        <div class="cp-error" id="cpError" role="alert"></div>
        <div class="cp-actions">
          <button type="button" class="cp-pill" id="cpCancel">Cancel</button>
          <button type="submit" id="cpSubmit" class="cp-submit">Post comment</button>
        </div>
      </form>
    </div>
  </div>
</div>
<!-- Inline compose box (HARB-32): a docked shell box for anchored comments,
     positioned over the anchor; stays put while typing. Whole-doc compose
     stays in the drawer. Uses Tailwind utilities (app.css) — pageview CSS
     seeding (HARB-15/32). -->
<div class="fixed z-50 w-72 max-w-[92vw] overflow-hidden rounded-md border border-[var(--border)] bg-[var(--surface)] shadow-[0_1px_2px_rgba(46,52,64,.06),0_6px_16px_-4px_rgba(46,52,64,.16),0_20px_44px_-14px_rgba(46,52,64,.26)]" id="cpInline" role="dialog" aria-label="Comment" aria-hidden="true" hidden>
  <div class="flex items-center gap-2 border-b border-[var(--hair)] px-4 py-3">
    <span class="text-[13px] font-semibold text-[var(--strong)]" id="cpInlineTitle">Comment</span>
    <span class="flex-1 truncate text-[11px] text-[var(--muted)]" id="cpInlineWhere"></span>
    <button type="button" class="icon-btn ml-auto" id="cpInlineClose" aria-label="Discard comment" title="Cancel">` + iconClose() + `</button>
  </div>
  <div class="p-4">
    <textarea id="cpInlineBody" rows="3" required placeholder="Tell the agent what to change…" class="min-h-[72px] w-full resize-y rounded border border-[var(--border)] bg-[var(--surface)] px-3 py-2 text-[13px] leading-6 text-[var(--text)] outline-none focus:outline-2 focus:outline-[var(--acc)]"></textarea>
    <div class="mt-1 text-[12.5px] text-[var(--warn)]" id="cpInlineError" role="alert"></div>
    <div class="mt-2 flex items-center justify-end gap-2">
      <button type="button" class="cursor-pointer rounded border border-[var(--border)] px-4 py-2 text-[12.5px] font-semibold text-[var(--text)] hover:bg-[var(--surface2)]" id="cpInlineCancel">Cancel</button>
      <button type="button" class="cursor-pointer rounded bg-[var(--acc)] px-4 py-2 text-[12.5px] font-semibold text-white hover:brightness-95" id="cpInlinePost">Post comment</button>
    </div>
  </div>
</div>`
}
func iconClose() string {
	return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12"/></svg>`
}
func iconClear() string {
	return `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12"/></svg>`
}

// changeTourMarkup is the floating "what changed" walkthrough affordance: a
// chip shown only when the page records changes AND at least one has a matching
// data-cf-change marker in the DOM, plus a stepper card that tours each change.
// It lives entirely in the shell — the marker is read back from the agent's
// page, never injected into it.
func changeTourMarkup() string {
	return `<div class="cf">
  <button type="button" class="cf-btn" id="cfBtn" hidden aria-label="What changed" title="What changed"><i></i>What changed</button>
  <div class="cf-card" id="cfCard" role="dialog" aria-label="What changed" aria-hidden="true" hidden>
    <div class="cf-head" id="cfHead"><span class="cf-step" id="cfStep"></span><button type="button" class="icon-btn cf-close" id="cfClose" aria-label="Close what-changed" title="Close">` + iconClose() + `</button></div>
    <div class="cf-body"><div class="cf-title" id="cfTitle"></div><div class="cf-desc" id="cfDesc"></div></div>
    <div class="cf-actions">
      <button type="button" class="cf-pill" id="cfPrev">Prev</button>
      <button type="button" class="cf-pill" id="cfNext">Next</button>
      <button type="button" class="cf-pill cf-primary" id="cfDone">Done</button>
    </div>
  </div>
</div>`
}

// changeTourScript wires the what-changed walkthrough. It fetches the page's
// `changes`, locates each data-cf-change marker in the same-origin iframe DOM,
// and—only if at least one marker resolves—shows a "What changed" chip. Clicking
// it opens a stepper that scrolls to + highlights each change in sequence with
// its title/description. Honors reduced motion and is skipped entirely (no tour,
// no error) when the page has no changes or no matching markers.
func pageViewCSS() string {
	return `<style>
` + ThemeTokens() + `
:root{--rs:6px;--panelw:340px}
*{box-sizing:border-box}
body{margin:0;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;
background:var(--bg);color:var(--text);font-size:15px}
a{text-decoration:none;color:inherit}
.pv{display:flex;align-items:center;gap:9px;height:52px;padding:0 14px;background:var(--surface);border-bottom:1px solid var(--border)}
.pv .title{font-weight:600;color:var(--strong);font-size:14px;white-space:nowrap}
.pv a.crumb{color:var(--muted);font-size:13px;transition:color .1s}
.pv a.crumb:hover{color:var(--text)}
.pv .sep{color:var(--muted);opacity:.55;margin:0 1px}
.pv .chips{display:flex;align-items:center;gap:6px;overflow:hidden}
.pv .navr{margin-left:auto;display:flex;align-items:center;gap:6px;flex:none}
.badge{padding:2px 8px;border-radius:999px;font:600 11px var(--font)}
.badge.pub{color:var(--ok);background:var(--ok-soft)}
.badge.draft{color:var(--warn);background:var(--warn-soft)}
.badge.arch{color:var(--arch);background:var(--arch-soft)}
.pv-fb{display:inline-flex;align-items:center;gap:5px;color:var(--warn);font:600 11px var(--font);
background:var(--warn-soft);border-radius:999px;padding:2px 8px;white-space:nowrap}
.pv-fb i{width:7px;height:7px;border-radius:999px;background:var(--warn)}
.tag{font-size:11.5px;color:var(--muted);background:var(--surface);border:1px solid var(--hair);border-radius:4px;padding:0 6px}
.icon-btn{border:1px solid transparent;background:transparent;color:var(--muted);width:32px;height:32px;border-radius:var(--rs);
display:grid;place-items:center;transition:color .1s,transform 120ms var(--ease);cursor:pointer}
.icon-btn:hover{color:var(--strong);background:var(--surface2)}
.icon-btn:disabled,.icon-btn.disabled{opacity:.35}
.wrap{height:calc(100vh - 52px);transition:height .3s var(--ease),margin-right .3s var(--ease)}
#frame{width:100%;height:100%;border:0;background:#fff;transition:height .3s var(--ease)}
/* The comment panel docks rather than overlays the page: opening it compresses
   the content column by the panel width so the page stays fully visible and
   selectable beside it (no hidden right edge). On narrow screens the panel
   stays a full-width overlay — there is no room to dock. */
@media(min-width:681px){body.commenting .wrap{margin-right:var(--panelw)}}
/* container (default): the page fills the column below the header, edge-to-edge
   and seamless — like a harbor frame page living inside the shell. The header
   stays visible (sticky). */
body:not(.full) .pv{position:sticky;top:0;z-index:10}
/* full: the page fills the whole viewport, and the header becomes a floating
   bar that slides in from the top on intent (hover) and auto-hides — entering
   and leaving along the same symmetric path, so it never snaps over content. */
body.full .wrap{height:100vh}
body.full #frame{height:100vh}
body.full .pv{position:fixed;top:0;left:0;right:0;z-index:10;background:var(--surface);
transform:translateY(-100%);opacity:0;pointer-events:none;
transition:transform .25s var(--ease),opacity .25s var(--ease)}
body.full .pv.is-visible{transform:translateY(0);opacity:1;pointer-events:auto}
/* Reduced motion: cross-fade the header instead of sliding it, no height snap. */
@media(prefers-reduced-motion:reduce){
.wrap,#frame{transition:none}
body.full .pv{transition:opacity .2s ease;transform:none}
}
/* ── Comment panel ── */
.comment-panel{position:fixed;top:0;right:0;bottom:0;width:var(--panelw);max-width:92vw;z-index:40;
display:flex;flex-direction:column;background:var(--surface);border-left:1px solid var(--border);
box-shadow:-8px 0 24px rgba(0,0,0,.10);
transform:translateX(100%);opacity:0;pointer-events:none;
transition:transform .28s var(--ease),opacity .28s var(--ease)}
.comment-panel.open{transform:translateX(0);opacity:1;pointer-events:auto}
.cp-head{display:flex;align-items:center;gap:10px;padding:0 12px 0 16px;min-height:44px;border-bottom:1px solid var(--border)}
.cp-title{font-weight:600;color:var(--strong);font-size:13px;letter-spacing:.02em}
.cp-close{width:44px;height:44px;margin-left:auto}
.cp-body{flex:1;min-height:0;overflow-y:auto;padding:20px}
/* List-first pane (HARB-33): toolbar with filters + New comment, plus the
   compose pane shown only on demand. */
.cp-pane[hidden]{display:none}
.cp-toolbar{display:flex;align-items:center;gap:8px;margin-bottom:10px;flex-wrap:wrap}
.cp-filters{position:relative;display:flex;gap:4px;background:var(--surface2);border:1px solid var(--border);border-radius:999px;padding:3px}
.cp-thumb{position:absolute;top:3px;bottom:3px;left:3px;width:0;border-radius:999px;background:var(--surface);box-shadow:0 1px 2px rgba(46,52,64,.12);will-change:transform}
.cp-filters.thumb-ready .cp-thumb{transition:transform .18s var(--ease-in-out)}
.cp-chip{border:0;background:transparent;color:var(--muted);position:relative;z-index:1;font:600 11px var(--font);padding:5px 10px;
border-radius:999px;cursor:pointer;transition:color .1s}
.cp-chip:hover{color:var(--strong)}
.cp-chip.active{color:var(--strong)}
.cp-newbtn{margin-left:auto;border:1px solid var(--border);background:var(--surface);color:var(--acc);
border-radius:999px;padding:6px 12px;font:600 11.5px var(--font);cursor:pointer;white-space:nowrap;
transition:background .1s,border-color .1s}
.cp-newbtn:hover{background:var(--acc-soft);border-color:var(--acc)}
.cp-form{display:flex;flex-direction:column;gap:8px;margin-top:18px}
.cp-capture{font-size:12.5px;color:var(--muted);line-height:1.6;background:var(--surface2);
border:1px dashed var(--border);border-radius:var(--rs);padding:12px 14px;margin:2px 0 4px}
.cp-row{display:flex;align-items:center;gap:10px;margin-top:14px}
.cp-row select{flex:1;border:1px solid var(--border);background:var(--surface);color:var(--text);
border-radius:var(--rs);padding:7px 9px;font:400 13px var(--font)}
.cp-fields{display:flex;flex-direction:column;gap:12px;padding-top:2px;margin-top:14px}
.cp-label{font:500 11px var(--font);color:var(--muted);text-transform:uppercase;letter-spacing:.05em;margin-bottom:4px}
.cp-anchor{display:flex;align-items:center;gap:10px}
.cp-anchor code{font:11px ui-monospace,SFMono-Regular,Menlo,monospace;color:var(--acc);
background:var(--acc-soft);border-radius:4px;padding:2px 6px;overflow-wrap:anywhere;flex:0 1 auto}
.cp-clear{border:1px solid var(--border);background:var(--surface);color:var(--muted);border-radius:var(--rs);
width:32px;height:32px;display:grid;place-items:center;cursor:pointer;margin-left:auto;flex:none;
transition:color .1s,background-color .1s,border-color .1s}
.cp-clear:hover{color:var(--strong);background:var(--surface2);border-color:var(--text)}
.cp-clear[hidden]{display:none}
.cp-quote{display:flex;flex-direction:column;gap:6px;color:var(--text);margin-top:8px}
.cp-quote[hidden]{display:none}
.cp-quote blockquote{margin:0;font-size:12.5px;font-style:italic;color:var(--muted);line-height:1.6;
border-left:3px solid var(--border);padding:2px 0 2px 10px}
.cp-body textarea{width:100%;min-height:96px;resize:vertical;border:1px solid var(--border);
border-radius:var(--rs);background:var(--surface);color:var(--text);padding:10px 12px;
font:400 13px var(--font);line-height:1.6}
.cp-body textarea:focus-visible,.cp-close:focus-visible,.cp-submit:focus-visible,
.cp-row select:focus-visible{outline:2px solid var(--acc);outline-offset:1px}
.cp-actions{display:flex;align-items:center;justify-content:flex-end;gap:8px;padding-top:2px}
.cp-pill{border:1px solid var(--border);background:var(--surface);color:var(--text);border-radius:var(--rs);
padding:8px 16px;font:600 13px var(--font);cursor:pointer;
transition:background-color .1s,color .1s,border-color .1s,transform 120ms cubic-bezier(.23,1,.32,1)}
.cp-pill:hover{background:var(--surface2);border-color:var(--text)}
.cp-pill:active:not(:disabled){transform:scale(.97)}
.cp-pill:focus-visible{outline:2px solid var(--acc);outline-offset:1px}
.cp-submit{border:0;border-radius:var(--rs);background:var(--acc);color:#fff;padding:9px 18px;
font:600 13px var(--font);cursor:pointer}
.cp-submit:hover{filter:brightness(.96)}
.cp-submit:disabled{opacity:.5;cursor:not-allowed}
.cp-error{display:none;font-size:12.5px;color:var(--warn);background:var(--warn-soft);border-radius:var(--rs);padding:9px 12px}
.cp-error.show{display:block}
.cp-list-head{margin:28px 0 12px;font:600 12px var(--font);color:var(--muted);text-transform:uppercase;letter-spacing:.05em}
.cp-list{list-style:none;margin:0;padding:0;display:flex;flex-direction:column;gap:14px}
.cp-item{border:1px solid var(--border);border-radius:var(--rs);padding:13px 15px;background:var(--surface2)}
.cp-item.cp-item--done{border-style:dashed;border-color:var(--hair)}
.cp-item-top{display:flex;align-items:flex-start;gap:10px}
.cp-item-body{flex:1;min-width:0;font-size:13.5px;line-height:1.55;color:var(--text)}
.cp-item--done .cp-item-body{color:var(--muted)}
.cp-item-status{flex:none;display:inline-flex;align-items:center;gap:4px;margin-top:2px;color:var(--acc);font:600 10px var(--font);text-transform:uppercase;letter-spacing:.02em}
.cp-item-status::before{content:'';width:6px;height:6px;border-radius:999px;background:currentColor}
.cp-item-status.done{color:var(--ok)}
.cp-item-anchor{font-size:12px;color:var(--muted);margin-top:7px;overflow-wrap:anywhere}
.cp-item--done .cp-item-anchor{opacity:.75}
.cp-item-quote{font-size:12px;font-style:italic;color:var(--muted);line-height:1.55;margin-top:8px}
/* List-item actions (HARB-34): Jump / Edit / Done / Reply. */
.cp-item-actions{display:flex;gap:6px;margin-top:10px;flex-wrap:wrap}
.cp-act{border:1px solid var(--border);background:var(--surface);color:var(--muted);border-radius:var(--rs);padding:4px 9px;font:600 11px var(--font);cursor:pointer;transition:color .1s,background .1s,border-color .1s}
.cp-act.cp-act-go{color:var(--acc)}
.cp-act.cp-act-go:hover{border-color:var(--acc)}
.cp-act:hover{color:var(--strong);background:var(--surface2);border-color:var(--text)}
.cp-replyto{margin:0 0 10px;font-size:12px;color:var(--muted);background:var(--surface2);border-radius:var(--rs);padding:8px 10px}
.cp-replyto[hidden]{display:none}
/* Collect mode (HARB-35): header toggle, floating pins bar. */
/* Collect: unified selection-driven flow (picker cursor + N-spots chip). */
.cp-pins{position:fixed;left:16px;bottom:16px;z-index:80;display:flex;align-items:center;gap:6px;background:var(--surface);border:1px solid var(--border);border-radius:999px;padding:6px 6px 6px 12px;box-shadow:0 1px 3px rgba(46,52,64,.12);opacity:0;transform:translateY(8px);transition:opacity .12s var(--ease),transform .12s var(--ease)}
.cp-pins.cp-show{opacity:1;transform:translateY(0)}
.cp-pins[hidden]{display:none}
#cpInline{opacity:0;transform:scale(.95);transform-origin:bottom center;transition:opacity .15s var(--ease),transform .15s var(--ease)}
#cpInline.show{opacity:1;transform:scale(1)}
.pv-tooltip{position:fixed;z-index:70;padding:4px 10px;background:#2e3440;color:#f8fafc;font:500 12px var(--font);line-height:1.4;white-space:nowrap;border-radius:4px;pointer-events:none;opacity:0;transform:translateY(2px);transition:opacity .1s var(--ease),transform .1s var(--ease)}
.pv-tooltip.show{opacity:1;transform:translateY(0)}
.pv-tooltip-arrow{position:fixed;z-index:70;border:4px solid transparent;border-bottom-color:#2e3440;pointer-events:none;opacity:0;transition:opacity .1s var(--ease)}
@media(prefers-reduced-motion:reduce){.pv .icon-btn:active,.comment-panel .cp-act:active,.comment-panel .cp-chip:active,.comment-panel .cp-newbtn:active,.comment-panel .cp-clear:active,.comment-panel .cp-pill:active,.comment-panel #cpSubmit:active,.cf-btn:active,.cf-card .cf-pill:active,.cp-pins-clear:active,.cp-pins-post:active{transform:none}}
.pv-tooltip-arrow.show{opacity:1}
.pv-toast{position:fixed;left:16px;bottom:56px;z-index:90;max-width:min(420px,calc(100vw - 32px));background:var(--surface);color:var(--text);border:1px solid var(--border);border-radius:var(--rs);padding:9px 13px;font:500 12.5px var(--font);line-height:1.4;box-shadow:0 2px 10px rgba(46,52,64,.16);opacity:0;transform:translateY(4px);transition:opacity .14s var(--ease),transform .14s var(--ease)}
.pv-toast.show{opacity:1;transform:translateY(0)}
.cp-pins-count{font:600 12px var(--font);color:var(--strong);margin-right:2px}
.cp-pins-clear,.cp-pins-post{border:1px solid var(--border);background:var(--surface);border-radius:999px;padding:5px 11px;font:600 11.5px var(--font);cursor:pointer;transition:background .1s,color .1s}
.cp-pins-clear{color:var(--muted)}
.cp-pins-clear:hover{color:var(--strong);background:var(--surface2)}
.cp-pins-post{background:var(--acc);border-color:var(--acc);color:#fff}
.cp-pins-post:hover{filter:brightness(.95)}
.cp-empty{text-align:center;padding:34px 18px 32px;margin-top:4px;border:1px dashed var(--border);border-radius:var(--rs);background:var(--surface);color:var(--muted);line-height:1.55}
.cp-empty-title{font-size:14px;font-weight:600;color:var(--strong);margin:0 0 7px}
.cp-empty-note{margin:0 auto;max-width:248px;font-size:12.5px}
.cp-empty-note b{color:var(--text)}
.cp-empty-action{border:1px solid var(--border);background:var(--surface);color:var(--acc);border-radius:var(--rs);padding:7px 14px;font:600 12px var(--font);cursor:pointer;transition:background .1s,color .1s,border-color .1s}
.cp-empty-action:hover{background:var(--acc-soft);border-color:var(--acc)}
.cp-skeleton{border:1px solid var(--hair);border-radius:var(--rs);padding:13px 15px;background:var(--surface2);display:flex;flex-direction:column;gap:9px}
.cp-skeleton span{display:block;height:9px;border-radius:4px;background:var(--hair)}
.cp-skeleton span:nth-child(2){width:72%}
.cp-skeleton span:nth-child(3){width:45%}
.cp-empty-err .cp-empty-note{margin-bottom:12px}
/* ── What-changed walkthrough ──
   Styled to match the shell's flat Nord surfaces (comment panel + header):
   --surface body, --border edge, --hair dividers, --rs radius, subtle shadow,
   and an accent-soft primary that survives dark mode. */
.cf-btn{position:fixed;left:16px;bottom:16px;z-index:45;display:inline-flex;align-items:center;gap:7px;
border:1px solid var(--border);background:var(--surface);color:var(--text);border-radius:999px;
padding:8px 14px;font:600 12.5px var(--font);cursor:pointer;box-shadow:0 1px 3px rgba(0,0,0,.08);
transition:background .1s,color .1s,border-color .1s}
.cf-btn:hover{background:var(--surface2);border-color:var(--text)}
.cf-btn[hidden]{display:none}
.cf-btn i{width:8px;height:8px;border-radius:999px;background:var(--acc)}
.cf-card{position:fixed;z-index:46;width:380px;max-width:92vw;
background:var(--surface);border-radius:var(--rs);padding:0;
box-shadow:0 1px 2px rgba(46,52,64,.06),0 6px 16px -4px rgba(46,52,64,.16),0 20px 44px -14px rgba(46,52,64,.26);
animation:cf-in 160ms cubic-bezier(.23,1,.32,1)}
/* Elevation instead of a ghost 1px border + wide soft blur: layered shadows
   (contact + mid + ambient) carry the lift. In dark the black shadow is
   invisible on the dark surface, so the shadow deepens AND a lit top edge
   (inset highlight) is added — that top light is what makes a dark dialog
   read as floating rather than bleeding into its background. */
[data-theme="dark"] .cf-card{
box-shadow:0 2px 6px rgba(0,0,0,.3),0 12px 28px -8px rgba(0,0,0,.5),0 28px 60px -20px rgba(0,0,0,.6),inset 0 1px 0 rgba(236,239,244,.07)}
@keyframes cf-in{from{opacity:0;transform:translateY(4px) scale(.98)}to{opacity:1;transform:none}}
/* F1 (animation audit): symmetric exit so close doesn't pop the card out —
   contract back the way it came (fade + subtle scale-down, 120ms). Held with
   forwards until JS hides the element. */
@keyframes cf-exit{from{opacity:1;transform:none}to{opacity:0;transform:translateY(4px) scale(.985)}}
.cf-card.cf-exit{animation:cf-exit 120ms cubic-bezier(.23,1,.32,1) forwards}
.cf-card::before{content:'';position:absolute;bottom:-6px;left:50%;width:12px;height:12px;
margin-left:-6px;background:var(--surface);transform:rotate(45deg)}
.cf-card.cf-card--below::before{top:-6px;bottom:auto}
.cf-card[hidden]{display:none}
.cf-head{display:flex;align-items:center;gap:10px;padding:14px 16px 11px;border-bottom:1px solid var(--hair)}
.cf-step{font:600 11px var(--font);color:var(--muted);text-transform:uppercase;letter-spacing:.05em}
.cf-close{margin-left:auto}
.cf-body{padding:13px 16px}
.cf-title{font-weight:600;color:var(--strong);font-size:14px}
.cf-desc{color:var(--muted);font-size:12.5px;line-height:1.55;margin-top:3px}
.cf-actions{display:flex;align-items:center;gap:8px;padding:12px 16px;border-top:1px solid var(--hair)}
.cf-pill{border:1px solid var(--border);background:var(--surface);color:var(--text);border-radius:var(--rs);
padding:7px 14px;font:600 12.5px var(--font);cursor:pointer;
transition:background-color .1s,color .1s,border-color .1s,transform 120ms cubic-bezier(.23,1,.32,1)}
.cf-pill:hover{background:var(--surface2)}
.cf-pill:active:not(:disabled){transform:scale(.97)}
.cf-pill:disabled{opacity:.4;cursor:not-allowed}
.cf-pill.cf-primary{background:var(--acc-soft);border-color:var(--acc);color:var(--acc)}
.cf-pill.cf-primary:hover{background:var(--surface)}
.cf-pill:focus-visible,.cf-close:focus-visible{outline:2px solid var(--acc);outline-offset:1px}
@media(prefers-reduced-motion:reduce){.cf-btn{transition:none}.cf-card{animation:none;transition:none}.cf-card.cf-exit{animation:none}.cf-pill{transition:background-color .1s,color .1s,border-color .1s}.cf-pill:active{transform:none}}
[data-theme="dark"] .cf-btn{box-shadow:0 2px 8px rgba(0,0,0,.32),inset 0 1px 0 rgba(236,239,244,.06)}
/* Press feedback — match the library's physicality (library.go .link:active):
   every clickable compresses slightly. Values are the repo's shared ones. */
.pv .icon-btn:active,.comment-panel .cp-act:active,.comment-panel .cp-chip:active,
.comment-panel .cp-newbtn:active,.comment-panel .cp-clear:active,
.comment-panel .cp-pill:active,.comment-panel #cpSubmit:active,
.cf-btn:active,.cf-card .cf-pill:active,.cp-pins-clear:active,.cp-pins-post:active{transform:scale(.97)}
.pv .icon-btn,.comment-panel .cp-act,.comment-panel .cp-chip,.comment-panel .cp-newbtn,
.comment-panel .cp-clear,.comment-panel .cp-pill,.cf-btn,.cf-card .cf-pill,
.cp-pins-clear,.cp-pins-post{transition:color .1s,background-color .1s,border-color .1s,transform 120ms var(--ease)}

@media(prefers-reduced-motion:reduce){.comment-panel{transition:opacity .18s ease;transform:none}
.comment-panel.open{transform:none}
.cp-pins{transition:none;transform:none}
#cpInline{transition:none;transform:none}
.pv-toast{transition:none;transform:none}
.cp-filters.thumb-ready .cp-thumb{transition:none}
.pv-tooltip,.pv-tooltip-arrow{transition:none;transform:none}
}
@media(max-width:680px){.comment-panel{width:100%;max-width:100vw}}
</style>`
}
