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
		Use:   "listener",
		Short: "List listeners in server",
		Long:  help.GetHelpFor("listener"),
		Run: func(cmd *cobra.Command, args []string) {
			ListenerCmd(cmd, con)
			return
		},
	}

	jobCmd := &cobra.Command{
		Use:   "job",
		Short: "List jobs in server",
		Args:  cobra.ExactArgs(1),
		Long:  help.GetHelpFor("job"),
		Run: func(cmd *cobra.Command, args []string) {
			listJobsCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(jobCmd, nil, carapace.ActionValues().Usage("listener id"))

	tcpCmd := &cobra.Command{
		Use:   "tcp",
		Short: "Listener tcp pipeline ctrl manager",
		Long:  help.GetHelpFor("tcp"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			listTcpCmd(cmd, con)
			return
		},
		GroupID: consts.ListenerGroup,
	}

	common.BindArgCompletions(tcpCmd, nil, carapace.ActionValues().Usage("listener id"))

	tcpRegisterCmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new TCP pipeline",
		Args:  cobra.ExactArgs(4),
		Long:  help.GetHelpFor("tcp register"),
		Run: func(cmd *cobra.Command, args []string) {
			newTcpPipelineCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(tcpRegisterCmd, nil,
		carapace.ActionValues().Usage("tcp pipeline name"),
		carapace.ActionValues().Usage("listener id"),
		carapace.ActionValues().Usage("tcp pipeline host"),
		carapace.ActionValues().Usage("tcp pipeline port"))

	common.Bind("cert", false, tcpRegisterCmd, func(f *pflag.FlagSet) {
		f.String("cert_path", "", "tcp pipeline tls cert path")
		f.String("key_path", "", "tcp pipeline tls key path")
	})

	common.BindFlagCompletions(tcpRegisterCmd, func(comp *carapace.ActionMap) {
		(*comp)["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		(*comp)["key_path"] = carapace.ActionFiles().Usage("path to the key file")
	})

	tcpStartCmd := &cobra.Command{
		Use:   "start",
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
		carapace.ActionValues().Usage("listener id"))

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

	common.BindArgCompletions(tcpStopCmd, nil,
		carapace.ActionValues().Usage("tcp pipeline name"),
		carapace.ActionValues().Usage("listener id"))

	tcpCmd.AddCommand(tcpRegisterCmd, tcpStartCmd, tcpStopCmd)

	websiteCmd := &cobra.Command{
		Use:   "website",
		Short: "Listener website ctrl manager",
		Long:  help.GetHelpFor("website"),
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			listWebsitesCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(websiteCmd, nil, carapace.ActionValues().Usage("listener id"))

	websiteRegisterCmd := &cobra.Command{
		Use:   "register",
		Short: "register a website",
		Args:  cobra.ExactArgs(6),
		Long:  help.GetHelpFor("website Register"),
		Run: func(cmd *cobra.Command, args []string) {
			newWebsiteCmd(cmd, con)
			return
		},
	}

	common.BindArgCompletions(websiteRegisterCmd, nil,
		carapace.ActionValues().Usage("website name"),
		carapace.ActionValues().Usage("listener id"),
		carapace.ActionValues().Usage("website port"),
		carapace.ActionValues().Usage("website router root path"),
		carapace.ActionValues().Usage("website content path"),
		carapace.ActionValues().Usage("website content type"))

	common.Bind("cert", false, websiteRegisterCmd, func(f *pflag.FlagSet) {
		f.String("cert_path", "", "website tls cert path")
		f.String("key_path", "", "website tls key path")
	})

	common.BindFlagCompletions(websiteRegisterCmd, func(comp *carapace.ActionMap) {
		(*comp)["cert_path"] = carapace.ActionFiles().Usage("path to the cert file")
		(*comp)["key_path"] = carapace.ActionFiles().Usage("path to the key file")
	})

	websiteStartCmd := &cobra.Command{
		Use:   "start",
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
		carapace.ActionValues().Usage("listener id"))

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

	common.BindArgCompletions(websiteStopCmd, nil,
		carapace.ActionValues().Usage("website name"),
		carapace.ActionValues().Usage("listener id"))

	websiteCmd.AddCommand(websiteRegisterCmd, websiteStartCmd, websiteStopCmd)

	return []*cobra.Command{listenerCmd, jobCmd, tcpCmd, websiteCmd}

}
