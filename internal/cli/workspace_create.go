package cli

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/udit-001/harbor/internal/db"
)

var workspaceCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace",
	Long: `Create a new workspace.

The workspace is a directory under your data directory's workspaces/ containing:
  RESOURCES.md    — Curated sources and references
  NOTES.md        — Free-form notes
  assets/         — Shared assets (stylesheets, fonts, etc.)

Use '--dir <path>' to place the workspace elsewhere.

Examples:
  harbor workspace create "Payroll App"
  harbor workspace create "Design System" --dir ./my-workspace`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)

		displayName := args[0]
		slug := db.Slugify(displayName)

		// Workspace path — a deployment concern (data dir / --dir override).
		customDir, _ := cmd.Flags().GetString("dir")
		var wsPath string
		if customDir != "" {
			wsPath = customDir
		} else {
			wsPath = filepath.Join(defaultWorkspacesDir(), slug)
		}

		topic, _ := cmd.Flags().GetString("topic")
		if topic == "" {
			topic = displayName
		}

		description, _ := cmd.Flags().GetString("description")

		// CreateWorkspace owns the full row ⇔ dir tree invariant: subdirs,
		// seed templates, and the DB row. The CLI only decides the path.
		created, err := s.CreateWorkspace(slug, topic, description, wsPath)
		if err != nil {
			return formatError("failed to create workspace", err)
		}

		// Auto-set as current workspace, and auto-bind the folder harbor was
		// run from: creating a workspace from a project directory is the
		// one-time setup that makes later commands need no --workspace.
		_ = s.SetCurrentWorkspace(slug)
		if _, berr := s.BindCwd(slug); berr != nil {
			fmt.Printf("  ⚠ Could not bind %s to this folder (%v) — use 'harbor workspace bind %s'.\n", slug, berr, slug)
		}

		notifyServer("home", "changed", 0)

		fmt.Println()
		fmt.Printf("  ✓ Created workspace: %s\n", created.DisplayName())
		fmt.Printf("    Path: %s\n", wsPath)
		fmt.Println()
		fmt.Println("  Next steps:")
		fmt.Println("    cd " + wsPath)
		fmt.Println("    harbor page add <your.html> --workspace " + slug + " --description \"what it shows\"")
		fmt.Println("    harbor tag create <name> --description \"why it exists\"")
		fmt.Println()

		return nil
	},
}

func init() {
	workspaceCmd.AddCommand(workspaceCreateCmd)
	workspaceCreateCmd.Flags().String("dir", "", "Create workspace at a custom path")
	workspaceCreateCmd.Flags().String("topic", "", "Friendly display title for the workspace (default: the name you passed)")
	workspaceCreateCmd.Flags().String("description", "", "Semantic description of this body of work (powers disambiguation and search)")
}
