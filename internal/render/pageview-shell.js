
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
})();