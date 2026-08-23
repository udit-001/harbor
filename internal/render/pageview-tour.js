// What-changed tour (HARB-12/36). Extracted out of pageview.go into a real JS
// file and //go:embed'd, inlined into the page. Reads per-page data from the
// window.__harbor context seam set by Go.
(function () {
  const slug = (window.__harbor && window.__harbor.slug) || '';
  const frame = document.getElementById('frame');
  const btn = document.getElementById('cfBtn');
  const card = document.getElementById('cfCard');
  const stepEl = document.getElementById('cfStep');
  const titleEl = document.getElementById('cfTitle');
  const descEl = document.getElementById('cfDesc');
  const prevBtn = document.getElementById('cfPrev');
  const nextBtn = document.getElementById('cfNext');
  const doneBtn = document.getElementById('cfDone');
  const closeBtn = document.getElementById('cfClose');
  const headEl = document.getElementById('cfHead');
  const LIST_URL = '/api/pages/' + encodeURIComponent(slug) + '/changes';
  const REDUCED = matchMedia('(prefers-reduced-motion: reduce)').matches;
  // The what-changed tour walks data-cf-change markers in the iframe DOM —
  // only HTML artifacts carry them. Other formats never show the chip.
  const DOM_FORMAT = (window.__harbor && window.__harbor.format) === 'html';
  if (!DOM_FORMAT) { btn.hidden = true; return; }

  // Single-mode coordinator (HARB-31): READER | TOUR | COMMENT are mutually
  // exclusive. A shared window.harborModes records the active mode and fires a
  // 'harbor-mode' event; each subsystem tears its own UI down when the OTHER
  // mode claims the stage. Closers only revert to READER if they still own the
  // mode, so a close-triggered-by-a-mode-claim never clobbers the claimer.
  if (!window.harborModes) {
    window.harborModes = HarborCore.createModes(function (d) {
      document.dispatchEvent(new CustomEvent('harbor-mode', { detail: d }));
    });
  }
  function claimReader() { if (window.harborModes && window.harborModes.get() === 'tour') window.harborModes.set('reader'); }
  // If COMMENT claims the stage, the tour yields (closes cleanly).
  document.addEventListener('harbor-mode', function (e) {
    if (e.detail.next === 'comment') { chipSuppressed = true; finish(); }
    else if (e.detail.prev === 'comment') {
      chipSuppressed = false;
      if (matched.length && card.hidden) btn.hidden = false;
    }
  });

  let all = [], matched = [], idx = -1, iframeReady = false;
  let chipSuppressed = false; // comment mode claimed the stage; chip returns when it releases
  let hlEl = null, styleTag = null;

  function locate(id) {
    try { const d = frame.contentDocument; if (!d) return null; return d.querySelector('[data-cf-change="' + String(id).replace(/"/g, '\\"') + '"]'); }
    catch (_) { return null; }
  }
  function injectStyle() {
    if (styleTag && styleTag.parentNode) return;
    try {
      const d = frame.contentDocument; if (!d || !d.head) return;
      const theme = document.documentElement.dataset.theme || 'light';
      const accent = (theme === 'dark') ? '#88c0d0' : '#5e81ac';
      const tint = (theme === 'dark') ? 'rgba(136,192,208,.22)' : 'rgba(94,129,172,.15)';
      styleTag = d.createElement('style'); styleTag.textContent =
        '.cf-hl{outline:2px solid ' + accent + '!important;outline-offset:1px!important;' +
        'box-shadow:0 0 0 2px ' + tint + '!important;border-radius:2px}';
      d.head.appendChild(styleTag);
    } catch (_) { }
  }
  function clearHl() { if (hlEl) { hlEl.classList.remove('cf-hl'); hlEl = null; } }
  function render() {
    if (idx < 0 || idx >= matched.length) return;
    const s = matched[idx];
    const single = matched.length <= 1;
    stepEl.hidden = single;
    if (!single) stepEl.textContent = 'Change ' + (idx + 1) + ' of ' + matched.length;
    prevBtn.hidden = single;
    nextBtn.hidden = single;
    headEl.hidden = single;
    titleEl.textContent = s.change.title || s.change.changeId;
    descEl.textContent = s.change.description || (s.el ? ('Marked element: ' + s.change.changeId) : '');
    clearHl();
    if (s.el) {
      injectStyle(); hlEl = s.el; s.el.classList.add('cf-hl');
      try { s.el.scrollIntoView(REDUCED ? { block: 'center' } : { behavior: 'smooth', block: 'center' }); } catch (_) { try { s.el.scrollIntoView(); } catch (_2) { } }
    }
    positionCard();
    if (!REDUCED) setTimeout(positionCard, 350);
    prevBtn.disabled = (idx === 0);
    nextBtn.disabled = (idx === matched.length - 1);
  }
  function positionCard() {
    card.classList.remove('cf-card--below');
    if (idx < 0 || idx >= matched.length) return;
    const el = matched[idx].el;
    if (!el) { card.style.left = '16px'; card.style.top = ''; card.style.bottom = '16px'; card.style.right = ''; return; }
    let fr, er;
    try { fr = frame.getBoundingClientRect(); er = el.getBoundingClientRect(); } catch (_) { return; }
    const tx = fr.left + er.left, ty = fr.top + er.top;
    const cw = card.offsetWidth || 380, ch = card.offsetHeight || 190, margin = 12;
    let left = tx + er.width / 2 - cw / 2;
    left = Math.max(8, Math.min(left, innerWidth - cw - 8));
    card.style.right = ''; card.style.bottom = ''; card.style.top = '';
    if (ty > ch + margin) {
      card.style.top = (ty - ch - margin) + 'px';
    } else {
      card.style.top = (ty + er.height + margin) + 'px';
      card.classList.add('cf-card--below');
    }
    card.style.left = left + 'px';
  }
  function go(i) { if (i < 0 || i >= matched.length) return; idx = i; render(); }
  function doHide() { card.classList.remove('cf-exit'); card.hidden = true; card.setAttribute('aria-hidden', 'true'); btn.hidden = chipSuppressed; }
  function finish() {
    claimReader();
    clearHl();
    if (REDUCED) { doHide(); return; }
    if (card.classList.contains('cf-exit')) { doHide(); return; }
    card.classList.add('cf-exit');
    card.addEventListener('animationend', function once() { card.removeEventListener('animationend', once); doHide(); });
    setTimeout(doHide, 150);
  }
  function tryReady() {
    if (!iframeReady || !all.length) return;
    matched = all.map(function (c) { return { change: c, el: locate(c.changeId) }; }).filter(function (p) { return p.el; });
    if (matched.length && !chipSuppressed) btn.hidden = false;
  }
  prevBtn.addEventListener('click', function () { go(idx - 1); });
  nextBtn.addEventListener('click', function () { go(idx + 1); });
  doneBtn.addEventListener('click', finish);
  closeBtn.addEventListener('click', finish);
  btn.addEventListener('click', function () {
    if (window.harborModes) window.harborModes.set('tour');
    btn.hidden = true; card.classList.remove('cf-exit'); card.hidden = false; card.setAttribute('aria-hidden', 'false');
    go(0); document.activeElement && document.activeElement.blur && document.activeElement.blur();
  });
  function onTourKey(e) { if (e.key === 'Escape' && !card.hidden) { e.preventDefault(); finish(); } }
  document.addEventListener('keydown', onTourKey);
  function wireFrameKeys() { try { if (frame.contentDocument) { frame.contentDocument.removeEventListener('keydown', onTourKey); frame.contentDocument.addEventListener('keydown', onTourKey); } } catch (_) { } }
  frame.addEventListener('load', function () { iframeReady = true; wireFrameKeys(); if (btn.hidden) tryReady(); });
  wireFrameKeys();
  if (frame.contentDocument && frame.contentDocument.readyState === 'complete') { iframeReady = true; }
  fetch(LIST_URL).then(function (r) { return r.json(); }).then(function (list) { all = Array.isArray(list) ? list : []; tryReady(); }).catch(function () { });
})();