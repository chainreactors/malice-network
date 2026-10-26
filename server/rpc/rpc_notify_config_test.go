package rpc

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	config "github.com/gookit/config/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestUpdateNotifyConfigTransaction pins the transactional contract of
// UpdateNotifyConfig: the persisted configuration and the running
// notification service change together or not at all.
func TestUpdateNotifyConfigTransaction(t *testing.T) {
	configs.InitTestConfigRuntime(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  enable: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Mirror the persistence hook PrepareConfig installs at server boot:
	// every set.* event rewrites the config file.
	config.WithOptions(config.WithHookFunc(func(event string, c *config.Config) {
		if strings.HasPrefix(event, "set.") {
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0o600)
			if err != nil {
				return
			}
			defer f.Close()
			_, _ = config.DumpTo(f, config.Yaml)
		}
	}))
	if err := config.LoadFiles(path); err != nil {
		t.Fatal(err)
	}

	oldBroker := core.EventBroker
	core.EventBroker = core.NewBroker()
	t.Cleanup(func() {
		core.EventBroker.Stop()
		core.EventBroker = oldBroker
	})

	var deliveries atomic.Int64
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deliveries.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"StatusCode":0,"StatusMessage":"success"}`))
	}))
	defer receiver.Close()

	waitDeliveries := func(want int64) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if deliveries.Load() >= want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("deliveries = %d, want >= %d", deliveries.Load(), want)
	}

	rpcServer := &Server{}
	ctx := context.Background()

	if _, err := rpcServer.UpdateNotifyConfig(ctx, nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil request: got %v, want InvalidArgument", err)
	}

	if _, err := rpcServer.UpdateNotifyConfig(ctx, &clientpb.Notify{
		LarkEnable:     true,
		LarkWebhookUrl: receiver.URL,
	}); err != nil {
		t.Fatalf("valid update failed: %v", err)
	}
	if got := configs.GetNotifyConfig(); got == nil || got.Lark == nil || !got.Enable || got.Lark.WebHookUrl != receiver.URL {
		t.Fatalf("loaded configuration not updated: %#v", got)
	}
	persisted, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(persisted), receiver.URL) {
		t.Fatalf("persisted configuration missing the new lark webhook url:\n%s", persisted)
	}
	core.EventBroker.Notify(core.Event{EventType: "test", Op: "op", Message: "message"})
	waitDeliveries(1)

	// A failing update must leave both the persisted configuration and the
	// running service untouched.
	if _, err := rpcServer.UpdateNotifyConfig(ctx, &clientpb.Notify{
		LarkEnable:     true,
		LarkWebhookUrl: "not-a-url",
	}); err == nil {
		t.Fatal("invalid update unexpectedly succeeded")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(persisted, after) {
		t.Fatal("failed update altered the persisted configuration")
	}
	if got := configs.GetNotifyConfig(); got == nil || got.Lark == nil || got.Lark.WebHookUrl != receiver.URL {
		t.Fatalf("failed update altered the loaded configuration: %#v", got)
	}
	core.EventBroker.Notify(core.Event{EventType: "test", Op: "op", Message: "message"})
	waitDeliveries(2)
}
