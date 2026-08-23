package render

import (
	"github.com/udit-001/harbor/internal/markdown"
)

// TextFrameView renders a text-frame artifact (markdown, text) as a complete
// HTML document for the pageview iframe: markdown goes through the goldmark
// renderer, text wraps in a <pre>. The artifact's stored bytes are never
// modified — this view is derived on every read; /raw keeps serving the
// source. One function per view family: callers pass the format + source,
// everything about styling and dispatch stays here.
func TextFrameView(page PageMeta, source string) string {
	var body string
	switch page.Format {
	case "markdown":
		body = `<article class="prose">` + markdown.Render(source) + `</article>`
	default: // text
		body = `<pre class="plain">` + esc(source) + `</pre>`
	}
	return `<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + esc(page.Title) + `</title>` + ThemeBootScript + textFrameCSS() + `</head><body>` + body + `</body></html>`
}

// PageMeta is the subset of a page the view needs: identity + format. Keeping
// it tiny (not the whole db.Page) makes the render package's interface say
// exactly what the view consumes — no accidental coupling to store fields.
type PageMeta struct {
	Slug   string
	Title  string
	Format string
}

func textFrameCSS() string {
	return `<style>` + ThemeTokens() + `
/* Surface extras: a paper-white reading surface in light mode; dark inherits
   --bg from the tokens. The iframe follows the shell's manual theme via
   ThemeBootScript (same-origin localStorage) instead of only the OS preference. */
/* paper-white reading surface, light theme only */
:root:not([data-theme="dark"]){--bg:#fff}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--text);font:400 15px/1.7 var(--font);
-webkit-font-smoothing:antialiased;padding:40px max(24px,calc((100% - 760px)/2)) 80px}
.prose{max-width:760px}
.prose h1,.prose h2,.prose h3{color:var(--strong);line-height:1.3;margin:1.6em 0 .5em}
.prose h1{font-size:1.7em}.prose h2{font-size:1.3em}.prose h3{font-size:1.1em}
.prose p{margin:.7em 0}
.prose a{color:var(--acc)}
.prose code{font:500 .88em ui-monospace,SFMono-Regular,Menlo,monospace;background:var(--surface);
border:1px solid var(--hair);border-radius:4px;padding:.1em .35em}
.prose pre{background:var(--surface);border:1px solid var(--hair);border-radius:8px;
padding:14px 16px;overflow-x:auto}
.prose pre code{background:none;border:0;padding:0}
.prose blockquote{margin:.8em 0;padding:.1em 0 .1em 14px;border-left:3px solid var(--border);color:var(--muted)}
.prose ul,.prose ol{padding-left:1.5em}
.prose li{margin:.25em 0}
.prose hr{border:0;border-top:1px solid var(--hair);margin:1.8em 0}
.prose table{border-collapse:collapse;margin:1em 0}
.prose th,.prose td{border:1px solid var(--border);padding:6px 12px;text-align:left}
.prose img{max-width:100%}
pre.plain{font:400 13.5px/1.65 ui-monospace,SFMono-Regular,Menlo,monospace;color:var(--text);
white-space:pre-wrap;overflow-wrap:anywhere}
</style>`
}
