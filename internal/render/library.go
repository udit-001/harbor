package render

import (
	"encoding/json"
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

// PageRow is one row in the library list. JSON tags drive the client-side
// filter API (/api/pages); the HTML renderer uses the struct directly.
type PageRow struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Desc         string   `json:"desc"`
	Workspace    string   `json:"workspace"`
	Format       string   `json:"format"`
	Status       string   `json:"status"`
	Tags         []string `json:"tags"`
	Updated      string   `json:"updated"`
	FeedbackOpen int      `json:"feedbackOpen"`
}

func e(tagName string) string { return esc(tagName) }

// escJS renders a Go string as a JS string-literal body (JSON encoding covers
// quotes and newlines; < escaped so "</script>" can never terminate the tag).
func escJS(s string) string {
	b, _ := json.Marshal(s)
	return strings.ReplaceAll(string(b), "<", "\\u003c")
}

// Library renders the full Harbor library page (its own shell: topbar + sidebar
// of workspaces/tags + status segment + hairline list). Self-contained — inline
// CSS + JS — so it does not depend on the Harbor lesson shell.
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
<title>` + esc(title) + `</title>` +
		`<script>(function(){try{var t=localStorage.getItem('harbor_theme');if(!t||t==='system'){t=window.matchMedia('(prefers-color-scheme:dark)').matches?'dark':'light'}document.documentElement.dataset.theme=t}catch(e){document.documentElement.dataset.theme='light'}})();</script>` +
		libraryCSS() + `</head><body>`
}

func libraryShell(data LibraryData) string {
	var b strings.Builder
	b.WriteString(`<div class="app">`)
	b.WriteString(librarySidebar(data))
	b.WriteString(`<main class="main">`)
	b.WriteString(libraryTopbar(data))
	b.WriteString(libraryContent(data))
	b.WriteString(`</main></div>`)
	return b.String()
}

func librarySidebar(data LibraryData) string {
	var b strings.Builder
	b.WriteString(`<aside class="side"><div class="brand">` + logo() + ` harbor</div><nav class="nav">`)

	// All pages
	allActive := data.Filter.Workspace == "" && data.Filter.Tag == ""
	addLink := func(href, label string, count int, active bool, icon string, data string) {
		cls := "link"
		if active {
			cls += " active"
		}
		fmt.Fprintf(&b, `<a class="%s" href="%s" %s><span class="ic">%s</span>%s<span class="cnt">%d</span></a>`,
			cls, esc(href), data, icon, esc(label), count)
	}

	allHref := "/?status=" + esc(data.Filter.Status)
	if data.Filter.Q != "" {
		allHref += "&q=" + esc(data.Filter.Q)
	}
	addLink(allHref, "All pages", data.Total, allActive, iconList(), `data-all="1"`)

	if len(data.Workspaces) > 0 {
		b.WriteString(`<div class="secwrap"><div class="sec" data-sec="ws" role="button" tabindex="0" aria-expanded="true">Workspaces` + chevron() + `</div><div class="secitems"><div>`)
		for _, w := range data.Workspaces {
			active := data.Filter.Workspace == w.Name
			addLink("/?workspace="+e(w.Name)+"&status="+e(data.Filter.Status), w.Name, w.Count, active, iconFolder(), `data-ws="`+e(w.Name)+`"`)
		}
		b.WriteString(`</div></div></div>`)
	}
	if len(data.Tags) > 0 {
		b.WriteString(`<div class="secwrap"><div class="sec" data-sec="tags" role="button" tabindex="0" aria-expanded="true">Tags` + chevron() + `</div><div class="secitems"><div>`)
		for _, t := range data.Tags {
			active := data.Filter.Tag == t.Name
			addLink("/?tag="+e(t.Name)+"&status="+e(data.Filter.Status), t.Name, t.Count, active, iconTag(), `data-tag="`+e(t.Name)+`"`)
		}
		b.WriteString(`</div></div></div>`)
	}

	b.WriteString(`</nav></aside>`)
	return b.String()
}

func libraryTopbar(data LibraryData) string {
	searchVal := ""
	if data.Filter.Q != "" {
		searchVal = ` value="` + e(data.Filter.Q) + `"`
	}
	// Home is the root of the shell, so it carries no breadcrumb — the page
	// title ("All pages") is the single location label and lives here in the
	// topbar (with the count beside it), not duplicated in the body. The status
	// filter is a pill switch next to the title; the search box rides right.
	return `<div class="topbar"><h1 class="title">All pages<span class="count">` + fmt.Sprintf("%d", data.Total) + ` pages</span></h1>` + libraryStatusSeg(data.Filter.Status) + `<div class="search"><input id="q" type="search" placeholder="Search pages…" aria-label="Search pages"` + searchVal + `></div><button id="theme-toggle" type="button" class="theme-toggle" title="Toggle theme" aria-label="Toggle theme">
