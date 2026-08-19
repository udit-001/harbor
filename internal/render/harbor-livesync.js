// Shared live-sync SSE client (HARB-36 cleanup). Extracted from the Go
// liveSyncScript helper into a //go:embed'd file. Reads its config from the
// window.__harbor.live seam:
//
//   live: { topic, mode: "workspace" | "library", slug? }
//
// Library ("changed" → window.__harborReload) and workspace ("page-changed" for
// this slug, "navigate") pages both reuse it.
//
// Connection hygiene: the server is plain HTTP → HTTP/1.1, so the browser caps
// this origin at 6 parallel connections. Every tab holding an EventSource
// permanently pins one of them; 6 open tabs = every navigation queues = the
// dashboard "freezes". So the connection is only held while the tab is
// visible: hidden tabs close it and resubscribe on focus. The broker has no
// replay, so a tab hidden longer than RESYNC_MS also resyncs once on return
// (library refetch / view reload); quick tab switches skip the resync to
// avoid flicker. Trade-off: events pushed while hidden — including
// "navigate" — are not seen live; the resync covers data, not navigation.
(function () {
  var live = (window.__harbor && window.__harbor.live) || {};
  var topic = live.topic || '';
  if (!topic) return;

  var RESYNC_MS = 5000;
  var es = null;
  var hiddenAt = 0;

  function reloadView() {
    var f = document.getElementById('frame');
    if (!f || !f.contentWindow) return;
    try {
      var y = f.contentWindow.scrollY;
      f.addEventListener('load', function rst() { f.removeEventListener('load', rst); try { f.contentWindow.scrollTo(0, y); } catch (_) { } });
      f.contentWindow.location.reload();
    } catch (_) { }
  }

  function handle(ev) {
    if (!ev || !ev.type) return;
    if (ev.type === 'changed') {
      if (window.__harborReload) window.__harborReload();
      return;
    }
    if (ev.type === 'page-changed' && live.mode === 'workspace' && live.slug && ev.slug === live.slug) {
      reloadView();
      return;
    }
    if (ev.type === 'navigate' && ev.url) { location.href = ev.url; }
  }

  // Catch-up after being hidden: no replay in the broker, so redo the
  // equivalent of the events that may have been missed.
  function resync() {
    if (live.mode === 'library') {
      if (window.__harborReload) window.__harborReload();
      return;
    }
    reloadView();
  }

  function subscribe() {
    if (es || document.hidden) return; // never hold a connection while hidden
    es = new EventSource('/api/events?topic=' + encodeURIComponent(topic));
    es.addEventListener('message', function (e) {
      var ev; try { ev = JSON.parse(e.data); } catch (_) { return; }
      handle(ev);
    });
  }

  function unsubscribe() {
    if (!es) return;
    es.close();
    es = null;
  }

  document.addEventListener('visibilitychange', function () {
    if (document.hidden) {
      hiddenAt = Date.now();
      unsubscribe();
    } else {
      if (hiddenAt && Date.now() - hiddenAt > RESYNC_MS) resync();
      hiddenAt = 0;
      subscribe();
    }
  });

  subscribe();
})();
