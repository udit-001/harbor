# CLI Reference

## Initialize

```bash
harbor init                              # Create database and run migrations
harbor init --force                      # Recreate database from scratch
```

## Dev Server (hot-reload)

```bash
harbor dev                               # Start dev server with live Go + CSS rebuild
harbor dev --port 9090                   # Custom port
harbor dev --no-open                     # Don't auto-open browser
```

## Web UI

```bash
harbor start                             # Start read-only web dashboard (default :9090)
harbor start --port 9090                 # Custom port (auto-increments if busy)
harbor start --no-open                   # Don't auto-open browser
harbor start --foreground / -f           # Run in foreground
harbor start --background / -b           # Run in background (default)
harbor start --dev-css                   # Serve CSS from disk (dev mode)
harbor stop                              # Stop the running web server
```

## Workspaces

```bash
harbor workspace create "<name>"         # Create a new workspace
harbor workspace create "<name>" --dir <path>    # Create at custom path
harbor workspace create "<name>" --topic "<title>"  # Override display title

harbor workspace list                    # List all workspaces
harbor workspace stats                   # Show learning statistics (with bar charts)
harbor workspace use "<name>"            # Set as current workspace
harbor workspace current                 # Show current workspace
harbor workspace rename "<new name>"     # Update display name (directory slug unchanged)
harbor workspace delete "<name>"         # Delete workspace and directory
harbor workspace delete "<name>" --force # Skip confirmation prompt
```

## Lessons

```bash
harbor lesson create "<title>" -w "<workspace>" --body-file <path>   # Create lesson with content
harbor lesson list -w "<workspace>"                                   # List lessons
harbor lesson list -w "<workspace>" --search "<q>"                    # Search lessons
harbor lesson read <seq> -w "<workspace>"                             # Read lesson content
harbor lesson read <seq> -w "<workspace>" --meta-only                 # Show metadata only
harbor lesson show <seq> -w "<workspace>"                             # Open in web dashboard
harbor lesson revise <seq> -w "<workspace>" --body-file <path>        # Revise lesson content
harbor lesson revise <seq> -w "<workspace>" --title "<new>"           # Update lesson title
harbor lesson revise <seq> -w "<workspace>" --summary "<new>"         # Update lesson summary
```

## Learning Records

```bash
harbor record create "<title>" -w "<workspace>" --body-file <path>    # Create a learning record
harbor record create "<title>" -w "<workspace>" --body-file <path> \  # With summary
  --summary "..."
harbor record list -w "<workspace>"                                    # List records
harbor record list -w "<workspace>" --search "<q>"                     # Search records
harbor record read <seq> -w "<workspace>"                              # Read record content
harbor record read <seq> -w "<workspace>" --meta-only                  # Show metadata only
harbor record show <seq> -w "<workspace>"                              # Open in web dashboard
harbor record supersede <seq> -w "<workspace>" --title "<new>" \      # Supersede with new understanding
  --body-file <path>
```

## References

```bash
harbor reference create "<title>" -w "<workspace>" --body-file <path>  # Create a reference
harbor reference list -w "<workspace>"                                  # List references
harbor reference list -w "<workspace>" --search "<q>"                   # Search references
harbor reference read <slug> -w "<workspace>"                           # Read reference content
harbor reference read <slug> -w "<workspace>" --meta-only               # Show metadata only
harbor reference show <slug> -w "<workspace>"                           # Open in web dashboard
harbor reference revise <slug> -w "<workspace>" --body-file <path>     # Revise reference content
harbor reference revise <slug> -w "<workspace>" --title "<new>"        # Update reference title
harbor reference revise <slug> -w "<workspace>" --summary "<new>"      # Update reference summary
```

## Workspace Documents

```bash
harbor mission read -w "<workspace>"                                 # Read mission
harbor mission read -w "<workspace>" --json                          # Read mission as JSON
harbor mission edit -w "<workspace>"                                 # Edit mission in $EDITOR
harbor mission edit -w "<workspace>" --body-file <path>               # Write mission from file

harbor resources read -w "<workspace>"                                # Read resources
harbor resources read -w "<workspace>" --json                         # Read resources as JSON
harbor resources edit -w "<workspace>"                                # Edit resources in $EDITOR
harbor resources edit -w "<workspace>" --body-file <path>             # Write resources from file

harbor notes read -w "<workspace>"                                    # Read notes
harbor notes read -w "<workspace>" --json                             # Read notes as JSON
harbor notes edit -w "<workspace>"                                    # Edit notes in $EDITOR
harbor notes edit -w "<workspace>" --body-file <path>                 # Write notes from file
harbor notes edit -w "<workspace>" --append --body-file <path>        # Append to notes

harbor glossary list                                                  # Show glossary
harbor glossary list --json                                           # Show glossary as JSON
harbor glossary create "<term>" "<definition>" -w "<workspace>"       # Add or update a term
harbor glossary create "<term>" "<definition>" --category "<name>"    # Group under a heading
harbor glossary create "<term>" "<definition>" --avoid "<synonym>"    # Flag a synonym to avoid
harbor glossary delete "<term>" -w "<workspace>"                      # Remove a term (idempotent)
```

## Assets

```bash
harbor asset list -w "<workspace>"                  # List workspace assets (seeded, vendored, user)
harbor asset create <filename> -w "<workspace>" --body-file <path>  # Create or overwrite asset file
harbor asset add <name> -w "<workspace>"            # Install a vendored/seeded asset (skips if present)
harbor asset redeploy <name> -w "<workspace>"       # Force-sync asset to current binary version
harbor asset delete <filename> -w "<workspace>"     # Remove an asset file
```

