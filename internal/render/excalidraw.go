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
//
// HarborBridge implements the cross-frame anchor protocol the shell speaks
// (HARB-PLAN-C): the shell cannot reach into this canvas with querySelector,
// so anchoring is negotiated over postMessage. Messages carry
// source:"harbor-ex" (viewer → shell) or "harbor-shell" (shell → viewer):
//
//	ready                          viewer loaded, scene mounted
//	pick {id,label,rect}           user clicked an element while capturing
//	geometry {id,rect}             answer to a geometry request; rect is
//	                               iframe-viewport coords
//	capture / capture-off          arm/disarm element-pick clicking
//	select {ids} / clear           highlight via the editor's own selection
//	jump {id}                      scroll into view, then report geometry
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

    // The 0.17 UMD build hands the editor API through the excalidrawAPI prop
    // (the useExcalidrawAPI hook isn't exported on the UMD global).
    function bindApi(api){
      var capturing = false;

      function post(msg){ msg.source='harbor-ex'; parent.postMessage(msg,'*'); }
      function els(){ return api.getSceneElements().filter(function(e){ return !e.isDeleted; }); }
      function st(){ return api.getAppState(); }
        // Screen = (scene + scroll) * zoom; the #app div is the draw surface.
        function surface(){ return document.getElementById('app').getBoundingClientRect(); }
        function toScene(cx,cy){
          var s=st(), z=s.zoom.value, r=surface();
          return { x:(cx-r.left-s.scrollX)/z, y:(cy-r.top-s.scrollY)/z };
        }
        function rectOf(e){
          var s=st(), z=s.zoom.value, r=surface();
          return { left:r.left+(e.x+s.scrollX)*z, top:r.top+(e.y+s.scrollY)*z,
                   width:e.width*z, height:e.height*z };
        }
        // Topmost element whose bbox contains the point. Rotation is ignored —
        // good enough for anchoring; the pick is confirmed by selection anyway.
        function hitAt(p){
          var list=els();
          for(var i=list.length-1;i>=0;i--){
            var e=list[i];
            if(p.x>=e.x && p.x<=e.x+e.width && p.y>=e.y && p.y<=e.y+e.height) return e;
          }
          return null;
        }
        function byId(id){ return els().find(function(e){ return e.id===id; }); }
        function label(e){ return (e.type==='text' && e.text) ? String(e.text).slice(0,48) : e.type; }

        function select(ids){
          var sel={};
          (ids||[]).forEach(function(id){ sel[id]=true; });
          api.updateScene({ selectedElementIds: sel });
        }

        // View mode suppresses the editor's own selection UI, so highlights are
        // drawn by us: one overlay div over the element's screen rect, kept
        // aligned through onChange (pan/zoom/scroll all fire it).
        var hlId = null;
        var hl = document.createElement('div');
        hl.style.cssText = 'position:fixed;pointer-events:none;z-index:9999;display:none;'+
          'border:2px solid '+(window.matchMedia('(prefers-color-scheme: dark)').matches?'#88c0d0':'#5e81ac')+';'+
          'background:'+(window.matchMedia('(prefers-color-scheme: dark)').matches?'rgba(136,192,208,.18)':'rgba(94,129,172,.14)')+';'+
          'border-radius:6px;transition:border-color .12s ease';
        document.body.appendChild(hl);
        function moveHl(){
          if(!hlId) return;
          var e=byId(hlId);
          if(!e){ hl.style.display='none'; return; }
          var r=rectOf(e);
          hl.style.display='block';
          hl.style.left=r.left+'px'; hl.style.top=r.top+'px';
          hl.style.width=r.width+'px'; hl.style.height=r.height+'px';
        }
        window.exSetHl = function(id){ hlId=id; moveHl(); };
        window.exClearHl = function(){ hlId=null; hl.style.display='none'; };
        window.exMoveHl = moveHl;

        function onMsg(ev){
          var m=ev.data||{}; if(m.source!=='harbor-shell') return;
          if(m.type==='capture'){ capturing=true; }
          else if(m.type==='capture-off'){ capturing=false; }
          else if(m.type==='select'){ select(m.ids); window.exSetHl((m.ids||[])[0]||null); }
          else if(m.type==='clear'){ select([]); window.exClearHl(); }
          else if(m.type==='jump'){
            var e=byId(m.id);
            if(e){ try{ api.scrollToContent(e,{fitToViewport:false}); }catch(_){ }
              setTimeout(function(){ window.exSetHl(m.id); post({type:'geometry', id:m.id, rect:rectOf(byId(m.id)||e)}); }, 380);
            }
          }
          else if(m.type==='geometry'){
            var g=byId(m.id);
            if(g) post({type:'geometry', id:m.id, rect:rectOf(g)});
          }
        }
        function onDown(ev){
          if(!capturing) return;
          var e=hitAt(toScene(ev.clientX, ev.clientY));
          if(e) post({type:'pick', id:e.id, label:label(e), rect:rectOf(e)});
        }
        window.addEventListener('message', onMsg);
        document.getElementById('app').addEventListener('pointerdown', onDown, true);
        post({type:'ready'});
    }

    fetch(` + "`" + rawURL + "`" + `, { credentials:'same-origin' })
      .then(function(r){ if(!r.ok) throw new Error('fetch '+r.status); return r.json(); })
      .then(function(scene){
        ReactDOM.createRoot(document.getElementById('app')).render(
          React.createElement(window.ExcalidrawLib.Excalidraw, {
            initialData: { elements: scene.elements || [], appState: scene.appState || {}, files: scene.files || {} },
            viewModeEnabled: true,
            zenModeEnabled: true,
            gridModeEnabled: false,
            theme: (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) ? 'dark' : 'light',
            excalidrawAPI: function(api){ try{ bindApi(api); }catch(e){ fail(); } },
            // Pan/zoom/scroll all fire onChange — keep the highlight overlay glued
            onChange: function(){ if(window.exMoveHl) window.exMoveHl(); }
          })
        );
      })
      .catch(fail);
  } catch (e) { fail(); }
})();
</script>
</body></html>`
}
