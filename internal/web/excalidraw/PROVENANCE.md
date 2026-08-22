# Vendored excalidraw viewer (read-only)

Pinned upstream builds, downloaded from unpkg.com. Do not edit by hand —
re-download to upgrade.

| File | Source | Version |
|------|--------|---------|
| react.production.min.js | unpkg.com/react/umd/react.production.min.js | react@18.3.1 |
| react-dom.production.min.js | unpkg.com/react-dom/umd/react-dom.production.min.js | react-dom@18.3.1 |
| excalidraw.production.min.js | unpkg.com/@excalidraw/excalidraw/dist/excalidraw.production.min.js | @excalidraw/excalidraw@0.17.6 |
| excalidraw-assets/*.woff2 | unpkg.com/@excalidraw/excalidraw/dist/excalidraw-assets/ | @excalidraw/excalidraw@0.17.6 |

The bundle expects `window.React` / `window.ReactDOM` and resolves its font
assets relative to its own script URL (`/excalidraw/excalidraw-assets/...`),
which the server route preserves.
