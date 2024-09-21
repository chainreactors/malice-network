package main

import (
	"bytes"
	"fmt"
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/armory"
	"github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/explorer"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/file"
	"github.com/chainreactors/malice-network/client/command/filesystem"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/modules"
	"github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/sys"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"io"
	"os"
	"sort"
	"strings"
)

var markdownExtension = ".md"

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
		buf.WriteString(cmd.Example + "\n\n")
	}

	if err := printOptions(buf, cmd, name); err != nil {
		return err
	}
	if hasSeeAlso(cmd) {
		buf.WriteString("**SEE ALSO**\n\n")
		if cmd.HasParent() {
			parent := cmd.Parent()
			pname := parent.CommandPath()
			link := strings.ReplaceAll(pname, " ", "_")
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
			link := cname + markdownExtension
			link = strings.ReplaceAll(link, " ", "_")
			buf.WriteString(fmt.Sprintf("* [%s](%s)\t - %s\n", cname, linkHandler(link), child.Short))
		}
		buf.WriteString("\n")
	}
	_, err := buf.WriteTo(w)
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

func GenGroupHelp(writer io.Writer, con *repl.Console, groupId string, binds ...func(con *repl.Console) []*cobra.Command) {
	writer.Write([]byte(fmt.Sprintf("## %s\n", groupId)))
	for _, b := range binds {
		for _, c := range b(con) {
			c.SetHelpCommand(nil)
			_ = GenMarkdownTreeCustom(c, writer, func(s string) string {
				return "#" + s
			})
		}
	}
}

func GenImplantHelp(con *repl.Console) {
	implantMd, err := os.Create("implant.md")
	if err != nil {
		panic(err)
	}
	GenGroupHelp(implantMd, con, consts.ImplantGroup,
		tasks.Commands,
		modules.Commands,
		explorer.Commands,
		addon.Commands,
	)

	GenGroupHelp(implantMd, con, consts.ExecuteGroup,
		exec.Commands)

	GenGroupHelp(implantMd, con, consts.SysGroup,
		sys.Commands)

	GenGroupHelp(implantMd, con, consts.FileGroup,
		file.Commands,
		filesystem.Commands)
}

func GenClientHelp(con *repl.Console) {
	clientMd, err := os.Create("client.md")
	if err != nil {
		panic(err)
	}
	GenGroupHelp(clientMd, con, consts.GenericGroup,
		generic.Commands)

	GenGroupHelp(clientMd, con, consts.ManageGroup,
		sessions.Commands,
		alias.Commands,
		extension.Commands,
		armory.Commands,
		mal.Commands,
	)

	GenGroupHelp(clientMd, con, consts.ListenerGroup,
		listener.Commands,
	)
}

func main() {
	con, err := repl.NewConsole()
	if err != nil {
		fmt.Println(err)
		return
	}

	GenClientHelp(con)
	GenImplantHelp(con)
}
