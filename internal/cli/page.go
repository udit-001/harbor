package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/udit-001/harbor/internal/artifacts"
	"github.com/udit-001/harbor/internal/db"
	"github.com/udit-001/harbor/internal/extract"
)

// Managed-store layout and the stage→commit invariant live in
// internal/artifacts — one seam for every writer of the served copy.

var pageCmd = &cobra.Command{
	Use:   "page",
	Short: "Manage library pages",
	Args:  cobra.NoArgs,
	RunE:  runShowHelp,
	Long: `Manage harbor pages — the atomic, standalone HTML artifacts the agent
produces and imports.

A page is imported from a real HTML source (page add copies it into the managed
store, safe from temp wipes and out of project folders), carries searchable
provenance (description/context), belongs to exactly one workspace, and is
labeled with status (draft/published/archived).

Example:
  harbor page add dashboard.html --workspace income-tracker \
      --tag finance --description "monthly totals chart" \
      --context "built from /api/reports; prototype v2"`,
}

var pageAddCmd = &cobra.Command{
	Use:   "add <source> --workspace <name>",
	Short: "Import an HTML page into the managed store",
	Long: `Import a page: copy <source> (an HTML file) into the managed store under
<workspace>, record its provenance (origin path, description, context, tags),
and index its body text.

The workspace must already exist — page add fails fast rather than auto-creating
a phantom/typo workspace.

Examples:
  harbor page add dashboard.html --workspace income-tracker --description "totals chart"
  harbor page add report.html --workspace finance --title "Q3 Report" --tag report --tag finance`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		source := args[0]

		workspaceName, _ := cmd.Flags().GetString("workspace")
		if workspaceName == "" {
			return fmt.Errorf("--workspace is required\n  harbor page add <source> --workspace <name>")
		}

		// Fail fast: the workspace must exist (deliberate grouping name).
		ws, err := s.GetWorkspaceByName(workspaceName)
		if err != nil {
			return fmt.Errorf("workspace %q not found\n  Use 'harbor workspace create %q --description \"...\"' first",
				workspaceName, workspaceName)
		}

		title, _ := cmd.Flags().GetString("title")
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
		}
		slug := db.Slugify(title)
		if slug == "" {
			return fmt.Errorf("page title must produce a slug (got %q)", title)
		}

		formatFlag, _ := cmd.Flags().GetString("format")
		format := extract.ArtifactFormat(source, formatFlag)
		if !extract.ValidArtifactFormat(format) {
			return fmt.Errorf("cannot determine format of %q\n  Supported: html, markdown, pdf, text, svg, image, excalidraw\n  Or pass --format explicitly", source)
		}

		data, err := os.ReadFile(source)
		if err != nil {
			return formatError("failed to read source file", err)
		}

		staged, serr := artifacts.Stage(resolveDataDir(), workspaceName, slug, format, data)
		if serr != nil {
			return serr
		}
		defer os.Remove(staged)
		if err := artifacts.Commit(resolveDataDir(), workspaceName, slug, format, staged); err != nil {
			return err
		}
		managedPath := artifacts.Path(resolveDataDir(), workspaceName, slug, format)

		description, _ := cmd.Flags().GetString("description")
		context, _ := cmd.Flags().GetString("context")
		var tags []string
		if cmd.Flags().Changed("tag") {
			tags, _ = cmd.Flags().GetStringArray("tag")
		}

		page, err := s.CreatePage(ws.ID, title, description, context, "", format, source, extract.ArtifactBodyText(source, format, data), tags)
		if err != nil {
			return formatError("failed to create page", err)
		}
		notifyPage(workspaceName, page.Slug)

		if jsonEnabled(cmd) {
			return printPageJSON(s, page)
		}
		fmt.Println()
		fmt.Printf("  ✓ Page imported: %s\n", page.Slug)
		fmt.Printf("    Where: %s\n", managedPath)
		if description == "" {
			fmt.Println()
			fmt.Println("  ⚠ No description set — pages without a description are hard to find later.")
			fmt.Println("    Run:  harbor page update " + page.Slug + " --description \"what it shows\"")
			fmt.Println()
		}
		fmt.Println()
		return nil
	},
}

