package core

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/mcptest"
	"github.com/mark3labs/mcp-go/server"
	lua "github.com/yuin/gopher-lua"
)

// TestLuaVMBasicExecution verifies the Lua VM can execute scripts and return results.
func TestLuaVMBasicExecution(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()

	tests := []struct {
		name     string
		script   string
		wantTop  int
		wantVal  string
		wantErr  bool
	}{
		{
			name:    "return_string",
			script:  `return "hello"`,
			wantTop: 1,
			wantVal: "hello",
		},
		{
			name:    "return_number",
			script:  `return 42`,
			wantTop: 1,
			wantVal: "42",
		},
		{
			name:    "return_concat",
			script:  `return "IoM" .. " " .. "v1.0"`,
			wantTop: 1,
			wantVal: "IoM v1.0",
		},
		{
			name:    "return_table_length",
			script:  `local t = {1,2,3}; return #t`,
			wantTop: 1,
			wantVal: "3",
		},
		{
			name:    "return_multiple",
			script:  `return "a", "b"`,
			wantTop: 2,
		},
		{
			name:    "no_return",
			script:  `local x = 1 + 1`,
			wantTop: 0,
		},
		{
			name:    "syntax_error",
			script:  `if then end`,
			wantErr: true,
		},
		{
			name:    "runtime_error",
			script:  `error("test error")`,
			wantErr: true,
		},
		{
			name:    "string_operations",
			script:  `return string.format("user:%s pid:%d", "admin", 1234)`,
			wantTop: 1,
			wantVal: "user:admin pid:1234",
		},
		{
			name:    "table_concat",
			script:  `return table.concat({"a","b","c"}, ",")`,
			wantTop: 1,
			wantVal: "a,b,c",
		},
		{
			name:    "math_operations",
			script:  `return math.floor(3.7)`,
			wantTop: 1,
			wantVal: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset stack before each test
			vm.SetTop(0)

			err := vm.DoString(tt.script)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			top := vm.GetTop()
			if top != tt.wantTop {
				t.Errorf("stack top = %d, want %d", top, tt.wantTop)
			}

			if tt.wantVal != "" && top > 0 {
				got := vm.Get(1).String()
				if got != tt.wantVal {
					t.Errorf("result = %q, want %q", got, tt.wantVal)
				}
			}
		})
	}
}

// TestLuaVMStdLibAvailability confirms standard Lua libraries are available.
func TestLuaVMStdLibAvailability(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()

	libs := []struct {
		name   string
		script string
	}{
		{"string", `return type(string.format)`},
		{"table", `return type(table.concat)`},
		{"math", `return type(math.floor)`},
		{"os", `return type(os.time)`},
		{"io", `return type(io.open)`},
	}

	for _, lib := range libs {
		t.Run(lib.name, func(t *testing.T) {
			vm.SetTop(0)
			if err := vm.DoString(lib.script); err != nil {
				t.Fatalf("stdlib %s not available: %v", lib.name, err)
			}
			if vm.GetTop() < 1 || vm.Get(1).String() != "function" {
				t.Errorf("stdlib %s: expected function type, got %v", lib.name, vm.Get(1))
			}
		})
	}
}

// TestMCPExecuteLuaTool tests the execute_lua MCP tool protocol behavior.
func TestMCPExecuteLuaTool(t *testing.T) {
	ctx := context.Background()

	// Create a mock execute_lua tool that runs real Lua scripts
	srv, err := mcptest.NewServer(t, server.ServerTool{
		Tool: mcp.NewTool("execute_lua",
			mcp.WithDescription("Execute Lua script"),
			mcp.WithString("script", mcp.Required(), mcp.Description("Lua script to execute")),
			mcp.WithString("session_id", mcp.Description("Optional session ID")),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			script, err := request.RequireString("script")
			if err != nil || script == "" {
				return mcp.NewToolResultError("script is required"), nil
			}

			// Execute with a real Lua VM
			vm := lua.NewState()
			defer vm.Close()

			if err := vm.DoString(script); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Lua error: %v", err)), nil
			}

			top := vm.GetTop()
			if top == 0 {
				return mcp.NewToolResultText("Script executed successfully (no return value)"), nil
			}

			var results []string
			for i := 1; i <= top; i++ {
				results = append(results, fmt.Sprintf("%v", vm.Get(i)))
			}

			return mcp.NewToolResultText(strings.Join(results, "\n")), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	// Verify tool registration
	listResult, err := srv.Client().ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatal("ListTools:", err)
	}
	if len(listResult.Tools) != 1 || listResult.Tools[0].Name != "execute_lua" {
		t.Fatalf("expected execute_lua tool, got %v", listResult.Tools)
	}

	t.Run("simple_return", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "execute_lua"
		req.Params.Arguments = map[string]any{"script": `return "hello from lua"`}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if got != "hello from lua" {
			t.Errorf("got %q, want %q", got, "hello from lua")
		}
	})

	t.Run("string_format", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "execute_lua"
		req.Params.Arguments = map[string]any{
			"script": `return string.format("sessions: %d", 5)`,
		}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if got != "sessions: 5" {
			t.Errorf("got %q, want %q", got, "sessions: 5")
		}
	})

	t.Run("no_return_value", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "execute_lua"
		req.Params.Arguments = map[string]any{"script": `local x = 1`}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, "no return value") {
			t.Errorf("got %q, want 'no return value'", got)
		}
	})

	t.Run("lua_error", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "execute_lua"
		req.Params.Arguments = map[string]any{"script": `error("test failure")`}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		if !result.IsError {
			t.Error("expected error result for Lua runtime error")
		}
	})

	t.Run("syntax_error", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "execute_lua"
		req.Params.Arguments = map[string]any{"script": `if then`}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		if !result.IsError {
			t.Error("expected error for syntax error")
		}
	})

	t.Run("missing_script", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "execute_lua"
		req.Params.Arguments = map[string]any{}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		if !result.IsError {
			t.Error("expected error for missing script")
		}
	})

	t.Run("table_operations", func(t *testing.T) {
		var req mcp.CallToolRequest
		req.Params.Name = "execute_lua"
		req.Params.Arguments = map[string]any{
			"script": `
local t = {}
for i = 1, 5 do table.insert(t, "item" .. i) end
return table.concat(t, ", ")
`,
		}

		result, err := srv.Client().CallTool(ctx, req)
		if err != nil {
			t.Fatal("CallTool:", err)
		}

		got, err := extractTextContent(result)
		if err != nil {
			t.Fatal(err)
		}
		if got != "item1, item2, item3, item4, item5" {
			t.Errorf("got %q", got)
		}
	})
}