<svg class="moon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
<svg class="sun" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/></svg>
</button></div>`
}

func libraryContent(data LibraryData) string {
	var b strings.Builder
	b.WriteString(`<div class="body">`)
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
	b.WriteString(`<div class="status-seg"><span class="seg-thumb" aria-hidden="true"></span>`)
	for _, it := range items {
		cls := ""
		if active == it.val {
			cls = " class=\"active\""
		}
		if it.val == "" {
			fmt.Fprintf(&b, `<a%s href="/" data-status="">%s</a>`, cls, it.label)
		} else {
			fmt.Fprintf(&b, `<a%s href="/?status=%s" data-status="%s">%s</a>`, cls, it.val, it.val, it.label)
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
		b.WriteString(`<div class="right"><span class="updated">` + esc(r.Updated) + `</span><span class="badge fmt">` + esc(r.Format) + `</span><span class="badge ` + statusClass(r.Status) + `">` + esc(r.Status) + `</span></div>`)
		b.WriteString(`</a>`)
	}
	return b.String()
}

// chevron is the collapsible-section affordance; CSS rotates it closed/open.
func chevron() string {
	return `<svg class="chev" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M6 9l6 6 6-6"/></svg>`
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

// emptyLibraryCopy is the single source of truth for the empty-library state —
// the server-rendered first paint and the client-side filter swap must never
// drift apart (they did: the copy still said "HTML pages" after artifacts
// shipped). Rendered as HTML; the JS side embeds it verbatim via %s.
const emptyLibraryCopy = `Harbor gathers the artifacts your agent builds — dashboards, reports,
diagrams, drawings. When the agent makes something worth keeping, it lands here
automatically for you to browse and review — nothing for you to do.`

func libraryEmpty() string {
	return `<div class="empty"><h2>No pages yet</h2>
<p>` + emptyLibraryCopy + `</p></div>`
}

