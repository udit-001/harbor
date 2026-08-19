package extract

import (
	"path/filepath"
	"strings"
)

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
		return "html"
	case FormatMarkdown:
		return "markdown"
	case FormatPDF:
		return "pdf"
	case FormatText:
		return "text"
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
// Kept beside ArtifactFormat so the format vocabulary has one home.
func ValidArtifactFormat(f string) bool {
	switch f {
	case "html", "markdown", "pdf", "text", "svg", "image", "excalidraw":
		return true
	}
	return false
}
