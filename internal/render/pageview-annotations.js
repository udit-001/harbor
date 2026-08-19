(function(){
  const slug = (window.__harbor && window.__harbor.slug) || '';
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
  const listPane=document.getElementById('cpListPane');
  const composePane=document.getElementById('cpComposePane');
  const newBtn=document.getElementById('cpNew');
  const cancelBtn=document.getElementById('cpCancel');
  const chips=document.querySelectorAll('.cp-chip');
  const titleEl=document.getElementById('commentPanelTitle');
  const REDUCED=window.matchMedia&&window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  // Artifact formats without a DOM (pdf, image, svg, excalidraw, markdown,
  // text) can only anchor whole-page comments: there is no element or text
  // selection inside the iframe to path against. HTML keeps all three kinds.
  const DOM_FORMAT=(window.__harbor&&window.__harbor.format)==='html';
  if(!DOM_FORMAT){ typeSel.hidden=true; typeSel.value='general'; }
  // Transient status toast (bottom of the shell). Live region so assistive tech hears it.
  function toast(msg){
    var t=document.createElement('div'); t.className='pv-toast'; t.setAttribute('role','status'); t.textContent=msg;
    document.body.appendChild(t);
    requestAnimationFrame(function(){ t.classList.add('show'); });
    if(toast._timers){ for(var i=0;i<toast._timers.length;i++) clearTimeout(toast._timers[i]); } else { toast._timers=[]; }
    toast._timers.push(setTimeout(function(){ t.classList.remove('show'); }, 2200));
    toast._timers.push(setTimeout(function(){ if(t.parentNode) t.parentNode.removeChild(t); }, 2350));
  }
  // List-first view (HARB-33): the drawer opens to the history; compose is
  // opt-in. Closed comments are hidden by default (filter=open).
  let filter='open';
  let view='list';
  function showView(v){
    view=v;
    listPane.hidden=(v!=='list');
    composePane.hidden=(v!=='compose');
    listHeadEl.hidden=(v!=='list');
    titleEl.textContent=(v==='compose') ? 'New comment' : 'Comments';
  }
  function refreshChips(){ chips.forEach(function(ch){ ch.classList.toggle('active', ch.dataset.filter===filter); }); moveChipThumb(); }
  // Sliding active thumb for the Open/Done/All filter chips (mirrors the home status pills).
  var filterSeg=document.querySelector('.cp-filters');
  var chipThumb=filterSeg&&filterSeg.querySelector('.cp-thumb');
  function moveChipThumb(){
    if(!filterSeg||!chipThumb) return;
    var c=filterSeg.querySelector('.cp-chip.active')||filterSeg.querySelector('.cp-chip');
    if(!c) return;
    chipThumb.style.width=c.offsetWidth+'px';
    chipThumb.style.transform='translateX('+(c.offsetLeft-3)+'px)';
  }
  if(filterSeg){ moveChipThumb(); filterSeg.classList.add('thumb-ready'); }
  window.addEventListener('resize',function(){ moveChipThumb(); });
  chips.forEach(function(ch){ ch.addEventListener('click',function(){ filter=ch.dataset.filter; refreshChips(); loadList(); }); });
  newBtn.addEventListener('click',function(){ resetCompose(); showView('compose'); body.focus(); });
  cancelBtn.addEventListener('click',function(){ showView('list'); loadList(); });
  refreshChips();

  // ── Inline compose box (HARB-32) ────────────────────────────────────
  // Anchored comments compose in a docked shell box over the anchor (drawer
  // stays closed; whole-doc compose stays in the drawer). Stays put while
  // typing; Cancel/Escape/outside-click discards with no draft.
  const inlineBox=document.getElementById('cpInline');
  const inlineBody=document.getElementById('cpInlineBody');
  const inlineErr=document.getElementById('cpInlineError');
  const inlineWhere=document.getElementById('cpInlineWhere');
  const inlinePost=document.getElementById('cpInlinePost');
  const inlineCancel=document.getElementById('cpInlineCancel');
  const inlineClose=document.getElementById('cpInlineClose');
  const inlineTitle=document.getElementById('cpInlineTitle');
  let replyParent=0;   // comment id this draft replies to (reply thread, HARB-20/34)
  let editID=0;        // when >0 the inline box edits this open comment (PATCH)
  function mapStateAnchor(){ return {kind:state.type==='selection'?'text':(state.type==='element'?'element':'document'), path:state.anchor, quote:state.quote}; }
  function inlineResolved(){ return window.harborResolveAnchor ? window.harborResolveAnchor(mapStateAnchor()) : null; }
  function positionInlineFor(el){
    if(!el) return;
    var fr=frame.getBoundingClientRect(), er=el.getBoundingClientRect();
    var cw=inlineBox.offsetWidth||288, ch=inlineBox.offsetHeight||212, margin=12;
    var left=Math.max(8,Math.min(fr.left+er.left+er.width/2-cw/2, window.innerWidth-cw-8));
    inlineBox.style.transform=''; inlineBox.style.top=''; inlineBox.style.left='';
    var ty=fr.top+er.top;
    if(ty>ch+margin){ inlineBox.style.top=(ty-ch-margin)+'px'; } else { inlineBox.style.top=(ty+er.height+margin)+'px'; }
    inlineBox.style.left=left+'px';
  }
  function positionInline(){
    var r=inlineResolved(); if(!r||!r.el) return false;
    positionInlineFor(r.el); return true;
  }
  function showInline(){
    if(!state.anchor && state.type!=='element') return; // needs an anchor
    if(window.harborModes) window.harborModes.set('comment');
    hideAfford();
    inlineErr.hidden=true; inlineErr.textContent='';
    inlineWhere.textContent=state.quote||state.anchor||'whole page';
    clearTimeout(hideInline._t);
    inlineBox.hidden=false; inlineBox.setAttribute('aria-hidden','false');
    inlineBox.classList.remove('show'); void inlineBox.offsetWidth; inlineBox.classList.add('show');
    positionInline(); // stays put; does not follow on scroll
    inlineBody.focus();
  }
  function hideInline(){
    inlineBox.classList.remove('show');
    clearTimeout(hideInline._t);
    hideInline._t=setTimeout(function(){ inlineBox.hidden=true; inlineBox.setAttribute('aria-hidden','true'); }, 150);
  }
  function cancelInline(){
    hideInline(); editID=0; replyParent=0; setReplyContext(0); discardCollect(); resetCompose();
    if(window.harborModes && window.harborModes.get()==='comment') window.harborModes.set('reader');
  }
  function postInline(){
    var text=inlineBody.value.trim(); if(!text) return;
    inlineErr.hidden=true; inlinePost.disabled=true;
    var method='POST', url=LIST_URL, payload=null;
    if(editID){ method='PATCH'; url=LIST_URL+'/'+editID; payload={body:text, anchors:[mapStateAnchor()]}; }
    else if(collectSet.length){ payload={body:text, anchors:collectSet.map(function(p){ return {kind:p.kind,path:p.path,quote:p.quote||''}; })}; }
    else { payload={type:state.type,anchor:state.anchor,quote:state.quote,body:text,replyTo:replyParent}; }
    fetch(url,{method:method,headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)})
      .then(function(r){ if(!r.ok) return r.json().then(function(j){ return Promise.reject(j&&j.error); }).catch(function(){ return Promise.reject('could not save comment ('+r.status+')'); }); return r.json(); })
      .then(function(){ hideInline(); editID=0; replyParent=0; setReplyContext(0); discardCollect(); resetCompose(); dropPickStyle(); setOpen(true); bumpOpenFb(); }) // drawer opens to the list so it lands visibly
      .catch(function(m){ inlineErr.textContent=m; inlineErr.hidden=false; })
      .finally(function(){ inlinePost.disabled=false; });
  }
  inlinePost.addEventListener('click',postInline);
  inlineCancel.addEventListener('click',cancelInline);
  inlineClose.addEventListener('click',cancelInline);

  // ── Collect mode (HARB-35) ────────────────────────────────────────────
  // With Collect on, element clicks add and each completed text drag snapshots
  // one span into a multi-anchor set. All stay pinned (dashed outline); clicking
  // a pinned element toggles it off. A pins bar offers Comment / Clear; the
  // gathered set is discarded on Escape / Clear (no draft).
  let collecting=false; // a multi-spot set is being gathered (＋Add / element clicks)
  let collectSet=[]; // [{kind,path,quote,el}]
  function setPageCursor(on){ try{ var d=frame.contentDocument; if(d&&d.body){ d.body.style.cursor = on ? 'crosshair' : ''; } }catch(_){} }
  function ensurePinStyle(){
    try{ var d=frame.contentDocument; if(!d||!d.head) return;
      if(!window.__cpPinStyle){ window.__cpPinStyle=d.createElement('style'); window.__cpPinStyle.textContent='.cp-collect-pin{outline:2px dashed rgba(191,97,106,.9)!important;outline-offset:1px!important;box-shadow:inset 0 0 0 3000px rgba(191,97,106,.06)!important;cursor:pointer!important;transition:box-shadow .12s ease,outline-color .12s ease}'; d.head.appendChild(window.__cpPinStyle); }
    }catch(_){}
  }
  function reindexPins(){ for(var i=0;i<collectSet.length;i++){ if(collectSet[i].el) collectSet[i].el.setAttribute('data-cp-pin',String(i)); } }
  function addPin(pin){
    ensurePinStyle();
    for(var i=0;i<collectSet.length;i++){ var e=collectSet[i]; if(e.kind===pin.kind && e.path===pin.path) return; }
    var elRes=window.harborResolveAnchor&&window.harborResolveAnchor({kind:pin.kind,path:pin.path,quote:pin.quote||''});
    if(elRes&&elRes.el) pin.el=elRes.el;
    collectSet.push(pin);
    if(pin.el){ pin.el.classList.add('cp-collect-pin'); pin.el.setAttribute('data-cp-pin',String(collectSet.length-1)); }
    refreshPins();
  }
  function removePinAt(i){
    if(i<0||i>=collectSet.length) return;
    var p=collectSet[i]; if(p.el){ p.el.classList.remove('cp-collect-pin'); p.el.removeAttribute('data-cp-pin'); }
    collectSet.splice(i,1); reindexPins(); refreshPins();
  }
  function clearPins(){ for(var i=0;i<collectSet.length;i++){ var p=collectSet[i]; if(p.el){ p.el.classList.remove('cp-collect-pin'); p.el.removeAttribute('data-cp-pin'); } } collectSet=[]; refreshPins(); }
  var pinsBar=null;
  function ensurePinsBar(){
    if(pinsBar&&pinsBar.parentNode) return pinsBar;
    pinsBar=document.createElement('div'); pinsBar.className='cp-pins';
    pinsBar.innerHTML='<span class="cp-pins-count"></span><button type="button" class="cp-pins-clear">Clear</button><button type="button" class="cp-pins-post">Write comment</button>';
    document.body.appendChild(pinsBar);
    pinsBar.querySelector('.cp-pins-clear').addEventListener('click',function(){ discardCollect(); });
    pinsBar.querySelector('.cp-pins-post').addEventListener('click',commentSet);
    return pinsBar;
  }
  var pinsHideTimer=null;
  function refreshPins(){
    var bar=ensurePinsBar(); var n=collectSet.length;
    if(n){
      clearTimeout(pinsHideTimer);
      bar.hidden=false; bar.classList.add('cp-show');
      bar.querySelector('.cp-pins-count').textContent=n+' spot'+(n===1?'':'s');
      bar.querySelector('.cp-pins-post').textContent='Write comment';
    } else {
      bar.classList.remove('cp-show');
      bar.hidden=false;
      pinsHideTimer=setTimeout(function(){ if(!collectSet.length) bar.hidden=true; }, 130);
    }
  }
  function setCollecting(on){ collecting=on; setPageCursor(on); refreshPins(); }
  function discardCollect(){ collecting=false; setPageCursor(false); clearPins(); }
  function pinFromAnchor(a){
    var el=null;
    try{ var r=window.harborResolveAnchor&&window.harborResolveAnchor(a); if(r&&r.el) el=r.el; }catch(_){}
    return {kind:a.kind,path:a.path||'',quote:a.quote||'',el:el};
  }
  function escDefault(){ if(open) return setOpen(false); if(!inlineBox.hidden) return cancelInline(); if(collectSet.length){ var n=collectSet.length; discardCollect(); toast('Discarded '+n+' spot'+(n===1?'':'s')+'.'); return; } hideAfford(); }
  // Selection popup "＋ Add": append this selection to a multi-spot set and
  // enter the gathering state (picker cursor + N-spots chip + dashed pins).
  function addToSetFromSelection(){
    // Prefer the snapshot captured when the pill appeared (showAfford) over a
    // live re-read: a shell mousedown on the pill can collapse the iframe
    // selection in some browsers, so the live selection is not reliable here.
    var pc=pendingComment, text='', path='';
    if(pc && pc.quote){ text=pc.quote; path=pc.anchor; }
    else {
      const w=frame.contentWindow, sel=w&&w.getSelection();
      text=sel?sel.toString().trim():'';
      if(text){ const an=sel.anchorNode; const el=an?(an.nodeType===1?an:an.parentElement):null; path=cssPath(el||(frame.contentDocument&&frame.contentDocument.body)); }
    }
    hideAfford();
    if(!text) return;
    addPin({kind:'text', path:path, quote:text});
    setCollecting(true);
    showHeader();
  }
  // Comment the collected set (opens the inline box with an N-spot header).
  function commentSet(){
    if(!collectSet.length) return;
    collecting=false; setPageCursor(false);
    var _b=pinsBar; if(_b){ _b.classList.remove('cp-show'); }
    if(window.harborModes) window.harborModes.set('comment');
    hideAfford(); editID=0; replyParent=0; inlineBody.value=''; inlineErr.hidden=true;
    inlineTitle.textContent='Commenting on '+collectSet.length+' spot'+(collectSet.length===1?'':'s');
    inlineWhere.textContent=''; // title carries the count; where is for a single anchor's context
    clearTimeout(hideInline._t);
    inlineBox.hidden=false; inlineBox.setAttribute('aria-hidden','false');
    inlineBox.classList.remove('show'); void inlineBox.offsetWidth; inlineBox.classList.add('show');
    var last=collectSet[collectSet.length-1]; var t=last&&last.el;
    if(t){ positionInlineFor(t); } else { inlineBox.style.left='50%'; inlineBox.style.top='40%'; inlineBox.style.transform='translateX(-50%)'; }
    inlineBody.focus();
  }
  const clearBtn=document.getElementById('cpClear');
  const LIST_URL='/api/pages/'+encodeURIComponent(slug)+'/comments';

  // Single-mode coordinator (HARB-31) — see change tour script for the shared
  // window.harborModes. During TOUR, commenting is fully suppressed: the
  // comment button disables, the selection pill hides, and the panel cannot
  // open.
  if(!window.harborModes){
    window.harborModes={m:'reader',
      get:function(){ return this.m; },
      set:function(next){ if(next===this.m) return; var prev=this.m; this.m=next;
        document.dispatchEvent(new CustomEvent('harbor-mode',{detail:{prev:prev,next:next}})); } };
  }
  function syncCommentSuppression(){
    var tour=window.harborModes && window.harborModes.get()==='tour';
    btn.disabled=tour;
    if(tour){ hideAfford(); }
  }
  document.addEventListener('harbor-mode',function(e){
    if(e.detail.next==='tour'){ hideAfford(); setOpen(false); }
    syncCommentSuppression();
  });
  syncCommentSuppression();

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
    panel.inert=!v;
    btn.setAttribute('aria-expanded',String(v));
    document.body.classList.toggle('commenting',v);
    if(v){ if(window.harborModes) window.harborModes.set('comment'); hideAfford(); showHeader(); showView('list'); loadList(); if(!panel.hasAttribute('tabindex')) panel.tabIndex=-1; panel.focus(); }
    else {
      if(window.harborModes && window.harborModes.get()==='comment') window.harborModes.set('reader');
      if(document.activeElement&&document.activeElement.closest('#commentPanel')) btn.focus();
      resetCompose();
    }
  }
  // Open the comment panel DIRECTLY into the compose pane (anchored paths: the
  // selection pill, in-flow re-anchoring). List-first open goes through setOpen.
  function openCompose(){
    open=true;
    panel.classList.add('open'); panel.setAttribute('aria-hidden','false'); panel.inert=false; btn.setAttribute('aria-expanded','true'); document.body.classList.add('commenting');
    if(window.harborModes) window.harborModes.set('comment');
    hideAfford(); showHeader();
    showView('compose');
    body.focus();
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
        'box-shadow:0 0 0 2px '+tint+',inset 0 0 0 600px '+tint+'!important;border-radius:2px;transition:box-shadow .12s ease,outline-color .12s ease}';
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
  // Adopt the reader's current live text selection (if any) as a selection anchor
  // when entering the commenting flow, so a preceding selection is captured
  // without the selection itself ever opening the panel (interaction-flow: the
  // only affordance that opens the panel is the comment button).
  function adoptSelection(){
    const win=frame.contentWindow; if(!win) return;
    const sel=win.getSelection(); if(!sel) return;
    const text=sel.toString().trim(); if(!text) return;
    const an=sel.anchorNode;
    const el=an?(an.nodeType===1?an:an.parentElement):null;
    state={type:'selection',anchor:cssPath(el),quote:text};
    renderState();
  }
  // ── "Comment on selection" affordance ────────────────────────────────
  // A reader's selection stays inert (no surprise drawer), but one small pill
  // appears at the selection's end offering to anchor a comment — giving the
  // commenter one-click intent exactly where they're looking (Medium/GDocs
  // pattern). The pill lives in the SHELL (not the agent's page), positioned
  // over the selection by translating the iframe's rect into shell coords, and
  // reads the shell theme so it adapts to dark mode.
  let afford=null, pendingComment=null;
  function shellVar(n){ try{ return getComputedStyle(document.documentElement).getPropertyValue(n).trim()||''; }catch(_){ return ''; } }
  function ensureAfford(){
    if(afford&&afford.parentNode) return afford;
    const bg=shellVar('--surface')||'#ffffff', fg=shellVar('--strong')||'#2e3440', bd=shellVar('--border')||'#d8dee9', fn=shellVar('--font')||'system-ui,sans-serif';
    afford=document.createElement('div');
    afford.setAttribute('role','group');
    afford.style.cssText='position:fixed;z-index:999999;display:flex;align-items:center;gap:3px;padding:3px;border-radius:999px;background:'+bg+';border:1px solid '+bd+';box-shadow:0 2px 10px rgba(46,52,64,.22);opacity:0;'+
      (REDUCED
        ? 'transition:opacity .1s ease'
        : 'transform-origin:top center;transform:scale(.9);transition:opacity .14s var(--ease),transform .14s var(--ease)');
    function addAct(label, fn){
      var b=document.createElement('button'); b.type='button'; b.textContent=label;
      b.style.cssText='border:0;background:transparent;color:'+fg+';font:600 11.5px '+fn+';letter-spacing:.01em;padding:4px 10px;border-radius:999px;cursor:pointer;white-space:nowrap;transition:background .1s,color .1s';
      b.addEventListener('mousedown',function(e){ e.preventDefault(); }); // don't collapse the selection
      b.addEventListener('mouseenter',function(){ b.style.background='rgba(0,0,0,.06)'; });
      b.addEventListener('mouseleave',function(){ b.style.background='transparent'; });
      b.addEventListener('click',function(e){ e.preventDefault(); e.stopPropagation(); fn(); });
      afford.appendChild(b); return b;
    }
    addAct('Comment', openCommentFromSelection);
    addAct('＋ Add', addToSetFromSelection);
    document.body.appendChild(afford);
    return afford;
  }
  function showAfford(){
    if(open){ hideAfford(); return; }
    if(window.harborModes && window.harborModes.get()==='tour'){ hideAfford(); return; }
    const w=frame.contentWindow, s=w&&w.getSelection();
    if(!s||s.isCollapsed||!s.rangeCount) return;
    const r=s.getRangeAt(0).getBoundingClientRect(); if(!r||(!r.width&&!r.height)) return;
    const an=s.anchorNode; const el=an?(an.nodeType===1?an:an.parentElement):null;
    pendingComment={anchor:cssPath(el), quote:s.toString().trim()}; // capture now; a shell click may collapse the selection
    const pill=ensureAfford(); if(!pill) return;
    const fr=frame.getBoundingClientRect();
    const fw=window.innerWidth, fh=window.innerHeight;
    let top=fr.top+r.bottom+8, left=fr.left+r.right-pill.offsetWidth;
    if(top+pill.offsetHeight>fh-8 && (fr.top+r.top)>pill.offsetHeight+10){ top=fr.top+r.top-pill.offsetHeight-8; }
    pill.style.top=Math.max(8,top)+'px'; pill.style.left=Math.max(8,Math.min(left,fw-pill.offsetWidth-8))+'px';
    requestAnimationFrame(function(){ pill.style.opacity='1'; if(!REDUCED) pill.style.transform='scale(1)'; });
  }
  function hideAfford(){
    if(!afford) return;
    var el=afford; afford=null;
    if(!el.parentNode) return;
    if(REDUCED){ el.parentNode.removeChild(el); return; }
    el.style.opacity='0'; el.style.transform='scale(.9)';
    setTimeout(function(){ if(el.parentNode) el.parentNode.removeChild(el); }, 80);
  }
  // pendingComment is the snapshot captured at showAfford (anchor + quote). It is
  // deliberately NOT cleared when the pill fades: a shell mousedown on the pill (or
  // a browser collapsing the iframe selection on it) fires selectionchange ->
  // hideAfford before the click lands, and the Comment/→Add actions must still be
  // able to open from the snapshot instead of a now-empty live selection. It is
  // always overwritten by the next showAfford.
  function openCommentFromSelection(){
    if(pendingComment){ const p=pendingComment; hideAfford(); state={type:'selection',anchor:p.anchor,quote:p.quote}; renderState(); showInline(); return; }
    // Fallback: read a still-live selection if the pill's captured snapshot is gone.
    const w=frame.contentWindow, sel=w&&w.getSelection(); const text=sel?sel.toString().trim():'';
    if(!text) return;
    const an=sel.anchorNode; const el=an?(an.nodeType===1?an:an.parentElement):null;
    hideAfford(); state={type:'selection',anchor:cssPath(el),quote:text}; renderState(); showInline();
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

  // The panel opens only from an explicit affordance (the comment button) —
  // never from a passive text selection. On open, adopt a live selection if one
  // exists so "select, then click comment" still anchors; otherwise whole-page.
  function toggleOpen(){
    if(window.harborModes && window.harborModes.get()==='tour') return; // suppressed during the tour (HARB-31)
    if(!open){ adoptSelection(); }
    setOpen(!open);
  }
  btn.addEventListener('click',toggleOpen);
  close.addEventListener('click',()=>setOpen(false));
  document.addEventListener('keydown',(e)=>{ if(e.key==='Escape'){ escDefault(); } });

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

  // ── Anchor resolution (HARB-30) ────────────────────────────────────────
  // Resolve an {kind, path, quote?, markerID?} anchor to a LIVE DOM target in
  // the iframe, preferring stable identity (change-markerID) over selector over
  // quote re-location — so an anchor survives edits that change selectors.
  // Returns {el, range?, quoteEl?} or null when nothing matches ("not found").
  // Exposed as window.harborResolveAnchor for the review/jump surfaces.
  function attrEscape(v){ return String(v||'').replace(/\\/g,'\\\\').replace(/"/g,'\\"'); }
  function locateQuote(quote){
    if(!doc||!quote) return null;
    if(!doc.createTreeWalker) return null;
    var walker=doc.createTreeWalker(doc.body,4); // NodeFilter.SHOW_TEXT
    while(walker.nextNode()){
      var n=walker.currentNode;
      var idx=n.textContent.indexOf(quote);
      if(idx>=0){
        var range=doc.createRange();
        range.setStart(n,idx); range.setEnd(n,idx+quote.length);
        return {el:n.parentElement, range:range, quoteEl:n};
      }
    }
    return null;
  }
  function resolveAnchor(anchor){
    if(!doc||!anchor) return null;
    // 1) Stable identity: the best-change-marker id survives edits.
    if(anchor.markerID){ try{ var m=doc.querySelector('[data-cf-change="'+attrEscape(anchor.markerID)+'"]'); if(m) return {el:m}; }catch(_){} }
    // 2) Fallback: the recorded selector.
    if(anchor.path){ try{ var p=doc.querySelector(anchor.path); if(p) return {el:p}; }catch(_){} }
    // 3) Last resort: re-locate the quoted text.
    if(anchor.kind==='text' && anchor.quote){ var t=locateQuote(anchor.quote); if(t) return t; }
    return null;
  }
  window.harborResolveAnchor=resolveAnchor;

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
    doc.addEventListener('keydown',(ev)=>{ if(ev.key==='Escape'){ escDefault(); } },true);
    doc.addEventListener('mouseup',(ev)=>{
      const win=frame.contentWindow, sel=win&&win.getSelection();
      const text=sel?sel.toString().trim():'';
      if(collecting && !open){
        if(text && sel && !sel.isCollapsed){ const an=sel.anchorNode; const pe=an?(an.nodeType===1?an:an.parentElement):null; addPin({kind:'text',path:cssPath(pe),quote:text}); }
        return;
      }
      if(open){
        if(!text) return;
        // The anchor of a text selection is usually a text node (use its
        // parent), but for an element selection it is the element itself.
        const an=(sel&&sel.anchorNode)||null;
        const el=an?(an.nodeType===1?an:an.parentElement):null;
        state={type:'selection',anchor:cssPath(el),quote:text};
        renderState(); // in-flow re-anchor
        showView('compose'); // surface the compose pane for the just-captured anchor
        return;
      }
      // Reader mode: selection stays inert, but a passing "Comment" pill is
      // offered at its end so commenting is one click from where they look.
      if(text && sel && !sel.isCollapsed) showAfford(); else hideAfford();
    },true);
    // Dismiss the affordance on any other interaction inside the page.
    doc.addEventListener('mousedown',()=>{ hideAfford(); if(!inlineBox.hidden && !collecting) cancelInline(); },true);
    doc.addEventListener('scroll',()=>hideAfford(),true);
    doc.addEventListener('selectionchange',()=>{ const s=frame.contentWindow&&frame.contentWindow.getSelection(); if(!s||s.isCollapsed) hideAfford(); });
    doc.addEventListener('mousemove',trackHover,true);
    doc.addEventListener('click',(ev)=>{
      // Collecting (no panel): a click adds the element to the set;
      // clicking an already-pinned element toggles it off (removal).
      if(collecting && !open){
        if(hasTextSelection()) return;
        var pinEl=ev.target&&ev.target.closest?ev.target.closest('[data-cp-pin]'):null;
        if(pinEl){ removePinAt(parseInt(pinEl.getAttribute('data-cp-pin'),10)); return; }
        var tg=ev.target; if(!tg||tg.nodeType!==1) return;
        addPin({kind:'element',path:cssPath(tg),quote:''});
        return;
      }
      if(!canPickElement()) return;
      if(hasTextSelection()) return; // a drag-selection just became a quote; don't override it
      setOpen(false); // anchored compose is inline; drop the drawer
      state={type:'element',anchor:cssPath(ev.target),quote:''};
      renderState();
      showInline();
    },true);
  }
  frame.addEventListener('load',wire);
  wire();
  // When the cursor leaves the page (into the panel, header, or beyond) stop
  // the candidate highlight — the pointer is no longer over a pickable element.
  frame.addEventListener('mouseleave',clearHover);
  // Clicking anywhere in the shell chrome (or outside the inline box while it's
  // open) dismisses the affordance / discards the inline draft (HARB-23).
  // Dismiss the affordance / discard the inline draft on a shell mousedown
  // anywhere EXCEPT inside the affordance pill or the inline box. The pill's
  // action buttons are children of `afford`, so a press on them must be
  // treated as inside — otherwise this would hide the pill and discard
  // `pendingComment` before the click, falling back to a live selection that a
  // shell mousedown may have collapsed (selection comment + collect stuck).
  document.addEventListener('mousedown',(e)=>{
    const inAfford=!!(afford&&afford.contains(e.target));
    const inInline=!!(inlineBox&&inlineBox.contains(e.target));
    if(!inAfford && !inInline){ hideAfford(); if(!inlineBox.hidden && !collecting) cancelInline(); }
  },true);

  // Submit the pending comment.
  form.addEventListener('submit',(e)=>{
    e.preventDefault();
    const text=body.value.trim();
    if(!text) return;
    errEl.classList.remove('show');
    const submit=document.getElementById('cpSubmit');
    submit.disabled=true;
    fetch(LIST_URL,{method:'POST',headers:{'Content-Type':'application/json'},
      body:JSON.stringify({type:state.type,anchor:state.anchor,quote:state.quote,body:text,replyTo:replyParent})})
      .then(r=>{ if(!r.ok) return r.json().then(j=>Promise.reject(j&&j.error)).catch(()=>Promise.reject('could not save comment ('+r.status+')')); return r.json(); })
      .then(()=>{ resetCompose(); replyParent=0; setReplyContext(0); showView('list'); loadList(); bumpOpenFb(); })
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
  // List-item actions (HARB-34): anchor line + Jump/Edit/Done/Reply, wired via
  // one delegated click handler. Edit is open-only; done comments are never
  // revised (use Reply instead); Reply starts a new comment citing the parent.
  var itemsById={};
  var jumpIdx={};
  function firstAnchorLabel(a){ if(!a) return ''; if(a.kind==='text') return a.quote||a.path||'text selection'; if(a.kind==='element') return a.path||'element'; return 'whole page'; }
  function anchorLabel(c){
    var a=c.anchors||[];
    if(a.length>1) return a.length+' spots';
    if(a.length===1) return firstAnchorLabel(a[0]);
    if(c.quote) return c.quote;
    if(c.type==='selection') return c.anchor||'text selection';
    if(c.type==='element') return c.anchor||'element';
    return 'whole page';
  }
  function aKindFor(c){ var a=(c.anchors&&c.anchors[0])||{}; return a.kind||(c.type==='selection'?'text':(c.type==='element'?'element':'document')); }
  function itemActions(c){
    var out='<button type="button" class="cp-act cp-act-go" data-act="jump">Jump</button>';
    if(c.status==='open') out+='<button type="button" class="cp-act" data-act="edit">Edit</button>';
    out+='<button type="button" class="cp-act" data-act="'+(c.status==='done'?'reopen':'done')+'">'+(c.status==='done'?'Reopen':'Done')+'</button>';
    out+='<button type="button" class="cp-act" data-act="reply">Reply</button>';
    return out;
  }
  function setReplyContext(id){
    var el=document.getElementById('cpReplyTo'); if(!el) return;
    if(id){ el.textContent='Replying to comment #'+id; el.hidden=false; }
    else { el.textContent=''; el.hidden=true; }
  }
  function openReply(c){
    hideInline(); replyParent=c.id; editID=0; resetCompose(); setReplyContext(c.id);
    showView('compose'); titleEl.textContent='Reply'; body.focus();
  }
  function openEdit(c){
    if(c.status!=='open') return;
    hideInline(); editID=c.id; replyParent=0;
    var a=(c.anchors&&c.anchors[0])||{};
    state={type:aKindFor(c)==='text'?'selection':(aKindFor(c)==='element'?'element':'general'), anchor:a.path||'', quote:a.quote||''};
    inlineBody.value=c.body; inlineTitle.textContent='Edit comment'; inlineWhere.textContent=anchorLabel(c);
    renderState(); showInline();
  }
  function patchStatus(id,status){
    fetch(LIST_URL+'/'+id,{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({status:status})})
      .then(function(r){ if(!r.ok) return Promise.reject('update failed'); return r.json(); })
      .then(function(){ loadList(); })
      .catch(function(){});
  }
  function jumpTo(c){
    var a=c.anchors||[];
    if(!a.length && c.type){ a=[{kind:c.type==='selection'?'text':(c.type==='element'?'element':'document'), path:c.anchor, quote:c.quote}]; }
    if(!a.length) return;
    var i=(jumpIdx[c.id]||0)%a.length; jumpIdx[c.id]=i+1;
    var res=window.harborResolveAnchor&&window.harborResolveAnchor(a[i]);
    if(!res||!res.el){ toast("Can't find that spot in the page — it may have moved since a rebuild."); return; }
    var el=res.el;
    try{ el.scrollIntoView({behavior:REDUCED?'auto':'smooth',block:'center'}); }catch(_){ try{el.scrollIntoView();}catch(_2){} }
    flashEl(el);
  }
  var jumpStyle=null;
  function flashEl(el){
    try{
      var d=frame.contentDocument; if(!d||!d.head) return;
      if(!jumpStyle){ jumpStyle=d.createElement('style'); jumpStyle.textContent='.cp-jump{outline:2px solid #5e81ac!important;outline-offset:1px!important;box-shadow:0 0 0 2px rgba(94,129,172,.28),inset 0 0 0 300px rgba(94,129,172,.18)!important;border-radius:2px;transition:box-shadow .12s ease,outline-color .12s ease}.cp-jump-leave{box-shadow:none!important;outline-width:0!important}'; d.head.appendChild(jumpStyle); }
      el.classList.add('cp-jump');
      setTimeout(function(){
        if(REDUCED){ el.classList.remove('cp-jump'); return; }
        el.classList.add('cp-jump-leave');
        setTimeout(function(){ el.classList.remove('cp-jump'); el.classList.remove('cp-jump-leave'); }, 120);
      }, 1500);
    }catch(_){}
  }
  listEl.addEventListener('click',function(e){
    var btn=e.target.closest&&e.target.closest('.cp-act'); if(!btn) return;
    var li=btn.closest('.cp-item'); if(!li) return;
    var c=itemsById[parseInt(li.dataset.id,10)]; if(!c) return;
    var act=btn.dataset.act;
    if(act==='jump') jumpTo(c);
    else if(act==='edit') openEdit(c);
    else if(act==='done'||act==='reopen') patchStatus(c.id, act==='reopen'?'open':'done');
    else if(act==='reply') openReply(c);
  });
  function skeletonList(){ return '<li class="cp-skeleton"><span></span><span></span><span></span></li><li class="cp-skeleton"><span></span><span></span><span></span></li>'; }
  function loadList(){
    listHeadEl.hidden=false;
    listEl.innerHTML=skeletonList();
    var url=LIST_URL + (filter==='all' ? '' : '?status='+encodeURIComponent(filter));
    fetch(url).then(function(r){ if(!r.ok) throw new Error('load failed'); return r.json(); }).then(items=>{
      listHeadEl.hidden = (items.length===0);
      listEl.innerHTML=''; itemsById={};
      if(!items.length){
        var title='', note='';
        // Copy follows the app's empty-state pattern (state + what to do next)
        // and the model's ubiquitous language: the object is a "comment";
        // status is "Open" / "Done" (not "resolved"); anchors are a text
        // selection, an element, or the whole page; the affordance is
        // "+ New comment". One boxed card, no inline CTA.
        if(filter==='all'){ title='No comments yet'; note='Select a bit of text in the page to comment on it, or use the <b>+ New comment</b> button above to comment on the whole page.'; }
        else if(filter==='open'){ title='No open comments'; note='Open comments show up here. Switch to <b>All</b> or <b>Done</b> to see the rest.'; }
        else { title='No done comments'; note='Comments you mark <b>Done</b> show up here. Switch to <b>All</b> or <b>Open</b> to see the rest.'; }
        // Boxed empty state matching the app's library empty pattern (dashed
        // frame, title + note). No inline CTA — "+ New comment" already lives
        // in the sidebar toolbar above this list.
        listEl.innerHTML='<li class="cp-empty"><div class="cp-empty-title">'+title+'</div><p class="cp-empty-note">'+note+'</p></li>';
        return;
      }
      for(const c of items){
        const li=document.createElement('li');
        li.className='cp-item'+(c.status==='done'?' cp-item--done':''); li.dataset.id=String(c.id);
        // Content-first hierarchy: feedback body is the hero (flex-1), the
        // status is compact queue-meta at top-right, then the spot + actions.
        const top=document.createElement('div'); top.className='cp-item-top';
        const b=document.createElement('div'); b.className='cp-item-body'; b.textContent=c.body;
        const st=document.createElement('span'); st.className='cp-item-status'+(c.status==='done'?' done':'');
        st.textContent=(c.status==='done'?'done':'open');
        top.appendChild(b); top.appendChild(st);
        li.appendChild(top);
        const aL=document.createElement('div'); aL.className='cp-item-anchor'; aL.textContent=anchorLabel(c); li.appendChild(aL);
        if(c.quote && !(c.anchors||[]).length){ const q=document.createElement('div'); q.className='cp-item-quote'; q.textContent=c.quote; li.appendChild(q); }
        const acts=document.createElement('div'); acts.className='cp-item-actions'; acts.innerHTML=itemActions(c); li.appendChild(acts);
        itemsById[c.id]=c;
        listEl.appendChild(li);
      }
    }).catch(function(){
      listHeadEl.hidden=false;
      listEl.innerHTML='<li class="cp-empty cp-empty-err"><div class="cp-empty-title">Can\'t load comments</div><p class="cp-empty-note">There was a problem reaching the server.</p><button type="button" class="cp-empty-action" id="cpRetry">Retry</button></li>';
      var rb=document.getElementById('cpRetry'); if(rb) rb.addEventListener('click',loadList);
    });
  }
})();