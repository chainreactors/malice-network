package notify

import (
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestNotifierInitServiceUsesConfiguredChannels(t *testing.T) {
	n := NewNotifier()
	cfg := &configs.NotifyConfig{
		Enable: true,
		DingTalk: &configs.DingTalkConfig{
			Enable: true,
			Secret: "ding-secret",
			Token:  "ding-token",
		},
		Lark: &configs.LarkConfig{
			Enable:     true,
			WebHookUrl: "https://lark.example/hook",
			Secret:     "lark-secret",
		},
		ServerChan: &configs.ServerChanConfig{
			Enable: true,
			URL:    "https://serverchan.example/send",
		},
		PushPlus: &configs.PushPlusConfig{
			Enable:  true,
			Token:   "push-token",
			Topic:   "soc",
			Channel: "wechat",
		},
	}

	if err := n.InitService(cfg); err != nil {
		t.Fatalf("InitService failed: %v", err)
	}
	if !n.enable {
		t.Fatal("expected notifier to be enabled after loading config")
	}
}

func TestNotifierInitServiceSkipsDisabledConfig(t *testing.T) {
	n := NewNotifier()

	if err := n.InitService(&configs.NotifyConfig{Enable: false}); err != nil {
		t.Fatalf("InitService failed: %v", err)
	}
	if n.enable {
		t.Fatal("expected notifier to remain disabled")
	}
}
