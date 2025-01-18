package config

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func Commands(con *repl.Console) []*cobra.Command {
	configCmd := &cobra.Command{
		Use:   consts.CommandConfig,
		Short: "Config operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	configRefreshCmd := &cobra.Command{
		Use:   consts.CommandRefresh,
		Short: "Refresh config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RefreshCmd(cmd, con)
		},
	}

	common.BindFlag(configRefreshCmd, func(f *pflag.FlagSet) {
		f.Bool("client", false, "Refresh client config")
	})

	githubCmd := &cobra.Command{
		Use:   consts.CommandGithub,
		Short: "Show Github config and more operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetGithubConfigCmd(cmd, con)
		},
	}
	githubUpdateCmd := &cobra.Command{
		Use:   consts.CommandConfigUpdate,
		Short: "Update Github config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateGithubConfigCmd(cmd, con)
		},
	}

	common.BindFlag(githubUpdateCmd, common.GithubFlagSet)

	githubCmd.AddCommand(githubUpdateCmd)

	notifyCmd := &cobra.Command{
		Use:   consts.CommandNotify,
		Short: "Show Notify config and more operations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return GetNotifyCmd(cmd, con)
		},
	}

	notifyUpdateCmd := &cobra.Command{
		Use:   consts.CommandConfigUpdate,
		Short: "Update Notify config",
		RunE: func(cmd *cobra.Command, args []string) error {
			return UpdateNotifyCmd(cmd, con)
		},
	}

	common.BindFlag(notifyUpdateCmd, common.TelegramSet, common.DingTalkSet, common.LarkSet, common.ServerChanSet)

	notifyCmd.AddCommand(notifyUpdateCmd)

	configCmd.AddCommand(configRefreshCmd, githubCmd, notifyCmd)
	return []*cobra.Command{configCmd}
}
