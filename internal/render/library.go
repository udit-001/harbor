package render

import (
	"fmt"
	"strings"
)

// LibraryData drives the Harbor Library home — the "decide" surface. Workspaces
// and Tags populate the sidebar; Pages is the current filtered list; Filter
// reflects the active query params so links can round-trip.
type LibraryData struct {
	Workspaces []LibrarySection
	Tags       []LibrarySection
	Filter     LibraryFilter
	Pages      []PageRow
	Total      int
	// HrefQuery is the "?status=…&workspace=…&tag=…&q=…" suffix appended to page
	// row links so the page view's prev/next stays scoped to the current set.
	HrefQuery string
}

// LibrarySection is one sidebar entry with its page count.
type LibrarySection struct {
	Name  string
	Count int
}

// LibraryFilter is the active narrowing. Q is the live search box; the rest are
// sidebar/segment selections (all rendered as query-param links).
type LibraryFilter struct {
	Status    string // "" | draft | published | archived
	Workspace string // "" = all
	Tag       string // "" = all
	Q         string
}

// PageRow is one row in the library list.
type PageRow struct {
	Slug         string
	Title        string
	Desc         string
	Workspace    string
	Status       string
	Tags         []string
	Updated      string
	FeedbackOpen int // 0 for now; derived feedback lands with M2
}

func e(tagName string) string { return esc(tagName) }

// Library renders the full Harbor library page (its own shell: topbar + sidebar
// of workspaces/tags + status segment + hairline list). Self-contained — inline
// CSS + JS — so it does not depend on the Pharos lesson shell.
func Library(data LibraryData) string {
	var b strings.Builder
	b.WriteString(libraryHead(data))
	b.WriteString(libraryShell(data))
	b.WriteString(libraryScript(data))
	return b.String()
}

