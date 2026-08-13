// Shared live-sync SSE client (HARB-36 cleanup). Extracted from the Go
// liveSyncScript helper into a //go:embed'd file. Reads its config from the
// window.__harbor.live seam:
//
//   live: { topic, mode: "workspace" | "library", slug? }
//
// Library ("changed" → window.__harborReload) and workspace ("page-changed" for
// this slug, "navigate") pages both reuse it; EventSource auto-reconnects.
(function () {
  var live = (window.__harbor && window.__harbor.live) || {};
  var topic = live.topic || '';
  if (!topic) return;
  var es = new EventSource('/api/events?topic=' + encodeURIComponent(topic));
  es.addEventListener('message', function (e) {
    var ev; try { ev = JSON.parse(e.data); } catch (_) { return; }
    if (!ev || !ev.type) return;
    if (ev.type === 'changed') { if (window.__harborReload) window.__harborReload(); return; }
    if (ev.type === 'page-changed' && live.mode === 'workspace' && live.slug && ev.slug === live.slug) {
      var f = document.getElementById('frame');
      if (f && f.contentWindow) {
        try {
          var y = f.contentWindow.scrollY;
          f.addEventListener('load', function rst() { f.removeEventListener('load', rst); try { f.contentWindow.scrollTo(0, y); } catch (_) { } });
          f.contentWindow.location.reload();
        } catch (_) { }
      }
      return;
    }
    if (ev.type === 'navigate' && ev.url) { location.href = ev.url; }
  });
})();