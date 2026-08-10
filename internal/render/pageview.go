package render

import (
	"strings"
)

// PageViewData drives the Harbor page view — the "consume" surface. One shared
// header for both container and full modes; the page renders in an iframe that
// fills the content column (container) or the whole viewport (full).
type PageViewData struct {
	Slug      string
	Title     string
	Status    string
	Workspace string
	Tags      []string
	RawURL    string // /page/<slug>/raw — the iframe src and pop-out target
	BackURL   string // library URL preserving the current filter
	PrevURL   string // "" when no previous page in the current set
	NextURL   string // "" when no next page in the current set
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
	b.WriteString(`<script>(function(){try{var t=localStorage.getItem('harbor_theme');if(!t||t==='system'){t=window.matchMedia('(prefers-color-scheme:dark)').matches?'dark':'light'}document.documentElement.dataset.theme=t}catch(e){document.documentElement.dataset.theme='light'}})();</script>`)
	b.WriteString(pageViewCSS())
	b.WriteString(`</head><body data-slug="` + e(d.Slug) + `">`)
	b.WriteString(pageViewHeader(d))
	b.WriteString(`<div class="wrap"><iframe id="frame" src="` + e(d.RawURL) + `" title="` + esc(d.Title) + `"></iframe></div>`)
	b.WriteString(pageViewScript(d))
	if d.Workspace != "" {
		// Live-sync: reload this page's iframe (preserving scroll) when its
		// content changes, and follow agent-driven navigation.
		b.WriteString(liveSyncScript("workspace:"+d.Workspace, `{
  pageChanged: function(ev){ if(ev.slug==='`+d.Slug+`'){ var f=document.getElementById('frame'); if(f&&f.contentWindow){ try{ var y=f.contentWindow.scrollY; f.addEventListener('load',function rst(){ f.removeEventListener('load',rst); try{ f.contentWindow.scrollTo(0,y); }catch(_){} }); f.contentWindow.location.reload(); }catch(_){} } } },
  navigate: function(ev){ if(ev.url) location.href=ev.url; }
}`))
	}
	b.WriteString(`</body></html>`)
	return b.String()
}

func pageViewHeader(d PageViewData) string {
	var b strings.Builder
	b.WriteString(`<div class="pv" id="pv">`)
	// Breadcrumb: back to the library (parent), then the page title as the
	// current crumb — one integrated location region, not chrome + heading in
	// two separate rows.
	b.WriteString(`<a class="icon-btn back" href="` + e(d.BackURL) + `" title="Back to library" aria-label="Back to library">` + iconChevronLeft() + `</a>`)
	b.WriteString(`<a class="crumb" href="` + e(d.BackURL) + `">Library</a>`)
	b.WriteString(`<span class="sep">/</span>`)
	b.WriteString(`<span class="title">` + esc(d.Title) + `</span>`)
	b.WriteString(`<span class="badge ` + statusClass(d.Status) + `">` + esc(d.Status) + `</span>`)
	b.WriteString(`<div class="chips">`)
	b.WriteString(`<span class="tag">` + e(d.Workspace) + `</span>`)
	for _, t := range d.Tags {
		b.WriteString(`<span class="tag">` + e(t) + `</span>`)
	}
	b.WriteString(`</div><div class="navr">`)
	// prev/next within the current set
	if d.PrevURL != "" {
		b.WriteString(`<a class="icon-btn" href="` + e(d.PrevURL) + `" title="Previous" aria-label="Previous">` + iconPrev() + `</a>`)
	} else {
		b.WriteString(`<span class="icon-btn disabled" title="No previous">` + iconPrev() + `</span>`)
	}
	if d.NextURL != "" {
		b.WriteString(`<a class="icon-btn" href="` + e(d.NextURL) + `" title="Next" aria-label="Next">` + iconNext() + `</a>`)
	} else {
		b.WriteString(`<span class="icon-btn disabled" title="No next">` + iconNext() + `</span>`)
	}
	b.WriteString(`<button class="icon-btn" id="modeBtn" title="Toggle full / container" aria-label="Toggle view mode">` + iconExpand() + `</button>`)
	b.WriteString(`<a class="icon-btn" href="` + e(d.RawURL) + `" target="_blank" rel="noopener" title="Pop out" aria-label="Pop out">` + iconPopOut() + `</a>`)
	b.WriteString(`</div></div>`)
	return b.String()
}

