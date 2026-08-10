package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workspaceStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show workspaces and their page counts",
	Long: `Show a summary of workspaces in the data directory, including how many
pages each body of work holds.

Examples:
  harbor workspace stats
  harbor workspace stats --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		workspaces, err := s.GetWorkspaces()
		if err != nil {
			return formatError("failed to get stats", err)
		}

		type wsStat struct {
			Name  string `json:"name"`
			Pages int    `json:"pages"`
		}
		st := make([]wsStat, 0, len(workspaces))
		for _, w := range workspaces {
			count, err := s.PageCountForWorkspace(w.Name)
			if err != nil {
				return formatError("failed to count pages", err)
			}
			st = append(st, wsStat{Name: w.DisplayName(), Pages: count})
		}

		if jsonEnabled(cmd) {
			printJSON(map[string]any{"totalWorkspaces": len(workspaces), "workspaces": st})
			return nil
		}

		fmt.Println()
		fmt.Printf("  Total workspaces: %d\n", len(workspaces))
		fmt.Println()

		if len(workspaces) > 0 {
			fmt.Println("  Workspaces:")
			for _, s := range st {
				fmt.Printf("    %-30s %d page(s)\n", s.Name, s.Pages)
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceStatsCmd)
}
