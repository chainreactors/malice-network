package search

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func Commands(con *core.Console) []*cobra.Command {
	searchCmd := &cobra.Command{
		Use:   "search [query...]",
		Short: "Full-text search across commands and plugins",
		Long:  `Search for commands, plugins, and their documentation using FTS5 full-text search. Supports Chinese text and FTS5 syntax (AND, OR, NOT, "phrases").`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  searchRunE(con),
		Annotations: map[string]string{
			"static": "true",
		},
		Example: `~~~
search screenshot
search "credential dump" --type command
search pivot --group implant --limit 10
~~~`,
	}

	searchCmd.Flags().StringP("type", "t", "", "filter by type: command, plugin")
	searchCmd.Flags().StringP("group", "g", "", "filter by command group")
	searchCmd.Flags().IntP("limit", "n", 20, "maximum results to return")
	common.BindFlagCompletions(searchCmd, func(comp carapace.ActionMap) {
		comp["type"] = carapace.ActionValues("command", "plugin").Usage("search result type")
		comp["group"] = searchGroupCompleter(con)
		comp["limit"] = carapace.ActionValues("10", "20", "50", "100").Usage("maximum results")
	})

	return []*cobra.Command{searchCmd}
}

func searchGroupCompleter(con *core.Console) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		if con == nil || con.SearchIndex == nil {
			return carapace.ActionValues().Tag("search group")
		}
		categories, err := con.SearchIndex.Categories("")
		if err != nil {
			return carapace.ActionValues().Tag("search group")
		}
		return carapace.ActionValues(categories...).Tag("search group")
	})
}

func searchRunE(con *core.Console) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if con.SearchIndex == nil {
			return fmt.Errorf("search index not initialized")
		}

		query := strings.Join(args, " ")
		typeFilter, _ := cmd.Flags().GetString("type")
		group, _ := cmd.Flags().GetString("group")
		limit, _ := cmd.Flags().GetInt("limit")

		// Use hybrid search when vector index has precomputed embeddings
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		results, err := core.HybridSearch(ctx, con.SearchIndex, con.VectorIndex, query, typeFilter, group, limit)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(results) == 0 {
			if common.ShouldUseStaticOutput(con) {
				fmt.Fprintf(cmd.OutOrStdout(), "No results found for: %s\n", query)
			} else {
				con.Log.Infof("No results found for: %s\n", query)
			}
			return nil
		}

		if !common.ShouldUseStaticOutput(con) {
			tui.Down(0)
		}
		renderResults(cmd.OutOrStdout(), query, results)
		return nil
	}
}

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	nameStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	groupStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	snippetStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	ttpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
)

func opsecStyle(opsecStr string) lipgloss.Style {
	opsec, _ := strconv.Atoi(opsecStr)
	switch {
	case opsec >= 9:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	case opsec >= 7:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	case opsec >= 4:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // orange
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // red
	}
}

func renderResults(writer io.Writer, query string, results []core.SearchResult) {
	fmt.Fprintf(writer, "%s\n\n", headerStyle.Render(fmt.Sprintf("Found %d results for \"%s\":", len(results), query)))

	for i, r := range results {
		// Name and type badge
		badge := "[cmd]"
		if r.Type == "plugin" {
			badge = "[plugin]"
		}

		line := fmt.Sprintf("  %s %s", groupStyle.Render(badge), nameStyle.Render(r.Name))

		if r.Category != "" {
			line += " " + groupStyle.Render("["+r.Category+"]")
		}
		if r.TTP != "" {
			line += " " + ttpStyle.Render(r.TTP)
		}
		if r.Opsec != "" && r.Opsec != "0" {
			line += " " + opsecStyle(r.Opsec).Render("opsec:"+r.Opsec)
		}

		fmt.Fprintln(writer, line)

		if r.Description != "" {
			fmt.Fprintf(writer, "    %s\n", r.Description)
		}

		if r.Snippet != "" && r.Snippet != r.Description {
			fmt.Fprintf(writer, "    %s\n", snippetStyle.Render(r.Snippet))
		}

		if r.Usage != "" {
			fmt.Fprintf(writer, "    %s\n", groupStyle.Render(r.Usage))
		}

		if i < len(results)-1 {
			fmt.Fprintln(writer)
		}
	}
	fmt.Fprintln(writer)
}
