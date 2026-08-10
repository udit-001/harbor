# PLAN — Harbor

A local, agent-driven library for HTML pages. An AI agent produces standalone HTML
pages across its work; Harbor imports, organizes, and serves them so a human can
browse, search, review, and comment. Landing a page costs one command; finding it
again later costs one search.

> Product name: **harbor** (single canonical spelling; never "harbour").
> This repo is the result of **Option 1**: rework the Harbor codebase — reuse
> its plumbing (cobra/templ/FTS/iframe-frame-mode), strip the learning-only
> surface, and rebuild the domain model around `pages`.

---

## 1. Problem & goal

- Agent builds HTML pages as part of many bodies of work. They land in temp dirs
  that get wiped, or in projects where they pollute the tree.
- Neither Trilium nor the project folders give local HTML pages a browsable,
  searchable home.
- Goal: a calm, curated home for agent-produced pages — organized by work and by
  tag, full-text searchable, viewable in a neutral full-bleed shell, and
  revisable through a human→comment→agent loop.

## 2. Design principles (inherited + repurposed)

1. **The page is the product, not the frame.** Harbor never restyles, wraps, or
   injects into a page. Pages are served byte-for-byte as the agent wrote them.
2. **Production is agent-driven; curation is human.** The agent writes; the
   human browses, searches, reviews, comments (v1 read-only).
3. **Don't lose the work.** Anything worth showing a human is imported into the
   managed store — safe from temp wipes, away from project pollution.
4. **Findability by provenance.** Every page carries enough "what/where/why" to
   resurface later, for both a human and a future agent session.
5. **Calm, focused, expert.** No dopamine theater (inherited from Harbor).
6. **Empty states teach the next command,** so the tool is one prompt from useful.

Dropped from Harbor entirely: in-page theme toggle, iframe theme-sync, seeded
`seed/style.css`, Nord token duality in the page path, lessons/quizzes/records/
missions/glossary, mermaid/vega/katex/highlight assets, and the in-page tooling
that wraps agent content in a styled container.

## 3. Conceptual model & vocabulary

