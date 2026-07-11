package rpc

import (
	"context"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRetireListenerPushesCtrlAndRevokesOperator(t *testing.T) {
	initForwardRPCTestDB(t)
	withIsolatedListenersAndJobs(t)
	withIsolatedPipelinesCh(t)
	seedForwardAdminOperator(t, "admin-client", "admin-retire-fp")
	seedForwardListenerOperator(t, "listener-retire", "listener-retire-fp")
	oldBroker := core.EventBroker
	core.EventBroker = core.NewBroker()
	t.Cleanup(func() {
		core.EventBroker.Stop()
		core.EventBroker = oldBroker
	})

	lns := core.NewListener("listener-retire", "10.0.0.8")
	core.Listeners.Add(lns)
	events := subscribeEventBrokerReady(t, core.EventBroker)
	defer core.EventBroker.Unsubscribe(events)
	go func() {
		ctrl := <-lns.Ctrl
		if ctrl.GetCtrl() != consts.CtrlListenerRetire {
			t.Errorf("ctrl = %q, want %q", ctrl.GetCtrl(), consts.CtrlListenerRetire)
		}
		if retire := ctrl.GetRetire(); retire.GetListenerId() != "listener-retire" || !retire.GetPurgeConfig() || retire.GetNoRevoke() {
			t.Errorf("retire = %#v, want listener-retire purge-config revoke", retire)
		}
		lns.CtrlJob.Store(ctrl.GetId(), &clientpb.JobStatus{
			ListenerId: "listener-retire",
			Ctrl:       ctrl.GetCtrl(),
			CtrlId:     ctrl.GetId(),
			Status:     consts.CtrlStatusSuccess,
		})
	}()

	ctx := contextWithIdentity(context.Background(), &PeerIdentity{Fingerprint: "admin-retire-fp"})
	reply, err := (&Server{}).RetireListener(ctx, &clientpb.ListenerRetire{
		ListenerId:     "listener-retire",
		PurgeConfig:    true,
		TimeoutSeconds: 1,
	})
	if err != nil {
		t.Fatalf("RetireListener failed: %v", err)
	}
	if reply.GetListenerId() != "listener-retire" || reply.GetActive() {
		t.Fatalf("reply = %#v, want inactive listener-retire", reply)
	}
	op, err := db.FindOperatorByName("listener-retire")
	if err != nil {
		t.Fatalf("FindOperatorByName failed: %v", err)
	}
	if !op.Revoked || op.Type != mtls.Listener {
		t.Fatalf("operator = %#v, want revoked listener operator", op)
	}
	event := waitForLifecycleEvent(t, events, consts.CtrlListenerStop)
	if event.EventType != consts.EventListener || event.Listener.GetId() != "listener-retire" || event.Listener.GetActive() {
		t.Fatalf("listener retire event = %#v, want inactive listener_stop", event)
	}
}

func TestRetireListenerRevokesBeforeSendingCtrl(t *testing.T) {
	initForwardRPCTestDB(t)
	withIsolatedListenersAndJobs(t)
	withIsolatedPipelinesCh(t)
	seedForwardAdminOperator(t, "admin-client", "admin-retire-order-fp")

	lns := core.NewListener("listener-retire-order", "10.0.0.8")
	core.Listeners.Add(lns)

	events := make(chan string, 2)
	originalRevoke := revokeRetiredListenerOperator
	revokeRetiredListenerOperator = func(name string) error {
		if name != "listener-retire-order" {
			t.Errorf("revoke name = %q, want listener-retire-order", name)
		}
		events <- "revoke"
		return nil
	}
	t.Cleanup(func() { revokeRetiredListenerOperator = originalRevoke })

	go func() {
		ctrl := <-lns.Ctrl
		if ctrl.GetCtrl() != consts.CtrlListenerRetire {
			t.Errorf("ctrl = %q, want %q", ctrl.GetCtrl(), consts.CtrlListenerRetire)
		}
		events <- "ctrl"
		lns.CtrlJob.Store(ctrl.GetId(), &clientpb.JobStatus{
			ListenerId: "listener-retire-order",
			Ctrl:       ctrl.GetCtrl(),
			CtrlId:     ctrl.GetId(),
			Status:     consts.CtrlStatusSuccess,
		})
	}()

	ctx := contextWithIdentity(context.Background(), &PeerIdentity{Fingerprint: "admin-retire-order-fp"})
	_, err := (&Server{}).RetireListener(ctx, &clientpb.ListenerRetire{
		ListenerId:     "listener-retire-order",
		TimeoutSeconds: 1,
	})
	if err != nil {
		t.Fatalf("RetireListener failed: %v", err)
	}

	first := <-events
	second := <-events
	if first != "revoke" || second != "ctrl" {
		t.Fatalf("event order = [%s %s], want [revoke ctrl]", first, second)
	}
}

func TestRetireListenerRequiresAdmin(t *testing.T) {
	initForwardRPCTestDB(t)
	seedForwardListenerOperator(t, "listener-retire-noadmin", "listener-retire-noadmin-fp")
	ctx := contextWithIdentity(context.Background(), &PeerIdentity{Fingerprint: "listener-retire-noadmin-fp"})

	_, err := (&Server{}).RetireListener(ctx, &clientpb.ListenerRetire{
		ListenerId:     "listener-retire-noadmin",
		TimeoutSeconds: uint32(time.Second / time.Second),
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("RetireListener error = %v, want PermissionDenied", err)
	}
}
