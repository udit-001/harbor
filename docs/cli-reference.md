# CLI Reference

## Initialize

```bash
harbor init                              # Create config + SQLite database (idempotent)
harbor init --force                      # Recreate the database from scratch
```

## Web UI

```bash
harbor start                             # Start the read-only dashboard (background by default)
harbor start --port 9090                 # Custom port (explicit flag wins; else config port; default 9090)
harbor start --no-open                   # Don't auto-open the browser
harbor start --foreground / -f           # Run in the foreground
harbor start --dev-css                   # Serve CSS from disk (dev mode; run from project root)
harbor stop                              # Stop the running web server
harbor dev                               # Hot-reload dev server (live Go + CSS rebuild)
harbor dev --port 9090 --no-open         # Custom port; don't auto-open
```

If a server is already running, `harbor start` prints its URL and returns.

## Workspaces

A workspace is a named body of work. Pages belong to one workspace, and
when only one workspace exists most commands use it automatically without
`--workspace`.

```bash
harbor workspace list                          # List workspaces (current marked with *)
harbor workspace stats                         # Workspaces and their page counts
harbor workspace create "<name>"               # Create a workspace (auto-sets current)
harbor workspace create "<name>" --dir <path>  # Place the workspace elsewhere
harbor workspace create "<name>" --topic "<title>" --description "..."  # Display title + description
harbor workspace use "<name>"                  # Set as current workspace
harbor workspace current                       # Show current workspace
harbor workspace rename "<new name>" -w <name> # Rename (directory slug unchanged unless -w given)
harbor workspace delete "<name>"               # Delete workspace + its directory (prompts)
harbor workspace delete "<name>" --force       # Skip the confirmation prompt
```

## Pages

A page is an atomic artifact the agent produces: HTML, markdown, pdf, text,
svg, image, or excalidraw (format inferred from the source file; `--format`
overrides). Importing copies the file into the managed store
(`<data_dir>/store/<workspace>/<slug>.<format>`), safe from temp wipes, and
records provenance (description, context, tags) plus an FTS-indexed body.
Pages carry a status: draft / published / archived.

```bash
harbor page add dashboard.html --workspace income-tracker \
    --description "monthly totals chart" --context "prototype v2" --tag finance
harbor page add notes.md --workspace research --description "findings"
harbor page add diagram.png --workspace design
harbor page add report.html --workspace finance --title "Q3 Report" \
    --tag report --tag finance
harbor page list                              # List pages (FORMAT column)
harbor page list --workspace income-tracker --status published
harbor page list --tag finance --search "totals chart"
harbor page read <slug>                       # Metadata, tags, origin, body excerpt
harbor page update <slug> --description "..."
harbor page update <slug> --status published
harbor page update <slug> --tag finance --tag chart   # Replaces the full tag set
harbor page update <slug> --file <new.md>            # Replace the content (must match the page's format)
harbor page delete <slug>                     # Remove page + its managed file
```

`harbor page add` requires an existing workspace (`--workspace`); it fails fast
rather than auto-creating a phantom one. Pages without a description get a
warning — descriptions power search.

`harbor page update --file` is how content changes reach the **served copy**:
after `page add`, what the dashboard serves is the stored file, and `--file`
replaces it (copying the new file into the managed store under the page's
existing format, re-extracting body text, and bumping `updated_at`). Without
`--file`, `page update` only touches metadata.

## Tags

Tags are the scratchpad's first-class labels. A tag is a name plus a semantic
**description** (the description is what powers tag search and disambiguates
one tag from another).

```bash
harbor tag list
harbor tag create "ml" --description "machine learning career goal"
harbor tag update "ml" --description "revised description"
harbor tag delete "ml"                        # Detaches from all pages
```

## Search

```bash
harbor search "income tracker dashboard"
harbor search "chart" --workspace income-tracker --status published
harbor search --rebuild-index                 # Re-harvest body text for pages missing it (idempotent)
harbor search --rebuild-index --force         # Clear body_text first, re-index everything
```

Full-text search covers title, description, context, body text, and tag
names/descriptions. Search without `--rebuild-index` requires a `<query>`.

## Comments (feedback loop)

