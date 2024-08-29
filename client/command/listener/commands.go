package listener

import (
	"github.com/chainreactors/malice-network/client/command/flags"
	"github.com/chainreactors/malice-network/client/command/help"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *console.Console) []*cobra.Command {
	listenerCmd := &cobra.Command{
		Use:   "listener",
		Short: "List listeners in server",
		Long:  help.GetHelpFor("listener"),
		Run: func(cmd *cobra.Command, args []string) {
			ListenerCmd(cmd, con)
			return
		},
		GroupID: consts.ListenerGroup,
	}

	tcpCmd := &cobra.Command{
		Use:   "tcp",
		Short: "Listener tcp pipeline ctrl manager",
		Long:  help.GetHelpFor("tcp"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			listTcpPipelines(cmd, con)
			return
		},
		GroupID: consts.ListenerGroup,
	}
	carapace.Gen(tcpCmd).PositionalCompletion(carapace.ActionValues().Usage("listener id"))

	tcpStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a TCP pipeline",
		Args:  cobra.ExactArgs(4),
		Long:  help.GetHelpFor("tcp start"),
		Run: func(cmd *cobra.Command, args []string) {
			startTcpPipelineCmd(cmd, con)
			return
		},
	}

	carapace.Gen(tcpStartCmd).PositionalCompletion(
		carapace.ActionValues().Usage("tcp pipeline name"),
		carapace.ActionValues().Usage("listener id"),
		carapace.ActionValues().Usage("tcp pipeline host"),
		carapace.ActionValues().Usage("tcp pipeline port"),
	)

	flags.Bind("cert", false, tcpStartCmd, func(f *pflag.FlagSet) {
		f.String("cert_path", "", "tcp pipeline tls cert path")
		f.String("key_path", "", "tcp pipeline tls key path")
	})

	flags.BindFlagCompletions(tcpStartCmd, func(comp *carapace.ActionMap) {
		(*comp)["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		(*comp)["key_path"] = carapace.ActionFiles().Usage("path to the key file")
	})

	tcpStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a TCP pipeline",
		Args:  cobra.ExactArgs(2),
		Long:  help.GetHelpFor("tcp stop"),
		Run: func(cmd *cobra.Command, args []string) {
			stopTcpPipelineCmd(cmd, con)
			return
		},
	}
	carapace.Gen(tcpStopCmd).PositionalCompletion(
		carapace.ActionValues().Usage("tcp pipeline name"),
		carapace.ActionValues().Usage("listener id"),
	)

	tcpCmd.AddCommand(tcpStartCmd, tcpStopCmd)

	websiteCmd := &cobra.Command{
		Use:   "website",
		Short: "Listener website ctrl manager",
		Long:  help.GetHelpFor("website"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			listWebsitesCmd(cmd, con)
			return
		},
		GroupID: consts.ListenerGroup,
	}
	carapace.Gen(websiteCmd).PositionalCompletion(carapace.ActionValues().Usage("listener id"))

	websiteStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a website",
		Args:  cobra.ExactArgs(6),
		Long:  help.GetHelpFor("website start"),
		Run: func(cmd *cobra.Command, args []string) {
			startWebsiteCmd(cmd, con)
			return
		},
	}
	carapace.Gen(websiteStartCmd).PositionalCompletion(
		carapace.ActionValues().Usage("website name"),
		carapace.ActionValues().Usage("listener id"),
		carapace.ActionValues().Usage("website port"),
		carapace.ActionValues().Usage("website router root path"),
		carapace.ActionValues().Usage("website content path"),
		carapace.ActionValues().Usage("website content type"),
	)

	flags.Bind("cert", false, websiteStartCmd, func(f *pflag.FlagSet) {
		f.String("cert_path", "", "website tls cert path")
		f.String("key_path", "", "website tls key path")
	})

	flags.BindFlagCompletions(websiteStartCmd, func(comp *carapace.ActionMap) {
		(*comp)["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		(*comp)["key_path"] = carapace.ActionFiles().Usage("path to the key file")
	})

	websiteStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a website",
		Args:  cobra.ExactArgs(2),
		Long:  help.GetHelpFor("website stop"),
		Run: func(cmd *cobra.Command, args []string) {
			stopWebsitePipelineCmd(cmd, con)
			return
		},
	}
	carapace.Gen(websiteStopCmd).PositionalCompletion(
		carapace.ActionValues().Usage("website name"),
		carapace.ActionValues().Usage("listener id"),
	)

	websiteCmd.AddCommand(websiteStartCmd, websiteStopCmd)

	return []*cobra.Command{listenerCmd, tcpCmd, websiteCmd}

}
