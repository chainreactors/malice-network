package core

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
)

// newTestConsoleWithCommands creates a minimal Console with cobra commands registered in menus.
func newTestConsoleWithCommands(t testing.TB) *Console {
	t.Helper()

	app := console.New("test")

	implantMenu := app.NewMenu(consts.ImplantMenu)
	implantMenu.AddGroup(&cobra.Group{ID: consts.ImplantGroup, Title: "Implant"})
	implantMenu.AddGroup(&cobra.Group{ID: consts.ExecuteGroup, Title: "Execute"})
	implantMenu.AddGroup(&cobra.Group{ID: consts.SysGroup, Title: "System"})

	implantMenu.AddCommand(&cobra.Command{
		Use:     "whoami",
		Short:   "Display the current user identity",
		GroupID: consts.SysGroup,
		Annotations: map[string]string{
			"ttp":   "T1033",
			"opsec": "9",
		},
	})
	implantMenu.AddCommand(&cobra.Command{
		Use:     "sysinfo",
		Short:   "Get system information including OS, hostname, and architecture",
		GroupID: consts.SysGroup,
	})
	implantMenu.AddCommand(&cobra.Command{
		Use:     "shell",
		Short:   "Execute a shell command via cmd.exe or /bin/sh",
		GroupID: consts.ExecuteGroup,
		Annotations: map[string]string{
			"ttp":   "T1059",
			"opsec": "3",
		},
	})
	uacCmd := &cobra.Command{
		Use:     "uac-bypass",
		Short:   "Bypass User Account Control",
		GroupID: consts.ImplantGroup,
		Annotations: map[string]string{
			"ttp":   "T1548.002",
			"opsec": "8",
		},
	}
	uacCmd.AddCommand(&cobra.Command{
		Use:   "elevatedcom",
		Short: "UAC bypass via elevated COM interface",
	})
	uacCmd.AddCommand(&cobra.Command{
		Use:   "sspi",
		Short: "UAC bypass via SSPI datagram",
	})
	implantMenu.AddCommand(uacCmd)
	implantMenu.AddCommand(&cobra.Command{
		Use:     "hashdump",
		Short:   "Dump SAM database hashes",
		GroupID: consts.ImplantGroup,
		Annotations: map[string]string{
			"ttp":   "T1003.002",
			"opsec": "9",
		},
	})
	implantMenu.AddCommand(&cobra.Command{
		Use:     "hidden-cmd",
		Short:   "This should not appear in search results",
		GroupID: consts.SysGroup,
		Hidden:  true,
	})

	clientMenu := app.NewMenu(consts.ClientMenu)
	clientMenu.AddGroup(&cobra.Group{ID: consts.GenericGroup, Title: "Generic"})
	clientMenu.AddCommand(&cobra.Command{
		Use:     "session",
		Short:   "List or manage sessions",
		GroupID: consts.GenericGroup,
	})

	con := &Console{
		App: app,
	}
	return con
}

// --- matchCommand tests ---

func TestMatchCommand_ByName(t *testing.T) {
	cmd := &cobra.Command{Use: "whoami", Short: "Display user"}
	if !matchCommand(cmd, "who") {
		t.Error("expected match on command name substring")
	}
}

func TestMatchCommand_ByDescription(t *testing.T) {
	cmd := &cobra.Command{Use: "sysinfo", Short: "Get system information"}
	if !matchCommand(cmd, "system") {
		t.Error("expected match on description substring")
	}
}

func TestMatchCommand_ByAlias(t *testing.T) {
	cmd := &cobra.Command{Use: "whoami", Aliases: []string{"id"}, Short: "Display user"}
	if !matchCommand(cmd, "id") {
		t.Error("expected match on alias")
	}
}

func TestMatchCommand_BySubcommandName(t *testing.T) {
	parent := &cobra.Command{Use: "uac-bypass", Short: "Bypass UAC"}
	parent.AddCommand(&cobra.Command{Use: "elevatedcom", Short: "COM interface"})
	if !matchCommand(parent, "elevated") {
		t.Error("expected match on subcommand name")
	}
}

func TestMatchCommand_CaseInsensitive(t *testing.T) {
	cmd := &cobra.Command{Use: "HashDump", Short: "Dump SAM hashes"}
	if !matchCommand(cmd, "hashdump") {
		t.Error("expected case-insensitive match")
	}
}

