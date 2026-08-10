# Harbor

CLI + read-only web dashboard for organizing and finding the HTML pages your
agent builds.

## Commands

- `harbor tailwind download` — download the Tailwind CLI binary to `.bin/tailwindcss`
- `harbor build` — rebuild CSS + Go binary into `bin/harbor`
- `make test` — `go test ./...` (real SQLite temp files, no mocks)
- `harbor start` / `harbor stop` / `harbor dev` — run / stop / hot-reload the web dashboard
- `harbor page`, `harbor workspace`, `harbor search`, `harbor scrap`, `harbor tag` — CLI surface

## Layout

- `cmd/harbor/` — the `main` package (module root has none). Install via
  `go install github.com/udit-001/harbor/cmd/harbor@latest`.
- `internal/cli/` — Cobra commands, one file each.
- `internal/server/` — `NewMux(store, dataDir, devCSS)` HTTP mux (the testable seam).
- `internal/render/` — all HTML output.
- `internal/db/` — SQLite via sqlx; `internal/migrate/` is goose-managed `*.sql`.
- `internal/{config,docutil,extract,markdown,skills,urls,version,web}` — supporting packages.

## Rendering

Two distinct families in `internal/render/`:

- **Library home + page view** (the primary surface) are *self-contained string
  builders* with inline CSS + JS — `library.go` (the `/` Library shell) and
  `pageview.go` (the `/page/{slug}` view + raw iframe). No templ, no Tailwind.
- **Workspace / doc pages** use `templ` components (`frame.templ`, `views.templ`)
  compiled to committed `*_templ.go`. Their CSS comes from the Tailwind-built
  `internal/web/app.css` (`//go:embed`'d). `*.templ` → run `templ generate`.

### Client-side filtering

The library filters entirely in the browser: `/api/pages` returns the *full*
page set as JSON, and the page's JS narrows it by status pill, search, and
sidebar workspace/tag — no reloads. Filters stay in the URL via
`history.pushState`; `popstate` re-applies them (parse query params back into
the state object — assigning a `URLSearchParams` directly breaks `state.q`).

### Page view modes (container / full)

`/page/{slug}` renders the page in a seamless iframe. Container keeps the header
sticky and visible; **full** fills the viewport with a floating header that
slides in from the top on a ~60px top "peek" band and dismisses on scroll /
interaction / pointer-down inside the iframe (wired via capture listeners on the
iframe's `contentWindow`). View mode persists per slug in
`localStorage['harbor_view_<slug>']`.

### Theming

Ligh/dark is driven by `data-theme` on `<html>` and `localStorage['harbor_theme']`
(set in `<head>` before paint). All three surfaces (library, page view, frame)
define a `[data-theme="dark"]` block overriding the same CSS variables
(`--bg`, `--surface`, … Nord palette). Keep the names consistent across surfaces.

## Conventions & gotchas

- Run `harbor stop && harbor build && harbor start` after rebuilding; `harbor build`
  writes `bin/harbor` (gitignored). The PATH binary is `~/go/bin/harbor` — after a
  rebuild, `cp bin/harbor ~/go/bin/` so the running server picks up new code.
- **`--font` is a real CSS variable** in `internal/render/library.go` /
  `pageview.go` `:root` — if dropped, `font: <weight> <size> var(--font)`
  shorthands become invalid and every "small" label collapses to inherited 15px,
  making the whole shell look oversized.
- `internal/migrate/*.sql` are goose v3 migrations with **checksum verification**:
  never edit an already-applied migration (even a comment) — it breaks existing DBs.
- `scripts/gen-winres.sh` regenerates Windows PE metadata into
  `cmd/harbor/*.syso` (committed) from `winres/winres.json` + `winres/harbor.ico`.
- Pre-commit hook in `.githooks/pre-commit` runs `gofmt` on staged `.go`; CI runs
  `test -z "$(gofmt -l .)"`. Install with `git config core.hooksPath .githooks`.
- **Never commit browser screenshots.** Playwright artifacts land in
  `.playwright-mcp/` (gitignored); manual debug shots saved to the repo root
  must be added to `.gitignore`.

## Releases

`git tag v0.x.x && git push --tags` triggers `.github/workflows/release.yml`.
`.goreleaser.yaml` cross-compiles linux/darwin/windows × amd64/arm64 and produces
archives, `checksums.txt`, and Linux packages. Cosign keyless signing of
`checksums.txt` is enabled. Local dry-run: `goreleaser release --snapshot --clean`.

## Repo

`github.com/udit-001/harbor`