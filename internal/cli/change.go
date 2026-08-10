package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/harbor/internal/db"
)

var changeCmd = &cobra.Command{
	Use:   "change",
	Short: "Record and list agent changes",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Record the edits the agent makes in response to feedback, so the human can
be walked through what changed.

A change carries a data-cf-change marker id (embeddable in the page HTML), an
optional comment it resolves, and a human title/description for the walkthrough.

Examples:
  harbor change add monthly-totals --change-id cf-1 --comment 3 \
      --title "Widen the chart" --description "expanded the totals chart to full width"`,
}

var changeAddCmd = &cobra.Command{
	Use:   "add <slug>",
	Short: "Record a change on a page",
	Long: `Record a change the agent made on a page. The change_id matches the
data-cf-change="<id>" marker placed in the page HTML so the walkthrough can
locate and highlight what changed. Pass --comment <id> to link it to the
comment it resolves; --title/--description give the human-readable summary.

Examples:
  harbor change add monthly-totals --change-id cf-1 --comment 3 \\
      --title "Widen the chart" --description "expanded the chart to full width"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		changeID, _ := cmd.Flags().GetString("change-id")
		title, _ := cmd.Flags().GetString("title")
		desc, _ := cmd.Flags().GetString("description")
		commentRaw, _ := cmd.Flags().GetString("comment")
		var commentID int64
		if commentRaw != "" {
			parsed, perr := parseCommentID(commentRaw) // accepts "#3" or "3"
			if perr != nil {
				return perr
			}
			commentID = parsed
		}
		if changeID == "" {
			return fmt.Errorf("--change-id is required\n  harbor change add %q --change-id <marker> --title ...", slug)
		}
		if title == "" {
			return fmt.Errorf("--title is required (the walkthrough reads it)\n  harbor change add %q --change-id %q --title ...", slug, changeID)
		}

		ch, err := s.CreateChange(slug, changeID, commentID, title, desc)
		if err != nil {
			return formatError("failed to record change", err)
		}

		if jsonEnabled(cmd) {
			printJSON(ch)
			return nil
		}
		fmt.Println()
		fmt.Printf("  ✓ Change recorded: %s\n", ch.ChangeID)
		fmt.Printf("    %s — %s\n", ch.Title, ch.Description)
		fmt.Printf("    Add the marker:  data-cf-change=\"%s\" on the edited element\n", ch.ChangeID)
		// Completion guard: a change whose marker isn't in the page is skipped by
		// the walkthrough — warn so the agent knows to place data-cf-change.
		if page, perr := s.PageBySlug(slug); perr == nil {
			if ws, werr := s.GetWorkspace(page.WorkspaceID); werr == nil {
				path := managedPagePath(ws.Name, slug)
				if data, rerr := os.ReadFile(path); rerr == nil {
					if !strings.Contains(string(data), `data-cf-change="`+ch.ChangeID+`"`) {
						fmt.Printf("  ⚠ Marker not found in %s — the walkthrough skips a change with no matching element.\n", path)
						fmt.Printf("    Place  data-cf-change=\"%s\" on the element you edited, then run this again.\n", ch.ChangeID)
					}
				}
			}
		}
		fmt.Println()
		return nil
	},
}

var changeListCmd = &cobra.Command{
	Use:   "list <slug>",
	Short: "List a page's changes",
	Long: `List the changes recorded for a page (what the walkthrough will tour), newest
last.

Examples:
  harbor change list monthly-totals`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		changes, err := s.ListChanges(args[0], 0)
		if err != nil {
			return formatError("failed to list changes", err)
		}

		if jsonEnabled(cmd) {
			if changes == nil {
				changes = []db.Change{}
			}
			printJSON(changes)
			return nil
		}
		if len(changes) == 0 {
			fmt.Println()
			fmt.Println("  No changes recorded for this page.")
			fmt.Println()
			return nil
		}
		rows := make([][]string, 0, len(changes))
		for _, c := range changes {
			title := strings.ReplaceAll(c.Title, "\n", " ")
			desc := strings.ReplaceAll(c.Description, "\n", " ")
			if clipped := truncate(desc, 40); clipped != "" {
				title += " — " + clipped
			}
			rows = append(rows, []string{c.ChangeID, title, formatDateShort(c.CreatedAt)})
		}
		fmt.Println()
		fmt.Print(formatTable([]string{"MARKER", "CHANGE", "CREATED"}, rows))
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(changeCmd)
	changeCmd.AddCommand(changeAddCmd)
	changeCmd.AddCommand(changeListCmd)
	changeAddCmd.Flags().String("change-id", "", "data-cf-change marker id (required)")
	changeAddCmd.Flags().String("comment", "", "Comment id this change resolves (\"#3\" or \"3\")")
	changeAddCmd.Flags().String("title", "", "Short human title (required)")
	changeAddCmd.Flags().String("description", "", "Longer what-changed description")
}