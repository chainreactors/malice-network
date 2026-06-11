package config

import (
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func TestMergeGithubUpdatePreservesUnchangedFields(t *testing.T) {
	cmd := &cobra.Command{Use: "update"}
	GithubFlagSet(cmd.Flags())
	if err := cmd.ParseFlags([]string{"--repo", "new-repo"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	merged := mergeGithubUpdate(&clientpb.GithubActionBuildConfig{
		Owner:      "old-owner",
		Repo:       "old-repo",
		Token:      "old-token",
		WorkflowId: "old.yml",
	}, cmd)

	if merged.Owner != "old-owner" || merged.Token != "old-token" || merged.WorkflowId != "old.yml" {
		t.Fatalf("expected unchanged github fields to be preserved: %#v", merged)
	}
	if merged.Repo != "new-repo" {
		t.Fatalf("expected repo override, got %#v", merged)
	}
}

func TestMergeGithubUpdateAllowsExplicitWorkflowClear(t *testing.T) {
	cmd := &cobra.Command{Use: "update"}
	GithubFlagSet(cmd.Flags())
	if err := cmd.ParseFlags([]string{"--workflowFile", ""}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
	if err := cmd.Flags().Set("workflowFile", ""); err != nil {
		t.Fatalf("failed to set workflow flag: %v", err)
	}

	merged := mergeGithubUpdate(&clientpb.GithubActionBuildConfig{
		Owner:      "old-owner",
		Repo:       "old-repo",
		Token:      "old-token",
		WorkflowId: "old.yml",
	}, cmd)

	if merged.WorkflowId != "" {
		t.Fatalf("expected workflow to be cleared, got %#v", merged)
	}
}

func TestMergeNotifyUpdatePreservesUnchangedFields(t *testing.T) {
	cmd := &cobra.Command{Use: "update"}
	TelegramSet(cmd.Flags())
	DingTalkSet(cmd.Flags())
	LarkSet(cmd.Flags())
	ServerChanSet(cmd.Flags())
	PushPlusSet(cmd.Flags())
	if err := cmd.ParseFlags([]string{"--lark-enable", "--lark-webhook-url", "https://new.example/hook"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	merged := mergeNotifyUpdate(&clientpb.Notify{
		TelegramEnable:   true,
		TelegramApiKey:   "telegram-token",
		TelegramChatId:   123,
		DingtalkEnable:   true,
		DingtalkSecret:   "ding-secret",
		DingtalkToken:    "ding-token",
		LarkEnable:       false,
		LarkWebhookUrl:   "https://old.example/hook",
		LarkSecret:       "old-secret",
		ServerchanEnable: true,
		ServerchanUrl:    "https://serverchan.example/send",
		PushplusEnable:   true,
		PushplusToken:    "push-token",
		PushplusTopic:    "ops",
		PushplusChannel:  "wechat",
	}, cmd)

	if !merged.TelegramEnable || merged.TelegramApiKey != "telegram-token" || merged.TelegramChatId != 123 {
		t.Fatalf("expected telegram config preserved: %#v", merged)
	}
	if !merged.DingtalkEnable || merged.DingtalkSecret != "ding-secret" || merged.DingtalkToken != "ding-token" {
		t.Fatalf("expected dingtalk config preserved: %#v", merged)
	}
	if !merged.ServerchanEnable || merged.ServerchanUrl != "https://serverchan.example/send" {
		t.Fatalf("expected serverchan config preserved: %#v", merged)
	}
	if !merged.PushplusEnable || merged.PushplusToken != "push-token" || merged.PushplusTopic != "ops" || merged.PushplusChannel != "wechat" {
		t.Fatalf("expected pushplus config preserved: %#v", merged)
	}
	if !merged.LarkEnable || merged.LarkWebhookUrl != "https://new.example/hook" || merged.LarkSecret != "old-secret" {
		t.Fatalf("expected lark override with preserved secret: %#v", merged)
	}
}

func TestMergeNotifyUpdateAllowsExplicitDisable(t *testing.T) {
	cmd := &cobra.Command{Use: "update"}
	TelegramSet(cmd.Flags())
	if err := cmd.ParseFlags([]string{}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
	if err := cmd.Flags().Set("telegram-enable", "false"); err != nil {
		t.Fatalf("failed to set telegram-enable: %v", err)
	}

	merged := mergeNotifyUpdate(&clientpb.Notify{
		TelegramEnable: true,
		TelegramApiKey: "telegram-token",
		TelegramChatId: 123,
	}, cmd)

	if merged.TelegramEnable {
		t.Fatalf("expected telegram to be disabled, got %#v", merged)
	}
	if merged.TelegramApiKey != "telegram-token" || merged.TelegramChatId != 123 {
		t.Fatalf("expected telegram credentials preserved when toggling enable: %#v", merged)
	}
}
