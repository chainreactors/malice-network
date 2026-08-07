package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/rootpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	config "github.com/gookit/config/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"
)

func TestAddListenerAuthUsesConfiguredServerAddress(t *testing.T) {
	_ = newRPCTestEnv(t)
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	config.Set("server.ip", "198.51.100.10")
	config.Set("server.grpc_port", 7443)

	resp, err := (&Server{}).AddListener(context.Background(), &rootpb.Operator{
		Args: []string{"remote-listener"},
	})
	if err != nil {
		t.Fatalf("AddListener failed: %v", err)
	}

	auth := &mtls.ClientConfig{}
	if err := yaml.Unmarshal([]byte(resp.Response), auth); err != nil {
		t.Fatalf("unmarshal listener auth: %v", err)
	}
	if auth.Host != "198.51.100.10" || auth.Port != 7443 {
		t.Fatalf("listener auth address = %s, want 198.51.100.10:7443", auth.Address())
	}
}

func TestRemoveListenerRejectsActiveRuntimeListener(t *testing.T) {
	_ = newRPCTestEnv(t)
	seedForwardListenerOperator(t, "active-runtime-listener", "active-runtime-fp")
	core.Listeners.Add(core.NewListener("active-runtime-listener", "10.0.0.5"))

	resp, err := (&Server{}).RemoveListener(context.Background(), &rootpb.Operator{
		Args: []string{"active-runtime-listener"},
	})

	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("RemoveListener error = %v, want FailedPrecondition", err)
	}
	if resp == nil || resp.Status != 1 || resp.Error == "" {
		t.Fatalf("response = %#v, want error response", resp)
	}
	if _, err := db.FindOperatorByName("active-runtime-listener"); err != nil {
		t.Fatalf("listener operator should still exist after rejected remove: %v", err)
	}
}

func TestRemoveListenerRejectsActiveForwardRuntime(t *testing.T) {
	_ = newRPCTestEnv(t)
	t.Cleanup(resetForwardListenerRuntimes)
	seedForwardListenerOperator(t, "active-forward-listener", "active-forward-fp")
	forwardListenerRuntimes.Store("active-forward-listener", &forwardListenerRuntime{
		listenerID:  "active-forward-listener",
		connectHost: "127.0.0.1",
		connectPort: 5005,
		address:     "127.0.0.1:5005",
		fingerprint: "active-forward-fp",
	})

	resp, err := (&Server{}).RemoveListener(context.Background(), &rootpb.Operator{
		Args: []string{"active-forward-listener"},
	})

	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("RemoveListener error = %v, want FailedPrecondition", err)
	}
	if resp == nil || resp.Status != 1 || resp.Error == "" {
		t.Fatalf("response = %#v, want error response", resp)
	}
	if _, err := db.FindOperatorByName("active-forward-listener"); err != nil {
		t.Fatalf("listener operator should still exist after rejected remove: %v", err)
	}
}

func TestRemoveListenerPublishesIdentityLifecycleEvent(t *testing.T) {
	_ = newRPCTestEnv(t)
	seedForwardListenerOperator(t, "inactive-listener", "inactive-listener-fp")
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)

	resp, err := (&Server{}).RemoveListener(context.Background(), &rootpb.Operator{
		Args: []string{"inactive-listener"},
	})
	if err != nil || resp == nil || resp.Status != 0 {
		t.Fatalf("RemoveListener response = %#v, error = %v", resp, err)
	}

	event := waitForLifecycleEvent(t, events, consts.CtrlListenerRemove)
	if event.EventType != consts.EventListener || event.Listener.GetId() != "inactive-listener" {
		t.Fatalf("unexpected listener identity event: %#v", event)
	}
}
