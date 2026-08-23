package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/udit-001/harbor/internal/artifacts"
	"github.com/udit-001/harbor/internal/db"
	"github.com/udit-001/harbor/internal/extract"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search library pages",
	Long: `Full-text search across library pages: title, description, context, body
text, and the names/descriptions of attached tags.

Search is broad — it indexes everything a page "was for" — so a future agent
session can find a page by describing it. Narrow results with --workspace,
--status, or --tag.

Use --rebuild-index to (re)harvest body text for pages missing it (idempotent:
pages with text are skipped), and --force with it to clear body_text first and
re-index everything.

Examples:
  harbor search "income tracker dashboard"
  harbor search "chart" --workspace income-tracker --status published
  harbor search --rebuild-index
  harbor search --rebuild-index --force`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)

		rebuild, _ := cmd.Flags().GetBool("rebuild-index")
		if rebuild {
			return runRebuildIndex(cmd, s)
		}
		if len(args) == 0 {
			return fmt.Errorf("a search query is required (or pass --rebuild-index to rebuild)\n  harbor search \"<query>\"")
		}

		filter := db.PageFilter{}
		filter.Status, _ = cmd.Flags().GetString("status")
		filter.WorkspaceSlug, _ = cmd.Flags().GetString("workspace")
		filter.TagName, _ = cmd.Flags().GetString("tag")

		pages, err := s.SearchPages(args[0], filter)
		if err != nil {
			return formatError("search failed", err)
		}

		if jsonEnabled(cmd) {
			return printPagesJSON(s, pages)
		}

		if len(pages) == 0 {
			fmt.Println()
			fmt.Printf("  No results for %q.\n", args[0])
			fmt.Println()
			return nil
		}

		fmt.Println()
		fmt.Printf("  Results for %q:\n\n", args[0])

		rows := make([][]string, 0, len(pages))
		for _, p := range pages {
			ws, _ := s.GetWorkspace(p.WorkspaceID)
			rows = append(rows, []string{p.Slug, truncate(p.Title, 40), ws.Name, p.Status})
		}
		fmt.Println(formatTable([]string{"Slug", "Title", "Workspace", "Status"}, rows))
		fmt.Println()
		return nil
	},
}

func runRebuildIndex(cmd *cobra.Command, s *db.Store) error {
	force, _ := cmd.Flags().GetBool("force")
	if force {
		if err := s.ClearPageBodies(); err != nil {
			return formatError("failed to clear page bodies", err)
		}
	}

	needs, err := s.PagesNeedingReindex()
	if err != nil {
		return formatError("failed to scan for reindex", err)
	}

	indexed, skipped := 0, 0
	type result struct {
		Slug      string `json:"slug"`
		Indexed   bool   `json:"indexed"`
		Reason    string `json:"reason,omitempty"`
		BodyChars int    `json:"bodyChars,omitempty"`
	}
	results := make([]result, 0, len(needs))
	for _, p := range needs {
		r := result{Slug: p.Slug}
		ws, err := s.GetWorkspace(p.WorkspaceID)
		if err != nil {
			r.Reason = "no workspace"
			skipped++
			results = append(results, r)
			continue
		}
		data, err := os.ReadFile(artifacts.Path(resolveDataDir(), ws.Name, p.Slug, p.Format))
		if err != nil {
			r.Reason = "no managed file"
			skipped++
			results = append(results, r)
			continue
		}
		bt := extract.FromHTML(string(data))
		if _, err := s.UpdatePage(p.Slug, nil, nil, nil, nil, nil, nil, &bt, nil); err != nil {
			r.Reason = "update failed"
			skipped++
			results = append(results, r)
			continue
		}
		r.Indexed = true
		r.BodyChars = len(bt)
		indexed++
		results = append(results, r)
	}

	if jsonEnabled(cmd) {
		printJSON(map[string]any{"indexed": indexed, "skipped": skipped, "pages": results})
		return nil
	}

	fmt.Println()
	fmt.Printf("  Reindexed %d page(s); skipped %d.\n", indexed, skipped)
	fmt.Println()
	return nil
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().Bool("rebuild-index", false, "Re-harvest body text for pages missing it")
	searchCmd.Flags().Bool("force", false, "With --rebuild-index: clear body_text first, re-index everything")
	searchCmd.Flags().String("status", "", "Filter by status: draft, published, archived")
	searchCmd.Flags().String("workspace", "", "Filter by workspace name")
	searchCmd.Flags().String("tag", "", "Filter by tag name")
}
