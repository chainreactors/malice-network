package listener_test

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestForwardConnectCommandSendsTypedRPC(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Recorder.OnForwardListenerStatus("ConnectForwardListener", func(_ context.Context, _ any) (*clientpb.ForwardListenerStatus, error) {
		return &clientpb.ForwardListenerStatus{
			ListenerId:  "listener-a",
			ConnectHost: "10.0.0.5",
			ConnectPort: 5005,
			Active:      true,
		}, nil
	})

	if err := h.ExecuteClient("listener", "forward", "connect", "listener-a", "--host", "10.0.0.5", "--port", "5005", "--timeout", "9"); err != nil {
		t.Fatalf("forward connect failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.ForwardListenerConnect](t, h, "ConnectForwardListener")
	if req.ListenerId != "listener-a" || req.ConnectHost != "10.0.0.5" || req.ConnectPort != 5005 || req.TimeoutSeconds != 9 {
		t.Fatalf("request = %#v, want listener-a 10.0.0.5:5005 timeout=9", req)
	}
}

func TestForwardConnectCommandRequiresHost(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	if err := h.ExecuteClient("listener", "forward", "connect", "listener-a"); err == nil {
		t.Fatal("forward connect without --host succeeded")
	}
	if calls := h.Recorder.Calls(); len(calls) != 0 {
		t.Fatalf("RPC calls = %#v, want none", calls)
	}
}

func TestForwardDisconnectCommandSendsTypedRPC(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Recorder.OnForwardListenerStatus("DisconnectForwardListener", func(_ context.Context, _ any) (*clientpb.ForwardListenerStatus, error) {
		return &clientpb.ForwardListenerStatus{ListenerId: "listener-a"}, nil
	})

	if err := h.ExecuteClient("listener", "forward", "disconnect", "listener-a"); err != nil {
		t.Fatalf("forward disconnect failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.Listener](t, h, "DisconnectForwardListener")
	if req.Id != "listener-a" {
		t.Fatalf("request = %#v, want listener-a", req)
	}
}
