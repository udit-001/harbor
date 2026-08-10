# Harbor

Use `harbor` (single canonical spelling — never write `harbour`) when a human
asks you to save, organize, find, or review HTML pages it builds. Harbor is a
local library for agent-produced HTML pages: you create the pages, Harbor
imports a copy, and a human browses, searches, and reviews them in a calm web
dashboard.

The leading idea: **a page is worth keeping if a human will want to see it.**
When you make a standalone HTML artifact as part of any work, `harbor page add`
imports a copy into a managed store — safe from temp-dir wipes, out of project
folders — and makes it searchable and browseable. Anything you don't register
is simply not in the library.

## Workflow (do these in order)

1. **Create the workspace first.** A page belongs in a named body of work.
   `harbor workspace create "<name>" --description "<what this work is>"`.
   The description is the semantic payload that powers disambiguation and
   search — always give one.
2. **Check it exists** with `harbor workspace list`. Never assume; a typo'd
   workspace creates a phantom grouping.
3. **Create any tags first** (they cannot auto-create): `harbor tag create
   "<name>" --description "<why it exists>"`.
4. **Import the page** with provenance so it can be found later:
   ```
   harbor page add <your.html> --workspace <name> \
       --description "what the page shows" \
       --context "where it came from / why I made it" \
       --tag <tag1> --tag <tag2>
   ```
   - `description` = *what the page shows* (a reader summary).
   - `context` = *where it came from / why it exists* (provenance).
   - Both are full-text searched, so a future session can rediscover the page
     by describing it.
5. **Publish when it's ready:** `harbor page update <slug> --status published`.

## Deterministic behaviors

- `harbor page add` **fails fast** if the workspace doesn't exist — the error
  prints the exact `harbor workspace create` command to run. Run it; do not
  auto-create behind the tool's back.
- **Search first, then update.** Before making a new page, find an existing one
  by what it was for: `harbor search "<what it is about>"`. If a page already
  covers it, update it by slug instead of duplicating.
- **Status is manual** (`draft` / `published` / `archived`). Open
  feedback is derived from the comments queue — never set a "has feedback" flag.

## Finding things

- `harbor search "<query>"` — full-text across title, description, context,
  page body, and tag names/descriptions. Use it to find anything you built in a
  past session.
- `harbor page list [--workspace X] [--tag Y] [--status Z] [--search Q]`.
- `harbor page read <slug>` — full record (provenance, tags).

## Browser

`harbor start` serves the Library dashboard. Humans browse there; you operate
through the CLI above. Pages render byte-for-byte as made — Harbor never
restyles or injects into them.

## Empty states

The dashboard's empty states show the exact command that fills them. If a human
reports a blank library, run `harbor page add <file> --workspace <work>
--description "..."`.

## Editing this skill / the tool

The CLI and stores follow a **two-seam contract**:
- All data operations go through the single `db.Store` seam
  (`internal/db/pages.go`, `workspaces.go`, `scraps.go`). Tests there never
  mock — they run against real temp SQLite files (`pages_test.go`).
- CLI commands are thin Cobra wrappers over the store; CLI tests are
  HOME-sandboxed via `t.Setenv("HOME", t.TempDir())` so imports write to
  scratch space.
Keep the leading word `harbor`, the domain vocabulary (page / workspace / tag /
library), and the canonical-spelling rule intact when you edit.