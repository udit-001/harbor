package extract

import (
	"path/filepath"
	"strings"
)

// The canonical artifact-format vocabulary. Every consumer — db validation,
// CLI inference, serving content types — reads from here; adding a format is
// one edit in this file plus whatever serving behavior it needs.
const (
	ArtifactHTML       = "html"
	ArtifactMarkdown   = "markdown"
	ArtifactPDF        = "pdf"
	ArtifactText       = "text"
	ArtifactSVG        = "svg"
	ArtifactImage      = "image"
	ArtifactExcalidraw = "excalidraw"
)

// AllArtifactFormats is the closed set, in family order.
var AllArtifactFormats = []string{ArtifactHTML, ArtifactMarkdown, ArtifactPDF, ArtifactText, ArtifactSVG, ArtifactImage, ArtifactExcalidraw}

// artifactFormats maps file extensions (lower-cased, with dot) to the artifact
// formats Harbor's library accepts. These are the byte formats the learn-tool
// detector never needed; they join the source-document formats in ArtifactFormat.
var artifactFormats = map[string]string{
	".svg": "svg",
	".png": "image", ".jpg": "image", ".jpeg": "image", ".gif": "image", ".webp": "image", ".bmp": "image",
	".excalidraw": "excalidraw",
}

// ArtifactFormat resolves an artifact's format from its source file:
// explicit override wins, then extension, then the Detect() sniffing
// (extension + magic bytes) the learn-tool extraction already uses.
// Returns "" when the format cannot be determined — callers decide whether
// that's an error. The knowledge of "which file kinds exist" lives here, in
// the extract package; nothing upstream re-derives it.
func ArtifactFormat(source, override string) string {
	if override != "" {
		return override
	}
	if f, ok := artifactFormats[strings.ToLower(filepath.Ext(source))]; ok {
		return f
	}
	switch Detect(source) {
	case FormatHTML:
		return ArtifactHTML
	case FormatMarkdown:
		return ArtifactMarkdown
	case FormatPDF:
		return ArtifactPDF
	case FormatText:
		return ArtifactText
	}
	return ""
}

// ArtifactBodyText pulls the FTS body for an artifact from its raw bytes.
// Text-bearing formats reuse the learn-tool extractors (html strips markup,
// svg is xml so the same stripper works, pdf/md/txt go through FromFile);
// binary image formats and excalidraw scenes return no prose — their title
// and description carry the searchable signal. Callers pass the bytes they
// already read; nothing is read twice.
func ArtifactBodyText(source, format string, data []byte) string {
	switch format {
	case "html", "svg":
		return FromHTML(string(data))
	case "text", "markdown", "pdf":
		if r, err := FromFile(source); err == nil {
			return r.Text
		}
	}
	return ""
}

// ValidArtifactFormat reports whether f is one of Harbor's artifact formats.
func ValidArtifactFormat(f string) bool {
	for _, known := range AllArtifactFormats {
		if known == f {
			return true
		}
	}
	return false
}
