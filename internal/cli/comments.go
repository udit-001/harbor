package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/udit-001/harbor/internal/db"
)

var commentsCmd = &cobra.Command{
	Use:   "comments",
	Short: "Manage page comments",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Manage anchored feedback on pages — the human side of the feedback loop.

A comment is feedback anchored to a page (by a text selection, a specific
element, or the page as a whole). It lives in the database and never edits the
page file; the agent reads the open queue and acts on it.

Examples:
  harbor comments list
  harbor comments list --page monthly-totals --status open
  harbor comments watch
  harbor comments update 34 --status in-progress
  harbor comments update 34 --status done`,
}

var commentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List page comments",
	Long: `List comments. Without --status, only OPEN comments are shown (the agent's
default context read — the queue it must respond to). Pass --status
in-progress or --status done to see later states, or --page <slug> to narrow
to one page.

Examples:
  harbor comments list
  harbor comments list --page monthly-totals --status open
  harbor comments list --status done`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		status, _ := cmd.Flags().GetString("status")
		page, _ := cmd.Flags().GetString("page")

		comments, err := s.ListComments(db.CommentFilter{PageSlug: page, Status: status})
		if err != nil {
			return formatError("failed to list comments", err)
		}

		if jsonEnabled(cmd) {
			return printCommentsJSON(comments)
		}
		printCommentsTable(comments)
		return nil
	},
}

var commentsWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Tail new open comments as they arrive",
	Long: `Block and print open comments as they land, echoing the current open queue
first, then new arrivals as the human writes them.

Prints one line per open comment:
  [#id] page / "quote" / body

Exits on Ctrl-C.

Examples:
  harbor comments watch`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		return commentWatcher(os.Stdout,
			func() ([]db.CommentView, error) {
				return s.ListComments(db.CommentFilter{Status: db.CommentStatusOpen})
			},
			ticker.C, ctx.Done())
	},
}

var commentsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a comment's status",
	Long: `Move a comment through its lifecycle by id: open → in-progress → done.
Setting done records the resolution time.

Examples:
  harbor comments update 34 --status in-progress
  harbor comments update 34 --status done`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		id, err := parseCommentID(args[0])
		if err != nil {
			return err
		}
		status, _ := cmd.Flags().GetString("status")
		if status == "" {
			return fmt.Errorf("--status is required (open, in-progress, done)\n  harbor comments update %d --status done", id)
		}

		updated, err := s.UpdateCommentStatus(id, status)
		if err != nil {
			return formatError("failed to update comment", err)
		}

		if jsonEnabled(cmd) {
			return printCommentJSON(updated)
		}
		fmt.Println()
		fmt.Printf("  ✓ Comment #%d (%s, %s) is now %s\n", updated.ID, updated.PageSlug, updated.Type, updated.Status)
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commentsCmd)
	commentsCmd.AddCommand(commentsListCmd)
	commentsCmd.AddCommand(commentsWatchCmd)
	commentsCmd.AddCommand(commentsUpdateCmd)
	commentsListCmd.Flags().String("page", "", "Filter by page slug")
	commentsListCmd.Flags().String("status", db.CommentStatusOpen, "Filter by status: open (default), in-progress, done")
	commentsUpdateCmd.Flags().String("status", "", "Set status: open, in-progress, done")
}

// parseCommentID parses a comment id argument (a positive integer).
func parseCommentID(raw string) (int64, error) {
	// list/watch render ids as "#3"; accept that form so the agent can paste
	// the id straight back into update / change add --comment.
	raw = strings.TrimPrefix(strings.TrimSpace(raw), "#")
	var id int64
	if _, err := fmt.Sscanf(raw, "%d", &id); err != nil || id < 1 {
		return 0, fmt.Errorf("invalid comment id %q", raw)
	}
	return id, nil
}

// commentWatcher is the tail loop behind `comments watch`, factored behind a
// seam so tests can drive it synchronously — no real timers, no signals. It
// echoes the current open queue, then prints each new open comment exactly once
// (deduped by id) on every poll tick, returning when done is closed.
func commentWatcher(out io.Writer, fetch func() ([]db.CommentView, error), poll <-chan time.Time, done <-chan struct{}) error {
	seen := make(map[int64]bool)

	flush := func() error {
		comments, err := fetch()
		if err != nil {
			return err
		}
		for _, c := range comments {
			if seen[c.ID] {
				continue
			}
			seen[c.ID] = true
			fmt.Fprintf(out, "%s\n", commentLine(c))
		}
		return nil
	}

	if err := flush(); err != nil {
		return err
	}

	for {
		select {
		case <-done:
			return nil
		case <-poll:
			if err := flush(); err != nil {
				return err
			}
		}
	}
}

// commentLine renders one comment as a single watch line:
//
//	[#12] monthly-totals / "the quote" / the body
//
// The quote is elided when absent; the body is collapsed to one line.
func commentLine(c db.CommentView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[#%d] %s", c.ID, c.PageSlug)
	if c.Quote != "" {
		fmt.Fprintf(&b, " / %q", c.Quote)
	}
	body := strings.Join(strings.Fields(c.Body), " ")
	if clipped := truncate(body, 120); clipped != "" {
		fmt.Fprintf(&b, " / %s", clipped)
	}
	return b.String()
}

func printCommentsJSON(comments []db.CommentView) error {
	out := make([]map[string]any, 0, len(comments))
	for _, c := range comments {
		out = append(out, commentMap(c))
	}
	printJSON(out)
	return nil
}

func printCommentJSON(c db.CommentView) error {
	printJSON(commentMap(c))
	return nil
}

func commentMap(c db.CommentView) map[string]any {
	m := map[string]any{
		"id":        c.ID,
		"page":      c.PageSlug,
		"type":      c.Type,
		"status":    c.Status,
		"body":      c.Body,
		"createdAt": c.CreatedAt,
	}
	if c.Anchor != "" {
		m["anchor"] = c.Anchor
	}
	if c.Quote != "" {
		m["quote"] = c.Quote
	}
	if c.ResolvedAt != nil {
		m["resolvedAt"] = *c.ResolvedAt
	}
	return m
}

func printCommentsTable(comments []db.CommentView) {
	if len(comments) == 0 {
		fmt.Println()
		fmt.Println("  No open comments. Review a page and leave feedback, or run:")
		fmt.Println()
		fmt.Println("    harbor comments list --status done")
		fmt.Println()
		return
	}
	rows := make([][]string, 0, len(comments))
	for _, c := range comments {
		rows = append(rows, []string{
			fmt.Sprintf("#%d", c.ID),
			c.PageSlug,
			c.Type,
			c.Status,
			strings.Join(strings.Fields(c.Body), " "),
			formatDateShort(c.CreatedAt),
		})
	}
	fmt.Println()
	fmt.Print(formatTable([]string{"ID", "PAGE", "TYPE", "STATUS", "BODY", "CREATED"}, rows))
	fmt.Println()
}