func libraryScript(data LibraryData) string {
	// Client-side filtering over one fetched dataset. The server still renders
	// the first paint (correct for the current URL), then this script takes
	// over: it fetches the full page list once and narrows it in the browser for
	// instant feedback — no reloads. Filters stay in the URL via history.pushState,
	// so back/forward and deep links keep working (popstate re-applies them).
	return `<script>
(function(){
  var input=document.getElementById('q');
  var listEl=document.getElementById('list');
  var countEl=document.querySelector('.topbar .title .count');
  var ready=false;
  var ALL=[];

  function params(){ return new URLSearchParams(location.search); }
  var H=window.__harbor||{};
  var state={ q:params().get('q')||'', status:params().get('status')||'', workspace:params().get('workspace')||'', tag:params().get('tag')||'' };

  function esc(s){ return String(s==null?'':s).replace(/[&<>"']/g,function(c){return{'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c];}); }
  function statusClass(s){ s=s||'draft'; if(s==='published')return 'pub'; if(s==='archived')return 'arch'; return 'draft'; }
  function qs(){ var p=new URLSearchParams(); if(state.status)p.set('status',state.status); if(state.workspace)p.set('workspace',state.workspace); if(state.tag)p.set('tag',state.tag); if(state.q)p.set('q',state.q); return p.toString(); }
  function push(){ history.pushState({}, '', '/'+(qs()?'?'+qs():'')); }

  function filtered(){
    var q=state.q.trim().toLowerCase();
    return ALL.filter(function(p){
      if(state.status && p.status!==state.status) return false;
      if(state.workspace && p.workspace!==state.workspace) return false;
      if(state.tag && (!p.tags||p.tags.indexOf(state.tag)===-1)) return false;
      if(!q) return true;
      var hay=(p.title+' '+(p.tags||[]).join(' ')+' '+p.workspace+(p.desc?' '+p.desc:'')).toLowerCase();
      return hay.indexOf(q)!==-1;
    });
  }

  function rowHTML(r){
    return '<a class="row" href="/page/'+esc(r.slug)+(qs()?'?'+esc(qs()):'')+'">'
      +'<div class="grow"><div class="t"><span class="name">'+esc(r.title)+'</span>'
      +(r.feedbackOpen>0?'<span class="fb"><i></i>'+r.feedbackOpen+' open</span>':'')
      +'</div>'
      +(r.desc?'<div class="desc">'+esc(r.desc)+'</div>':'')
      +'<div class="meta"><span class="tag">'+esc(r.workspace)+'</span>'
      +(r.tags||[]).map(function(t){return '<span class="tag">'+esc(t)+'</span>';}).join('')
      +'</div></div>'
      +'<div class="right"><span class="updated">'+esc(r.updated)+'</span>'
      +'<span class="badge fmt">'+esc(r.format||'html')+'</span>'
      +'<span class="badge '+statusClass(r.status)+'">'+esc(r.status)+'</span></div></a>';
  }
  function emptyHTML(noPages){
    return noPages
      ? '<div class="empty"><h2>No pages yet</h2><p>'+H.emptyCopy+'</p></div>'
      : '<div class="empty"><h2>No pages match these filters</h2><p>Try a different status, workspace, tag, or query.</p><button type="button" class="clear" id="clear-filters">Clear all filters</button></div>';
  }

  function render(){
    if(!ready){ return; }
    var rows=filtered();
    countEl.textContent=rows.length+' pages';
    if(!rows.length){ listEl.innerHTML=emptyHTML(ALL.length===0); bindClear(); }
    else { listEl.innerHTML=rows.map(rowHTML).join(''); }
    // Brief fade-in so filter/search/clear don't teleport the whole list.
    listEl.classList.remove('swap'); void listEl.offsetWidth; listEl.classList.add('swap');
  }
  function bindClear(){
    var b=document.getElementById('clear-filters');
    if(b) b.addEventListener('click',function(){ state.q='';state.status='';state.workspace='';state.tag=''; input.value=''; render(); updateActive(); push(); });
  }

  function updateActive(){
    var all=document.querySelectorAll('.status-seg a');
    for(var i=0;i<all.length;i++) all[i].classList.toggle('active',(all[i].getAttribute('data-status')||'')===state.status);
    var links=document.querySelectorAll('.nav .link');
    for(var j=0;j<links.length;j++){
      var a=links[j];
      if(a.hasAttribute('data-all')) a.classList.toggle('active',!state.workspace&&!state.tag);
      else if(a.hasAttribute('data-ws')) a.classList.toggle('active',a.getAttribute('data-ws')===state.workspace&&!state.tag);
      else if(a.hasAttribute('data-tag')) a.classList.toggle('active',a.getAttribute('data-tag')===state.tag&&!state.workspace);
    }
  }

  function apply(){
    input.value=state.q;
    render();
    updateActive();
    moveThumb();
  }
  // Sliding active thumb for the status pills: the selected pill's highlight
  // glides to the active tab via a composited transform (interruptible, no
  // width/layout animation). Enabled after the first paint so the initial
  // position doesn't animate on load.
  var segEl=document.querySelector('.status-seg');
  var thumb=segEl&&segEl.querySelector('.seg-thumb');
  var segPad=3;
  function moveThumb(){
    if(!segEl||!thumb) return;
    var a=segEl.querySelector('a.active')||segEl.querySelector('a');
    if(!a) return;
    thumb.style.width=a.offsetWidth+'px';
    thumb.style.transform='translateX('+(a.offsetLeft-segPad)+'px)';
  }
  if(segEl){ moveThumb(); segEl.classList.add('thumb-ready'); }
  window.addEventListener('resize',function(){ if(segEl) moveThumb(); });
  // Collapsible sidebar sections (pattern from learn-tool): click label to
  // fold/unfold; state persists per browser; a section containing the ACTIVE
  // filter always expands — hiding where you are is worse than any tidiness.
  document.querySelectorAll('.sec[data-sec]').forEach(function(sec){
    var wrap=sec.closest('.secwrap');
    function setCollapsed(v){ wrap.classList.toggle('collapsed',v); sec.setAttribute('aria-expanded',String(!v)); }
    var saved=null; try{ saved=localStorage.getItem('harbor_sec_'+sec.dataset.sec); }catch(_){}
    if(saved==='closed' && !wrap.querySelector('.link.active')) setCollapsed(true);
    function toggle(){
      setCollapsed(!wrap.classList.contains('collapsed'));
      try{ localStorage.setItem('harbor_sec_'+sec.dataset.sec, wrap.classList.contains('collapsed')?'closed':'open'); }catch(_){}
    }
    sec.addEventListener('click',toggle);
    sec.addEventListener('keydown',function(ev){ if(ev.key==='Enter'||ev.key===' '){ ev.preventDefault(); toggle(); } });
  });

  // Status pills (topbar switch) — client-side.
  var pills=document.querySelectorAll('.status-seg a');
  for(var i=0;i<pills.length;i++){
    pills[i].addEventListener('click',function(e){
      if(!ready) return; // degrade to native navigation until data arrives
      e.preventDefault();
      state.status=this.getAttribute('data-status')||'';
      apply(); push();
    });
  }
  // Sidebar workspace / tag links — client-side.
  var links=document.querySelectorAll('.nav .link');
  for(var j=0;j<links.length;j++){
    links[j].addEventListener('click',function(e){
      if(!ready) return;
      e.preventDefault();
      if(this.hasAttribute('data-ws')){ state.workspace=this.getAttribute('data-ws'); state.tag=''; }
      else if(this.hasAttribute('data-tag')){ state.tag=this.getAttribute('data-tag'); state.workspace=''; }
      else { state.workspace=''; state.tag=''; }
      apply(); push();
    });
  }
  // Search — live, client-side.
  var debounce;
  input.addEventListener('input',function(){
    clearTimeout(debounce);
    debounce=setTimeout(function(){ state.q=input.value; apply(); push(); },120);
  });
  // Back / forward re-applies the prior URL (no reload).
  window.addEventListener('popstate',function(){
    var p=new URLSearchParams(location.search);
    state.q=p.get('q')||''; state.status=p.get('status')||''; state.workspace=p.get('workspace')||''; state.tag=p.get('tag')||'';
    apply();
  });

  // Fetch the full set once, then take over rendering.
  fetch('/api/pages').then(function(r){return r.json();}).then(function(list){
    ALL=Array.isArray(list)?list:[];
    ready=true;
    apply();
  }).catch(function(){ /* keep the server-rendered list; links still navigate */ });

  // Theme toggle (mirrors harbor): flips [data-theme] on <html> and persists
  // under harbor_theme so the choice is shared with the harbor frame pages.
  function setTheme(t){ document.documentElement.dataset.theme=t; try{localStorage.setItem('harbor_theme',t);}catch(e){} }
  var themeBtn=document.getElementById('theme-toggle');
  if(themeBtn) themeBtn.addEventListener('click',function(){
    // Crossfade the palette flip instead of snapping (occasional action, big
    // visual delta). Class is removed on the next frame after the transition;
    // prefers-reduced-motion users keep the instant swap (global rule).
    document.body.classList.add('theme-swap');
    setTheme(document.documentElement.dataset.theme==='dark'?'light':'dark');
    setTimeout(function(){ document.body.classList.remove('theme-swap'); },220);
  });

  // Live-sync hook: refetch the full dataset and re-apply the current filters.
  window.__harborReload=function(){ fetch('/api/pages').then(function(r){return r.json();}).then(function(list){ ALL=Array.isArray(list)?list:[]; ready=true; apply(); }).catch(function(){}); };
})();
</script>` + `
<script>window.__harbor={live:{topic:'home',mode:'library'},emptyCopy:` + escJS(emptyLibraryCopy) + `};</script>` + `
<script>` + harborLiveSyncJS + `</script>` + `</body></html>`
}