func pageViewScript(d PageViewData) string {
	return `<script>
(function(){
  const slug=document.body.dataset.slug;
  const KEY='harbor_view_'+slug;
  const pv=document.getElementById('pv');
  let mode=localStorage.getItem(KEY)||'container';
  // sync() sets the envelope mode AND the header state together, so toggling to
  // full immediately slides the floating header away (no stuck overlap) and, while
  // hidden, drops pointer-events so it never blocks the page's top strip. The
  // header re-enters on hover along the same path.
  const sync=()=>{
    const full=(mode==='full');
    document.body.classList.toggle('full', full);
    hide();
  };
  function show(){ if(mode==='full') pv.classList.add('is-visible'); }
  function hide(){ pv.classList.remove('is-visible'); }
  sync();
  document.getElementById('modeBtn').addEventListener('click',()=>{
    mode=(mode==='full')?'container':'full';
    localStorage.setItem(KEY,mode);
    sync(); // envelope only — iframe src untouched, page not reloaded
  });
  // Reveal on hovering the top "peek" band; dismissing chrome on peek-away.
  document.body.addEventListener('mouseenter', show);
  document.body.addEventListener('mouseleave', hide);
  // In full mode the iframe fills the viewport, so interacting with or scrolling
  // the content must dismiss the header (the pointer lives inside the iframe,
  // not the outer body). Same-origin: attach capture listeners on the iframe's
  // contentWindow — always available, and they catch events from within the
  // loaded document without depending on document-readiness timing.
  const PEEK=60;
  function wireFrame(){
    var w;
    try{ w=document.getElementById('frame').contentWindow; }catch(_){ return; }
    if(!w) return;
    w.addEventListener('scroll', hide, true);
    w.addEventListener('pointerdown', hide, true);
    w.addEventListener('keydown', hide, true);
    w.addEventListener('mousemove', function(e){ if(e.clientY<PEEK) show(); else hide(); }, true);
  }
  wireFrame();
})();</script>`
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

func pageViewCSS() string {
	return `<style>
:root{--bg:#eceff4;--surface:#fff;--surface2:#f6f8fb;--border:#d8dee9;--hair:#e5e9f0;
--text:#4c566a;--muted:#8891a0;--strong:#2e3440;--acc:#5e81ac;--acc-soft:#e0e7ff;
--ok:#4a7a2e;--ok-soft:#e6f0e6;--warn:#d08770;--warn-soft:#fadfd2;--arch:#8891a0;--arch-soft:#eceff4;
--font:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;
--rs:6px;--ease:cubic-bezier(.23,1,.32,1)}
[data-theme="dark"]{
--bg:#2e3440;--surface:#3b4252;--surface2:#434c5e;--border:#4c566a;--hair:#3b4252;
--text:#d8dee9;--muted:#81a1c1;--strong:#eceff4;--acc:#88c0d0;--acc-soft:rgba(136,192,208,.18);
--ok:#a3be8c;--ok-soft:rgba(163,190,140,.16);--warn:#d08770;--warn-soft:rgba(208,135,112,.16);--arch:#81a1c1;--arch-soft:rgba(129,161,193,.16)}
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
.badge{font:600 11px var(--font);padding:2px 8px;border-radius:999px}
.badge.pub{color:var(--ok);background:var(--ok-soft)}
.badge.draft{color:var(--warn);background:var(--warn-soft)}
.badge.arch{color:var(--arch);background:var(--arch-soft)}
.tag{font-size:11.5px;color:var(--muted);background:var(--surface);border:1px solid var(--hair);border-radius:4px;padding:0 6px}
.icon-btn{border:1px solid transparent;background:transparent;color:var(--muted);width:32px;height:32px;border-radius:var(--rs);
display:grid;place-items:center;transition:color .1s;cursor:pointer}
.icon-btn:hover{color:var(--strong);background:var(--surface2)}
.icon-btn:disabled,.icon-btn.disabled{opacity:.35}
.wrap{height:calc(100vh - 52px);transition:height .35s var(--ease)}
#frame{width:100%;height:100%;border:0;background:#fff;transition:height .35s var(--ease)}
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
</style>`
}