func libraryHead(data LibraryData) string {
	title := "harbor — Library"
	return `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + esc(title) + `</title>` + libraryCSS() + `</head><body>`
}

func libraryShell(data LibraryData) string {
	var b strings.Builder
	b.WriteString(`<div class="app">`)
	b.WriteString(librarySidebar(data))
	b.WriteString(`<div class="main">`)
	b.WriteString(libraryTopbar(data))
	b.WriteString(libraryContent(data))
	b.WriteString(`</div></div>`)
	return b.String()
}

func librarySidebar(data LibraryData) string {
	var b strings.Builder
	b.WriteString(`<aside class="side"><div class="brand">` + logo() + ` harbor</div><nav class="nav">`)

	// All pages
	allActive := data.Filter.Workspace == "" && data.Filter.Tag == ""
	addLink := func(href, label string, count int, active bool, section bool) {
		if section {
			b.WriteString(`<div class="sec">` + esc(label) + `</div>`)
			return
		}
		cls := "link"
		if active {
			cls += " active"
		}
		icon := "▦"
		if label == "All pages" {
			icon = "≡"
		}
		fmt.Fprintf(&b, `<a class="%s" href="%s"><span class="ic">%s</span>%s<span class="cnt">%d</span></a>`,
			cls, esc(href), icon, esc(label), count)
	}

	allHref := "/?status=" + esc(data.Filter.Status)
	if data.Filter.Q != "" {
		allHref += "&q=" + esc(data.Filter.Q)
	}
	addLink(allHref, "All pages", data.Total, allActive, false)

	b.WriteString(`<div class="sec">Workspaces</div>`)
	for _, w := range data.Workspaces {
		active := data.Filter.Workspace == w.Name
		addLink("/?workspace="+e(w.Name)+"&status="+e(data.Filter.Status), w.Name, w.Count, active, false)
	}
	b.WriteString(`<div class="sec">Tags</div>`)
	for _, t := range data.Tags {
		active := data.Filter.Tag == t.Name
		addLink("/?tag="+e(t.Name)+"&status="+e(data.Filter.Status), t.Name, t.Count, active, false)
	}

	b.WriteString(`</nav></aside>`)
	return b.String()
}

func libraryTopbar(data LibraryData) string {
	searchVal := ""
	if data.Filter.Q != "" {
		searchVal = ` value="` + e(data.Filter.Q) + `"`
	}
	return `<div class="topbar"><span class="crumb">Library</span>
<div class="search"><input id="q" type="search" placeholder="Search pages…" aria-label="Search pages"` + searchVal + `></div></div>`
}

func libraryContent(data LibraryData) string {
	var b strings.Builder
	b.WriteString(`<div class="body"><div class="pagehead"><h1>All pages</h1>`)
	b.WriteString(fmt.Sprintf(`<span class="count">%d pages</span></div>`, data.Total))
	b.WriteString(libraryStatusSeg(data.Filter.Status))
	b.WriteString(`<div id="list" class="list">`)
	if data.Total == 0 {
		b.WriteString(libraryEmpty())
	} else {
		b.WriteString(libraryRows(data.Pages, data.HrefQuery))
	}
	b.WriteString(`</div></div>`)
	return b.String()
}

func libraryStatusSeg(active string) string {
	items := []struct {
		val   string
		label string
	}{
		{"", "All"}, {draft, "Draft"}, {pub, "Published"}, {arch, "Archived"},
	}
	var b strings.Builder
	b.WriteString(`<div class="seg">`)
	for _, it := range items {
		cls := ""
		if active == it.val {
			cls = " class=\"active\""
		}
		if it.val == "" {
			fmt.Fprintf(&b, `<a%s href="/">%s</a>`, cls, it.label)
		} else {
			fmt.Fprintf(&b, `<a%s href="/?status=%s">%s</a>`, cls, it.val, it.label)
		}
	}
	b.WriteString(`</div>`)
	return b.String()
}

const (
	draft = "draft"
	pub   = "published"
	arch  = "archived"
)

func libraryRows(rows []PageRow, hrefQuery string) string {
	var b strings.Builder
	if len(rows) == 0 {
		return libraryEmpty()
	}
	for _, r := range rows {
		b.WriteString(`<a class="row" href="/page/` + e(r.Slug) + hrefQuery + `">`)
		b.WriteString(`<div class="grow">`)
		b.WriteString(`<div class="t"><span class="name">` + esc(r.Title) + `</span>`)
		if r.FeedbackOpen > 0 {
			fmt.Fprintf(&b, `<span class="fb"><i></i>%d open</span>`, r.FeedbackOpen)
		}
		b.WriteString(`</div>`)
		if r.Desc != "" {
			b.WriteString(`<div class="desc">` + esc(r.Desc) + `</div>`)
		}
		b.WriteString(`<div class="meta">`)
		b.WriteString(`<span class="tag">` + e(r.Workspace) + `</span>`)
		for _, t := range r.Tags {
			b.WriteString(`<span class="tag">` + e(t) + `</span>`)
		}
		b.WriteString(`</div></div>`)
		b.WriteString(`<div class="right"><span class="updated">` + esc(r.Updated) + `</span><span class="badge ` + statusClass(r.Status) + `">` + esc(r.Status) + `</span></div>`)
		b.WriteString(`</a>`)
	}
	return b.String()
}

func statusClass(s string) string {
	switch s {
	case pub:
		return "pub"
	case arch:
		return "arch"
	default:
		return "draft"
	}
}

// LibraryRows returns just the list fragment (rows or empty state). Used by the
// live-search endpoint so the client can swap the list without reloading.
func LibraryRows(rows []PageRow, hrefQuery string) string { return libraryRows(rows, hrefQuery) }

func libraryEmpty() string {
	return `<div class="empty"><h2>No pages yet</h2>
<p>Harbor holds the HTML pages your agent builds. When the agent makes a page worth keeping, it imports it here with one command.</p>
<code>harbor page add your-page.html --workspace my-work --description "what it shows"</code></div>`
}

func libraryScript(data LibraryData) string {
	// Live search: debounced fetch of /api/pages, swapping the list in place.
	// Sidebar + status segments are plain links → server re-renders.
	return `<script>
const input=document.getElementById('q');
let t;
input.addEventListener('input',()=>{clearTimeout(t);t=setTimeout(()=>{
  const q=input.value;
  const base=new URL('/api/pages',location.origin);
  base.searchParams.set('q',q);
  const st=new URLSearchParams(location.search).get('status');
  const ws=new URLSearchParams(location.search).get('workspace');
  const tag=new URLSearchParams(location.search).get('tag');
  if(st)base.searchParams.set('status',st);
  if(ws)base.searchParams.set('workspace',ws);
  if(tag)base.searchParams.set('tag',tag);
  fetch(base).then(r=>r.json()).then(j=>{
    // j is the HTML string for the updated list; swap it in.
    document.getElementById('list').innerHTML=j;
  });
},150);});</script></body></html>`
}

func logo() string {
	return `<svg width="22" height="22" viewBox="0 0 24 24" fill="none"><rect x="4" y="3" width="16" height="18" rx="1.5"/><path d="M8 9l4 4 4-4M8 14l4 4 4-4" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/></svg>`
}

func libraryCSS() string {
	return `<style>
:root{--bg:#eceff4;--surface:#fff;--surface2:#f6f8fb;--border:#d8dee9;--hair:#e5e9f0;
--text:#4c566a;--muted:#8891a0;--strong:#2e3440;--acc:#5e81ac;--acc-soft:#e0e7ff;
--ok:#4a7a2e;--ok-soft:#e6f0e6;--warn:#d08770;--warn-soft:#fadfd2;--arch:#8891a0;--arch-soft:#eceff4;
--r:8px;--rs:6px;--ease:cubic-bezier(.23,1,.32,1)}
*{box-sizing:border-box}
body{margin:0;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:var(--bg);
color:var(--text);font-size:15px;line-height:1.6;-webkit-font-smoothing:antialiased}
a{text-decoration:none;color:inherit}
.app{display:grid;grid-template-columns:232px 1fr;min-height:100vh}
.side{background:var(--surface);border-right:1px solid var(--border);padding:16px 12px}
.brand{display:flex;align-items:center;gap:9px;color:var(--strong);font-weight:700;font-size:15px;padding:4px 8px 16px}
.brand svg{color:var(--acc)}
.nav{display:flex;flex-direction:column;gap:2px}
.sec{padding:14px 8px 6px;font-size:11px;letter-spacing:.04em;color:var(--muted);font-weight:600}
.link{display:flex;align-items:center;gap:9px;padding:7px 8px;border-radius:var(--rs);color:var(--text);font-size:14px}
.link:hover{background:var(--surface2)}
.link.active{background:var(--acc-soft);color:var(--acc);font-weight:600}
.link .ic{width:15px;color:var(--muted);flex:none}
.link .cnt{margin-left:auto;font-size:11px;color:var(--muted);background:rgba(0,0,0,.05);border-radius:999px;padding:0 7px}
.main{min-width:0}
.topbar{display:flex;align-items:center;gap:16px;height:52px;padding:0 24px;background:var(--surface);border-bottom:1px solid var(--border)}
.search{flex:1;max-width:460px;margin-left:auto}
.search input{width:100%;padding:8px 12px;border:1px solid var(--border);border-radius:var(--rs);
background:var(--bg);font:400 14px var(--font);color:var(--text);outline:none}
.search input:focus{border-color:var(--acc)}
.body{padding:22px 24px 60px}
.pagehead{display:flex;align-items:baseline;gap:12px;padding-bottom:6px}
.pagehead h1{font-size:20px;font-weight:700;color:var(--strong);margin:0}
.count{color:var(--muted);font-size:13px}
.seg{display:flex;gap:2px;padding:12px 0 10px;border-bottom:1px solid var(--hair)}
.seg a{border:1px solid transparent;color:var(--muted);font:600 13px var(--font);padding:6px 13px;border-radius:var(--rs)}
.seg a.active{background:var(--surface);border-color:var(--border);color:var(--strong);box-shadow:0 1px 2px rgba(0,0,0,.04)}
.list{margin-top:4px}
.row{display:flex;align-items:center;gap:14px;padding:14px 12px;border-bottom:1px solid var(--hair);border-radius:var(--rs);cursor:pointer}
.row:hover{background:var(--surface2)}
.row .grow{min-width:0;flex:1}
.row .t{display:flex;align-items:center;gap:9px}
.row .name{font-weight:600;color:var(--strong);font-size:14.5px}
.row .desc{color:var(--muted);font-size:13px;margin-top:2px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.row .meta{display:flex;align-items:center;gap:7px;margin-top:5px;flex-wrap:wrap}
.tag{font-size:11.5px;color:var(--muted);background:var(--surface);border:1px solid var(--hair);border-radius:4px;padding:0 6px}
.row .right{display:flex;align-items:center;gap:12px;flex:none}
.badge{font:600 11px var(--font);padding:2px 8px;border-radius:999px}
.badge.pub{color:var(--ok);background:var(--ok-soft)}
.badge.draft{color:var(--warn);background:var(--warn-soft)}
.badge.arch{color:var(--arch);background:var(--arch-soft)}
.fb{display:inline-flex;align-items:center;gap:5px;color:var(--warn);font:600 11px var(--font)}
.fb i{width:7px;height:7px;border-radius:999px;background:var(--warn)}
.updated{color:var(--muted);font-size:12px;white-space:nowrap}
.empty{text-align:center;padding:68px 20px 54px;border:1px dashed var(--border);border-radius:var(--r);background:var(--surface);margin-top:6px}
.empty h2{margin:0 0 6px;font-size:16px;color:var(--strong)}
.empty p{margin:0 auto 18px;color:var(--muted);font-size:13.5px;max-width:400px}
.empty code{background:var(--bg);border:1px solid var(--hair);border-radius:var(--rs);padding:9px 14px;display:inline-block;
font:500 12.5px ui-monospace,SFMono-Regular,Menlo,monospace;color:var(--text)}
@media(prefers-reduced-motion:reduce){*{transition:none}}
</style>`
}