func logo() string {
	return `<svg width="22" height="22" viewBox="0 0 24 24" fill="none"><path d="M7 9l5 5 5-5M7 14l5 5 5-5" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>`
}

func libraryCSS() string {
	return `<style>` + ThemeTokens() + `
:root{--chip:rgba(0,0,0,.05);--r:8px;--rs:6px}
[data-theme="dark"]{--chip:rgba(255,255,255,.09)}
*{box-sizing:border-box}
body{margin:0;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:var(--bg);
color:var(--text);font-size:15px;line-height:1.6;-webkit-font-smoothing:antialiased}
a{text-decoration:none;color:inherit}
.app{display:grid;grid-template-columns:232px 1fr;height:100vh;overflow:hidden}
.side{background:var(--surface);border-right:1px solid var(--border);padding:16px 12px;height:100vh;overflow:hidden;display:flex;flex-direction:column}
.brand{display:flex;align-items:center;gap:9px;color:var(--strong);font-weight:700;font-size:15px;padding:4px 8px 16px}
.brand svg{color:var(--acc)}
.nav{display:flex;flex-direction:column;gap:2px;flex:1;min-height:0;overflow-y:auto}
.sec{display:flex;align-items:center;gap:5px;padding:14px 8px 6px;font-size:11px;letter-spacing:.04em;color:var(--muted);font-weight:600;cursor:pointer;user-select:none;border-radius:var(--rs);transition:color .1s}
.sec:hover{color:var(--text)}
.sec .chev{width:11px;height:11px;flex:none;transition:transform .18s var(--ease);opacity:.8}
.secwrap:not(.collapsed) .sec .chev{transform:rotate(180deg)}
/* Collapse via grid-template-rows: animates to auto height with zero JS measuring */
.secitems{display:grid;grid-template-rows:1fr;transition:grid-template-rows .22s var(--ease)}
.secwrap.collapsed .secitems{grid-template-rows:0fr}
.secitems>div{min-height:0;overflow:hidden}
@media (prefers-reduced-motion:reduce){.secitems,.sec .chev{transition:none!important}}
/* Custom scrollbars: slim, themed, invisible track */
.nav{scrollbar-width:thin;scrollbar-color:var(--border) transparent}
.nav::-webkit-scrollbar{width:8px}
.nav::-webkit-scrollbar-track{background:transparent}
.nav::-webkit-scrollbar-thumb{background:var(--border);border-radius:4px}
.nav::-webkit-scrollbar-thumb:hover{background:var(--muted)}
.link{display:flex;align-items:center;gap:9px;padding:7px 8px;border-radius:var(--rs);color:var(--text);font-size:14px;transition:background .1s,color .1s,transform 120ms var(--ease)}
.link:hover{background:var(--surface2)}
.link.active{background:var(--acc-soft);color:var(--acc);font-weight:600}
.link:active{transform:scale(.97)}
.link .ic{width:16px;height:16px;color:var(--muted);flex:none;display:inline-flex;align-items:center;justify-content:center}
.link .cnt{margin-left:auto;font-size:11px;color:var(--muted);background:var(--chip);border-radius:999px;padding:0 7px}
.main{min-width:0;height:100vh;display:flex;flex-direction:column;overflow:hidden}
.topbar{flex:none;display:flex;align-items:center;gap:16px;height:52px;padding:0 24px;background:var(--surface);border-bottom:1px solid var(--border)}
.topbar .title{display:flex;align-items:baseline;gap:8px;font-size:15px;font-weight:700;color:var(--strong);white-space:nowrap}
.topbar .title .count{color:var(--muted);font:400 12.5px var(--font)}
.search{flex:1;max-width:460px;margin-left:auto}
.search input{width:100%;padding:5px 12px;border:1px solid var(--border);border-radius:var(--rs);
background:var(--bg);font:400 14px var(--font);color:var(--text);outline:none;transition:border-color .1s}
.search input:focus{border-color:var(--acc)}
:focus-visible{outline:2px solid var(--acc);outline-offset:1px}
.link:focus-visible,.row:focus-visible,.status-seg a:focus-visible,.clear:focus-visible,.theme-toggle:focus-visible{border-radius:var(--rs)}
.theme-toggle{flex:none;position:relative;width:32px;height:32px;border:1px solid var(--border);
background:var(--surface);color:var(--muted);border-radius:var(--rs);cursor:pointer;transition:color .12s,transform 120ms var(--ease)}
.theme-toggle:hover{color:var(--strong)}
.theme-toggle:active{transform:scale(.97)}
.theme-toggle svg{position:absolute;inset:0;margin:auto;pointer-events:none;transition:opacity .15s ease-out,transform .15s ease-out}
[data-theme="dark"] .theme-toggle .sun{opacity:1;transform:rotate(0)}
[data-theme="light"] .theme-toggle .moon{opacity:1;transform:rotate(0)}
[data-theme="dark"] .theme-toggle .moon{opacity:0;transform:scale(.4) rotate(-90deg)}
[data-theme="light"] .theme-toggle .sun{opacity:0;transform:scale(.4) rotate(90deg)}
.body{flex:1;min-height:0;overflow-y:auto;padding:22px 24px 60px}
.status-seg{position:relative;flex:none;display:flex;align-items:center;gap:2px;margin-left:24px;background:var(--bg);
border:1px solid var(--border);border-radius:999px;padding:3px}
.seg-thumb{position:absolute;top:3px;bottom:3px;left:3px;width:0;border-radius:999px;background:var(--surface);
box-shadow:0 1px 2px rgba(0,0,0,.08);will-change:transform}
.status-seg.thumb-ready .seg-thumb{transition:transform .18s var(--ease-in-out)}
.status-seg a{position:relative;z-index:1;font:600 12.5px var(--font);color:var(--muted);padding:5px 14px;border-radius:999px;white-space:nowrap;transition:color .12s,transform 120ms var(--ease)}
.status-seg a:hover{color:var(--strong)}
.status-seg a:active{transform:scale(.97)}
.status-seg a.active{color:var(--acc)}
.list{margin-top:4px}
@keyframes listfade{from{opacity:0}to{opacity:1}}
.list.swap{animation:listfade 130ms var(--ease)}
.row{display:flex;align-items:center;gap:14px;padding:14px 12px;border-bottom:1px solid var(--hair);border-radius:var(--rs);cursor:pointer;transition:background .1s,transform 120ms var(--ease)}
.row:hover{background:var(--surface2)}
.row:active{transform:scale(.985)}
.row .grow{min-width:0;flex:1}
.row .t{display:flex;align-items:center;gap:9px}
.row .name{font-weight:600;color:var(--strong);font-size:14.5px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.row .desc{color:var(--muted);font-size:13px;margin-top:2px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.row .meta{display:flex;align-items:center;gap:7px;margin-top:5px;flex-wrap:wrap}
.tag{font-size:11.5px;color:var(--muted);background:var(--surface);border:1px solid var(--hair);border-radius:4px;padding:0 6px}
.row .right{display:flex;align-items:center;gap:12px;flex:none}
.badge{font:600 11px var(--font);padding:2px 8px;border-radius:999px}
.badge.fmt{color:var(--muted);background:var(--surface2);border:1px solid var(--hair)}
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
.clear{display:inline-block;font:600 12.5px var(--font);color:var(--acc);background:var(--surface);
border:1px solid var(--border);border-radius:var(--rs);padding:7px 14px;cursor:pointer;transition:color .1s,transform 120ms var(--ease)}
.clear:hover{color:var(--strong)}
.clear:active{transform:scale(.97)}
body.theme-swap, body.theme-swap *{transition:background-color .18s var(--ease),color .18s var(--ease),border-color .18s var(--ease)!important}
@media(prefers-reduced-motion:reduce){*{transition:none}.list.swap{animation:none}.link:active,.row:active,.status-seg a:active,.theme-toggle:active,.clear:active{transform:none}}
</style>`
}