First-class objects (own terms, not Harbor's):

| Object | Definition | Key attributes |
|---|---|---|
| **Page** | A standalone HTML artifact the agent produced, owned by Harbor | `title` (defaults to filename), `description`, `context`, `workspace`, `tags`, `status`, `origin_path` (soft), timestamps, slug |
| **Workspace** | A named body of work; the organizing folder | `name` (slugified), `description`, page count |
| **Tag** | Global semantic label, cross-cutting | `name`, `description` |
| **Library** | The read surface: browse + search + filter all pages | n/a |
| **Status** | Page readiness: `draft` / `published` / `archived` | manual |
| **Comment** | Human feedback anchored to a page | `page`, `anchor` (CSS selector), `quote`, `type`, `body`, `status` |
| **Change** | Agent's response record, joined to comments + `data-cf-change` | `page`, `comment_id`, `change_id` |

Relationships: **Workspace (1) → Page (n); Tag (n) ↔ Page (n); Page (1) → Comment (n); Comment (1) → Change (0..1).**

Provenance (agent-findability): four single-meaning fields, all FTS-indexed —
`description` (what the page shows), `context` (where it came from / why),
`workspace` (which body of work), `tags`. Plus a soft `origin_path` (as-was,
never dereferenced). **No origin bucket** (overlaps `context`).

Status is manual (`draft`/`published`/`archived`); **"has open feedback" is
derived** from the comments queue, never a manual flag.

## 4. Storage model

- **Pages = real `.html` files in the managed store** (`~/.harbor/store/<work>/…`),
  owned by Harbor. The database holds only metadata + an FTS index over
  extracted body text. `page add` **imports/copies** the source into the store.
- **Single storage path: `managed`.** No `referenced`/disposition/re-sync
  machinery in v1. Rule: *if a page is worth showing a human, it's worth
  importing.* A page the agent doesn't register is simply not in the library.
- DB (SQLite): `workspaces`, `tags`, `pages`, `page_tags`, `comments`, `changes`.
  Indexable/`CREATE VIRTUAL TABLE … fts5` over page text fields.
- Read-only for humans in v1 (agent owns writes).

## 5. Command surface (CLI verb: `harbor`)

```
harbor init                                  # create store + DB + install the skill
harbor start / stop                          # localhost server for the library
harbor workspace create <name> [--description ...]
harbor workspace list | rename | delete | stats
harbor page add <source> --workspace <name>
              [--title ...] [--description ...] [--context ...] [--tag ...]
harbor page list [--workspace ...] [--tag ...] [--status ...] [--search ...]
harbor page read <slug>
harbor page update <slug> [--title ...] [--description ...] [--status ...]
harbor page delete <slug>
harbor search <query> [--rebuild-index]
# M2:
harbor comments list [--page <slug>] [--status open]
harbor comments watch                       # tail new open comments
harbor comments update <id> [--status ...]
harbor tag create/list/update/delete
harbor skills install                       # install the agent skill
```

Deterministic behaviors:
- `page add --workspace X` **fails fast** if `X` does not exist; the error prints
  the exact `harbor workspace create X --description …` to run. No auto-create
  (prevents phantom/typo workspaces).
- `page add` ends **checkably** (prints slug) and **warns when `description` is
  empty** — an nudge to write the metadata that powers later search.
- Search is broad (all text fields); field definitions are strict (one meaning).
- Canonical spelling `harbor` enforced as a hard rule in the skill; `harbour`
  never emitted.

## 6. Rendering & the Library (human surface)

- **Two zones:** the *Library* (decide: browse/search/filter) and the *Page*
  (consume). Chrome is the Library's job; the page is the product.
- **Opening a page** renders it in a **neutral, full-bleed iframe** (Harbor
  "frame mode"): page as designed, no restyle, no injected chrome. Same-origin
  so the shell can read the page DOM.
- **View modes** (CodePen-style, per-page, `localStorage` keyed by slug):
  - `full` — immersive; auto-hide chrome (back, workspace/tags/status, prev/next,
    pop-out). No search in-page.
  - `container` (default fallback) — content column inside the shell; metadata
    visible (workspace/tags/status, description), prev/next, comment access.
  - Switching modes never reloads/restyles the page — only the envelope resizes.
  - `prefers-reduced-motion` fallback on all transitions.
- **Store per-page viewer preference in `localStorage`** (`{slug: mode}` +
  `defaultView = container` for never-opened pages). Toggling sets only that page.
- Shell look = calm authority (Harbor/Nord heritage) but **only for the shell**;
  pages are untouched.

## 7. Feedback loop (M2) — no injection

- The **server does not inject anything into served pages** (verified intent: a
  page is byte-for-byte the agent's). Commenting logic lives in **the shell's
  JS**, which reads the page DOM across the same-origin iframe boundary.
- Human clicks/highlights inside the iframe; the shell captures the element's
  existing `id`/selector + quote; comment typed in a **shell panel**.
- `comments` + `changes` live in the SQLite DB (not per-page JSON). Agent writes
  `data-cf-change` markers into the HTML only as an authoring convention when it
  responds; the shell reads them back for a "what changed" walkthrough.
- Agent CLI: `comments list` / `comments watch` (real tailing, prints
  `page / quote / body` as they arrive; blocks until Ctrl-C) / `comments update`.
- Tradeoff: pop-out (own tab) loses live walkthrough (shell has no reach) —
  accepted.

## 8. Architecture & reuse map (from Harbor)

**Reuse (adapt):** cobra CLI + per-command files (`internal/cli/`); `templ`
rendering (`internal/render/`); SQLite store seam (`internal/db/` — the
`Store` single-seam pattern); FTS query + index-refresh idempotency; workspace
store / scrap+tag store (rework into `pages`+`tags`+`workspaces`); server +
daemon + `start`/`stop` (`internal/server`, `internal/cli/start.go`); frame-mode
full-bleed iframe (`frame.go` frameMaxWidthClass); command palette; empty-states
pattern; `Page`-file-on-disk + `body_text` FTS extraction (`internal/extract`).

**Strip:** `lessons`/`quizzes`/`learning_records`/`references`/`glossary_terms`/
`source_docs`; `mission`; `asset add` for mermaid/vega/katex/highlight;
`mermaid-theme.js`/`vega-theme.js`/`katex-render.js`/`glossary-tooltip.js`;
theme-sync JS (`harbor-theme.js`) + in-page seed stylesheet; quiz/attempt/record
commands + views; any `data-theme`/`postMessage` page-path code.

**Add:** `pages`/`workspaces`/`comments`/`changes` tables + stores; view-mode
(full/container) shell + per-page pref; Library home (grid/list) with
workspace/tag/status filters + search; provenance capture on `page add`;
`comments list/watch/update`; skill for the agent; `page`/`workspace`/`comments`
command groups; static-export (stretch, v2).

**Rename:** module `github.com/udit-001/harbor` → `github.com/udit-001/harbor`
(placeholder); binary `harbor` → `harbor`; select `internal/*` naming.

## 9. Milestones

### M1 — Core library (usable now)
1. Set up module/rename; copy Harbor tree; strip learning-only files & commands.
2. Build `pages`/`workspace`/`tag` stores + migrations; rework scrap+tag stores.
3. `harbor workspace …` group (+ fail-fast required for `page add`).
4. `harbor page add/list/read/update/delete/delete`; provenance capture; FTS
   indexing + `search`.
5. Library home: browse grid/list, search, workspace/tag/status filters.
6. Page view: neutral full-bleed iframe (frame mode), view-mode toggle
   (full/container) + per-page localStorage pref.
7. `skills install` + agent skill for producing/registering/finding pages.
8. Empty states teach every next command; WCAG-AA; reduced-motion.

### M2 — Feedback loop
9. `comments`/`changes` tables + stores + CLI (`list`/`watch`/`update`).
10. Shell-side comment panel; same-origin anchor read; submit → DB.
11. Agent edits page + writes `data-cf-change`; shell "what changed" tour.
12. Counts/badges derived from open feedback; status polish.

### Stretch (v2, separate)
- Static export of library + pages for sharing/hosting.
- Optional per-page `referenced` disposition if a project should own the artifact.

## 10. Open next steps
- Git-init `html-organizer`; copy Harbor tree as the starting skeleton.
- Confirm module path (placeholder `github.com/udit-001/harbor`) before pushing.
- Define exact DB schema + migration versioning in the first M1 increment.
- Wire the `harbor` skill name + canonical-spelling rule into the agent skill.