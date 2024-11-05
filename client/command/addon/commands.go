package addon

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strings"
)

func Commands(con *repl.Console) []*cobra.Command {
	listaddonCmd := &cobra.Command{
		Use:   consts.ModuleListAddon + " [addon]",
		Short: "List all addons",
		Run: func(cmd *cobra.Command, args []string) {
			AddonListCmd(cmd, con)
			return
		},
	}

	loadaddonCmd := &cobra.Command{
		Use:   consts.ModuleLoadAddon,
		Short: "Load an addon",
		Long:  `Load an executable into the implant's memory for reuse`,
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			LoadAddonCmd(cmd, con)
			return
		},
		Example: `addon default name is filename, default module is selected based on the file extension
~~~	
load_addon gogo.exe
~~~
assigns an alias name gogo to the addon, and the specified module is execute_exe
~~~
load_addon gogo.exe -n gogo -m execute_exe
~~~
`,
	}

	common.BindFlag(loadaddonCmd, func(f *pflag.FlagSet) {
		f.StringP("module", "m", "", "module type")
		f.StringP("name", "n", "", "addon name")
		//f.StringP("method", "t", "inline", "method type")
	})

	common.BindArgCompletions(loadaddonCmd, nil,
		carapace.ActionFiles().Usage("path the addon file to load"))

	common.BindFlagCompletions(loadaddonCmd, func(comp carapace.ActionMap) {
		comp["module"] = carapace.ActionValues(consts.ExecuteModules...).Usage("executable module")
		//comp["method"] = carapace.ActionValues("inline", "execute", "shellcode").Usage("method types")
	})

	execAddonCmd := &cobra.Command{
		Use:   consts.ModuleExecuteAddon,
		Short: "Execute the loaded addon",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ExecuteAddonCmd(cmd, con)
			return
		},
		Example: `Execute the addon without "-" arguments
~~~
execute_addon httpx 1.1.1.1
~~~
execute the addon file with "-" arguments, you need add "--" before the arguments
~~~
execute_addon gogo.exe -- -i 127.0.0.1 -p http
~~~
if you specify the addon name, you need to use the alias name
~~~
execute_addon gogo -- -i 127.0.0.1 -p http
~~~
`,
	}
	common.BindFlag(execAddonCmd, common.ExecuteFlagSet, common.SacrificeFlagSet)
	common.BindArgCompletions(execAddonCmd, nil, common.SessionAddonComplete(con))

	return []*cobra.Command{listaddonCmd, loadaddonCmd, execAddonCmd}
}

func Register(con *repl.Console) {
	con.RegisterImplantFunc(consts.ModuleListAddon,
		ListAddon,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			exts := content.Spite.GetAddons()
			if len(exts.Addons) == 0 {
				return "", fmt.Errorf("no addon found")
			}
			con.UpdateSession(content.Session.SessionId)
			var s strings.Builder
			s.WriteString("\n")
			for _, ext := range exts.Addons {
				s.WriteString(fmt.Sprintf("%s\t%s\t%s\n", ext.Name, ext.Type, ext.Depend))
			}
			return s.String(), nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			exts := content.Spite.GetAddons()
			if len(exts.Addons) == 0 {
				return "", fmt.Errorf("no addon found")
			}
			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				{Title: "Name", Width: 25},
				{Title: "Type", Width: 10},
				{Title: "Depend", Width: 35},
			}, true)
			for _, ext := range exts.Addons {
				row = table.Row{ext.Name, ext.Type, ext.Depend}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})
	con.RegisterImplantFunc(consts.ModuleLoadAddon,
		LoadAddon,
		"",
		nil,
		func(content *clientpb.TaskContext) (interface{}, error) {
			con.UpdateSession(content.Session.SessionId)
			return "addon loaded", nil
		}, nil)

	con.RegisterImplantFunc(consts.ModuleExecuteAddon, ExecuteAddon, "", nil, common.ParseAssembly, nil)
}
