---
name: harbor
description: Use harbor when you build a standalone HTML page — register it (harbor page add), group it into a workspace (harbor workspace), or tag it (harbor tag); also when a human asks you to save, organize, find, or review pages (harbor search, page list/read), to run the dashboard (harbor start / stop), or to answer page feedback (harbor comments list/watch, harbor change add).
---

# Harbor

A local library for agent-produced HTML pages: you build the page, Harbor
imports a copy, and a human browses, searches, and reviews it in a web
dashboard.

Write it `harbor` — the British spelling `harbour` is wrong.

The leading idea: **any standalone HTML page you build belongs in harbor.**
Register it the moment you make one — `harbor page add` imports a copy into a
managed store (safe from temp-dir wipes, out of project folders) and makes it
searchable and browseable. The bar for importing is low and automatic: if a
human would want to see or keep it, it goes in the library without waiting to be
asked. A page you made that isn't in harbor is the exception, not the rule.

## Workflow (do these in order)

**First, the gate — decided before importing:** is it a standalone HTML page a
human would want to see or keep? That's the worth-keeping test. Yes → it belongs
in the library (import it). A truly throwaway fragment (a one-off scratch
layout, a dead-end mockup) → skip harbor; don't pollute the library.

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

- **Every standalone HTML page you build belongs in harbor by default.** Import
  it the moment you produce one — don't wait to be asked. The worth-keeping
  test is the low, automatic gate: if a human would want to see it, it's in.
- `harbor page add` **fails fast** if the workspace doesn't exist — the error
  prints the exact `harbor workspace create` command. Run that command yourself;
  the import won't auto-create the workspace behind your back.
- **Search first, then update.** Before making a new page, find an existing one
  by what it was for: `harbor search "<what it is about>"`. If a page already
  covers it, update it by slug instead of duplicating.
- **Status is manual** (`draft` / `published` / `archived`). Open feedback is
  derived from the comments queue — there is no stored "has feedback" flag.

## Responding to feedback

Humans leave anchored comments on your pages; you answer them by editing the
page. Pull the open queue with `harbor comments list` (defaults to open; each
row shows `page / quote / body`). Do that once, handle every comment, then
re-list to confirm the queue is empty. Never run a blocking tail (`harbor
comments watch` blocks until Ctrl-C) as an agent step — snapshot with `list`
instead.

For each open comment, in order:

1. **Read it** — the `harbor comments list` row shows
   `page / quote / body` so you know exactly what to change.
2. **Edit the page** to address it.
3. **Mark the changed element** with a stable id on the element you edited:
   `data-cf-change="<id>"` (a short slug, e.g. `cf-1`).
4. **Record the change** so the human is walked through what you did:
   `harbor change add <slug> --change-id <id> --title "what I changed" \
   --description "why" [--comment <commentId>]`
5. **Close the comment** once addressed: `harbor comments update <commentId> --status done`.

Repeat until re-running `harbor comments list` returns no open comments — that
queue-empty check is when the work is done.

The change's `--title`/`--description` is what the walkthrough shows — write it
for someone who hasn't seen the diff. `--description` optional; `--title` is
required.

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