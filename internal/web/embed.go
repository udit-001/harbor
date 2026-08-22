package web

import (
	"embed"
	_ "embed"
)

//go:embed app.css
var CSS []byte

// Excalidraw holds the vendored read-only viewer bundles (react, react-dom,
// @excalidraw/excalidraw 0.17.6 production UMD builds + font assets), served
// under /excalidraw/. Vendored so the binary stays self-contained — no CDN.
// Provenance: unpkg.com, pinned versions, see internal/web/excalidraw/PROVENANCE.md.
//
//go:embed all:excalidraw
var Excalidraw embed.FS

//go:embed favicon.ico
var FaviconICO []byte

//go:embed favicon.png
var FaviconPNG []byte

//go:embed favicon.svg
var FaviconSVG []byte

//go:embed icon-192.png
var Icon192 []byte

//go:embed icon-512.png
var Icon512 []byte

//go:embed manifest.webmanifest
var Manifest []byte

//go:embed sw.js
var ServiceWorker []byte

//go:embed stopped.html
var StoppedPage []byte
