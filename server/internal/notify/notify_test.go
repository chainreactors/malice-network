package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestLarkWebhookService_SendWithoutSecret(t *testing.T) {
	var body map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"StatusCode":0,"StatusMessage":"success"}`))
	}))
	defer srv.Close()

	svc := NewLarkWebhookService(srv.URL, "")
	err := svc.Send(context.Background(), "test title", "test message")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Without secret, sign should be empty and timestamp should be 0
	if sign, ok := body["sign"]; ok && sign != "" {
		t.Errorf("expected empty sign without secret, got %q", sign)
	}
}

func TestLarkWebhookService_SendWithSecret(t *testing.T) {
	var body map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &body)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"StatusCode":0,"StatusMessage":"success"}`))
	}))
	defer srv.Close()

	svc := NewLarkWebhookService(srv.URL, "test-secret-key")
	err := svc.Send(context.Background(), "test title", "test message")
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	sign, ok := body["sign"]
	if !ok || sign == "" {
		t.Fatal("expected non-empty sign with secret")
	}

	ts, ok := body["timestamp"]
	if !ok {
		t.Fatal("expected timestamp with secret")
	}
	// go-lark serializes timestamp as string via strconv.FormatInt
	tsStr, isStr := ts.(string)
	if isStr {
		if tsStr == "" || tsStr == "0" {
			t.Error("expected non-zero timestamp")
		}
	} else if tsFloat, isFloat := ts.(float64); isFloat {
		if tsFloat == 0 {
			t.Error("expected non-zero timestamp")
		}
	} else {
		t.Errorf("unexpected timestamp type: %T", ts)
	}
}

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
	active := n.active.Load()
	if active == nil || active.Disabled {
		t.Fatal("expected notifier to be enabled after loading config")
	}
}

func TestNotifierInitServiceSkipsDisabledConfig(t *testing.T) {
	n := NewNotifier()

	if err := n.InitService(&configs.NotifyConfig{Enable: false}); err != nil {
		t.Fatalf("InitService failed: %v", err)
	}
	active := n.active.Load()
	if active == nil || !active.Disabled {
		t.Fatal("expected notifier to be disabled")
	}
}

func TestBuildServicesValidatesEnabledChannels(t *testing.T) {
	cases := []struct {
		name string
		cfg  *configs.NotifyConfig
	}{
		{"telegram missing credentials", &configs.NotifyConfig{
			Enable:   true,
			Telegram: &configs.TelegramConfig{Enable: true},
		}},
		{"dingtalk missing token", &configs.NotifyConfig{
			Enable:   true,
			DingTalk: &configs.DingTalkConfig{Enable: true},
		}},
		{"lark invalid webhook", &configs.NotifyConfig{
			Enable: true,
			Lark:   &configs.LarkConfig{Enable: true, WebHookUrl: "not-a-url"},
		}},
		{"serverchan invalid url", &configs.NotifyConfig{
			Enable:     true,
			ServerChan: &configs.ServerChanConfig{Enable: true, URL: "not-a-url"},
		}},
		{"pushplus missing token", &configs.NotifyConfig{
			Enable:   true,
			PushPlus: &configs.PushPlusConfig{Enable: true},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := buildServices(tc.cfg); err == nil {
				t.Fatal("expected a validation error")
			}
		})
	}
}

func TestNotifierPrepareKeepsActiveServiceOnFailure(t *testing.T) {
	n := NewNotifier()
	good := &configs.NotifyConfig{
		Enable:     true,
		ServerChan: &configs.ServerChanConfig{Enable: true, URL: "https://serverchan.example/send"},
	}
	if err := n.InitService(good); err != nil {
		t.Fatalf("InitService failed: %v", err)
	}
	active := n.active.Load()
	if _, err := n.Prepare(&configs.NotifyConfig{
		Enable:     true,
		ServerChan: &configs.ServerChanConfig{Enable: true, URL: "not-a-url"},
	}); err == nil {
		t.Fatal("expected a validation error")
	}
	if n.active.Load() != active {
		t.Fatal("failed prepare replaced the active service")
	}
}

func TestNotifierInitServiceReplacesPreviousServices(t *testing.T) {
	var deliveries int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&deliveries, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"StatusCode":0,"StatusMessage":"success"}`))
	}))
	defer srv.Close()

	n := NewNotifier()
	if err := n.InitService(&configs.NotifyConfig{
		Enable: true,
		Lark:   &configs.LarkConfig{Enable: true, WebHookUrl: srv.URL},
	}); err != nil {
		t.Fatalf("InitService failed: %v", err)
	}
	n.Send("test", "op", "message")
	if atomic.LoadInt32(&deliveries) != 1 {
		t.Fatal("expected the configured channel to deliver")
	}

	// A disabled replacement must swap the old service set out, not leave it active.
	if err := n.InitService(&configs.NotifyConfig{Enable: false}); err != nil {
		t.Fatalf("InitService failed: %v", err)
	}
	n.Send("test", "op", "message")
	if atomic.LoadInt32(&deliveries) != 1 {
		t.Fatal("disabled config left the previous service active")
	}
}
