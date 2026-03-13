package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func waitEventBrokerReady(t *testing.T, broker interface{ TryPublish(core.Event) error }) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		err := broker.TryPublish(core.Event{EventType: "test", Op: "ready"})
		if err == nil {
			return
		}
		if !errors.Is(err, core.ErrEventBrokerUnavailable) {
			t.Fatalf("unexpected broker readiness error: %v", err)
		}
		select {
		case <-deadline:
			t.Fatal("broker did not become ready")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func TestGenericRequestHandlerResponsePublishesTaskError(t *testing.T) {
	oldBroker := core.EventBroker
	oldTicker := core.GlobalTicker
	defer func() {
		core.EventBroker = oldBroker
		core.GlobalTicker = oldTicker
	}()

	testTicker := core.NewTicker()
	defer testTicker.RemoveAll()
	core.GlobalTicker = testTicker

	broker := core.NewBroker()
	defer broker.Stop()
	waitEventBrokerReady(t, broker)
	sub := broker.Subscribe()
	defer broker.Unsubscribe(sub)

	req := &GenericRequest{
		Task: &core.Task{
			Id:        7,
			SessionId: "session-a",
			Type:      "exec",
			Ctx:       context.Background(),
			Cancel:    func() {},
			DoneCh:    make(chan bool),
		},
	}
	ch := make(chan *implantpb.Spite, 1)

	req.HandlerResponse(ch, types.MsgExec)
	ch <- nil

	deadline := time.After(2 * time.Second)
	for {
		select {
		case evt := <-sub:
			if evt.Op != consts.CtrlTaskError {
				continue
			}
			if evt.Task == nil || evt.Task.TaskId != req.Task.Id {
				t.Fatalf("unexpected task payload: %#v", evt.Task)
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for task error event")
		}
	}
}