## Questions

Questions are DB-only entities used to build quizzes. Two modes:
- **choice** — `--body-file` is JSON `{"options": [...], "key": N}` (0-based correct index)
- **recall** — `--body-file` is the reveal text shown after self-grading

```bash
harbor question create "<title>" -w "<ws>" --mode choice --body-file <path>   # Create a choice question
harbor question create "<title>" -w "<ws>" --mode recall --body-file <path>   # Create a recall question
harbor question create "<title>" -w "<ws>" --mode choice --body-file <path> --stimulus-file <html>  # With stimulus

harbor question list -w "<workspace>"               # List questions
harbor question list -w "<workspace>" --weak        # Sort by accuracy ascending (struggles first)
harbor question list -w "<workspace>" --limit N     # Max results
harbor question read <slug> -w "<workspace>"        # Read question content and metadata
harbor question revise <slug> -w "<workspace>" --title "<new>"       # Update title
harbor question revise <slug> -w "<workspace>" --body-file <path>    # Update config
harbor question revise <slug> -w "<workspace>" --mode recall --body-file <path>  # Change mode
harbor question revise <slug> -w "<workspace>" --stimulus-file <path>  # Add/replace stimulus
harbor question revise <slug> -w "<workspace>" --clear-stimulus       # Remove stimulus
harbor question delete <slug> -w "<workspace>"      # Delete (blocks if a quiz references it)
```

## Quizzes

Quizzes are DB-only ordered lists of question slugs. Taken in the web dashboard.

```bash
harbor quiz create "<title>" -w "<ws>" --items "slug1,slug2"             # Create a quiz
harbor quiz create "<title>" -w "<ws>" --items "slug1" --description "..."  # With description
harbor quiz create "<title>" -w "<ws>" --items "slug1" --lesson <seq>    # Link to a lesson

harbor quiz list -w "<workspace>"                   # List quizzes with best scores
harbor quiz list -w "<workspace>" --weak            # Sort by lowest accuracy first
harbor quiz list -w "<workspace>" --limit N         # Max results
harbor quiz read <slug> -w "<workspace>"            # Read quiz metadata and items
harbor quiz show <slug> -w "<workspace>"            # Open quiz in web dashboard
harbor quiz revise <slug> -w "<ws>" --items "slug1,slug2"  # Update items (blocks if in-progress attempts)
harbor quiz revise <slug> -w "<ws>" --lesson <seq>         # Link/unlink lesson (0 to unlink)
harbor quiz attempts <slug> -w "<workspace>"        # Show attempt history and trend
harbor quiz delete <slug> -w "<workspace>"          # Delete (blocks if in-progress attempts)
```

## Migrations

```bash
harbor migrate up                     # Apply all pending migrations
harbor migrate down                   # Roll back most recent migration
harbor migrate up-to <version>        # Run migrations up to a specific version
harbor migrate down-to <version>      # Roll back to a specific version
harbor migrate status                 # Show migration status
```

## Search

```bash
harbor search "<query>"                          # Search across all workspaces
harbor search "<query>" -w "<workspace>"         # Search within one workspace
harbor search --rebuild-index                     # Index the current workspace's content
harbor search --rebuild-index --all               # Rebuild index across all workspaces
```

## Configuration

```bash
harbor config read                                # Read current configuration
harbor config set data_dir ~/my-harbor            # Change the data directory
harbor config set auto_submit_choice on           # Auto-submit choice questions on selection
```

## Skills

```bash
harbor skills install                 # Interactively install harbor skill into AI agent
harbor skills install --agent opencode  # Install for a specific agent
harbor skills install --project       # Install at project level (not global)
harbor skills check                   # Check installed skills and their status
harbor skills uninstall               # Remove installed skills (interactive)
harbor skills uninstall --orphans     # Remove only orphaned installs at old locations
harbor skills uninstall --all         # Remove all discovered installs
```

## Maintenance

```bash
harbor upgrade                        # Upgrade harbor via 'go install ...@latest'
harbor tailwind download              # Download the Tailwind CLI binary to .bin/tailwindcss
harbor build                          # Rebuild CSS + Go binary
harbor build --no-css                 # Go-only build (skip CSS rebuild)
harbor dev                            # Hot-reload dev server
```

## Global Flags

```bash
--json      # Machine-readable JSON output (most commands)
```

## File Naming

The CLI generates filenames automatically from titles:

| Type       | Pattern                        | Example                         |
|------------|--------------------------------|---------------------------------|
| Lesson     | `0001-dash-case-name.html`      | `0001-sql-joins.html`           |
| Record     | `0001-dash-case-title.md`       | `0001-understood-inner-join.md` |
| Reference  | Slug-based (from title)         | `notation-cheat-sheet.html`     |
| Question   | Slug-based (from title)         | `what-is-a-join` (DB-only; stimulus HTML in `questions/`) |

## Workspace Layout

```
<name>/
├── MISSION.md            # Why you're learning
├── RESOURCES.md          # Curated sources and communities
├── GLOSSARY.md           # Canonical terminology (built over time)
├── NOTES.md              # Preferences and working notes (scratchpad)
├── lessons/              # Self-contained lesson HTML files
├── learning-records/     # ADR-style records of what was learned
├── reference/            # Cheat sheets and reference documents
├── questions/            # Stimulus HTML files for questions (optional)
└── assets/               # Reusable components (style.css, etc.)
```

## Data

SQLite database at `~/.harbor/harbor.db` (configurable via `config set data_dir`).

FTS5 full-text search (Porter tokenizer) on lessons, records, and references.
All mutations happen through the CLI — the web UI is read-only.
