
(function(){
  if(!(window.__harbor&&window.__harbor.workspace)) return;
  var es=new EventSource('/api/events?topic='+encodeURIComponent('workspace:'+((window.__harbor&&window.__harbor.workspace)||'')));
  var H={
  pageChanged: function(ev){ if(ev.slug==="(window.__harbor&&window.__harbor.slug)||''"){ var f=document.getElementById('frame'); if(f&&f.contentWindow){ try{ var y=f.contentWindow.scrollY; f.addEventListener('load',function rst(){ f.removeEventListener('load',rst); try{ f.contentWindow.scrollTo(0,y); }catch(_){} }); f.contentWindow.location.reload(); }catch(_){} } } },
  navigate: function(ev){ if(ev.url) location.href=ev.url; }
};
  es.addEventListener('message',function(e){
    var ev; try{ ev=JSON.parse(e.data); }catch(_){ return; }
    if(!ev||!ev.type) return;
    if(ev.type==='changed'&&H.changed){ H.changed(ev); return; }
    if(ev.type==='page-changed'&&H.pageChanged){ H.pageChanged(ev); return; }
    if(ev.type==='navigate'&&ev.url){ if(H.navigate){ H.navigate(ev); } else { location.href=ev.url; } }
  });
})();
