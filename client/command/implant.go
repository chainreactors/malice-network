package command

import (
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/tui"
	"github.com/reeflective/console"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"os"

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
	"github.com/chainreactors/malice-network/client/command/modules"
	"github.com/chainreactors/malice-network/client/command/pipe"
	"github.com/chainreactors/malice-network/client/command/pivot"
	"github.com/chainreactors/malice-network/client/command/privilege"
	"github.com/chainreactors/malice-network/client/command/pty"
	"github.com/chainreactors/malice-network/client/command/reg"
	"github.com/chainreactors/malice-network/client/command/service"
	"github.com/chainreactors/malice-network/client/command/sys"
	"github.com/chainreactors/malice-network/client/command/tasks"
	"github.com/chainreactors/malice-network/client/command/taskschd"
	"github.com/chainreactors/malice-network/client/command/third"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/plugin"
)

func ImplantCmd(con *core.Console) *cobra.Command {
	makeCommands := BindImplantCommands(con)
	cmd := makeCommands()
	cmd.Use = consts.ImplantMenu
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	common.Bind(cmd.Use, true, cmd, func(f *pflag.FlagSet) {
		f.String("use", "", "set session context")
		f.Bool("wait", false, "wait task finished")
		f.Bool("yes", false, "skip confirmation prompts")
	})
	cobra.MarkFlagRequired(cmd.Flags(), "use")
	cmd.PersistentPreRunE, cmd.PersistentPostRunE = makeRunners(cmd, con)
	makeCompleters(cmd, con)
	return cmd
}

func makeRunners(implantCmd *cobra.Command, con *core.Console) (pre, post func(cmd *cobra.Command, args []string) error) {
	allowGroups := map[string]struct{}{
		consts.GenericGroup:   {},
		consts.ManageGroup:    {},
		consts.ListenerGroup:  {},
		consts.GeneratorGroup: {},
	}
	isCompletion := func() bool {
		if os.Getenv("IOM_COMPLETING") == "1" {
			return true
		}
		if _, ok := os.LookupEnv("CARAPACE_COMPLINE"); ok {
			return true
		}
		if _, ok := os.LookupEnv("COMP_LINE"); ok {
			return true
		}
		return false
	}
	getGroupID := func(cmd *cobra.Command) string {
		for c := cmd; c != nil; c = c.Parent() {
			if c.GroupID != "" {
				return c.GroupID
			}
		}
		return ""
	}

	// so we can have access to active sessions/beacons, and other stuff needed.
	pre = func(cmd *cobra.Command, args []string) error {
		if isCompletion() {
			return nil
		}
		if cmd.Annotations["resource"] == "true" {
			return nil
		}
		// Set the active target.
		if implantCmd.Parent() != nil {
			err := implantCmd.Parent().PersistentPreRunE(implantCmd, args)
			if err != nil {
				return err
			}
		}
		if _, ok := allowGroups[getGroupID(cmd)]; ok {
			return nil
		}
		sid, _ := cmd.Flags().GetString("use")
		if sid == "" && con.ActiveTarget.Session == nil {
			return fmt.Errorf("no implant to run command on")
		} else if sid == "" && con.ActiveTarget.Session != nil {
			sid = con.ActiveTarget.Session.SessionId
		}

		var session *client.Session
		var err error
		session, err = con.GetOrUpdateSession(sid)
		if err != nil || session == nil {
			if con.ActiveTarget != nil && con.ActiveTarget.Get() != nil && con.ActiveTarget.Get().SessionId == sid {
				session = con.ActiveTarget.Get()
			} else {
				return fmt.Errorf("session %s not found", sid)
			}
		}

		//if !session.IsAlive {
		//	con.Log.Warnf("Session %s is marked dead, continuing anyway\n", sid)
		//}

		con.ActiveTarget.Set(session)
		con.App.SwitchMenu(consts.ImplantMenu)

		return nil
	}
	post = func(cmd *cobra.Command, args []string) error {
		sess := con.GetInteractive()
		wait, _ := cmd.Flags().GetBool("wait")
		if !wait {
			if implantCmd.Parent() != nil {
				return implantCmd.Parent().PersistentPostRunE(implantCmd, args)
			}
			return nil
		}
		if sess.LastTask != nil {
			if wait {
				RegisterImplantFunc(con)
				context, err := con.WaitTaskFinish(sess.Context(), sess.LastTask)
				if err != nil {
					return err
				}
				core.HandlerTask(sess, sess.Log, context, nil, consts.CalleeCMD, true)
			} else {
				con.Log.Console(tui.RendStructDefault(sess.LastTask))
			}
		}
		if implantCmd.Parent() != nil {
			return implantCmd.Parent().PersistentPostRunE(implantCmd, args)
		}
		return nil
	}

	return pre, post
}

