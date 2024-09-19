package listener

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	listenerCmd := &cobra.Command{
		Use:   consts.CommandListener,
		Short: "List listeners in server",
		Long:  help.GetHelpFor(consts.CommandListener),
		Run: func(cmd *cobra.Command, args []string) {
			ListenerCmd(cmd, con)
			return
		},
	}

	jobCmd := &cobra.Command{
		Use:   consts.CommandJob,
		Short: "List jobs in server",
		Long:  help.GetHelpFor(consts.CommandJob),
		Run: func(cmd *cobra.Command, args []string) {
			listJobsCmd(cmd, con)
			return
		},
	}

	tcpCmd := &cobra.Command{
		Use:   consts.CommandTcp,
		Short: "Listener tcp pipeline ctrl manager",
		Long:  help.GetHelpFor(consts.CommandTcp),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			listTcpCmd(cmd, con)
			return
		},
		GroupID: consts.ListenerGroup,
	}

	common.BindArgCompletions(tcpCmd, nil, common.ListenerIDCompleter(con))

	tcpRegisterCmd := &cobra.Command{
		Use:   consts.CommandRegister,
		Short: "Register a new TCP pipeline",
		Args:  cobra.ExactArgs(1),
		Long:  help.GetHelpFor("tcp register"),
		Run: func(cmd *cobra.Command, args []string) {
			newTcpPipelineCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(tcpRegisterCmd, nil,
		carapace.ActionValues().Usage("tcp pipeline name"))

	common.BindFlag(tcpRegisterCmd, common.TlsCertFlagSet, common.PipelineFlagSet)

	common.BindFlagCompletions(tcpRegisterCmd, func(comp carapace.ActionMap) {
		comp["listener_id"] = common.ListenerIDCompleter(con)
		comp["host"] = carapace.ActionValues().Usage("tcp host")
		comp["port"] = carapace.ActionValues().Usage("tcp port")
		comp["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key_path"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
	})

	tcpStartCmd := &cobra.Command{
		Use:   consts.CommandPipelineStart,
		Short: "Start a TCP pipeline",
		Args:  cobra.ExactArgs(2),
		Long:  help.GetHelpFor("tcp start"),
		Run: func(cmd *cobra.Command, args []string) {
			startTcpPipelineCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(tcpStartCmd, nil,
		carapace.ActionValues().Usage("tcp pipeline name"),
		common.ListenerIDCompleter(con))

	tcpStopCmd := &cobra.Command{
		Use:   consts.CommandPipelineStop,
		Short: "Stop a TCP pipeline",
		Args:  cobra.ExactArgs(2),
		Long:  help.GetHelpFor("tcp stop"),
		Run: func(cmd *cobra.Command, args []string) {
			stopTcpPipelineCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(tcpStopCmd, nil,
		carapace.ActionValues().Usage("tcp pipeline name"),
		common.ListenerIDCompleter(con))

	tcpCmd.AddCommand(tcpRegisterCmd, tcpStartCmd, tcpStopCmd)

	websiteCmd := &cobra.Command{
		Use:   consts.CommandWebsite,
		Short: "Listener website ctrl manager",
		Long:  help.GetHelpFor(consts.CommandWebsite),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			listWebsitesCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(websiteCmd, nil, common.ListenerIDCompleter(con))

	websiteRegisterCmd := &cobra.Command{
		Use:   consts.CommandRegister,
		Short: "register a website",
		Args:  cobra.ExactArgs(3),
		Long:  help.GetHelpFor("website Register"),
		Run: func(cmd *cobra.Command, args []string) {
			newWebsiteCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(websiteRegisterCmd, nil,
		carapace.ActionValues().Usage("website name"),
		carapace.ActionValues().Usage("website router root path"),
		carapace.ActionFiles().Usage("website content path"))

	common.BindFlag(websiteRegisterCmd, common.TlsCertFlagSet, common.PipelineFlagSet, func(f *pflag.FlagSet) {
		f.String("content_type", "", "website content type")
	})

	common.BindFlagCompletions(websiteRegisterCmd, func(comp carapace.ActionMap) {
		comp["listener_id"] = common.ListenerIDCompleter(con)
		comp["port"] = carapace.ActionValues().Usage("website port")
		comp["content_type"] = carapace.ActionFiles().Tag("website content type")
		comp["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		comp["key_path"] = carapace.ActionFiles().Usage("path to the key file")
		comp["tls"] = carapace.ActionValues().Usage("enable tls")
	})

	websiteStartCmd := &cobra.Command{
		Use:   consts.CommandPipelineStart,
		Short: "Start a website",
		Args:  cobra.ExactArgs(2),
		Long:  help.GetHelpFor("website start"),
		Run: func(cmd *cobra.Command, args []string) {
			startWebsitePipelineCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(websiteStartCmd, nil,
		carapace.ActionValues().Usage("website name"),
		common.ListenerIDCompleter(con))

	websiteStopCmd := &cobra.Command{
		Use:   consts.CommandPipelineStop,
		Short: "Stop a website",
		Args:  cobra.ExactArgs(2),
		Long:  help.GetHelpFor("website stop"),
		Run: func(cmd *cobra.Command, args []string) {
			stopWebsitePipelineCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(websiteStopCmd, nil,
		carapace.ActionValues().Usage("website name"),
		common.ListenerIDCompleter(con))

	websiteCmd.AddCommand(websiteRegisterCmd, websiteStartCmd, websiteStopCmd)

	return []*cobra.Command{listenerCmd, jobCmd, tcpCmd, websiteCmd}

}
