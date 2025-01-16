package config

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
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

	common.BindFlag(githubUpdateCmd, GithubFlagSet)

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

	common.BindFlag(notifyUpdateCmd, TelegramSet, DingTalkSet, LarkSet, ServerChanSet)

	notifyCmd.AddCommand(notifyUpdateCmd)

	configCmd.AddCommand(configRefreshCmd, githubCmd, notifyCmd)
	return []*cobra.Command{configCmd}
}

func GithubFlagSet(f *pflag.FlagSet) {
	f.String("owner", "", "github owner")
	f.String("repo", "", "github repo")
	f.String("token", "", "github token")
	f.String("workflowFile", "", "github workflow file")
}

func ParseGithubFlags(cmd *cobra.Command) (string, string, string, string) {
	owner, _ := cmd.Flags().GetString("owner")
	repo, _ := cmd.Flags().GetString("repo")
	token, _ := cmd.Flags().GetString("token")
	file, _ := cmd.Flags().GetString("workflowFile")
	return owner, repo, token, file
}

func TelegramSet(f *pflag.FlagSet) {
	f.Bool("telegram-enable", false, "enable telegram")
	f.String("telegram-token", "", "telegram token")
	f.Int64("telegram-chat-id", 0, "telegram chat id")
}

func DingTalkSet(f *pflag.FlagSet) {
	f.Bool("dingtalk-enable", false, "enable dingtalk")
	f.String("dingtalk-secret", "", "dingtalk secret")
	f.String("dingtalk-token", "", "dingtalk token")
}

func LarkSet(f *pflag.FlagSet) {
	f.Bool("lark-enable", false, "enable lark")
	f.String("lark-webhook-url", "", "lark webhook url")
}

func ServerChanSet(f *pflag.FlagSet) {
	f.Bool("serverchan-enable", false, "enable serverchan")
	f.String("serverchan-url", "", "serverchan url")
}

func ParseNotifyFlags(cmd *cobra.Command) *clientpb.Notify {
	telegramEnable, _ := cmd.Flags().GetBool("telegram-enable")
	dingTalkEnable, _ := cmd.Flags().GetBool("dingtalk-enable")
	larkEnable, _ := cmd.Flags().GetBool("lark-enable")
	serverChanEnable, _ := cmd.Flags().GetBool("serverchan-enable")

	telegramToken, _ := cmd.Flags().GetString("telegram-token")
	telegramChatID, _ := cmd.Flags().GetInt64("telegram-chat-id")
	dingTalkSecret, _ := cmd.Flags().GetString("dingtalk-secret")
	dingTalkToken, _ := cmd.Flags().GetString("dingtalk-token")
	larkWebhookURL, _ := cmd.Flags().GetString("lark-webhook-url")
	serverChanURL, _ := cmd.Flags().GetString("serverchan-url")

	notifyConfig := &clientpb.Notify{
		TelegramEnable: telegramEnable,
		TelegramApiKey: telegramToken,
		TelegramChatId: telegramChatID,

		DingtalkEnable: dingTalkEnable,
		DingtalkSecret: dingTalkSecret,
		DingtalkToken:  dingTalkToken,

		LarkEnable:     larkEnable,
		LarkWebhookUrl: larkWebhookURL,

		ServerchanEnable: serverChanEnable,
		ServerchanUrl:    serverChanURL,
	}

	return notifyConfig
}
