# Harbor

Your agent builds dashboards, reports, diagrams, drawings — then they vanish
into temp folders. Harbor gives every artifact a home: HTML pages, markdown,
PDFs, SVGs, images, and excalidraw files import once, stay full-text
searchable, and open in a local read-only dashboard where your comments loop
back to the agent.

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

- **Page** — one stored artifact (HTML page, markdown doc, PDF, SVG or image,
excalidraw drawing), kept byte-for-byte with its provenance. Draft / published /
archived.
- **Workspace** — the body of work a page belongs to (a directory seeded with
  `RESOURCES.md`, `NOTES.md`, and `assets/`).
- **Tags** — cross-workspace labels.

## Dashboard

`harbor start` serves the dashboard locally:

- **Find** — sidebar of workspaces and tags, status pills, live full-text search
- **Read** — container ⇄ full page view, prev/next within the current set
- **Comment** — anchored feedback: select text or an element in an HTML page
- **Review** — the what-changed walkthrough replays recorded `harbor change` entries
- **Theme** — light / dark, persisted

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
