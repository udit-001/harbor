package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the scratchpad",
	Long: `Full-text search across scraps (and their tag descriptions).

Searches the global scratchpad: scrap titles, bodies, and the names and
descriptions of attached tags.

Examples:
  harbor search "SQL joins"
  harbor search "career"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSearch(cmd, args[0])
	},
}

func runSearch(cmd *cobra.Command, q string) error {
	s := mustStore(cmd)
	scraps, err := s.SearchScraps(q, "")
	if err != nil {
		return formatError("search failed", err)
	}

	if jsonEnabled(cmd) {
		printJSON(scraps)
		return nil
	}

	if len(scraps) == 0 {
		fmt.Println()
		fmt.Printf("  No results for %q.\n", q)
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("  Results for %q:\n\n", q)

	rows := make([][]string, 0, len(scraps))
	for _, s := range scraps {
		rows = append(rows, []string{s.Slug, truncate(s.Title, 40), s.Status})
	}
	fmt.Println(formatTable([]string{"Slug", "Title", "Status"}, rows))
	fmt.Println()
	return nil
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
