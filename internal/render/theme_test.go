package render

import (
	"strings"
	"testing"
)

// Every surface must start from the shared tokens — a surface that stops
// embedding ThemeTokens silently drifts its palette (the --font incident).
func TestThemeTokensShared(t *testing.T) {
	for name, css := range map[string]string{
		"library":   libraryCSS(),
		"pageview":  pageViewCSS(),
		"textframe": textFrameCSS(),
	} {
		if !strings.Contains(css, "--ease:cubic-bezier(.23,1,.32,1)") || !strings.Contains(css, "--acc:#466286") {
			t.Errorf("%s CSS does not embed ThemeTokens", name)
		}
	}
	if n := strings.Count(libraryCSS(), "--acc:#466286"); n != 1 {
		t.Errorf("library embeds accent token %d times, want 1 (tokens defined once)", n)
	}
}

// IframeNotFound renders through templ without panicking and carries the
// theme boot script.
func TestIframeNotFoundRenders(t *testing.T) {
	out := IframeNotFound("page", "missing.html")
	if !strings.Contains(out, "data-theme") || !strings.Contains(out, "Not found") {
		t.Errorf("iframe not-found page missing theme boot or content")
	}
}
