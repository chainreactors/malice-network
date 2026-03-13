package config

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func mergeGithubUpdate(existing *clientpb.GithubActionBuildConfig, cmd *cobra.Command) *clientpb.GithubActionBuildConfig {
	merged := &clientpb.GithubActionBuildConfig{}
	if existing != nil {
		*merged = *existing
	}

	if cmd.Flags().Changed("owner") {
		merged.Owner, _ = cmd.Flags().GetString("owner")
	}
	if cmd.Flags().Changed("repo") {
		merged.Repo, _ = cmd.Flags().GetString("repo")
	}
	if cmd.Flags().Changed("token") {
		merged.Token, _ = cmd.Flags().GetString("token")
	}
	if cmd.Flags().Changed("workflowFile") {
		merged.WorkflowId, _ = cmd.Flags().GetString("workflowFile")
	}

	return merged
}

func mergeNotifyUpdate(existing *clientpb.Notify, cmd *cobra.Command) *clientpb.Notify {
	merged := &clientpb.Notify{}
	if existing != nil {
		*merged = *existing
	}

	mergeBoolFlag(cmd, "telegram-enable", &merged.TelegramEnable)
	mergeStringFlag(cmd, "telegram-token", &merged.TelegramApiKey)
	mergeInt64Flag(cmd, "telegram-chat-id", &merged.TelegramChatId)

	mergeBoolFlag(cmd, "dingtalk-enable", &merged.DingtalkEnable)
	mergeStringFlag(cmd, "dingtalk-secret", &merged.DingtalkSecret)
	mergeStringFlag(cmd, "dingtalk-token", &merged.DingtalkToken)

	mergeBoolFlag(cmd, "lark-enable", &merged.LarkEnable)
	mergeStringFlag(cmd, "lark-webhook-url", &merged.LarkWebhookUrl)
	mergeStringFlag(cmd, "lark-secret", &merged.LarkSecret)

	mergeBoolFlag(cmd, "serverchan-enable", &merged.ServerchanEnable)
	mergeStringFlag(cmd, "serverchan-url", &merged.ServerchanUrl)

	mergeBoolFlag(cmd, "pushplus-enable", &merged.PushplusEnable)
	mergeStringFlag(cmd, "pushplus-token", &merged.PushplusToken)
	mergeStringFlag(cmd, "pushplus-topic", &merged.PushplusTopic)
	mergeStringFlag(cmd, "pushplus-channel", &merged.PushplusChannel)

	return merged
}

func mergeBoolFlag(cmd *cobra.Command, name string, target *bool) {
	if cmd.Flags().Changed(name) {
		*target, _ = cmd.Flags().GetBool(name)
	}
}

func mergeStringFlag(cmd *cobra.Command, name string, target *string) {
	if cmd.Flags().Changed(name) {
		*target, _ = cmd.Flags().GetString(name)
	}
}

func mergeInt64Flag(cmd *cobra.Command, name string, target *int64) {
	if cmd.Flags().Changed(name) {
		*target, _ = cmd.Flags().GetInt64(name)
	}
}
