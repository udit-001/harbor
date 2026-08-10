package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workspaceStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show workspaces",
	Long: `Show a summary of workspaces in the data directory.

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

		if jsonEnabled(cmd) {
			printJSON(map[string]any{"totalWorkspaces": len(workspaces)})
			return nil
		}

		fmt.Println()
		fmt.Printf("  Total workspaces: %d\n", len(workspaces))
		fmt.Println()

		if len(workspaces) > 0 {
			fmt.Println("  Workspaces:")
			for _, w := range workspaces {
				fmt.Printf("    %s\n", w.DisplayName())
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceStatsCmd)
}
