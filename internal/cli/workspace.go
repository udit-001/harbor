package cli

import "github.com/spf13/cobra"

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage workspaces",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `List and manage workspaces.

Examples:
  harbor workspace list
  harbor workspace stats`,
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
}
