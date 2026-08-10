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
	Slug      string
	Title     string
	Status    string
	Workspace string
	Tags      []string
	RawURL    string // /page/<slug>/raw — the iframe src and pop-out target
	BackURL   string // library URL preserving the current filter
PrevURL   string   // "" when no previous page in the current set
	NextURL   string   // "" when no next page in the current set
	FeedbackOpen int   // open comments on this page (derived; drives the chrome badge)
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
	b.WriteString(changeTourMarkup())
	b.WriteString(pageViewScript(d))
	b.WriteString(commentPanelScript(d.Slug))
	b.WriteString(changeTourScript(d.Slug))
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
  const KEY='harbor_view_v2_'+slug; // v2: collapsed (container) is the default; ignores pre-v2 saved "full"
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
    <div class="cp-capture" id="cpCapture">Click an element in the page, or select its text, to anchor your comment — then type and post.</div>
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

// changeTourMarkup is the floating "what changed" walkthrough affordance: a
// chip shown only when the page records changes AND at least one has a matching
// data-cf-change marker in the DOM, plus a stepper card that tours each change.
// It lives entirely in the shell — the marker is read back from the agent's
// page, never injected into it.
func changeTourMarkup() string {
	return `<div class="cf">
  <button type="button" class="cf-btn" id="cfBtn" hidden aria-label="What changed" title="What changed"><i></i>What changed</button>
  <div class="cf-card" id="cfCard" role="dialog" aria-label="What changed" aria-hidden="true" hidden>
    <div class="cf-head"><span class="cf-step" id="cfStep"></span><button type="button" class="icon-btn cf-close" id="cfClose" aria-label="Close what-changed" title="Close">` + iconClose() + `</button></div>
    <div class="cf-title" id="cfTitle"></div>
    <div class="cf-desc" id="cfDesc"></div>
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
func changeTourScript(slug string) string {
	slugLit, _ := json.Marshal(slug)
	return `<script>
(function(){
  const slug=` + string(slugLit) + `;
  const frame=document.getElementById('frame');
  const btn=document.getElementById('cfBtn');
  const card=document.getElementById('cfCard');
  const stepEl=document.getElementById('cfStep');
  const titleEl=document.getElementById('cfTitle');
  const descEl=document.getElementById('cfDesc');
  const prevBtn=document.getElementById('cfPrev');
  const nextBtn=document.getElementById('cfNext');
  const doneBtn=document.getElementById('cfDone');
  const closeBtn=document.getElementById('cfClose');
  const LIST_URL='/api/pages/'+encodeURIComponent(slug)+'/changes';
  const REDUCED=matchMedia('(prefers-reduced-motion: reduce)').matches;
  let all=[], matched=[], idx=-1, iframeReady=false;
  let hlEl=null, styleTag=null;

  function locate(id){
    try{ const d=frame.contentDocument; if(!d) return null; return d.querySelector('[data-cf-change="'+String(id).replace(/"/g,'\\"')+'"]'); }
    catch(_){ return null; }
  }
  function injectStyle(){
    if(styleTag&&styleTag.parentNode) return;
    try{ const d=frame.contentDocument; if(!d||!d.head) return;
      const theme=document.documentElement.dataset.theme||'light';
      const accent=(theme==='dark')?'#88c0d0':'#5e81ac';
      const tint=(theme==='dark')?'rgba(136,192,208,.22)':'rgba(94,129,172,.15)';
      styleTag=d.createElement('style'); styleTag.textContent=
        '.cf-hl{outline:2px solid '+accent+'!important;outline-offset:1px!important;'+
        'box-shadow:0 0 0 2px '+tint+',inset 0 0 0 600px '+tint+'!important;border-radius:2px}';
      d.head.appendChild(styleTag);
    }catch(_){}
  }
  function clearHl(){ if(hlEl){ hlEl.classList.remove('cf-hl'); hlEl=null; } }
  function render(){
    if(idx<0||idx>=matched.length) return;
    const s=matched[idx];
    stepEl.textContent='Change '+(idx+1)+' of '+matched.length;
    titleEl.textContent=s.change.title||s.change.changeId;
    descEl.textContent=s.change.description||(s.el?('Marked element: '+s.change.changeId):'');
    clearHl();
    if(s.el){
      injectStyle(); hlEl=s.el; s.el.classList.add('cf-hl');
      try{ s.el.scrollIntoView(REDUCED?{block:'center'}:{behavior:'smooth',block:'center'}); }catch(_){ try{s.el.scrollIntoView();}catch(_2){} }
    }
    prevBtn.disabled=(idx===0);
    nextBtn.disabled=(idx===matched.length-1);
  }
  function go(i){ if(i<0||i>=matched.length) return; idx=i; render(); }
  function finish(){ clearHl(); card.hidden=true; card.setAttribute('aria-hidden','true'); btn.hidden=false; }
  function tryReady(){
    if(!iframeReady||!all.length) return;
    // Pair each change with its located marker (skip changes with no marker).
    matched=all.map(function(c){ return {change:c, el:locate(c.changeId)}; }).filter(function(p){ return p.el; });
    if(matched.length) btn.hidden=false;
    else { /* changes exist but no markers -> no tour, no error */ }
  }
  prevBtn.addEventListener('click',()=>go(idx-1));
  nextBtn.addEventListener('click',()=>go(idx+1));
  doneBtn.addEventListener('click',finish);
  closeBtn.addEventListener('click',finish);
  btn.addEventListener('click',function(){ btn.hidden=true; card.hidden=false; card.setAttribute('aria-hidden','false'); go(0); document.activeElement&&document.activeElement.blur&&document.activeElement.blur(); });
  frame.addEventListener('load',function(){ iframeReady=true; if(btn.hidden) tryReady(); });
  if(frame.contentDocument&&frame.contentDocument.readyState==='complete'){ iframeReady=true; }
  fetch(LIST_URL).then(r=>r.json()).then(function(list){ all=Array.isArray(list)?list:[]; tryReady(); }).catch(function(){});
})();</script>`
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
  const anchorEl=document.getElementById('cpAnchor');
  const quoteWrap=document.getElementById('cpQuoteWrap');
  const quoteEl=document.getElementById('cpQuote');
  const errEl=document.getElementById('cpError');
  const listEl=document.getElementById('cpList');
  const listHeadEl=document.getElementById('cpListHead');
  const clearBtn=document.getElementById('cpClear');
  const LIST_URL='/api/pages/'+encodeURIComponent(slug)+'/comments';

  let open=false;
  let doc=null;
  // state: what the comment is anchored to (captured from the iframe).
  let state={type:'general',anchor:'',quote:''};

  // Reset the compose area to a neutral whole-page draft. Called on Clear,
  // on submit, and when the panel closes so reopening always starts fresh.
  function resetCompose(){
    state={type:'general',anchor:'',quote:''};
    body.value='';
    typeSel.value='general';
    renderState();
    dropPickStyle();
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

  // ── Pick highlight: candidate + confirmed ─────────────────────────────
  // The pending target has two life-phases:
  //   • candidate  — while armed, hovering an element highlights it (cp-hover)
  //                  so the user sees exactly what a click will anchor to.
  //   • confirmed  — after a pick (or selection capture), the chosen element
  //                  stays highlighted (cp-anchored) to preserve context of
  //                  what the comment points at, until the target is changed
  //                  or cleared.
  // The styles are injected into the same-origin iframe (ephemeral, never
  // saved); the accent matches the shell theme.
  let pickStyle=null, hoverEl=null, anchoredEl=null;
  function accentTint(){
    const theme=document.documentElement.dataset.theme||'light';
    const accent=(theme==='dark')?'#88c0d0':'#5e81ac';
    const tint=(theme==='dark')?'rgba(136,192,208,.20)':'rgba(94,129,172,.14)';
    return {accent,tint};
  }
  function ensurePickStyle(){
    if(!doc||!doc.head) return;
    if(pickStyle&&pickStyle.parentNode) return;
    const {accent,tint}=accentTint();
    pickStyle=doc.createElement('style'); pickStyle.id='cp-pick-style';
    pickStyle.textContent=
      '.cp-hover{outline:2px solid '+accent+'!important;outline-offset:-2px!important;'+
        'box-shadow:inset 0 0 0 1000px '+tint+'!important;cursor:pointer!important}'+
      '.cp-anchored{outline:2px solid '+accent+'!important;outline-offset:1px!important;'+
        'box-shadow:0 0 0 2px '+tint+',inset 0 0 0 600px '+tint+'!important;border-radius:2px}';
    doc.head.appendChild(pickStyle);
  }
  function dropPickStyle(){
    if(pickStyle&&pickStyle.parentNode) pickStyle.parentNode.removeChild(pickStyle);
    pickStyle=null;
    clearHover();
  }
  // Element clicks are captured while the panel is open, letting the user pick
  // and — importantly — re-target: clicking a different element simply moves the
  // anchor (confirmed highlight follows). Hover shows the candidate so a click
  // is never a surprise. Clear detaches back to a whole-page comment; closing
  // the panel returns clicks to the page entirely.
  function canPickElement(){ return open; }
  // True when the user has a live text selection in the page — the signal that
  // a drag is quoting text, so a trailing click must NOT turn into an element
  // pick, and the element hover preview should stand aside.
  function hasTextSelection(){
    const win=frame.contentWindow, sel=win&&win.getSelection();
    return !!(sel && sel.toString().trim());
  }
  function clearHover(){ if(hoverEl){ hoverEl.classList.remove('cp-hover'); hoverEl=null; } }
  function clearAnchored(){ if(anchoredEl){ anchoredEl.classList.remove('cp-anchored'); anchoredEl=null; } }
  // Persist a marker on the element the current target points at (if it can be
  // resolved back in the page) — the confirmation context after a pick.
  function applyAnchorHighlight(){
    clearAnchored();
    if(!doc||!state.anchor) return;
    const el=doc.querySelector(state.anchor);
    if(el&&el.nodeType===1){ ensurePickStyle(); anchoredEl=el; el.classList.add('cp-anchored'); }
  }
  // Track the deepest element under the pointer while an element pick is live,
  // so hovering previews exactly what a click would anchor to.
  function trackHover(ev){
    if(!canPickElement()||hasTextSelection()){ clearHover(); return; }
    const el=doc.elementFromPoint(ev.clientX,ev.clientY);
    if(el===hoverEl) return;
    clearHover();
    hoverEl=el;
    if(hoverEl){ ensurePickStyle(); hoverEl.classList.add('cp-hover'); }
  }

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
    applyAnchorHighlight();
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
    doc.addEventListener('mousemove',trackHover,true);
    doc.addEventListener('click',(ev)=>{
      if(!canPickElement()) return;
      if(hasTextSelection()) return; // a drag-selection just became a quote; don't override it
      state={type:'element',anchor:cssPath(ev.target),quote:''};
      renderState(); setOpen(true);
    },true);
  }
  frame.addEventListener('load',wire);
  wire();
  // When the cursor leaves the page (into the panel, header, or beyond) stop
  // the candidate highlight — the pointer is no longer over a pickable element.
  frame.addEventListener('mouseleave',clearHover);

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
      .then(()=>{ resetCompose(); loadList(); bumpOpenFb(); })
      .catch(m=>{ errEl.textContent=m; errEl.classList.add('show'); })
      .finally(()=>{ submit.disabled=false; });
  });

  // Reflect a newly-posted open comment in the page chrome badge (created from
  // scratch if this was the page's first open comment).
  function bumpOpenFb(){
    var fb=document.getElementById('pvFb');
    var n=fb? (parseInt(fb.getAttribute('data-n'),10)||0) : 0;
    n+=1;
    if(fb){ fb.setAttribute('data-n',String(n)); fb.setAttribute('title',n+' open comment(s)'); fb.innerHTML='<i></i>'+n+' open'; }
    else {
      var pv=document.getElementById('pv'); if(!pv||!document.querySelector('.pv .chips')) return;
      var s=document.createElement('span'); s.id='pvFb'; s.className='pv-fb';
      s.setAttribute('data-n','1'); s.setAttribute('title','1 open comment'); s.innerHTML='<i></i>1 open';
      pv.insertBefore(s, document.querySelector('.pv .chips'));
    }
  }

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
.badge{padding:2px 8px;border-radius:999px;font:600 11px var(--font)}
.badge.pub{color:var(--ok);background:var(--ok-soft)}
.badge.draft{color:var(--warn);background:var(--warn-soft)}
.badge.arch{color:var(--arch);background:var(--arch-soft)}
.pv-fb{display:inline-flex;align-items:center;gap:5px;color:var(--warn);font:600 11px var(--font);
background:var(--warn-soft);border-radius:999px;padding:2px 8px;white-space:nowrap}
.pv-fb i{width:7px;height:7px;border-radius:999px;background:var(--warn)}
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
/* ── What-changed walkthrough ── */
.cf-btn{position:fixed;left:16px;bottom:16px;z-index:45;display:inline-flex;align-items:center;gap:7px;
border:1px solid var(--border);background:var(--surface);color:var(--text);border-radius:999px;
padding:8px 14px;font:600 12.5px var(--font);cursor:pointer;box-shadow:0 2px 10px rgba(0,0,0,.10);
transition:background .1s,color .1s,border-color .1s}
.cf-btn:hover{background:var(--surface2);border-color:var(--text)}
.cf-btn[hidden]{display:none}
.cf-btn i{width:8px;height:8px;border-radius:999px;background:var(--acc)}
.cf-card{position:fixed;z-index:46;left:50%;top:64px;transform:translateX(-50%);width:380px;max-width:92vw;
background:var(--surface);border:1px solid var(--border);border-radius:var(--r);
box-shadow:0 10px 34px rgba(0,0,0,.18);padding:14px 16px}
.cf-card[hidden]{display:none}
.cf-head{display:flex;align-items:center;gap:10px;margin-bottom:8px}
.cf-step{font:600 11px var(--font);color:var(--muted);text-transform:uppercase;letter-spacing:.05em}
.cf-close{margin-left:auto}
.cf-title{font-weight:600;color:var(--strong);font-size:14px}
.cf-desc{color:var(--muted);font-size:12.5px;line-height:1.55;margin-top:3px}
.cf-actions{display:flex;align-items:center;gap:8px;margin-top:12px}
.cf-pill{border:1px solid var(--border);background:var(--surface);color:var(--text);border-radius:var(--rs);
padding:7px 14px;font:600 12.5px var(--font);cursor:pointer}
.cf-pill:hover{background:var(--surface2)}
.cf-pill:disabled{opacity:.4;cursor:not-allowed}
.cf-pill.cf-primary{background:var(--acc);border-color:var(--acc);color:#fff}
.cf-pill.cf-primary:hover{filter:brightness(.96)}
@media(prefers-reduced-motion:reduce){.cf-btn,.cf-card{transition:none}}
@media(prefers-reduced-motion:reduce){.comment-panel{transition:opacity .18s ease;transform:none}
.comment-panel.open{transform:none}
}
@media(max-width:680px){.comment-panel{width:100%;max-width:100vw}}
</style>`
}
