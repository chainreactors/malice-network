package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/rootpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
