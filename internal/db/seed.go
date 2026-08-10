package db

import (
	_ "embed"
	"os"
	"path/filepath"
)

// Workspace seed assets — the default files written into a freshly created
// workspace. Real files (lintable, syntax-highlighted, previewable) embedded
// at compile time; one source of truth for what a new workspace contains.
//
// The remaining asset embeds are exported so the CLI's asset registry can
// offer them via `harbor asset add <name>` (install-if-absent) and
// `redeploy` (force-sync) — see internal/cli/asset_registry.go.

//go:embed seed/copy-code.js
var SeedCopyCodeJS string

//go:embed seed/fonts/inter-latin.woff2
var SeedInterLatinWOFF2 []byte

// seedWorkspaceDefaults writes the default workspace assets (JS/CSS, fonts)
// into the given layout's root. Existing files are preserved — the seed only
// writes when the target is absent, so re-running on an existing workspace
// won't clobber user edits.
func seedWorkspaceDefaults(layout Layout) error {
	// Text assets (shared JS helpers)
	files := []struct {
		path    string
		content string
	}{
		{layout.AssetPath("copy-code.js"), SeedCopyCodeJS},
	}
	for _, f := range files {
		if _, err := os.Stat(f.path); err == nil {
			continue // file exists — preserve
		}
		if err := writeToFile(f.path, f.content); err != nil {
			return err
		}
	}

	// Bigger assets (fonts, etc.)
	bins := []struct {
		path string
		data []byte
	}{
		{layout.AssetPath(filepath.Join("fonts", "inter-latin.woff2")), SeedInterLatinWOFF2},
	}
	for _, f := range bins {
		if _, err := os.Stat(f.path); err == nil {
			continue // file exists — preserve
		}
		if err := writeBytesToFile(f.path, f.data); err != nil {
			return err
		}
	}
	return nil
}
