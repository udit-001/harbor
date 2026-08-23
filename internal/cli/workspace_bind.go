package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// workspaceBindCmd maps the caller's current folder (or --dir) to a
// workspace. After binding, workspace-scoped commands in that folder tree
// need no --workspace: resolveWorkspace finds the deepest matching binding.
var workspaceBindCmd = &cobra.Command{
	Use:   "bind <name>",
	Short: "Bind this folder to a workspace",
	Long: `Map your current directory (or --dir) to a workspace. After binding,
commands run anywhere inside that folder resolve to the workspace without
--workspace — per-project truth outranks the global current-workspace setting.

Re-binding a folder replaces its mapping.

Examples:
  cd ~/Dev/payroll && harbor workspace bind payroll-app
  harbor workspace bind research --dir ~/notes/research`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		dir, _ := cmd.Flags().GetString("dir")
		var bound string
		var err error
		if dir != "" {
			if err := s.BindWorkspaceDir(dir, args[0]); err != nil {
				return err
			}
			bound = dir
		} else {
			bound, err = s.BindCwd(args[0])
			if err != nil {
				return err
			}
		}
		fmt.Println()
		fmt.Printf("  ✓ Bound %s → %s\n", bound, args[0])
		fmt.Printf("    Commands run inside it now default to this workspace.\n")
		fmt.Println()
		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceBindCmd)
	workspaceBindCmd.Flags().String("dir", "", "Bind this folder instead of the current one")
}