A comment is feedback anchored to a page (by text selection, element, or the
page — HTML pages support all three; other formats anchor to the whole page,
since there's no DOM inside them to select against). It lives in the database
and never edits the page file; the agent reads the open queue and acts on it.

```bash
harbor comments list                                # List open comments (default)
harbor comments list --page <slug>                  # One page
harbor comments list --status in-progress|done      # Other states
harbor comments watch                               # Tail new open comments as they arrive
harbor comments update <id> --status in-progress
harbor comments update <id> --status done           # Close once addressed
```

## Changes (what-changed walkthrough)

The agent answers a comment by editing the page, placing a
`data-cf-change="<id>"` marker on the edited element, and recording the
change — which the dashboard walks the human through ("What changed"). Markers
live in HTML pages; non-HTML formats have no walkthrough.

```bash
harbor change add <slug> --change-id cf-1 --title "Widen the chart" \
    --description "expanded the chart to full width" --comment 3
harbor change list <slug>                            # What the walkthrough will tour
```

`--change-id` and `--title` are required. A change whose marker isn't present
in the page HTML is skipped by the walkthrough; `change add` warns if it can't
find the marker.

## Navigate

```bash
harbor nav <url>              # Navigate the open dashboard tab to a URL
```

Broadcasts a "navigate" event to dashboard subscribers (used by the agent to
drive the browser). Exits with code 2 if no dashboard tab is open.

## Assets

Reusable components (stylesheets, scripts, images) in the workspace's
`assets/` directory. They're raw files — no DB tracking — referenced via
root-relative URLs (`assets/style.css`). `--workspace`/`-w` defaults to the
current workspace when only one exists.

```bash
harbor asset list -w "<name>"                        # Seeded / vendored / user assets
harbor asset create style.css -w "<name>" --body-file /tmp/style.css
```

`asset list` groups assets by source — **seeded** (universal defaults every
workspace starts with), **vendored** (third-party libs added on demand), and
**user** (authored via `asset create`) — shows the action for each, and prints
the absolute assets directory (assets are plain files; edit them in place).
The dashboard serves a workspace's assets to that workspace's pages:
reference them relatively (`<link href="assets/style.css">`) and
`/page/<slug>/assets/style.css` resolves against the page's own workspace.

## Configuration

The config file `harbor.toml` lives in the platform config dir
(`~/.config/harbor/` on Linux) and points at the data directory.

```bash
harbor config read                    # Show config file, data_dir, database, port
harbor config set data_dir ~/my-harbor
harbor config set port 8080
```

Supported keys: `data_dir`, `port` (1-65535). The port is an identity decision
(no auto-increment).

## Migrations

```bash
harbor migrate up                     # Apply all pending migrations
harbor migrate down                   # Roll back the most recent migration
harbor migrate up-to <version>        # Run migrations up to a specific version
harbor migrate down-to <version>      # Roll back to a specific version
harbor migrate status                 # Show migration status
```

## Skills

```bash
harbor skills check                   # Check installed skills and their status
harbor skills install                 # Interactively install into detected AI agents
harbor skills install --project       # Install at project level (not global)
harbor skills install --agents-only   # Only .agents/skills (opencode, codex, pi.dev)
harbor skills install --claude-only   # Only .claude/skills (claude-code)
harbor skills uninstall --all         # Remove all discovered installs
```

## Maintenance

```bash
harbor tailwind download              # Download the Tailwind CLI binary to .bin/tailwindcss
harbor tailwind download --force      # Re-download even if present
harbor tailwind download --version v4.3.1
harbor build                          # Rebuild CSS + templ + Go binary into bin/harbor
harbor build --no-css                 # Go-only build (skip CSS/templ rebuild)
harbor upgrade                        # Upgrade via 'go install ...@latest'
harbor upgrade --no-skills            # Skip the skill-upgrade prompt
harbor version                        # Show version
```

`harbor build` requires `go` on PATH and the Tailwind CLI at `.bin/tailwindcss`
(see `harbor tailwind download`). It runs `templ generate` when the templ CLI
is present, otherwise uses the committed `*_templ.go`.

## Global Flags

```bash
--json      # Machine-readable JSON output (most commands)
```

## File Naming

Imported pages are stored in the managed store as real files, named
`<slug>.<format>` where the extension is the page's format:

| Type        | Location                                          | Example                     |
|-------------|---------------------------------------------------|-----------------------------|
| Artifact    | `<data_dir>/store/<workspace>/<slug>.<format>`    | `monthly-totals.html`, `notes.md` |
| Asset       | `<workspace>/assets/<filename>`                   | `style.css`                 |

## Workspace Layout

```
<workspace>/
└── assets/              # Reusable components (seeded: copy-code.js, fonts/) + user assets
```

Pages do **not** live in the workspace directory — they're imported into the
managed store (`<data_dir>/store/<workspace>/`) and referenced by the dashboard.
The stored file is the artifact; the dashboard derives its view (raw bytes,
rendered markdown/text, or the embedded excalidraw viewer) from the format.

## Data

- SQLite database at `~/.harbor/harbor.db` (configurable via
  `harbor config set data_dir`).
- FTS5 full-text search (Porter tokenizer) on pages.
- Config at `~/.config/harbor/harbor.toml`; server PID at
  `~/.config/harbor/server.pid`.
- All mutations happen through the CLI — the web UI is read-only.