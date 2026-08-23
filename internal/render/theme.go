package render

// ThemeTokens is the single home for Harbor's shared design tokens — the Nord
// palette, the font stack, and the motion curves — in both themes. Every
// surface (library, page view, text-frame views, templ pages) starts its CSS
// with this block and appends only its own extras. The tokens were once pasted
// into each surface independently; they drifted, and the drift shipped (the
// --font collapse documented in AGENTS.md). Change palettes here, nowhere else.
func ThemeTokens() string {
	return `:root{--bg:#eceff4;--surface:#fff;--surface2:#f6f8fb;--border:#d8dee9;--hair:#e5e9f0;
--text:#4c566a;--muted:#5f6b7d;--strong:#2e3440;--acc:#466286;--acc-soft:#e0e7ff;
--ok:#3f6d25;--ok-soft:#e6f0e6;--warn:#9c4f37;--warn-soft:#fadfd2;--arch:#5f6b7d;--arch-soft:#eceff4;
--font:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;
--ease:cubic-bezier(.23,1,.32,1);--ease-in-out:cubic-bezier(.77,0,.175,1)}
[data-theme="dark"]{
--bg:#2e3440;--surface:#3b4252;--surface2:#434c5e;--border:#4c566a;--hair:#3b4252;
--text:#d8dee9;--muted:#81a1c1;--strong:#eceff4;--acc:#88c0d0;--acc-soft:rgba(136,192,208,.18);
--ok:#a3be8c;--ok-soft:rgba(163,190,140,.16);--warn:#d08770;--warn-soft:rgba(208,135,112,.16);--arch:#81a1c1;--arch-soft:rgba(129,161,193,.16)}`
}

// ThemeBootScript is the <head> snippet that resolves the theme before first
// paint: stored choice, else system. Same-origin iframes (text-frame views)
// run it too so they follow the shell's manual theme choice instead of only
// the OS preference.
const ThemeBootScript = `<script>(function(){try{var t=localStorage.getItem('harbor_theme');if(!t||t==='system'){t=window.matchMedia('(prefers-color-scheme:dark)').matches?'dark':'light'}document.documentElement.dataset.theme=t}catch(e){document.documentElement.dataset.theme='light'}})();</script>`
