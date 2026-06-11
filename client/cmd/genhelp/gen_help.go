package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/basic"
	"github.com/chainreactors/malice-network/client/command/build"
	"github.com/chainreactors/malice-network/client/command/cert"
	configCmd "github.com/chainreactors/malice-network/client/command/config"
	"github.com/chainreactors/malice-network/client/command/context"
	"github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/explorer"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/file"
	"github.com/chainreactors/malice-network/client/command/filesystem"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/modules"
	"github.com/chainreactors/malice-network/client/command/mutant"
	"github.com/chainreactors/malice-network/client/command/pipe"
	"github.com/chainreactors/malice-network/client/command/pipeline"
	"github.com/chainreactors/malice-network/client/command/pivot"
	"github.com/chainreactors/malice-network/client/command/privilege"
	"github.com/chainreactors/malice-network/client/command/reg"
	"github.com/chainreactors/malice-network/client/command/service"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/sys"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/command/taskschd"
	"github.com/chainreactors/malice-network/client/command/third"
	"github.com/chainreactors/malice-network/client/command/website"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/spf13/cobra"
)

const clientReferenceFrontmatter = `---
title: Client
---

`

const implantReferenceFrontmatter = `---
title: Implant
---

`

const communityReferenceFrontmatter = `---
title: Community MAL
---

`

func init() {
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	}, config.WithHookFunc(assets.HookFn))
	config.AddDriver(yaml.Driver)
}

type byName []*cobra.Command

func (s byName) Len() int           { return len(s) }
func (s byName) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s byName) Less(i, j int) bool { return s[i].Name() < s[j].Name() }

func hasSeeAlso(cmd *cobra.Command) bool {
	if cmd.HasParent() {
		return true
	}
	for _, c := range cmd.Commands() {
		if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
			continue
		}
		return true
	}
	return false
}

func printOptions(buf *bytes.Buffer, cmd *cobra.Command, name string) error {
	flags := cmd.NonInheritedFlags()
	flags.SetOutput(buf)
	if flags.HasAvailableFlags() {
		buf.WriteString("**Options**\n\n```\n")
		flags.PrintDefaults()
		buf.WriteString("```\n\n")
	}

	parentFlags := cmd.InheritedFlags()
	parentFlags.SetOutput(buf)
	if parentFlags.HasAvailableFlags() {
		buf.WriteString("**Options inherited from parent commands**\n\n```\n")
		parentFlags.PrintDefaults()
		buf.WriteString("```\n\n")
	}
	return nil
}

func normalizeExamples(example string) string {
	lines := strings.Split(example, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "~~~" {
			lines[i] = "~~~"
		}
	}
	return strings.Join(lines, "\n")
}

func isMarkdownSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n'
}

func normalizeBoldSegment(segment string) string {
	var out strings.Builder
	last := 0

	for i := 0; i < len(segment)-1; i++ {
		if segment[i] != '*' || segment[i+1] != '*' {
			continue
		}
		if i > 0 && segment[i-1] == '*' {
			continue
		}
		if i+2 < len(segment) && segment[i+2] == '*' {
			continue
		}

		close := -1
		for j := i + 2; j < len(segment)-1; j++ {
			if segment[j] != '*' || segment[j+1] != '*' {
				continue
			}
			if j > 0 && segment[j-1] == '*' {
				continue
			}
			if j+2 < len(segment) && segment[j+2] == '*' {
				continue
			}
			if strings.TrimSpace(segment[i+2:j]) == "" {
				continue
			}
			close = j
			break
		}
		if close == -1 {
			continue
		}

		out.WriteString(segment[last:i])
		if i > 0 && !isMarkdownSpace(segment[i-1]) {
			out.WriteByte(' ')
		}
		out.WriteString(segment[i : close+2])
		if close+2 < len(segment) && !isMarkdownSpace(segment[close+2]) {
			out.WriteByte(' ')
		}
		last = close + 2
		i = close + 1
	}

	if last == 0 {
		return segment
	}
	out.WriteString(segment[last:])
	return out.String()
}

