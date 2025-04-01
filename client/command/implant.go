package command

import (
	"fmt"
	"github.com/carapace-sh/carapace"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/addon"
	"github.com/chainreactors/malice-network/client/command/alias"
	"github.com/chainreactors/malice-network/client/command/basic"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/exec"
	"github.com/chainreactors/malice-network/client/command/explorer"
	"github.com/chainreactors/malice-network/client/command/extension"
	"github.com/chainreactors/malice-network/client/command/file"
	"github.com/chainreactors/malice-network/client/command/filesystem"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/command/mal"
	"github.com/chainreactors/malice-network/client/command/modules"
	"github.com/chainreactors/malice-network/client/command/pipe"
	"github.com/chainreactors/malice-network/client/command/pivot"
	"github.com/chainreactors/malice-network/client/command/privilege"
	"github.com/chainreactors/malice-network/client/command/reg"
	"github.com/chainreactors/malice-network/client/command/service"
	"github.com/chainreactors/malice-network/client/command/sys"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/command/taskschd"
	"github.com/chainreactors/malice-network/client/command/third"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/core/plugin"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/tui"
)

func ImplantCmd(con *repl.Console) *cobra.Command {
	makeCommands := BindImplantCommands(con)
	cmd := makeCommands()
	cmd.Use = consts.ImplantMenu
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	common.Bind(cmd.Use, true, cmd, func(f *pflag.FlagSet) {
		f.String("use", "", "set session context")
		f.Bool("wait", false, "wait task finished")
	})
	cobra.MarkFlagRequired(cmd.Flags(), "use")
	cmd.PersistentPreRunE, cmd.PersistentPostRunE = makeRunners(cmd, con)
	makeCompleters(cmd, con)
	return cmd
}

func makeRunners(implantCmd *cobra.Command, con *repl.Console) (pre, post func(cmd *cobra.Command, args []string) error) {
	// so we can have access to active sessions/beacons, and other stuff needed.
	pre = func(cmd *cobra.Command, args []string) error {
		// Set the active target.
		err := implantCmd.Parent().PersistentPreRunE(implantCmd, args)
		if err != nil {
			return err
		}
		sid, _ := cmd.Flags().GetString("use")
		if sid == "" {
			return fmt.Errorf("no implant to run command on")
		}

		var session *core.Session
		var ok bool

		if session, ok = con.GetLocalSession(sid); !ok {
			return fmt.Errorf("session %s not found", sid)
		}

		con.ActiveTarget.Set(session)
		con.App.SwitchMenu(consts.ImplantMenu)

		return nil
	}
	post = func(cmd *cobra.Command, args []string) error {
		sess := con.GetInteractive()
		if sess.LastTask != nil {
			if wait, _ := cmd.Flags().GetBool("wait"); wait {
				RegisterImplantFunc(con)
				context, err := con.WaitTaskFinish(sess.Context(), sess.LastTask)
				if err != nil {
					return err
				}
				core.HandlerTask(sess, context, nil, consts.CalleeCMD, true)
			} else {
				con.Log.Console(tui.RendStructDefault(sess.LastTask))
			}
		}

		return implantCmd.Parent().PersistentPostRunE(implantCmd, args)
	}

	return pre, post
}

func makeCompleters(cmd *cobra.Command, con *repl.Console) {
	comps := carapace.Gen(cmd)

	comps.PreRun(func(cmd *cobra.Command, args []string) {
		cmd.PersistentPreRunE(cmd, args)
	})

	// Bind completers to flags (wrap them to use the same pre-runners)
	common.BindFlagCompletions(cmd, func(comp carapace.ActionMap) {
		comp["use"] = carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			cmd.PersistentPreRunE(cmd, c.Args)
			return common.SessionIDCompleter(con)
		})
	})
}

func BindBuiltinCommands(con *repl.Console, root *cobra.Command) *cobra.Command {
	bind := MakeBind(root, con)
	BindCommonCommands(bind)
	bind(consts.ImplantGroup,
		basic.Commands,
		tasks.Commands,
		modules.Commands,
		explorer.Commands,
		addon.Commands,
	)

	bind(consts.ExecuteGroup,
		exec.Commands)

	bind(consts.SysGroup,
		sys.Commands,
		service.Commands,
		reg.Commands,
		taskschd.Commands,
		privilege.Commands,
		third.Commands,
	)

	bind(consts.FileGroup,
		file.Commands,
		filesystem.Commands,
		pipe.Commands,
	)

	bind(consts.PivotGroup,
		pivot.Commands,
	)
	bind(consts.ArmoryGroup)
	bind(consts.AddonGroup)
	bind(consts.MalGroup)

	root.InitDefaultHelpCmd()
	root.SetHelpCommandGroupID(consts.GenericGroup)
	return root
}

func BindImplantCommands(con *repl.Console) console.Commands {
	implantCommands := func() *cobra.Command {
		implant := &cobra.Command{
			Use:   "implant",
			Short: "implant commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
			//GroupID: consts.ImplantMenu,
		}
		BindBuiltinCommands(con, implant)

		// Load Aliases
		aliasManifests := assets.GetInstalledAliasManifests()
		for _, manifest := range aliasManifests {
			manifest, err := alias.LoadAlias(manifest, con)
			if err != nil {
				con.Log.Errorf("Failed to load alias: %s\n", err)
				continue
			}
			err = alias.RegisterAlias(manifest, implant, con)
			if err != nil {
				con.Log.Errorf("Failed to register alias: %s\n", err)
				continue
			}
		}
		// Load Extensions
		extensionManifests := assets.GetInstalledExtensionManifests()
		for _, manifest := range extensionManifests {
			mext, err := extension.LoadExtensionManifest(manifest)
			if err != nil {
				con.Log.Errorf("Failed to load extension: %s\n", err)
				continue
			}

			for _, ext := range mext.ExtCommand {
				extension.ExtensionRegisterCommand(ext, implant, con)
			}
		}

		if con.App.Menu(consts.ClientMenu).Commands() == nil {
			return implant
		}

		plugin.GlobalPlugins = plugin.LoadGlobalLuaPlugin()
		for _, malName := range plugin.GetPluginManifest() {
			_, err := mal.LoadMalWithManifest(con, implant, malName)
			if err != nil {
				con.Log.Errorf("Failed to load mal %s: %s\n", malName.Name, err)
				continue
			}
		}

		implant.SetUsageFunc(help.UsageFunc)
		implant.SetHelpFunc(help.HelpFunc)
		return implant
	}
	return implantCommands
}

func RegisterImplantFunc(con *repl.Console) {
	tasks.Register(con)
	basic.Register(con)
	sys.Register(con)
	file.Register(con)
	third.Register(con)
	filesystem.Register(con)
	modules.Register(con)
	exec.Register(con)
	alias.Register(con)
	extension.Register(con)
	addon.Register(con)
	service.Register(con)
	reg.Register(con)
	taskschd.Register(con)
	privilege.Register(con)
	pipe.Register(con)
	pivot.Register(con)
}
