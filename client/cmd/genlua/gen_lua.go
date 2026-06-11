package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	_ "github.com/chainreactors/malice-network/client/cmd/cli"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/mals"
	"github.com/spf13/cobra"
)

var luaReferenceTitles = map[string]string{
	"docs/reference/lua-api/builtin.md": "Builtin",
	"docs/reference/lua-api/rpc.md":     "RPC",
	"docs/reference/lua-api/beacon.md":  "Beacon",
}

func main() {
	con, err := core.NewConsole()
	if err != nil {
		fmt.Println(err)
		return
	}
	var cmd = &cobra.Command{
		Use:   "client",
		Short: "",
		Long:  ``,
	}
	cmd.TraverseChildren = true
	command.BindBuiltinCommands(con, cmd)
	command.BindClientsCommands(con)
	rpc := clientrpc.NewMaliceRPCClient(nil)
	intermediate.RegisterBuiltin(rpc)
	command.RegisterClientFunc(con)
	command.RegisterImplantFunc(con)
	vm := plugin.NewLuaVM()
	plug := &plugin.LuaPlugin{DefaultPlugin: &plugin.DefaultPlugin{MalManiFest: &plugin.MalManiFest{}}}
	plug.InitLuaContext(vm)
	plug.RegisterLuaFunction()

	if os.Getenv("MALICE_GENLUA_STUBS") == "1" {
		must(mals.GenerateLuaDefinitionFile(vm, intermediate.BuiltinPackage, plugin.ProtoPackage, intermediate.InternalFunctions.Package(intermediate.BuiltinPackage)))
		must(mals.GenerateLuaDefinitionFile(vm, intermediate.RpcPackage, plugin.ProtoPackage, intermediate.InternalFunctions.Package(intermediate.RpcPackage)))
		must(mals.GenerateLuaDefinitionFile(vm, intermediate.BeaconPackage, plugin.ProtoPackage, intermediate.InternalFunctions.Package(intermediate.BeaconPackage)))
	}

	must(mals.GenerateMarkdownDefinitionFile(vm, intermediate.BuiltinPackage, "docs/reference/lua-api/builtin.md", intermediate.InternalFunctions.Package(intermediate.BuiltinPackage)))
	must(mals.GenerateMarkdownDefinitionFile(vm, intermediate.RpcPackage, "docs/reference/lua-api/rpc.md", intermediate.InternalFunctions.Package(intermediate.RpcPackage)))
	must(mals.GenerateMarkdownDefinitionFile(vm, intermediate.BeaconPackage, "docs/reference/lua-api/beacon.md", intermediate.InternalFunctions.Package(intermediate.BeaconPackage)))
	must(addLuaReferenceFrontmatter("docs/reference/lua-api/builtin.md"))
	must(addLuaReferenceFrontmatter("docs/reference/lua-api/rpc.md"))
	must(addLuaReferenceFrontmatter("docs/reference/lua-api/beacon.md"))
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func addLuaReferenceFrontmatter(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if bytesHasFrontmatter(content) {
		normalized := []byte(normalizeMarkdownListSpacing(string(content)))
		if string(normalized) == string(content) {
			return nil
		}
		return os.WriteFile(path, normalized, 0o644)
	}
	title := luaReferenceTitles[filepath.ToSlash(path)]
	if title == "" {
		title = "Lua API"
	}
	frontmatter := fmt.Sprintf("---\ntitle: %s\n---\n\n", title)
	body := normalizeMarkdownListSpacing(string(content))
	return os.WriteFile(path, append([]byte(frontmatter), []byte(body)...), 0o644)
}

func bytesHasFrontmatter(content []byte) bool {
	return len(content) >= 4 && string(content[:4]) == "---\n"
}

func listItemIndent(line string) (int, bool) {
	width := 0
	i := 0
	for i < len(line) {
		switch line[i] {
		case ' ':
			width++
		case '\t':
			width += 4 - (width % 4)
		default:
			goto marker
		}
		i++
	}

marker:
	if i >= len(line) {
		return 0, false
	}

	markerEnd := i
	switch line[i] {
	case '-', '+', '*':
		markerEnd = i + 1
	default:
		if line[i] < '0' || line[i] > '9' {
			return 0, false
		}
		for markerEnd < len(line) && line[markerEnd] >= '0' && line[markerEnd] <= '9' {
			markerEnd++
		}
		if markerEnd >= len(line) || (line[markerEnd] != '.' && line[markerEnd] != ')') {
			return 0, false
		}
		markerEnd++
	}

	if markerEnd >= len(line) || (line[markerEnd] != ' ' && line[markerEnd] != '\t') {
		return 0, false
	}
	for markerEnd < len(line) && (line[markerEnd] == ' ' || line[markerEnd] == '\t') {
		markerEnd++
	}
	if markerEnd >= len(line) {
		return 0, false
	}
	return width, true
}

func hasListContext(active []int, indent int) bool {
	for _, value := range active {
		if value <= indent {
			return true
		}
	}
	return false
}

func updateListContext(active []int, indent int) []int {
	updated := active[:0]
	seen := false
	for _, value := range active {
		if value <= indent {
			updated = append(updated, value)
			if value == indent {
				seen = true
			}
		}
	}
	if !seen {
		updated = append(updated, indent)
	}
	return updated
}

func normalizeMarkdownListSpacing(content string) string {
	var out strings.Builder
	inFrontmatter := strings.HasPrefix(content, "---\n")
	inFence := false
	fenceMarker := byte(0)
	previousBlank := true
	previousList := false
	activeListIndents := []int{}

	for index, line := range strings.SplitAfter(content, "\n") {
		body := strings.TrimRight(line, "\r\n")
		newline := line[len(body):]
		stripped := strings.TrimLeft(body, " \t")

		if inFrontmatter {
			out.WriteString(line)
			if index > 0 && (strings.TrimSpace(body) == "---" || strings.TrimSpace(body) == "...") {
				inFrontmatter = false
				previousBlank = true
				previousList = false
				activeListIndents = activeListIndents[:0]
			}
			continue
		}

		if strings.HasPrefix(stripped, "```") || strings.HasPrefix(stripped, "~~~") {
			marker := stripped[0]
			if !inFence {
				inFence = true
				fenceMarker = marker
			} else if marker == fenceMarker {
				inFence = false
				fenceMarker = 0
			}
			out.WriteString(line)
			previousBlank = false
			previousList = false
			continue
		}

		if inFence {
			out.WriteString(line)
			previousBlank = false
			previousList = false
			continue
		}

		if strings.TrimSpace(body) == "" {
			out.WriteString(line)
			previousBlank = true
			previousList = false
			activeListIndents = activeListIndents[:0]
			continue
		}

		if indent, ok := listItemIndent(body); ok {
			inExistingList := previousList || hasListContext(activeListIndents, indent)
			if !previousBlank && !inExistingList {
				blankIndent := body[:len(body)-len(strings.TrimLeft(body, " \t"))]
				if newline != "" {
					out.WriteString(blankIndent + newline)
				} else {
					out.WriteString(blankIndent + "\n")
				}
				previousBlank = true
			}
			out.WriteString(line)
			previousBlank = false
			previousList = true
			activeListIndents = updateListContext(activeListIndents, indent)
			continue
		}

		out.WriteString(line)
		previousBlank = false
		previousList = false
		activeListIndents = activeListIndents[:0]
	}

	return out.String()
}