func normalizeBoldSpacingLine(line string) string {
	var out strings.Builder
	inCode := false
	delimiter := ""
	segmentStart := 0

	for i := 0; i < len(line); {
		if line[i] != '`' {
			i++
			continue
		}

		j := i + 1
		for j < len(line) && line[j] == '`' {
			j++
		}
		ticks := line[i:j]

		if !inCode {
			if segmentStart < i {
				out.WriteString(normalizeBoldSegment(line[segmentStart:i]))
			}
			inCode = true
			delimiter = ticks
			segmentStart = i
		} else if ticks == delimiter {
			out.WriteString(line[segmentStart:j])
			inCode = false
			delimiter = ""
			segmentStart = j
		}

		i = j
	}

	if segmentStart < len(line) {
		if inCode {
			out.WriteString(line[segmentStart:])
		} else {
			out.WriteString(normalizeBoldSegment(line[segmentStart:]))
		}
	}

	return out.String()
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

func normalizeMarkdown(content string) string {
	var out strings.Builder
	inFence := false
	fenceMarker := byte(0)
	previousBlank := true
	previousList := false
	activeListIndents := []int{}

	for _, line := range strings.SplitAfter(content, "\n") {
		body := strings.TrimRight(line, "\r\n")
		newline := line[len(body):]
		stripped := strings.TrimLeft(body, " \t")

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
			out.WriteString(normalizeBoldSpacingLine(body))
			out.WriteString(newline)
			previousBlank = false
			previousList = true
			activeListIndents = updateListContext(activeListIndents, indent)
			continue
		}

		out.WriteString(normalizeBoldSpacingLine(body))
		out.WriteString(newline)
		previousBlank = false
		previousList = false
		activeListIndents = activeListIndents[:0]
	}

	return out.String()
}

func GenMarkdownCustom(cmd *cobra.Command, w io.Writer, linkHandler func(string) string) error {
	//cmd.InitDefaultHelpCmd()
	//cmd.InitDefaultHelpFlag()

	buf := new(bytes.Buffer)
	name := cmd.CommandPath()
	if cmd.HasParent() {
		buf.WriteString("#### " + name + "\n\n")
	} else {
		buf.WriteString("### " + name + "\n\n")
	}
	buf.WriteString(cmd.Short + "\n\n")
	if len(cmd.Long) > 0 {
		buf.WriteString("**Description**\n\n")
		buf.WriteString(cmd.Long + "\n\n")
	}

	if cmd.Runnable() {
		buf.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.UseLine()))
	}

	if len(cmd.Example) > 0 {
		buf.WriteString("**Examples**\n\n")
		buf.WriteString(normalizeExamples(cmd.Example) + "\n\n")
	}

	if err := printOptions(buf, cmd, name); err != nil {
		return err
	}
	if hasSeeAlso(cmd) {
		buf.WriteString("**SEE ALSO**\n\n")
		if cmd.HasParent() {
			parent := cmd.Parent()
			pname := parent.CommandPath()
			link := strings.ReplaceAll(pname, " ", "-")
			buf.WriteString(fmt.Sprintf("* [%s](%s)\t - %s\n", pname, linkHandler(link), parent.Short))
			cmd.VisitParents(func(c *cobra.Command) {
				if c.DisableAutoGenTag {
					cmd.DisableAutoGenTag = c.DisableAutoGenTag
				}
			})
		}

		children := cmd.Commands()
		sort.Sort(byName(children))

		for _, child := range children {
			if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
				continue
			}
			cname := name + " " + child.Name()
			buf.WriteString(fmt.Sprintf("* [%s](%s)\t - %s\n", cname, linkHandler(cname), child.Short))
		}
		buf.WriteString("\n")
	}
	_, err := io.WriteString(w, normalizeMarkdown(buf.String()))
	if cmd.HasSubCommands() {
		for _, sub := range cmd.Commands() {
			if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
				continue
			}
			GenMarkdownCustom(sub, w, linkHandler)
		}
	}
	return err
}