func makeCompleters(cmd *cobra.Command, con *core.Console) {
	comps := carapace.Gen(cmd)

	comps.PreRun(func(cmd *cobra.Command, args []string) {
		_ = os.Setenv("IOM_COMPLETING", "1")
		defer os.Unsetenv("IOM_COMPLETING")
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

func BindCommand(cmds []*cobra.Command) func(con *core.Console) []*cobra.Command {
	return func(con *core.Console) []*cobra.Command {
		return cmds
	}
}

func BindBuiltinCommands(con *core.Console, root *cobra.Command) *cobra.Command {
	bind := MakeBind(root, con, "golang")
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

	bind(consts.ThirdGroup,
		pty.Commands,
	)

	root.InitDefaultHelpCmd()
	root.SetHelpCommandGroupID(consts.GenericGroup)
	return root
}

func BindImplantCommands(con *core.Console) console.Commands {
	implantCommands := func() *cobra.Command {
		implant := &cobra.Command{
			Use:   "implant",
			Short: "implant commands",
			CompletionOptions: cobra.CompletionOptions{
				HiddenDefaultCmd: true,
			},
			//GroupID: consts.ImplantMenu,
		}
		common.Bind(implant.Use, true, implant, func(f *pflag.FlagSet) {
			f.String("use", "", "set session context")
			f.Bool("wait", false, "wait task finished")
			f.Bool("yes", false, "skip confirmation prompts")
		})
		cobra.MarkFlagRequired(implant.Flags(), "use")
		implant.PersistentPreRunE, implant.PersistentPostRunE = makeRunners(implant, con)
		makeCompleters(implant, con)

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

		if con.MalManager == nil {
			con.MalManager = plugin.GetGlobalMalManager()
		}

		// 注册嵌入式插件命令
		embeddedBind := MakeBind(implant, con, "mal")
		customCommands := con.MalManager.GetEmbeddedCommandsByLevel(plugin.CustomLevel)
		if len(customCommands) > 0 {
			embeddedBind(plugin.CustomLevel.String(), BindCommand(customCommands))
		}

		communityCommands := con.MalManager.GetEmbeddedCommandsByLevel(plugin.CommunityLevel)
		if len(communityCommands) > 0 {
			embeddedBind(plugin.CommunityLevel.String(), BindCommand(communityCommands))
		}

		professionalCommands := con.MalManager.GetEmbeddedCommandsByLevel(plugin.ProfessionalLevel)
		if len(professionalCommands) > 0 {
			embeddedBind(plugin.ProfessionalLevel.String(), BindCommand(professionalCommands))
		}

		// 注册外部插件命令
		externalBind := MakeBind(implant, con, "mal")
		for _, plug := range con.MalManager.GetAllExternalPlugins() {
			externalBind(plug.Manifest().Name, BindCommand(plug.Commands().Commands()))
		}

		implant.SetUsageFunc(help.UsageFunc)
		implant.SetHelpFunc(help.HelpFunc)
		return implant
	}
	return implantCommands
}

func RegisterImplantFunc(con *core.Console) {
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
