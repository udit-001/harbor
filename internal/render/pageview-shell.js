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

  // Pharos-style tooltips on the header icon buttons (back / prev / next /
  // collect / comment / view-mode / pop-out / theme). Dark pill with an arrow,
  // appearing below the icon. Uses data-tooltip so it never duplicates the
  // native title (kept only on genuinely-disabled controls).
  var pvTip=null, pvTipArrow=null;
  function pvTipEls(){
    if(!pvTip){ pvTip=document.createElement('div'); pvTip.className='pv-tooltip'; document.body.appendChild(pvTip);
      pvTipArrow=document.createElement('div'); pvTipArrow.className='pv-tooltip-arrow'; document.body.appendChild(pvTipArrow); }
    return [pvTip, pvTipArrow];
  }
  function hidePvTips(){ if(pvTip){ pvTip.classList.remove('show'); pvTipArrow.classList.remove('show'); } }
  document.querySelectorAll('.pv [data-tooltip]').forEach(function(el){
    el.addEventListener('mouseenter', function(){
      const [t,a]=pvTipEls();
      t.textContent=el.getAttribute('data-tooltip');
      const r=el.getBoundingClientRect();
      t.style.left=(r.left+r.width/2 - t.offsetWidth/2)+'px';
      t.style.top=(r.bottom+8)+'px';
      a.style.left=(r.left+r.width/2)+'px';
      a.style.top=(r.bottom+4)+'px';
      void t.offsetWidth; t.classList.add('show'); a.classList.add('show');
    });
    el.addEventListener('mouseleave', hidePvTips);
  });
  document.addEventListener('scroll', hidePvTips, true);
})();