func GenMarkdownTreeCustom(cmd *cobra.Command, writer io.Writer, linkHandler func(string) string) error {
	//for _, c := range cmd.Commands() {
	//	if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() {
	//		continue
	//	}
	//	if err := GenMarkdownTreeCustom(c, writer, linkHandler); err != nil {
	//		return err
	//	}
	//}

	if err := GenMarkdownCustom(cmd, writer, linkHandler); err != nil {
		return err
	}
	return nil
}

func GenGroupHelp(writer io.Writer, con *core.Console, groupId string, binds ...func(con *core.Console) []*cobra.Command) {
	writer.Write([]byte(fmt.Sprintf("## %s\n", groupId)))
	for _, b := range binds {
		cmds := b(con)
		sort.Sort(byName(cmds))
		for _, c := range cmds {
			c.SetHelpCommand(nil)
			_ = GenMarkdownTreeCustom(c, writer, func(s string) string {
				return "#" + strings.ReplaceAll(s, " ", "-")
			})
		}
	}
}

func GenImplantHelp(con *core.Console) {
	implantMd, err := os.Create("docs/reference/commands/implant.md")
	if err != nil {
		panic(err)
	}
	implantMd.Write([]byte(implantReferenceFrontmatter))

	GenGroupHelp(implantMd, con, consts.ImplantGroup,
		basic.Commands,
		tasks.Commands,
		modules.Commands,
		explorer.Commands,
		addon.Commands,
	)

	GenGroupHelp(implantMd, con, consts.ExecuteGroup,
		exec.Commands)

	GenGroupHelp(implantMd, con, consts.SysGroup,
		sys.Commands,
		service.Commands,
		reg.Commands,
		taskschd.Commands,
		privilege.Commands,
		third.Commands,
	)

	GenGroupHelp(implantMd, con, consts.FileGroup,
		file.Commands,
		filesystem.Commands,
		pipe.Commands)

	GenGroupHelp(implantMd, con, consts.PivotGroup,
		pivot.Commands,
	)
}

func GenClientHelp(con *core.Console) {
	clientMd, err := os.Create("docs/reference/commands/client.md")
	if err != nil {
		panic(err)
	}
	clientMd.Write([]byte(clientReferenceFrontmatter))
	GenGroupHelp(clientMd, con, consts.GenericGroup,
		generic.Commands)

	GenGroupHelp(clientMd, con, consts.ManageGroup,
		sessions.Commands,
		alias.Commands,
		extension.Commands,
		armory.Commands,
		mal.Commands,
		configCmd.Commands,
		context.Commands,
		cert.Commands,
	)

	GenGroupHelp(clientMd, con, consts.ListenerGroup,
		listener.Commands,
		website.Commands,
		pipeline.Commands,
	)

	GenGroupHelp(clientMd, con, consts.GeneratorGroup,
		build.Commands,
		mutant.Commands)

}

func GenMalHelper(con *core.Console, name string) {
	clientMd, err := os.Create("docs/reference/commands/" + name + ".md")
	if err != nil {
		panic(err)
	}
	if name == "community" {
		clientMd.Write([]byte(communityReferenceFrontmatter))
	}

	rpc := clientrpc.NewMaliceRPCClient(nil)
	intermediate.RegisterBuiltin(rpc)
	command.RegisterClientFunc(con)
	command.RegisterImplantFunc(con)
	clientMd.Write([]byte(fmt.Sprintf("## %s\n", name)))
	for _, p := range plugin.GetGlobalMalManager().GetAllEmbeddedPlugins() {
		var cmds []*cobra.Command
		for _, cc := range p.CMDs {
			cmds = append(cmds, cc.Command)
		}
		sort.Sort(byName(cmds))
		for _, c := range cmds {
			c.SetHelpCommand(nil)
			_ = GenMarkdownTreeCustom(c, clientMd, func(s string) string {
				return "#" + strings.ReplaceAll(s, " ", "-")
			})
		}
	}
}

func main() {
	con, err := core.NewConsole()
	if err != nil {
		fmt.Println(err)
		return
	}

	GenClientHelp(con)
	GenImplantHelp(con)
	GenMalHelper(con, "community")
}
