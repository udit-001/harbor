package render

import _ "embed"

// Pageview JS kept as real files and //go:embed'd, then inlined into the page
// (same pattern as quiz-attempt.js / copy-code.js). Per-page dynamic data is
// written by Go into a tiny inline window.__harbor context object that these
// files read, so a static JS file never needs to be templated.
//
//go:embed pageview-tour.js
var pageviewTourJS string

//go:embed pageview-annotations.js
var pageviewAnnotationsJS string

//go:embed pageview-shell.js
var pageviewShellJS string

//go:embed pageview-livesync.js
var pageviewLiveSyncJS string
