package render

// ExcalidrawView is the app-family view for .excalidraw artifacts: a read-only
// shell that loads the vendored UMD bundles from /excalidraw/ and feeds the
// artifact's scene bytes (fetched same-origin from RawURL) into the embedded
// editor with editing disabled. The stored file is never touched — this view
// is derived on every read; /raw keeps serving the source.
//
// The bundle is served as separate static files (not inlined): ~1.3MB of JS
// would bloat every pageview response, and the browser caches it immutably
// across all excalidraw artifacts.
func ExcalidrawView(meta PageMeta, rawURL string) string {
	return `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + esc(meta.Title) + `</title>
<style>
  html,body{margin:0;height:100%;background:#fff}
  #app{height:100vh}
  .ex-err{font:400 14px/1.6 system-ui,sans-serif;color:#4c566a;padding:32px}
</style>
<script src="/excalidraw/react.production.min.js"></script>
<script src="/excalidraw/react-dom.production.min.js"></script>
<script src="/excalidraw/excalidraw.production.min.js"></script>
</head><body>
<div id="app"></div>
<div id="err" class="ex-err" hidden>Could not load this drawing.</div>
<script>
(function(){
  function fail(){ document.getElementById('err').hidden=false; }
  try {
    if (!window.ExcalidrawLib || !window.React || !window.ReactDOM) { fail(); return; }
    fetch(` + "`" + rawURL + "`" + `, { credentials:'same-origin' })
      .then(function(r){ if(!r.ok) throw new Error('fetch '+r.status); return r.json(); })
      .then(function(scene){
        ReactDOM.createRoot(document.getElementById('app')).render(
          React.createElement(window.ExcalidrawLib.Excalidraw, {
            initialData: { elements: scene.elements || [], appState: scene.appState || {}, files: scene.files || {} },
            viewModeEnabled: true,
            zenModeEnabled: true,
            gridModeEnabled: false,
            theme: (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) ? 'dark' : 'light'
          })
        );
      })
      .catch(fail);
  } catch (e) { fail(); }
})();
</script>
</body></html>`
}
