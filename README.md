# Harbor

Your agent builds HTML pages; they vanish into temp folders. Harbor keeps
them: import once, find any of them later — plus a read-only dashboard to
browse, comment, and review them.

```bash
go install github.com/udit-001/harbor/cmd/harbor@latest

harbor init                            # config + database in ~/.harbor
harbor workspace create "Payroll App"  # group related pages
harbor page add dashboard.html \
  --workspace "Payroll App" \
  --description "monthly totals chart" \
  --tag finance
harbor start                           # dashboard on :9090
```

## The model

- **Page** — one self-contained HTML artifact, stored byte-for-byte with its
  provenance (description, where it came from). Draft / published / archived.
- **Workspace** — the body of work a page belongs to (a directory seeded with
  `RESOURCES.md`, `NOTES.md`, and `assets/`).
- **Tags** — cross-workspace labels.

## Dashboard

`harbor start` serves the dashboard locally:

- Sidebar of workspaces and tags, status pills, live search
- Container ⇄ full page view, prev/next within the current set
- Anchored comments — select text or an element to leave feedback
- What-changed walkthrough — replays recorded `harbor change` entries against the page
- Light / dark theme, persisted

## Commands

```
harbor page        # add / list / read / update / delete pages
harbor workspace   # create / list / rename / delete workspaces
harbor comments    # list / watch / update anchored feedback
harbor change      # record a what-changed walkthrough entry
harbor tag         # manage tags
harbor search      # full-text search across pages
harbor config      # read / set configuration
harbor start|stop  # run the web dashboard
harbor skills      # manage agent skills
harbor upgrade     # self-update
```

Full reference: [docs/cli-reference.md](docs/cli-reference.md)

## Data

All local: SQLite at `~/.harbor/harbor.db`, pages in the managed store.
No account, nothing uploaded.

## Contributing

Setup, architecture, and the release ritual live in
[docs/project-setup.md](docs/project-setup.md).
