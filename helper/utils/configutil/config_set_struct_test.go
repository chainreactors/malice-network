package configutil

import (
	"testing"
	"time"

	"github.com/gookit/config/v2"
)

type testNotify struct {
	Enable bool      `config:"enable"`
	Lark   *testLark `config:"lark"`
}

type testLark struct {
	Enable     bool   `config:"enable"`
	WebHookUrl string `config:"webhook_url"`
}

func TestSetStructByTag_UsesConfigTagKey(t *testing.T) {
	prefix := "test.notify." + time.Now().Format("150405.000000")

	err := SetStructByTag(prefix, &testNotify{
		Enable: true,
		Lark: &testLark{
			Enable:     true,
			WebHookUrl: "https://example.com/hook/abc",
		},
	}, "config")
	if err != nil {
		t.Fatalf("SetStructByTag failed: %v", err)
	}

	if !config.Bool(prefix + ".enable") {
		t.Fatalf("%s.enable should be true", prefix)
	}

	if !config.Bool(prefix + ".lark.enable") {
		t.Fatalf("%s.lark.enable should be true", prefix)
	}

	got := config.String(prefix + ".lark.webhook_url")
	if got != "https://example.com/hook/abc" {
		t.Fatalf("%s.lark.webhook_url = %q, want %q", prefix, got, "https://example.com/hook/abc")
	}

	wrong := config.String(prefix + ".lark.webhookurl")
	if wrong != "" {
		t.Fatalf("%s.lark.webhookurl should be empty, got %q", prefix, wrong)
	}
}