var pageListCmd = &cobra.Command{
	Use:   "list",
	Short: "List library pages",
	Long: `List pages. Filter by --workspace, --tag, --status (draft/published/archived),
or search the full text with --search.

Examples:
  harbor page list
  harbor page list --workspace income-tracker --status published
  harbor page list --search "totals chart"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		status, _ := cmd.Flags().GetString("status")
		workspace, _ := cmd.Flags().GetString("workspace")
		tag, _ := cmd.Flags().GetString("tag")
		search, _ := cmd.Flags().GetString("search")

		filter := db.PageFilter{Status: status, WorkspaceSlug: workspace, TagName: tag}
		var pages []db.Page
		var err error
		if search != "" {
			pages, err = s.SearchPages(search, filter)
		} else {
			pages, err = s.ListPages(filter)
		}
		if err != nil {
			return formatError("failed to list pages", err)
		}

		if jsonEnabled(cmd) {
			return printPagesJSON(s, pages)
		}
		return printPagesTable(s, pages)
	},
}

var pageReadCmd = &cobra.Command{
	Use:   "read <slug>",
	Short: "Show a page's record",
	Long: `Read one page's metadata, tags, and origin by its stable slug.

Examples:
  harbor page read monthly-totals`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]
		page, err := s.PageBySlug(slug)
		if err != nil {
			return formatError("page not found", err)
		}
		tags, err := s.TagsForPage(slug)
		if err != nil {
			return formatError("failed to read page tags", err)
		}

		if jsonEnabled(cmd) {
			return printPageJSON(s, page, tags...)
		}
		printPageDetail(s, page, tags)
		return nil
	},
}

var pageUpdateCmd = &cobra.Command{
	Use:   "update <slug>",
	Short: "Update a page in place",
	Long: `Update a page by stable slug (find-then-update). The slug never changes.
Use --title to rename, --description/--context to refresh provenance, --status
to change readiness (draft/published/archived), and repeat --tag to replace the
full tag set. Pass --file <path> to replace the page's HTML content: the new
file is copied into the managed store (what the dashboard serves) and its body
text is re-indexed.

Examples:
  harbor page update monthly-totals --description "revised chart"
  harbor page update monthly-totals --status published
  harbor page update monthly-totals --tag finance --tag chart
  harbor page update monthly-totals --file /tmp/new.html
  harbor page update monthly-totals --file /tmp/new.html --status published`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]

		page, err := s.PageBySlug(slug)
		if err != nil {
			return formatError("page not found", err)
		}

		var title, description, context, status *string
		if v, _ := cmd.Flags().GetString("title"); v != "" {
			title = &v
		}
		if v, _ := cmd.Flags().GetString("description"); v != "" {
			description = &v
		}
		if v, _ := cmd.Flags().GetString("context"); v != "" {
			context = &v
		}
		if v, _ := cmd.Flags().GetString("status"); v != "" {
			if v != db.PageStatusDraft && v != db.PageStatusPublished && v != db.PageStatusArchived {
				return fmt.Errorf("--status must be one of: draft, published, archived (got %q)", v)
			}
			status = &v
		}
		var tags *[]string
		if cmd.Flags().Changed("tag") {
			ts, _ := cmd.Flags().GetStringArray("tag")
			tags = &ts
		}

		filePath, _ := cmd.Flags().GetString("file")

		if filePath == "" && title == nil && description == nil && context == nil && status == nil && tags == nil {
			return fmt.Errorf("nothing to update — pass --title, --description, --context, --status, --tag, and/or --file\n  harbor page update %q --description ...", slug)
		}

		// The managed file is the content harbor serves; resolve its workspace
		// before touching the store so failures surface loudly.
		ws, err := s.GetWorkspace(page.WorkspaceID)
		if err != nil {
			return formatError("failed to resolve page workspace", err)
		}

		// Content push: --file replaces the managed HTML and re-indexes its body
		// text in one step — the one command that makes an edit reach the human.
		// The new content is staged beside the managed path, then renamed into
		// place only after the DB update commits below, so a failed update can
		// never leave the served page ahead of the index (view/search disagreeing).
		var staged string
		var bodyText *string
		if filePath != "" {
			data, rerr := os.ReadFile(filePath)
			if rerr != nil {
				return formatError("failed to read --file", rerr)
			}
			if strings.TrimSpace(string(data)) == "" {
				return fmt.Errorf("--file %q is empty or blank — refusing to blank the page", filePath)
			}
			// The --file replacement must match the page's format: the stored
			// filename is <slug>.<format>. Stage under the same name so the
			// commit lands on the file the server serves for this format.
			var serr error
			staged, serr = artifacts.Stage(resolveDataDir(), ws.Name, slug, page.Format, data)
			if serr != nil {
				return serr
			}
			defer os.Remove(staged)
			bt := extract.ArtifactBodyText(filePath, page.Format, data)
			bodyText = &bt
		}

		updated, err := s.UpdatePage(slug, title, description, context, status, nil, nil, bodyText, tags)
		if err != nil {
			return formatError("failed to update page", err)
		}

		if staged != "" {
			if err := artifacts.Commit(resolveDataDir(), ws.Name, slug, page.Format, staged); err != nil {
				return err
			}
		}
		notifyPage(ws.Name, updated.Slug)

		if jsonEnabled(cmd) {
			return printPageJSON(s, updated)
		}
		fmt.Println()
		fmt.Printf("  ✓ Page updated: %s\n", updated.Slug)
		fmt.Println()
		return nil
	},
}

var pageDeleteCmd = &cobra.Command{
	Use:   "delete <slug>",
	Short: "Remove a page permanently",
	Long: `Permanently delete a page, its tag associations, and its managed HTML file.

Examples:
  harbor page delete monthly-totals`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mustStore(cmd)
		slug := args[0]

		page, err := s.PageBySlug(slug)
		if err != nil {
			return formatError("page not found", err)
		}
		ws, err := s.GetWorkspace(page.WorkspaceID)
		if err != nil {
			return formatError("failed to resolve page workspace", err)
		}

		if err := s.DeletePage(slug); err != nil {
			return formatError("failed to delete page", err)
		}
		_ = os.Remove(artifacts.Path(resolveDataDir(), ws.Name, slug, page.Format)) // best-effort file cleanup
		notifyPage(ws.Name, slug)

		if jsonEnabled(cmd) {
			printJSON(map[string]any{"slug": slug, "deleted": true})
			return nil
		}
		fmt.Println()
		fmt.Printf("  ✓ Page deleted: %s\n", slug)
		fmt.Println()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pageCmd)
	pageCmd.AddCommand(pageAddCmd)
	pageCmd.AddCommand(pageListCmd)
	pageCmd.AddCommand(pageReadCmd)
	pageCmd.AddCommand(pageUpdateCmd)
	pageCmd.AddCommand(pageDeleteCmd)
	pageAddCmd.Flags().String("workspace", "", "Workspace name (required)")
	pageAddCmd.Flags().String("format", "", "Artifact format override: html, markdown, pdf, text, svg, image, excalidraw (default: inferred from the source file)")
	pageAddCmd.Flags().String("title", "", "Page title (default: the source filename)")
	pageAddCmd.Flags().String("description", "", "What the page shows (reader summary)")
	pageAddCmd.Flags().String("context", "", "Where it came from / why it exists (provenance)")
	pageAddCmd.Flags().StringArray("tag", nil, "Attach a tag (repeatable; tags must already exist)")
	pageListCmd.Flags().String("status", "", "Filter by status: draft, published, archived")
	pageListCmd.Flags().String("workspace", "", "Filter by workspace name")
	pageListCmd.Flags().String("tag", "", "Filter by tag name")
	pageListCmd.Flags().String("search", "", "Full-text search across title, description, context, body, and tag descriptions")
	pageUpdateCmd.Flags().String("title", "", "Replace the page title (slug stays stable)")
	pageUpdateCmd.Flags().String("description", "", "Replace the page description")
	pageUpdateCmd.Flags().String("context", "", "Replace the page context")
	pageUpdateCmd.Flags().String("status", "", "Set status: draft, published, archived")
	pageUpdateCmd.Flags().StringArray("tag", nil, "Replace the full tag set (repeatable)")
	pageUpdateCmd.Flags().String("file", "", "Replace the page HTML content from this file (copied into the managed store, body re-indexed)")
}

func printPagesJSON(s *db.Store, pages []db.Page) error {
	out := make([]map[string]any, 0, len(pages))
	for _, p := range pages {
		tags, err := s.TagsForPage(p.Slug)
		if err != nil {
			return formatError("failed to read page tags", err)
		}
		ws, _ := s.GetWorkspace(p.WorkspaceID)
		out = append(out, pageMap(p, ws.Name, tags))
	}
	printJSON(out)
	return nil
}

func printPageJSON(s *db.Store, page db.Page, tags ...db.Tag) error {
	ws, _ := s.GetWorkspace(page.WorkspaceID)
	if len(tags) == 0 {
		var err error
		tags, err = s.TagsForPage(page.Slug)
		if err != nil {
			return formatError("failed to read page tags", err)
		}
	}
	printJSON(pageMap(page, ws.Name, tags))
	return nil
}

func pageMap(p db.Page, workspace string, tags []db.Tag) map[string]any {
	return map[string]any{
		"slug":        p.Slug,
		"managedPath": artifacts.Path(resolveDataDir(), workspace, p.Slug, p.Format),
		"title":       p.Title,
		"workspace":   workspace,
		"format":      p.Format,
		"status":      p.Status,
		"tags":        tagNames(tags),
		"description": p.Description,
		"context":     p.Context,
		"originPath":  p.OriginPath,
		"updatedAt":   p.UpdatedAt,
		"createdAt":   p.CreatedAt,
	}
}

func printPagesTable(s *db.Store, pages []db.Page) error {
	if len(pages) == 0 {
		fmt.Println()
		fmt.Println("  No pages found. Import one with:")
		fmt.Println()
		fmt.Println("    harbor page add <your.html> --workspace <name> --description \"what it shows\"")
		fmt.Println()
		return nil
	}
	rows := make([][]string, 0, len(pages))
	for _, p := range pages {
		tags, err := s.TagsForPage(p.Slug)
		if err != nil {
			return formatError("failed to read page tags", err)
		}
		ws, _ := s.GetWorkspace(p.WorkspaceID)
		rows = append(rows, []string{
			p.Slug,
			strings.ReplaceAll(p.Title, "\n", " "),
			ws.Name,
			p.Format,
			p.Status,
			"[" + strings.Join(tagNames(tags), ", ") + "]",
			formatDateShort(p.UpdatedAt),
		})
	}
	fmt.Println()
	fmt.Print(formatTable([]string{"SLUG", "TITLE", "WORKSPACE", "FORMAT", "STATUS", "TAGS", "UPDATED"}, rows))
	fmt.Println()
	return nil
}

func printPageDetail(s *db.Store, page db.Page, tags []db.Tag) {
	ws, _ := s.GetWorkspace(page.WorkspaceID)
	names := tagNames(tags)
	fmt.Println()
	fmt.Printf("  %s  [%s]  (%s)  [%s]\n", page.Title, page.Status, ws.Name, page.Format)
	if len(names) > 0 {
		fmt.Printf("  tags: %s\n", strings.Join(names, ", "))
	}
	fmt.Printf("  slug: %s\n", page.Slug)
	if page.Description != "" {
		fmt.Printf("  description: %s\n", page.Description)
	}
	if page.Context != "" {
		fmt.Printf("  context: %s\n", page.Context)
	}
	if page.OriginPath != "" {
		fmt.Printf("  origin: %s\n", page.OriginPath)
	}
	fmt.Printf("  updated: %s\n", formatDateShort(page.UpdatedAt))
	fmt.Println()
	if page.BodyText != "" {
		excerpt := page.BodyText
		if len(excerpt) > 600 {
			excerpt = excerpt[:600] + "…"
		}
		lines := strings.Split(excerpt, "\n")
		for _, ln := range lines {
			if strings.TrimSpace(ln) != "" {
				fmt.Println("  " + strings.TrimSpace(ln))
			}
		}
		fmt.Println()
	}
}
