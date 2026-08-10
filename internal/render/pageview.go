package render

import (
	"encoding/json"
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
	b.WriteString(commentPanelMarkup())
	b.WriteString(pageViewScript(d))
	b.WriteString(commentPanelScript(d.Slug))
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
	b.WriteString(`<button class="icon-btn" id="commentBtn" aria-haspopup="dialog" aria-expanded="false" aria-label="Comments" title="Comments">` + iconComment() + `</button>`)
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
func iconComment() string {
	return `<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>`
}

// commentPanelMarkup is the shell-side comment panel: a right-hand drawer where
// the human composes anchored feedback about the page. It lives entirely in the
// shell — the agent's HTML (in the iframe) is never injected into. Type/anchor/
// quote are captured client-side from the same-origin iframe and submitted via
// the JSON API; the page file itself is never touched.
func commentPanelMarkup() string {
	return `<div class="comment-panel" id="commentPanel" role="dialog" aria-modal="true" aria-labelledby="commentPanelTitle" aria-hidden="true">
  <div class="cp-head">
    <span class="cp-title" id="commentPanelTitle">Comment</span>
    <button class="icon-btn cp-close" id="commentClose" aria-label="Close comments" title="Close">` + iconClose() + `</button>
  </div>
  <div class="cp-body">
    <div class="cp-capture" id="cpCapture">Select text in the page to quote it, or pick an element to anchor your comment to.</div>
    <div class="cp-row">
      <button type="button" class="cp-pick" id="cpPick" aria-pressed="false">Pick element</button>
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
        <button type="submit" id="cpSubmit" class="cp-submit">Post comment</button>
      </div>
    </form>
    <div class="cp-list-head" id="cpListHead" hidden>Comments</div>
    <ol class="cp-list" id="cpList" aria-live="polite"></ol>
  </div>
</div>`
}

func iconClose() string {
	return `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12"/></svg>`
}
func iconClear() string {
	return `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12"/></svg>`
}

// commentPanelScript wires the comment panel: open/close with focus management,
// same-origin capture of text selections and element picks inside the iframe,
// POSTing submitted comments, and listing the page's existing comments. All
// feedback flows through the shell + JSON API — nothing is injected into the
// agent's page.
func commentPanelScript(slug string) string {
	slugLit, _ := json.Marshal(slug) // a JSON string literal == a valid JS string literal
	return `<script>(function(){
  const slug=` + string(slugLit) + `;
  const frame=document.getElementById('frame');
  const panel=document.getElementById('commentPanel');
  const btn=document.getElementById('commentBtn');
  const close=document.getElementById('commentClose');
  const form=document.getElementById('commentForm');
  const body=document.getElementById('cpBody');
  const typeSel=document.getElementById('cpType');
  const pickBtn=document.getElementById('cpPick');
  const anchorEl=document.getElementById('cpAnchor');
  const quoteWrap=document.getElementById('cpQuoteWrap');
  const quoteEl=document.getElementById('cpQuote');
  const errEl=document.getElementById('cpError');
  const listEl=document.getElementById('cpList');
  const listHeadEl=document.getElementById('cpListHead');
  const clearBtn=document.getElementById('cpClear');
  const LIST_URL='/api/pages/'+encodeURIComponent(slug)+'/comments';

  let open=false;
  let picking=false;
  let doc=null;
  // state: what the comment is anchored to (captured from the iframe).
  let state={type:'general',anchor:'',quote:''};

  // Reset the compose area to a neutral whole-page draft. Called on Clear,
  // on submit, and when the panel closes so reopening always starts fresh.
  function resetCompose(){
    state={type:'general',anchor:'',quote:''};
    body.value='';
    typeSel.value='general';
    picking=false;
    pickBtn.classList.remove('active');
    pickBtn.setAttribute('aria-pressed','false');
    errEl.classList.remove('show');
    renderState();
  }
  function setOpen(v){
    open=v;
    panel.classList.toggle('open',v);
    panel.setAttribute('aria-hidden',String(!v));
    btn.setAttribute('aria-expanded',String(v));
    document.body.classList.toggle('commenting',v);
    if(v){ showHeader(); body.focus(); loadList(); }
    else {
      if(document.activeElement&&document.activeElement.closest('#commentPanel')) btn.focus();
      resetCompose();
    }
  }
  function showHeader(){ const pv=document.getElementById('pv'); pv&&pv.classList.add('is-visible'); }
  btn.addEventListener('click',()=>setOpen(!open));
  close.addEventListener('click',()=>setOpen(false));
  document.addEventListener('keydown',(e)=>{ if(e.key==='Escape'&&open) setOpen(false); });

  // Update the preview UI from state. The Clear affordance is offered only
  // while a selection/element target is captured (anchor is set).
  function renderState(){
    typeSel.value=state.type;
    anchorEl.textContent=state.anchor||'whole page';
    if(state.quote){ quoteWrap.hidden=false; quoteEl.textContent=state.quote; }
    else quoteWrap.hidden=true;
    clearBtn.hidden=(state.anchor==='');
  }
  typeSel.addEventListener('change',()=>{ state.type=typeSel.value; });
  clearBtn.addEventListener('click',resetCompose);

  function cssPath(el){
    if(!el||el.nodeType!==1) return '';
    if(el.id) return '#'+CSS.escape(el.id);
    const parts=[]; let n=el;
    while(n&&n.nodeType===1&&n!==doc.body){
      if(n.id){ parts.unshift('#'+CSS.escape(n.id)); break; }
      let sel=n.tagName.toLowerCase();
      const parent=n.parentElement;
      if(parent){
        const same=[].filter.call(parent.children,ch=>ch.tagName===n.tagName);
        if(same.length>1) sel+=':nth-of-type('+(same.indexOf(n)+1)+')';
      }
      parts.unshift(sel); n=parent;
    }
    return parts.join(' > ');
  }

  // Same-origin: reach into the iframe document to capture selections and clicks.
  function wire(){
    try{ doc=frame.contentDocument; }catch(_){ doc=null; return; }
    if(!doc) return;
    // Forward Escape from inside the page: focus lives in the iframe after a
    // selection/pick, and its keydown never reaches the shell document.
    doc.addEventListener('keydown',(ev)=>{ if(ev.key==='Escape'&&open) setOpen(false); },true);
    doc.addEventListener('mouseup',(ev)=>{
      const win=frame.contentWindow, sel=win&&win.getSelection();
      const text=sel?sel.toString().trim():'';
      // The anchor of a text selection is usually a text node (use its
      // parent), but for an element selection it is the element itself.
      const an=(sel&&sel.anchorNode)||null;
      const el=an?(an.nodeType===1?an:an.parentElement):null;
      if(text){
        state={type:'selection',anchor:cssPath(el),quote:text};
        renderState(); setOpen(true);
      }
    },true);
    doc.addEventListener('click',(ev)=>{
      if(!picking) return;
      picking=false; pickBtn.classList.remove('active'); pickBtn.setAttribute('aria-pressed','false');
      errEl.classList.remove('show');
      state={type:'element',anchor:cssPath(ev.target),quote:''};
      renderState(); setOpen(true);
    },true);
  }
  frame.addEventListener('load',wire);
  wire();
  pickBtn.addEventListener('click',()=>{
    picking=!picking;
    pickBtn.classList.toggle('active',picking);
    pickBtn.setAttribute('aria-pressed',String(picking));
    if(picking){ errEl.textContent='Click an element in the page to anchor to it.'; errEl.classList.add('show'); }
    else errEl.classList.remove('show');
  });

  // Submit the pending comment.
  form.addEventListener('submit',(e)=>{
    e.preventDefault();
    const text=body.value.trim();
    if(!text) return;
    errEl.classList.remove('show');
    const submit=document.getElementById('cpSubmit');
    submit.disabled=true;
    fetch(LIST_URL,{method:'POST',headers:{'Content-Type':'application/json'},
      body:JSON.stringify({type:state.type,anchor:state.anchor,quote:state.quote,body:text})})
      .then(r=>{ if(!r.ok) return r.json().then(j=>Promise.reject(j&&j.error)).catch(()=>Promise.reject('could not save comment ('+r.status+')')); return r.json(); })
      .then(()=>{ resetCompose(); loadList(); })
      .catch(m=>{ errEl.textContent=m; errEl.classList.add('show'); })
      .finally(()=>{ submit.disabled=false; });
  });

  // List existing comments for the page.
  function escHTML(s){ return String(s||'').replace(/[&<>"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c])); }
  function loadList(){
    fetch(LIST_URL).then(r=>r.json()).then(items=>{
      listHeadEl.hidden = (items.length===0);
      listEl.innerHTML='';
      if(!items.length){ listEl.innerHTML='<li class="cp-empty">No comments yet.</li>'; return; }
      for(const c of items){
        const li=document.createElement('li');
        li.className='cp-item';
        const head=document.createElement('div'); head.className='cp-item-head';
        head.innerHTML='<span class="cp-item-type">'+escHTML(c.type)+'</span><span class="cp-item-status'+(c.status==='done'?' done':'')+'">'+escHTML(c.status)+'</span>';
        const b=document.createElement('div'); b.className='cp-item-body'; b.textContent=c.body;
        li.appendChild(head); li.appendChild(b);
        if(c.quote){ const q=document.createElement('div'); q.className='cp-item-quote'; q.textContent=c.quote; li.appendChild(q); }
        listEl.appendChild(li);
      }
    }).catch(()=>{});
  }
})();</script>`
}

func pageViewCSS() string {
	return `<style>
:root{--bg:#eceff4;--surface:#fff;--surface2:#f6f8fb;--border:#d8dee9;--hair:#e5e9f0;
--text:#4c566a;--muted:#8891a0;--strong:#2e3440;--acc:#5e81ac;--acc-soft:#e0e7ff;
--ok:#4a7a2e;--ok-soft:#e6f0e6;--warn:#d08770;--warn-soft:#fadfd2;--arch:#8891a0;--arch-soft:#eceff4;
--font:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;
--rs:6px;--panelw:340px;--ease:cubic-bezier(.23,1,.32,1)}
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
.wrap{height:calc(100vh - 52px);transition:height .35s var(--ease),margin-right .3s var(--ease)}
#frame{width:100%;height:100%;border:0;background:#fff;transition:height .35s var(--ease)}
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
.cp-form{display:flex;flex-direction:column;gap:8px;margin-top:18px}
.cp-capture{font-size:12.5px;color:var(--muted);line-height:1.6;background:var(--surface2);
border:1px dashed var(--border);border-radius:var(--rs);padding:12px 14px;margin:2px 0 4px}
.cp-row{display:flex;align-items:center;gap:10px;margin-top:14px}
.cp-pick{border:1px solid var(--border);background:var(--surface);color:var(--text);border-radius:var(--rs);
padding:7px 11px;font:500 12.5px var(--font);cursor:pointer}
.cp-pick:hover{background:var(--surface2)}
.cp-pick.active{color:var(--acc);border-color:var(--acc);background:var(--acc-soft)}
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
.cp-body textarea:focus-visible,.cp-pick:focus-visible,.cp-close:focus-visible,.cp-submit:focus-visible,
.cp-row select:focus-visible{outline:2px solid var(--acc);outline-offset:1px}
.cp-actions{display:flex;justify-content:flex-end;padding-top:2px}
.cp-submit{border:0;border-radius:var(--rs);background:var(--acc);color:#fff;padding:9px 18px;
font:600 13px var(--font);cursor:pointer}
.cp-submit:hover{filter:brightness(.96)}
.cp-submit:disabled{opacity:.5;cursor:not-allowed}
.cp-error{display:none;font-size:12.5px;color:var(--warn);background:var(--warn-soft);border-radius:var(--rs);padding:9px 12px}
.cp-error.show{display:block}
.cp-list-head{margin:28px 0 12px;font:600 12px var(--font);color:var(--muted);text-transform:uppercase;letter-spacing:.05em}
.cp-list{list-style:none;margin:0;padding:0;display:flex;flex-direction:column;gap:14px}
.cp-item{border:1px solid var(--border);border-radius:var(--rs);padding:13px 15px;background:var(--surface2)}
.cp-item-head{display:flex;align-items:center;gap:8px;margin-bottom:10px}
.cp-item-type{font:600 10px var(--font);color:var(--acc);background:var(--acc-soft);border-radius:4px;padding:2px 6px;text-transform:uppercase}
.cp-item-status{font:500 10px var(--font);color:var(--muted)}
.cp-item-status.done{color:var(--ok)}
.cp-item-body{font-size:13px;color:var(--text);line-height:1.6}
.cp-item-quote{font-size:12px;font-style:italic;color:var(--muted);line-height:1.55;border-left:2px solid var(--border);padding:1px 0 1px 9px;margin-top:10px}
.cp-empty{font-size:12.5px;color:var(--muted)}
@media(prefers-reduced-motion:reduce){
.comment-panel{transition:opacity .18s ease;transform:none}
.comment-panel.open{transform:none}
}
@media(max-width:680px){.comment-panel{width:100%;max-width:100vw}}
</style>`
}
