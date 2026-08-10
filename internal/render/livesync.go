package render

import "strconv"

// liveSyncScript returns a self-contained SSE live-sync client for the
// dashboard. It's a deep module behind a small seam: the caller gives it a
// subscribe topic and a JavaScript handlers object, and this script owns the
// entire wire protocol — EventSource setup, automatic reconnection, event
// parsing, and routing. Callers never touch EventSource directly, so the
// live-sync behaviour lives in exactly one place.
//
// handlers is a JavaScript object literal. Recognized keys:
//
//	changed(ev)     — a broadcast "changed" mutation event
//	pageChanged(ev) — a "page-changed" mutation event (pageType + seq/slug)
//	navigate(ev)    — optional; defaults to location.href = ev.url
//
// Omitted keys are simply unused. EventSource auto-reconnects on drop.
//
// topic is the /api/events channel ("home", "workspace:<name>", …). Keep it
// simple and put the page-specific handlers at the call site.
func liveSyncScript(topic, handlers string) string {
	return `<script>
(function(){
  var es=new EventSource('/api/events?topic='+encodeURIComponent(` + strconv.Quote(topic) + `));
  var H=` + handlers + `;
  es.addEventListener('message',function(e){
    var ev; try{ ev=JSON.parse(e.data); }catch(_){ return; }
    if(!ev||!ev.type) return;
    if(ev.type==='changed'&&H.changed){ H.changed(ev); return; }
    if(ev.type==='page-changed'&&H.pageChanged){ H.pageChanged(ev); return; }
    if(ev.type==='navigate'&&ev.url){ if(H.navigate){ H.navigate(ev); } else { location.href=ev.url; } }
  });
})();
</script>`
}