func TestMatchCommand_NoMatch(t *testing.T) {
	cmd := &cobra.Command{Use: "whoami", Short: "Display user"}
	if matchCommand(cmd, "lateral") {
		t.Error("expected no match for unrelated query")
	}
}

// --- commandToInfo tests ---

func TestCommandToInfo_BasicFields(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "whoami",
		Short:   "Display user identity",
		GroupID: "sys",
		Annotations: map[string]string{
			"ttp":   "T1033",
			"opsec": "9",
		},
	}

	info := commandToInfo(cmd)

	if info.Name != "whoami" {
		t.Errorf("Name = %q, want %q", info.Name, "whoami")
	}
	if info.Group != "sys" {
		t.Errorf("Group = %q, want %q", info.Group, "sys")
	}
	if info.Description != "Display user identity" {
		t.Errorf("Description = %q, want %q", info.Description, "Display user identity")
	}
	if info.Ttp != "T1033" {
		t.Errorf("Ttp = %q, want %q", info.Ttp, "T1033")
	}
	if info.Opsec != 9 {
		t.Errorf("Opsec = %d, want %d", info.Opsec, 9)
	}
}

func TestCommandToInfo_WithSubcommands(t *testing.T) {
	parent := &cobra.Command{Use: "uac-bypass", Short: "Bypass UAC"}
	parent.AddCommand(&cobra.Command{Use: "elevatedcom", Short: "COM"})
	parent.AddCommand(&cobra.Command{Use: "sspi", Short: "SSPI"})
	parent.AddCommand(&cobra.Command{Use: "hidden", Short: "Hidden", Hidden: true})

	info := commandToInfo(parent)

	if len(info.Subcommands) != 2 {
		t.Fatalf("Subcommands count = %d, want 2", len(info.Subcommands))
	}
	found := map[string]bool{}
	for _, s := range info.Subcommands {
		found[s] = true
	}
	if !found["elevatedcom"] || !found["sspi"] {
		t.Errorf("Subcommands = %v, want [elevatedcom, sspi]", info.Subcommands)
	}
	if found["hidden"] {
		t.Error("hidden subcommand should not be included")
	}
}

func TestCommandToInfo_NoAnnotations(t *testing.T) {
	cmd := &cobra.Command{Use: "ps", Short: "List processes", GroupID: "sys"}
	info := commandToInfo(cmd)

	if info.Ttp != "" {
		t.Errorf("Ttp = %q, want empty", info.Ttp)
	}
	if info.Opsec != 0 {
		t.Errorf("Opsec = %d, want 0", info.Opsec)
	}
}

// --- searchCommands tests ---

