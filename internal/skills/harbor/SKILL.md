---
name: harbor
description: Use harbor when a human asks you to save, organize, find, or review HTML pages an agent builds — keep a standalone HTML artifact for a human to view (harbor page add), group pages into or pull them out of a workspace (harbor workspace), tag pages (harbor tag), search the library (harbor search, page list/read), or run the dashboard (harbor start / stop). Use it too when asked to install, uninstall, or improve this skill (harbor skills).
---

# Harbor

A local library for agent-produced HTML pages: you build the page, Harbor
imports a copy, and a human browses, searches, and reviews it in a web
dashboard.

Write it `harbor` — the British spelling `harbour` is wrong.

The leading idea: **a page is worth keeping if a human will want to see it.**
When you make a standalone HTML artifact, `harbor page add` imports a copy into
a managed store — safe from temp-dir wipes and out of project folders — and
makes it searchable and browseable. Anything unregistered is simply not in the
library.

## Workflow (do these in order)

1. **Create the workspace first.** A page belongs in a named body of work:
   `harbor workspace create "<name>" --description "<what this work is>"`.
   Give every workspace a description — it powers disambiguation and search.
2. **Confirm it exists** with `harbor workspace list` — a typo'd workspace
   creates a phantom grouping.
3. **Create tags before pages** (tags cannot auto-create):
   `harbor tag create "<name>" --description "<why it exists>"`.
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
  prints the exact `harbor workspace create` command. Run that command yourself;
  the import won't auto-create the workspace behind your back.
- **Search first, then update.** Before making a new page, find an existing one
  by what it was for: `harbor search "<what it is about>"`. If a page already
  covers it, update it by slug instead of duplicating.
- **Status is manual** (`draft` / `published` / `archived`). Open feedback is
  derived from the comments queue — there is no stored "has feedback" flag.

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