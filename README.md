# Harbor

A CLI tool to organize and find the HTML pages your agent builds — plus a
read-only web dashboard to browse and open them.

Pages are imported once, grouped into **workspaces**, labeled with a status and
tags, and full-text indexed so you can search everything you've collected.

## Quick start

```bash
go install github.com/udit-001/harbor/cmd/harbor@latest

harbor init                            # create config + database in ~/.harbor
harbor workspace create "Payroll App"  # a workspace to group related pages
harbor page add dashboard.html \
  --workspace "Payroll App" \
  --description "monthly totals chart" \
  --tag finance
harbor start                           # open the web dashboard on :9090
```

## What you collect

A **page** is a self-contained HTML artifact your agent produced:

- Imported into a managed store — safe from temp wipes and project folders
- Kept **byte-for-byte** as the agent wrote it
- Carries provenance (description + context — where it came from and why it exists)
- Belongs to exactly **one workspace** and any number of **tags**
- Labeled **draft / published / archived**

A **workspace** is a named body of work pages belong to. It's a directory under
`~/.harbor/workspaces/` seeded with `RESOURCES.md` and `NOTES.md`, plus an
`assets/` directory for shared files.

## Web dashboard

`harbor start` opens the dashboard on `http://127.0.0.1:9090`:

- Sidebar of **workspaces** and **tags** with page counts
- **Status** filter (All / Draft / Published / Archived) as a pill switch
- **Live search** across titles, tags, and descriptions — all filtering is
  client-side, no page reloads
- A **page view** with container ⇄ full modes and prev/next within the current set
- Light / dark theme toggle, persisted per browser

## Commands

```
harbor init        # set up config + database
harbor workspace   # create / list / rename / delete workspaces
harbor page        # add / list / read / update / delete pages
harbor scrap       # global scratchpad scraps
harbor tag         # manage tags (name + semantic description)
harbor search      # full-text search across pages
harbor config      # read / set configuration
harbor start|stop  # run the web dashboard
harbor skills      # manage agent skills for this project
harbor migrate     # database migrations
harbor upgrade     # upgrade harbor to the latest release
```

## Data & privacy

All data is local: SQLite at `~/.harbor/harbor.db`, pages in the managed store
under the data directory. No telemetry, no accounts, no cloud.

## Development

```bash
make test            # go test ./... (real SQLite temp files, no mocks)
harbor build         # rebuild CSS + Go binary into bin/harbor
harbor dev           # hot-reload dev server on :9090
```

## Releases

Tag a version to trigger the GitHub release workflow:

```bash
git tag v0.1.0 && git push --tags
```

GoReleaser (`.goreleaser.yaml`) cross-compiles linux/darwin/windows ×
amd64/arm64 and produces archives, a checksums file, and Linux packages.