func TestSearchCommands_ByName(t *testing.T) {
	con := newTestConsoleWithCommands(t)

	results, err := searchCommands(con, "whoami", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "whoami" {
		t.Errorf("Name = %q, want %q", results[0].Name, "whoami")
	}
}

func TestSearchCommands_ByDescription(t *testing.T) {
	con := newTestConsoleWithCommands(t)

	results, err := searchCommands(con, "SAM", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'SAM', got %d", len(results))
	}
	if results[0].Name != "hashdump" {
		t.Errorf("Name = %q, want %q", results[0].Name, "hashdump")
	}
}

func TestSearchCommands_BySubcommand(t *testing.T) {
	con := newTestConsoleWithCommands(t)

	results, err := searchCommands(con, "elevatedcom", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for subcommand search, got %d", len(results))
	}
	if results[0].Name != "uac-bypass" {
		t.Errorf("Name = %q, want %q", results[0].Name, "uac-bypass")
	}
}

func TestSearchCommands_GroupFilter(t *testing.T) {
	con := newTestConsoleWithCommands(t)

	// "shell" is in execute group; searching with sys group filter should not find it
	results, err := searchCommands(con, "shell", consts.SysGroup, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results with group filter, got %d", len(results))
	}

	// searching with correct group should find it
	results, err = searchCommands(con, "shell", consts.ExecuteGroup, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestSearchCommands_HiddenExcluded(t *testing.T) {
	con := newTestConsoleWithCommands(t)

	results, err := searchCommands(con, "hidden", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for hidden command, got %d", len(results))
	}
}

func TestSearchCommands_CrossMenu(t *testing.T) {
	con := newTestConsoleWithCommands(t)

	// "session" is in client menu, "whoami" is in implant menu
	// both should be found when searching without group filter
	results, err := searchCommands(con, "session", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result from client menu, got %d", len(results))
	}
	if results[0].Name != "session" {
		t.Errorf("Name = %q, want %q", results[0].Name, "session")
	}
}

func TestSearchCommands_NoResults(t *testing.T) {
	con := newTestConsoleWithCommands(t)

	results, err := searchCommands(con, "nonexistent_xyz", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestSearchCommands_NilConsole(t *testing.T) {
	_, err := searchCommands(nil, "test", "", "")
	if err == nil {
		t.Error("expected error for nil console")
	}
}

// --- MCP protocol-level test ---

func TestMCPSearchCommandsTool(t *testing.T) {
	ctx := context.Background()

	// Simulate the search_commands tool with mock data via mcptest
	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool: mcp.NewTool("search_commands",
			mcp.WithDescription("Search for commands"),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search keyword")),
			mcp.WithString("group", mcp.Description("Optional group filter")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			query, err := request.RequireString("query")
			if err != nil || query == "" {
				return mcp.NewToolResultError("query is required"), nil
			}

			group, _ := request.GetArguments()["group"].(string)

			// Simulate search with mock results
			type mockCmd struct {
				name, grp, desc, ttp string
				opsec                int
				subs                 []string
			}
			mockCmds := []mockCmd{
				{"whoami", "sys", "Display user identity", "T1033", 9, nil},
				{"sysinfo", "sys", "Get system information", "", 0, nil},
				{"uac-bypass", "implant", "Bypass UAC", "T1548.002", 8, []string{"elevatedcom", "sspi"}},
				{"shell", "execute", "Execute shell command", "T1059", 3, nil},
				{"hashdump", "implant", "Dump SAM hashes", "T1003.002", 9, nil},
				{"session", "generic", "List sessions", "", 0, nil},
			}

			queryLower := strings.ToLower(query)
			var matched []mockCmd
			for _, c := range mockCmds {
				if group != "" && c.grp != group {
					continue
				}
				if strings.Contains(strings.ToLower(c.name), queryLower) ||
					strings.Contains(strings.ToLower(c.desc), queryLower) {
					matched = append(matched, c)
				}
			}

			if len(matched) == 0 {
				return mcp.NewToolResultText("No commands found matching: " + query), nil
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Found %d commands matching \"%s\":\n\n", len(matched), query))
			for _, cmd := range matched {
				sb.WriteString(fmt.Sprintf("- **%s** [%s]: %s\n", cmd.name, cmd.grp, cmd.desc))
			}
			return mcp.NewToolResultText(sb.String()), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Verify tool is listed
	listResult, err := srv.Client().ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatal("ListTools:", err)
	}
	if len(listResult.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(listResult.Tools))
	}
	if listResult.Tools[0].Name != "search_commands" {
		t.Errorf("tool name = %q, want %q", listResult.Tools[0].Name, "search_commands")
	}

	t.Run("search_by_name", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "search_commands"
		req.Params.Arguments = map[string]any{"query": "whoami"}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "whoami") {
			t.Errorf("result = %q, want to contain 'whoami'", got)
		}
		if !strings.Contains(got, "Found 1") {
			t.Errorf("result = %q, want to contain 'Found 1'", got)
		}
	})

	t.Run("search_by_description", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "search_commands"
		req.Params.Arguments = map[string]any{"query": "SAM"}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "hashdump") {
			t.Errorf("result = %q, want to contain 'hashdump'", got)
		}
	})

	t.Run("search_with_group_filter", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "search_commands"
		req.Params.Arguments = map[string]any{"query": "shell", "group": "sys"}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "No commands found") {
			t.Errorf("result = %q, want no results with wrong group", got)
		}
	})

	t.Run("search_no_match", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "search_commands"
		req.Params.Arguments = map[string]any{"query": "nonexistent"}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "No commands found") {
			t.Errorf("result = %q, want 'No commands found'", got)
		}
	})

	t.Run("missing_query", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "search_commands"
		req.Params.Arguments = map[string]any{}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}
		if !result.IsError {
			t.Error("expected error for missing query")
		}
	})
}
